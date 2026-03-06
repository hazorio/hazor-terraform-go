package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &DNSRecordResource{}
	_ resource.ResourceWithImportState = &DNSRecordResource{}
)

type DNSRecordResource struct {
	client *client.Client
}

type DNSRecordResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ZoneID     types.String `tfsdk:"zone_id"`
	Name       types.String `tfsdk:"name"`
	RecordType types.String `tfsdk:"record_type"`
	Value      types.String `tfsdk:"value"`
	TTL        types.Int64  `tfsdk:"ttl"`
}

func NewDNSRecordResource() resource.Resource {
	return &DNSRecordResource{}
}

func (r *DNSRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *DNSRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor DNS record.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the DNS record.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				Description: "The DNS zone ID this record belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The record name (e.g., www, @, mail).",
				Required:    true,
			},
			"record_type": schema.StringAttribute{
				Description: "The record type (A, AAAA, CNAME, MX, TXT, etc.).",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The record value.",
				Required:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "Time to live in seconds. Defaults to 300.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
			},
		},
	}
}

func (r *DNSRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *client.Client")
		return
	}
	r.client = c
}

func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DNSRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"zone_id":     plan.ZoneID.ValueString(),
		"name":        plan.Name.ValueString(),
		"record_type": plan.RecordType.ValueString(),
		"value":       plan.Value.ValueString(),
		"ttl":         plan.TTL.ValueInt64(),
	}

	result, err := r.client.Create(ctx, "/api/v1/dns/records", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS record", err.Error())
		return
	}

	plan.ID = types.StringValue(safeString(result["id"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DNSRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/dns/records/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading DNS record", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ZoneID = types.StringValue(safeString(result["zone_id"]))
	state.Name = types.StringValue(safeString(result["name"]))
	state.RecordType = types.StringValue(safeString(result["record_type"]))
	state.Value = types.StringValue(safeString(result["value"]))
	state.TTL = types.Int64Value(int64(safeFloat64(result["ttl"])))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DNSRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DNSRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"record_type": plan.RecordType.ValueString(),
		"value":       plan.Value.ValueString(),
		"ttl":         plan.TTL.ValueInt64(),
	}

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/dns/records/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating DNS record", err.Error())
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DNSRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/dns/records/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting DNS record", err.Error())
		return
	}
}

func (r *DNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
