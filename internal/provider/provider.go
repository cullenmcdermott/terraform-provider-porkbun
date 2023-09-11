package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
type PorkbunProviderModel struct {
	ApiKey     types.String `tfsdk:"api_key"`
	SecretKey  types.String `tfsdk:"secret_key"`
	BaseUrl    types.String `tfsdk:"base_url"`
	MaxRetries types.Int64  `tfsdk:"max_retries"`
}

func (p *porkbunProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "porkbun"
	resp.Version = p.version
}

func (p *porkbunProvider) Datasources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *porkbunProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PorkbunProviderModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var apiKey string
	if data.ApiKey.IsUnknown() {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as api_key",
		)
		return
	}

	apiKey = data.ApiKey.ValueString()

	if data.ApiKey.IsNull() {
		apiKey = os.Getenv("PORKBUN_API_KEY")
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
	if data.SecretKey.IsUnknown() {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as secret_key",
		)
		return
	}

	secretKey = data.SecretKey.ValueString()

	if data.SecretKey.IsNull() {
		secretKey = os.Getenv("PORKBUN_SECRET_KEY")
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

	if data.MaxRetries.IsNull() {
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
		p.MaxRetries = int(data.MaxRetries.ValueInt64())
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = p.MaxRetries
	c.HTTPClient = retryClient.StandardClient()

	p.configured = true
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *porkbunProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPorkbunDnsRecordResource,
	}
}

func (p *porkbunProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *porkbunProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API Key for Porkbun",
				Required:            false,
				Optional:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret Key for Porkbun",
				Required:            false,
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Override Porkbun Base URL",
				Required:            false,
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Should only be changed if needing to work around Porkbun API rate limits",
				Required:            false,
				Optional:            true,
			},
		},
	}
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
