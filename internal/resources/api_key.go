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
	_ resource.Resource                = &APIKeyResource{}
	_ resource.ResourceWithImportState = &APIKeyResource{}
)

type APIKeyResource struct{ client *client.Client }

type APIKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Permissions types.List   `tfsdk:"permissions"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
}

func NewAPIKeyResource() resource.Resource { return &APIKeyResource{} }

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor API key for programmatic access.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Description: "API key ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Description: "API key name.", Required: true},
			"key":         schema.StringAttribute{Description: "The API key value (only available at creation).", Computed: true, Sensitive: true},
			"permissions": schema.ListAttribute{Description: "List of permissions.", ElementType: types.StringType, Optional: true},
			"expires_at":  schema.StringAttribute{Description: "Expiration date (ISO 8601). Null = never.", Optional: true},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	r.client = req.ProviderData.(*client.Client)
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{"name": plan.Name.ValueString()}
	if !plan.Permissions.IsNull() {
		var perms []string
		plan.Permissions.ElementsAs(ctx, &perms, false)
		body["permissions"] = perms
	}
	setOptional(body, "expires_at", plan.ExpiresAt)

	result, err := r.client.Create(ctx, "/api/v1/api-keys", body)
	if err != nil { resp.Diagnostics.AddError("Error creating API key", err.Error()); return }

	data := result["data"].(map[string]interface{})
	plan.ID = types.StringValue(data["id"].(string))
	if v, ok := data["key"].(string); ok {
		plan.Key = types.StringValue(v)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/api-keys/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading API key", err.Error()); return }
	if result == nil { resp.State.RemoveResource(ctx); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.ExpiresAt = getOptionalString(data, "expires_at")
	// key is not returned after creation — preserve from state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	body := map[string]interface{}{"name": plan.Name.ValueString()}
	_, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/api-keys/%s", plan.ID.ValueString()), body)
	if err != nil { resp.Diagnostics.AddError("Error updating API key", err.Error()); return }
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/api-keys/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting API key", err.Error())
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
