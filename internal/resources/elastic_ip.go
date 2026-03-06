package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &ElasticIPResource{}
	_ resource.ResourceWithImportState = &ElasticIPResource{}
)

type ElasticIPResource struct {
	client *client.Client
}

type ElasticIPResourceModel struct {
	ID         types.String `tfsdk:"id"`
	IPAddress  types.String `tfsdk:"ip_address"`
	Region     types.String `tfsdk:"region"`
	Status     types.String `tfsdk:"status"`
	InstanceID types.String `tfsdk:"instance_id"`
}

func NewElasticIPResource() resource.Resource {
	return &ElasticIPResource{}
}

func (r *ElasticIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_elastic_ip"
}

func (r *ElasticIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor Elastic IP address.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the elastic IP.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				Description: "The allocated IP address.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region for the elastic IP.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the elastic IP.",
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: "The instance ID the elastic IP is associated with.",
				Optional:    true,
			},
		},
	}
}

func (r *ElasticIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ElasticIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ElasticIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"region": plan.Region.ValueString(),
	}
	if !plan.InstanceID.IsNull() {
		body["instance_id"] = plan.InstanceID.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/elastic-ips", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating elastic IP", err.Error())
		return
	}

	plan.ID = types.StringValue(safeString(result["id"]))
	plan.IPAddress = types.StringValue(safeString(result["ip_address"]))
	plan.Status = types.StringValue(safeString(result["status"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ElasticIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ElasticIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/elastic-ips/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading elastic IP", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.IPAddress = types.StringValue(safeString(result["ip_address"]))
	state.Region = types.StringValue(safeString(result["region"]))
	state.Status = types.StringValue(safeString(result["status"]))
	if v, ok := result["instance_id"].(string); ok && v != "" {
		state.InstanceID = types.StringValue(v)
	} else {
		state.InstanceID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ElasticIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ElasticIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ElasticIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}
	if !plan.InstanceID.IsNull() {
		body["instance_id"] = plan.InstanceID.ValueString()
	} else {
		body["instance_id"] = nil
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/elastic-ips/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating elastic IP", err.Error())
		return
	}

	plan.ID = state.ID
	plan.IPAddress = types.StringValue(safeString(result["ip_address"]))
	plan.Status = types.StringValue(safeString(result["status"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ElasticIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ElasticIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/elastic-ips/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting elastic IP", err.Error())
		return
	}
}

func (r *ElasticIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
