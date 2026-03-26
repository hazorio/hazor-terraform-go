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
	_ resource.Resource                = &LaunchTemplateResource{}
	_ resource.ResourceWithImportState = &LaunchTemplateResource{}
)

type LaunchTemplateResource struct {
	client *client.Client
}

type LaunchTemplateResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	InstanceType   types.String `tfsdk:"instance_type"`
	ImageID        types.String `tfsdk:"image_id"`
	VPCID          types.String `tfsdk:"vpc_id"`
	SubnetID       types.String `tfsdk:"subnet_id"`
	KeyPairID      types.String `tfsdk:"key_pair_id"`
	UserData       types.String `tfsdk:"user_data"`
}

func NewLaunchTemplateResource() resource.Resource {
	return &LaunchTemplateResource{}
}

func (r *LaunchTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_launch_template"
}

func (r *LaunchTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor launch template for instances and auto scaling groups.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Description: "Template ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":            schema.StringAttribute{Description: "Template name.", Required: true},
			"description":     schema.StringAttribute{Description: "Template description.", Optional: true},
			"instance_type":   schema.StringAttribute{Description: "Instance type (e.g. oc.small).", Required: true},
			"image_id":        schema.StringAttribute{Description: "Image ID.", Required: true},
			"vpc_id":          schema.StringAttribute{Description: "VPC ID.", Optional: true},
			"subnet_id":       schema.StringAttribute{Description: "Subnet ID.", Optional: true},
			"key_pair_id":     schema.StringAttribute{Description: "Key pair ID.", Optional: true},
			"user_data":       schema.StringAttribute{Description: "User data script (base64 or plain).", Optional: true},
		},
	}
}

func (r *LaunchTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	r.client = req.ProviderData.(*client.Client)
}

func (r *LaunchTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LaunchTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
		"instance_type": plan.InstanceType.ValueString(),
		"image_id": plan.ImageID.ValueString(),
	}
	setOptional(body, "description", plan.Description)
	setOptional(body, "vpc_id", plan.VPCID)
	setOptional(body, "subnet_id", plan.SubnetID)
	setOptional(body, "key_pair_id", plan.KeyPairID)
	setOptional(body, "user_data", plan.UserData)

	result, err := r.client.Create(ctx, "/api/v1/launch-templates", body)
	if err != nil { resp.Diagnostics.AddError("Error creating launch template", err.Error()); return }

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LaunchTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LaunchTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/launch-templates/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading launch template", err.Error()); return }
	if result == nil { resp.State.RemoveResource(ctx); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.InstanceType = types.StringValue(getString(data, "instance_type"))
	state.ImageID = types.StringValue(getString(data, "image_id"))
	state.VPCID = getOptionalString(data, "vpc_id")
	state.SubnetID = getOptionalString(data, "subnet_id")
	state.KeyPairID = getOptionalString(data, "key_pair_id")
	state.Description = getOptionalString(data, "description")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LaunchTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LaunchTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
		"instance_type": plan.InstanceType.ValueString(),
		"image_id": plan.ImageID.ValueString(),
	}
	setOptional(body, "description", plan.Description)
	setOptional(body, "vpc_id", plan.VPCID)
	setOptional(body, "subnet_id", plan.SubnetID)
	setOptional(body, "key_pair_id", plan.KeyPairID)
	setOptional(body, "user_data", plan.UserData)

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/launch-templates/%s", plan.ID.ValueString()), body)
	if err != nil { resp.Diagnostics.AddError("Error updating launch template", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LaunchTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LaunchTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/launch-templates/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting launch template", err.Error())
	}
}

func (r *LaunchTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
