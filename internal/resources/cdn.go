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
	_ resource.Resource                = &CDNDistributionResource{}
	_ resource.ResourceWithImportState = &CDNDistributionResource{}
)

type CDNDistributionResource struct {
	client *client.Client
}

type CDNDistributionResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Domain    types.String `tfsdk:"domain"`
	OriginURL types.String `tfsdk:"origin_url"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewCDNDistributionResource() resource.Resource {
	return &CDNDistributionResource{}
}

func (r *CDNDistributionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_distribution"
}

func (r *CDNDistributionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor CDN distribution.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the CDN distribution.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "The domain assigned to the CDN distribution.",
				Computed:    true,
			},
			"origin_url": schema.StringAttribute{
				Description: "The origin URL to pull content from.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the CDN distribution.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the CDN distribution was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the CDN distribution was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *CDNDistributionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CDNDistributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CDNDistributionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"origin_url": plan.OriginURL.ValueString(),
	}

	result, err := r.client.Create(ctx, "/api/v1/cdn/distributions", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CDN distribution", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/cdn/distributions/%s", id), "active", 10*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for CDN distribution", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Domain = types.StringValue(safeString(final["domain"]))
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CDNDistributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CDNDistributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/cdn/distributions/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading CDN distribution", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Domain = types.StringValue(safeString(result["domain"]))
	state.OriginURL = types.StringValue(safeString(result["origin_url"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CDNDistributionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CDNDistributionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CDNDistributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"origin_url": plan.OriginURL.ValueString(),
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/cdn/distributions/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating CDN distribution", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Domain = types.StringValue(safeString(result["domain"]))
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CDNDistributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CDNDistributionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/cdn/distributions/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting CDN distribution", err.Error())
		return
	}
}

func (r *CDNDistributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
