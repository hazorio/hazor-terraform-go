package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &AutoScalingGroupResource{}
	_ resource.ResourceWithImportState = &AutoScalingGroupResource{}
)

type AutoScalingGroupResource struct {
	client *client.Client
}

type AutoScalingGroupResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	LaunchTemplateID types.String `tfsdk:"launch_template_id"`
	MinSize          types.Int64  `tfsdk:"min_size"`
	MaxSize          types.Int64  `tfsdk:"max_size"`
	DesiredCapacity  types.Int64  `tfsdk:"desired_capacity"`
	VPCID            types.String `tfsdk:"vpc_id"`
	SubnetIDs        types.List   `tfsdk:"subnet_ids"`
	TargetGroupID    types.String `tfsdk:"target_group_id"`
	CooldownSeconds  types.Int64  `tfsdk:"cooldown_seconds"`
	Status           types.String `tfsdk:"status"`
}

func NewAutoScalingGroupResource() resource.Resource {
	return &AutoScalingGroupResource{}
}

func (r *AutoScalingGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auto_scaling_group"
}

func (r *AutoScalingGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor Auto Scaling Group.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Description: "ASG ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":   schema.StringAttribute{Description: "ASG name.", Required: true},
			"launch_template_id": schema.StringAttribute{Description: "Launch template ID.", Required: true},
			"min_size":          schema.Int64Attribute{Description: "Minimum number of instances.", Required: true},
			"max_size":          schema.Int64Attribute{Description: "Maximum number of instances.", Required: true},
			"desired_capacity":  schema.Int64Attribute{Description: "Desired number of instances.", Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"vpc_id":            schema.StringAttribute{Description: "VPC ID.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"subnet_ids":        schema.ListAttribute{Description: "Subnet IDs for instance placement.", ElementType: types.StringType, Optional: true},
			"target_group_id":   schema.StringAttribute{Description: "Target group ID for LB integration.", Optional: true},
			"cooldown_seconds":  schema.Int64Attribute{Description: "Cooldown period in seconds.", Optional: true},
			"status":            schema.StringAttribute{Description: "Current ASG status.", Computed: true},
		},
	}
}

func (r *AutoScalingGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	r.client = req.ProviderData.(*client.Client)
}

func (r *AutoScalingGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AutoScalingGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{
		"name":               plan.Name.ValueString(),
		"launch_template_id": plan.LaunchTemplateID.ValueString(),
		"min_size":           plan.MinSize.ValueInt64(),
		"max_size":           plan.MaxSize.ValueInt64(),
		"desired_capacity":   plan.DesiredCapacity.ValueInt64(),
		"vpc_id":             plan.VPCID.ValueString(),
	}
	setOptional(body, "target_group_id", plan.TargetGroupID)
	if !plan.CooldownSeconds.IsNull() {
		body["cooldown_seconds"] = plan.CooldownSeconds.ValueInt64()
	}
	if !plan.SubnetIDs.IsNull() {
		var ids []string
		plan.SubnetIDs.ElementsAs(ctx, &ids, false)
		body["subnet_ids"] = ids
	}

	result, err := r.client.Create(ctx, "/api/v1/auto-scaling/groups", body)
	if err != nil { resp.Diagnostics.AddError("Error creating ASG", err.Error()); return }

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	plan.Status = types.StringValue(getString(data, "status"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutoScalingGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AutoScalingGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/auto-scaling/groups/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading ASG", err.Error()); return }
	if result == nil { resp.State.RemoveResource(ctx); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.Status = types.StringValue(getString(data, "status"))
	if v, ok := data["min_size"].(float64); ok { state.MinSize = types.Int64Value(int64(v)) }
	if v, ok := data["max_size"].(float64); ok { state.MaxSize = types.Int64Value(int64(v)) }
	if v, ok := data["desired_capacity"].(float64); ok { state.DesiredCapacity = types.Int64Value(int64(v)) }
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AutoScalingGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AutoScalingGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{
		"name":             plan.Name.ValueString(),
		"min_size":         plan.MinSize.ValueInt64(),
		"max_size":         plan.MaxSize.ValueInt64(),
		"desired_capacity": plan.DesiredCapacity.ValueInt64(),
	}
	setOptional(body, "target_group_id", plan.TargetGroupID)

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/auto-scaling/groups/%s", plan.ID.ValueString()), body)
	if err != nil { resp.Diagnostics.AddError("Error updating ASG", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutoScalingGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AutoScalingGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/auto-scaling/groups/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting ASG", err.Error())
	}
}

func (r *AutoScalingGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
