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
	_ resource.Resource                = &SecretResource{}
	_ resource.ResourceWithImportState = &SecretResource{}
)

type SecretResource struct{ client *client.Client }

type SecretResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
}

func NewSecretResource() resource.Resource { return &SecretResource{} }

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor secret in the Secrets Manager.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Description: "Secret ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Description: "Secret name (unique per project).", Required: true},
			"value":       schema.StringAttribute{Description: "Secret value (encrypted at rest).", Required: true, Sensitive: true},
			"description": schema.StringAttribute{Description: "Description.", Optional: true},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	r.client = req.ProviderData.(*client.Client)
}

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{"name": plan.Name.ValueString(), "value": plan.Value.ValueString()}
	setOptional(body, "description", plan.Description)

	result, err := r.client.Create(ctx, "/api/v1/secrets", body)
	if err != nil { resp.Diagnostics.AddError("Error creating secret", err.Error()); return }

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/secrets/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading secret", err.Error()); return }
	if result == nil { resp.State.RemoveResource(ctx); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.Description = getOptionalString(data, "description")
	// Value is not returned by API for security — preserve from state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{"value": plan.Value.ValueString()}
	setOptional(body, "description", plan.Description)

	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/secrets/%s", plan.ID.ValueString()), body)
	if err != nil { resp.Diagnostics.AddError("Error updating secret", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/secrets/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting secret", err.Error())
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
