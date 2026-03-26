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
	_ resource.Resource                = &ImageResource{}
	_ resource.ResourceWithImportState = &ImageResource{}
)

type ImageResource struct{ client *client.Client }

type ImageResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	SourceInstanceID types.String `tfsdk:"source_instance_id"`
	Status           types.String `tfsdk:"status"`
	Architecture     types.String `tfsdk:"architecture"`
	MinDiskGB        types.Int64  `tfsdk:"min_disk_gb"`
}

func NewImageResource() resource.Resource { return &ImageResource{} }

func (r *ImageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (r *ImageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor custom machine image.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{Description: "Image ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":               schema.StringAttribute{Description: "Image name.", Required: true},
			"description":        schema.StringAttribute{Description: "Image description.", Optional: true},
			"source_instance_id": schema.StringAttribute{Description: "Source instance ID to create image from (must be stopped).", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"status":             schema.StringAttribute{Description: "Image status.", Computed: true},
			"architecture":       schema.StringAttribute{Description: "Architecture (x86_64, arm64).", Computed: true},
			"min_disk_gb":        schema.Int64Attribute{Description: "Minimum disk size in GB.", Computed: true},
		},
	}
}

func (r *ImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	r.client = req.ProviderData.(*client.Client)
}

func (r *ImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{
		"name":               plan.Name.ValueString(),
		"source_instance_id": plan.SourceInstanceID.ValueString(),
	}
	setOptional(body, "description", plan.Description)

	result, err := r.client.Create(ctx, "/api/v1/images", body)
	if err != nil { resp.Diagnostics.AddError("Error creating image", err.Error()); return }

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	plan.Status = types.StringValue(getString(data, "status"))
	plan.Architecture = getOptionalString(data, "architecture")
	if v, ok := data["min_disk_gb"].(float64); ok { plan.MinDiskGB = types.Int64Value(int64(v)) }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/images/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading image", err.Error()); return }
	if result == nil { resp.State.RemoveResource(ctx); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.Status = types.StringValue(getString(data, "status"))
	state.Architecture = getOptionalString(data, "architecture")
	state.Description = getOptionalString(data, "description")
	if v, ok := data["min_disk_gb"].(float64); ok { state.MinDiskGB = types.Int64Value(int64(v)) }
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{"name": plan.Name.ValueString()}
	setOptional(body, "description", plan.Description)

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/images/%s", plan.ID.ValueString()), body)
	if err != nil { resp.Diagnostics.AddError("Error updating image", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/images/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting image", err.Error())
	}
}

func (r *ImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
