package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nrdcg/porkbun"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.Provider = &porkbunProvider{}

type porkbunProvider struct {
	client     *porkbun.Client
	configured bool
	version    string
	MaxRetries int
}

// providerData can be used to store data from the Terraform configuration.
type providerData struct {
	ApiKey     types.String `tfsdk:"api_key"`
	SecretKey  types.String `tfsdk:"secret_key"`
	BaseUrl    types.String `tfsdk:"base_url"`
	MaxRetries types.Int64  `tfsdk:"max_retries"`
}

func (p *porkbunProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var apiKey string
	if data.ApiKey.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as api_key",
		)
		return
	}

	if data.ApiKey.Null {
		apiKey = os.Getenv("PORKBUN_API_KEY")
	} else {
		apiKey = data.ApiKey.Value
	}

	if apiKey == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find api_key",
			"api_key cannot be an empty string",
		)
		return
	}

	var secretKey string
	if data.SecretKey.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as secret_key",
		)
		return
	}

	if data.SecretKey.Null {
		secretKey = os.Getenv("PORKBUN_SECRET_KEY")
	} else {
		secretKey = data.SecretKey.Value
	}

	if secretKey == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find secret_key",
			"secret_key cannot be an empty string",
		)
		return
	}

	c := porkbun.New(secretKey, apiKey)

	if baseUrl, ok := os.LookupEnv("PORKBUN_BASE_URL"); ok {
		c.BaseURL, _ = url.Parse(baseUrl)
	}

	if data.MaxRetries.Null {
		if mr, ok := os.LookupEnv("PORKBUN_MAX_RETRIES"); ok {
			mri, err := strconv.Atoi(mr)
			if err != nil {
				resp.Diagnostics.AddError(
					"failed converting max retries",
					err.Error(),
				)
			}
			p.MaxRetries = mri
		} else {
			p.MaxRetries = 10
		}
	} else {
		p.MaxRetries = int(data.MaxRetries.Value)
	}

	p.client = c
	p.configured = true
}

func (p *porkbunProvider) GetResources(ctx context.Context) (map[string]provider.ResourceType, diag.Diagnostics) {
	return map[string]provider.ResourceType{
		"porkbun_dns_record": porkbunDnsRecordResourceType{},
	}, nil
}

func (p *porkbunProvider) GetDataSources(ctx context.Context) (map[string]provider.DataSourceType, diag.Diagnostics) {
	return map[string]provider.DataSourceType{}, nil
}

func (p *porkbunProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"api_key": {
				MarkdownDescription: "API Key for Porkbun",
				Required:            false,
				Optional:            true,
				Type:                types.StringType,
			},
			"secret_key": {
				MarkdownDescription: "Secret Key for Porkbun",
				Required:            false,
				Optional:            true,
				Type:                types.StringType,
			},
			"base_url": {
				MarkdownDescription: "Override Porkbun Base URL",
				Required:            false,
				Optional:            true,
				Type:                types.StringType,
			},
			"max_retries": {
				MarkdownDescription: "Should only be changed if needing to work around Porkbun API rate limits",
				Required:            false,
				Optional:            true,
				Type:                types.Int64Type,
			},
		},
	}, nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &porkbunProvider{
			version: version,
		}
	}
}

func convertProviderType(in provider.Provider) (porkbunProvider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*porkbunProvider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return porkbunProvider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return porkbunProvider{}, diags
	}

	return *p, diags
}
