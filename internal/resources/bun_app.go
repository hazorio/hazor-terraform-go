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
	_ resource.Resource                = &BunAppResource{}
	_ resource.ResourceWithImportState = &BunAppResource{}
)

type BunAppResource struct {
	client *client.Client
}

type BunAppResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VCPUs     types.Int64  `tfsdk:"vcpus"`
	MemoryMB  types.Int64  `tfsdk:"memory_mb"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewBunAppResource() resource.Resource {
	return &BunAppResource{}
}

func (r *BunAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bun_app"
}

func (r *BunAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor Bun application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Bun application.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Bun application.",
				Required:    true,
			},
			"vcpus": schema.Int64Attribute{
				Description: "The number of vCPUs allocated to the application.",
				Optional:    true,
			},
			"memory_mb": schema.Int64Attribute{
				Description: "The amount of memory in megabytes allocated to the application.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the Bun application.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the Bun application was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the Bun application was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *BunAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BunAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.VCPUs.IsNull() {
		body["vcpus"] = plan.VCPUs.ValueInt64()
	}
	if !plan.MemoryMB.IsNull() {
		body["memory_mb"] = plan.MemoryMB.ValueInt64()
	}

	result, err := r.client.Create(ctx, "/api/v1/bun-apps", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Bun application", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/bun-apps/%s", id), "running", 10*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Bun application", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BunAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/bun-apps/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bun application", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	if v := safeFloat64(result["vcpus"]); v != 0 {
		state.VCPUs = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["memory_mb"]); v != 0 {
		state.MemoryMB = types.Int64Value(int64(v))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BunAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state BunAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.VCPUs.IsNull() {
		body["vcpus"] = plan.VCPUs.ValueInt64()
	}
	if !plan.MemoryMB.IsNull() {
		body["memory_mb"] = plan.MemoryMB.ValueInt64()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/bun-apps/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Bun application", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BunAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/bun-apps/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Bun application", err.Error())
		return
	}
}

func (r *BunAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
