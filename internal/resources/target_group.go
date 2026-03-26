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
	_ resource.Resource                = &TargetGroupResource{}
	_ resource.ResourceWithImportState = &TargetGroupResource{}
)

type TargetGroupResource struct {
	client *client.Client
}

type TargetGroupResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Protocol        types.String `tfsdk:"protocol"`
	Port            types.Int64  `tfsdk:"port"`
	VPCID           types.String `tfsdk:"vpc_id"`
	HealthCheckPath types.String `tfsdk:"health_check_path"`
	HealthCheckPort types.Int64  `tfsdk:"health_check_port"`
	Algorithm       types.String `tfsdk:"algorithm"`
}

func NewTargetGroupResource() resource.Resource {
	return &TargetGroupResource{}
}

func (r *TargetGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target_group"
}

func (r *TargetGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor target group for load balancer routing.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the target group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the target group.",
				Required:    true,
			},
			"protocol": schema.StringAttribute{
				Description: "The protocol (HTTP, HTTPS, TCP).",
				Optional:    true,
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port the target group listens on.",
				Required:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"health_check_path": schema.StringAttribute{
				Description: "Health check path (e.g. /health).",
				Optional:    true,
			},
			"health_check_port": schema.Int64Attribute{
				Description: "Health check port.",
				Optional:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Load balancing algorithm (round_robin, least_connections, ip_hash).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *TargetGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *TargetGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TargetGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"port":    plan.Port.ValueInt64(),
		"vpc_id":  plan.VPCID.ValueString(),
	}
	if !plan.Protocol.IsNull() {
		body["protocol"] = plan.Protocol.ValueString()
	}
	if !plan.HealthCheckPath.IsNull() {
		body["health_check_path"] = plan.HealthCheckPath.ValueString()
	}
	if !plan.HealthCheckPort.IsNull() {
		body["health_check_port"] = plan.HealthCheckPort.ValueInt64()
	}
	if !plan.Algorithm.IsNull() {
		body["algorithm"] = plan.Algorithm.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/target-groups", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating target group", err.Error())
		return
	}

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	if v, ok := data["protocol"].(string); ok {
		plan.Protocol = types.StringValue(v)
	}
	if v, ok := data["algorithm"].(string); ok {
		plan.Algorithm = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TargetGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TargetGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/target-groups/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading target group", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(data["name"].(string))
	if v, ok := data["protocol"].(string); ok {
		state.Protocol = types.StringValue(v)
	}
	if v, ok := data["port"].(float64); ok {
		state.Port = types.Int64Value(int64(v))
	}
	if v, ok := data["vpc_id"].(string); ok {
		state.VPCID = types.StringValue(v)
	}
	if v, ok := data["algorithm"].(string); ok {
		state.Algorithm = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TargetGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TargetGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
		"port": plan.Port.ValueInt64(),
	}
	if !plan.Protocol.IsNull() {
		body["protocol"] = plan.Protocol.ValueString()
	}
	if !plan.HealthCheckPath.IsNull() {
		body["health_check_path"] = plan.HealthCheckPath.ValueString()
	}
	if !plan.Algorithm.IsNull() {
		body["algorithm"] = plan.Algorithm.ValueString()
	}

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/target-groups/%s", plan.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating target group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TargetGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TargetGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/target-groups/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting target group", err.Error())
	}
}

func (r *TargetGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
