package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/nrdcg/porkbun"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &porkbunDnsRecordResource{}
	_ resource.ResourceWithImportState = &porkbunDnsRecordResource{}
)

func NewPorkbunDnsRecordResource() resource.Resource {
	return &porkbunDnsRecordResource{}
}

type porkbunDnsRecordResource struct {
	client   *porkbun.Client
	provider *provider.Provider
}

// porkbunDnsRecordResourceModel describes the data model
type porkbunDnsRecordResourceModel struct {
	Id      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Content types.String `tfsdk:"content"`
	Ttl     types.String `tfsdk:"ttl"`
	Notes   types.String `tfsdk:"notes"`
	Prio    types.String `tfsdk:"prio"`
	Domain  types.String `tfsdk:"domain"`
}

func (r *porkbunDnsRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *porkbunDnsRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Porkbun DNS Record resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The subdomain for the record itself without the base domain",
				// Todo: If this is unset it shows a change all the time?
				// But it seems fine if its explicitly set to ""
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Changing should force recreation
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The base domain to to create the record on",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Optional:            false,
				Required:            false,
				MarkdownDescription: "The Porkbun ID of the Record",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Todo: Validate that its not set to less than 600
			"ttl": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ttl of the record, the minimum  is 600",
				Default:             stringdefault.StaticString("600"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of DNS Record to create",
			},
			"notes": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Notes to add to the record",
			},
			"prio": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The priority of the record",
				Default:             stringdefault.StaticString("0"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The content of the record",
			},
		},
	}
}

func (r *porkbunDnsRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*porkbun.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r porkbunDnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data porkbunDnsRecordResourceModel
	// attempts := req.Provider

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	record := porkbun.Record{
		Name:    data.Name.ValueString(),
		Type:    data.Type.ValueString(),
		Content: data.Content.ValueString(),
		TTL:     data.Ttl.ValueString(),  // Minimum is 600 according to porkbun docs
		Prio:    data.Prio.ValueString(), // Doesn't work on .com?
		Notes:   data.Notes.ValueString(),
	}

	id, err := r.client.CreateRecord(ctx, data.Domain.ValueString(), record)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DNS Record",
			fmt.Sprintf("Error: %s", err),
		)
	}

	data.Id = types.StringValue(fmt.Sprint(id))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r porkbunDnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data porkbunDnsRecordResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getRecordsResult, err := r.getRecords(ctx, data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(
				`Could not retrieve records for %s.`,
				data.Domain.ValueString(),
			),
			fmt.Sprintf("Error: %s", err.Error()),
		)
	}

	tflog.Info(ctx, fmt.Sprintf("Found records: %s", getRecordsResult))
	for _, record := range getRecordsResult {
		tflog.Info(ctx, fmt.Sprintf("This record is: %s", record.ID))
		if record.ID == data.Id.ValueString() {
			// This is to handle if there's no subdomain
			if data.Domain.ValueString() == record.Name {
				data.Name = types.StringValue("")
			} else {
				// The API returns the full record as the name so we'll strip off the domain at the end to keep it consistent
				data.Name = types.StringValue(strings.ReplaceAll(record.Name, fmt.Sprintf(".%s", data.Domain.ValueString()), ""))
			}

		}
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r porkbunDnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data porkbunDnsRecordResourceModel
	var recordId string

	diags := req.Plan.Get(ctx, &data)
	req.State.GetAttribute(ctx, path.Root("id"), &recordId)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	record := porkbun.Record{
		Name:    data.Name.ValueString(),
		Type:    data.Type.ValueString(),
		Content: data.Content.ValueString(),
		TTL:     data.Ttl.ValueString(),  // Minimum is 600 according to porkbun docs
		Prio:    data.Prio.ValueString(), // Doesn't work on .com?
		Notes:   data.Notes.ValueString(),
	}

	intId, err := strconv.Atoi(recordId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting ID to a string",
			fmt.Sprintf("Error: %s", err),
		)
	}

	err = r.client.EditRecord(ctx, data.Domain.ValueString(), intId, record)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating the record",
			fmt.Sprintf("Error %s", err),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.State.SetAttribute(ctx, path.Root("id"), &recordId)
	resp.Diagnostics.Append(diags...)
}

func (r porkbunDnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state porkbunDnsRecordResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	intId, err := strconv.Atoi(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting ID to a string",
			fmt.Sprintf("Error: %s", err),
		)
	}

	err = r.client.DeleteRecord(ctx, state.Domain.ValueString(), intId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting record",
			fmt.Sprintf("Error: %s", err),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r porkbunDnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r porkbunDnsRecordResource) getRecords(ctx context.Context, domain string) ([]porkbun.Record, error) {
	records, err := r.client.RetrieveRecords(ctx, domain)
	if err != nil {
		return []porkbun.Record{}, err
	}
	return records, nil
}
