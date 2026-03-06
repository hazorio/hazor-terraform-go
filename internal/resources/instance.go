package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

type InstanceResource struct {
	client *client.Client
}

type InstanceResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	InstanceType types.String `tfsdk:"instance_type"`
	ImageID      types.String `tfsdk:"image_id"`
	VPCID        types.String `tfsdk:"vpc_id"`
	SubnetID     types.String `tfsdk:"subnet_id"`
	Status       types.String `tfsdk:"status"`
	KeyPairID    types.String `tfsdk:"key_pair_id"`
	UserData     types.String `tfsdk:"user_data"`
	PublicIP     types.String `tfsdk:"public_ip"`
	PrivateIP    types.String `tfsdk:"private_ip"`
}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

func (r *InstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor compute instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the instance.",
				Required:    true,
			},
			"instance_type": schema.StringAttribute{
				Description: "The instance type (e.g., hz.small, hz.medium, hz.large).",
				Required:    true,
			},
			"image_id": schema.StringAttribute{
				Description: "The ID of the image to launch the instance from.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID to launch the instance in.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Description: "The subnet ID to launch the instance in.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the instance.",
				Computed:    true,
			},
			"key_pair_id": schema.StringAttribute{
				Description: "The key pair ID for SSH access.",
				Optional:    true,
			},
			"user_data": schema.StringAttribute{
				Description: "User data script to run on instance launch.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP address assigned to the instance.",
				Computed:    true,
			},
			"private_ip": schema.StringAttribute{
				Description: "The private IP address assigned to the instance.",
				Computed:    true,
			},
		},
	}
}

func (r *InstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"instance_type": plan.InstanceType.ValueString(),
		"image_id":      plan.ImageID.ValueString(),
		"vpc_id":        plan.VPCID.ValueString(),
		"subnet_id":     plan.SubnetID.ValueString(),
	}
	if !plan.KeyPairID.IsNull() {
		body["key_pair_id"] = plan.KeyPairID.ValueString()
	}
	if !plan.UserData.IsNull() {
		body["user_data"] = plan.UserData.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/instances", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating instance", err.Error())
		return
	}

	id := result["id"].(string)

	// Wait for the instance to become active
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/instances/%s", id), "active", 10*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for instance to become active", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.PublicIP = types.StringValue(safeString(final["public_ip"]))
	plan.PrivateIP = types.StringValue(safeString(final["private_ip"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.InstanceType = types.StringValue(safeString(result["instance_type"]))
	state.ImageID = types.StringValue(safeString(result["image_id"]))
	state.VPCID = types.StringValue(safeString(result["vpc_id"]))
	state.SubnetID = types.StringValue(safeString(result["subnet_id"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.PublicIP = types.StringValue(safeString(result["public_ip"]))
	state.PrivateIP = types.StringValue(safeString(result["private_ip"]))

	if v, ok := result["key_pair_id"].(string); ok && v != "" {
		state.KeyPairID = types.StringValue(v)
	}
	if v, ok := result["user_data"].(string); ok && v != "" {
		state.UserData = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"instance_type": plan.InstanceType.ValueString(),
	}
	if !plan.KeyPairID.IsNull() {
		body["key_pair_id"] = plan.KeyPairID.ValueString()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/instances/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating instance", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.PublicIP = types.StringValue(safeString(result["public_ip"]))
	plan.PrivateIP = types.StringValue(safeString(result["private_ip"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting instance", err.Error())
		return
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// safeString safely extracts a string from an interface{}.
func safeString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// safeFloat64 safely extracts a float64 from an interface{}.
func safeFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
