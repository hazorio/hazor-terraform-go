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
	_ resource.Resource                = &SupabaseInstanceResource{}
	_ resource.ResourceWithImportState = &SupabaseInstanceResource{}
)

type SupabaseInstanceResource struct {
	client *client.Client
}

type SupabaseInstanceResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	PlanVCPU      types.Int64  `tfsdk:"plan_vcpu"`
	PlanMemoryMB  types.Int64  `tfsdk:"plan_memory_mb"`
	PlanStorageGB types.Int64  `tfsdk:"plan_storage_gb"`
	Region        types.String `tfsdk:"region"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func NewSupabaseInstanceResource() resource.Resource {
	return &SupabaseInstanceResource{}
}

func (r *SupabaseInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_supabase_instance"
}

func (r *SupabaseInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor Supabase instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Supabase instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Supabase instance.",
				Required:    true,
			},
			"plan_vcpu": schema.Int64Attribute{
				Description: "The number of vCPUs for the plan.",
				Optional:    true,
			},
			"plan_memory_mb": schema.Int64Attribute{
				Description: "The amount of memory in megabytes for the plan.",
				Optional:    true,
			},
			"plan_storage_gb": schema.Int64Attribute{
				Description: "The amount of storage in gigabytes for the plan.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region for the Supabase instance.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the Supabase instance.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the Supabase instance was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the Supabase instance was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *SupabaseInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SupabaseInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SupabaseInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.PlanVCPU.IsNull() {
		body["plan_vcpu"] = plan.PlanVCPU.ValueInt64()
	}
	if !plan.PlanMemoryMB.IsNull() {
		body["plan_memory_mb"] = plan.PlanMemoryMB.ValueInt64()
	}
	if !plan.PlanStorageGB.IsNull() {
		body["plan_storage_gb"] = plan.PlanStorageGB.ValueInt64()
	}
	if !plan.Region.IsNull() {
		body["region"] = plan.Region.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/supabase/instances", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Supabase instance", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/supabase/instances/%s", id), "running", 15*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Supabase instance", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SupabaseInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SupabaseInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/supabase/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Supabase instance", err.Error())
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

	if v := safeFloat64(result["plan_vcpu"]); v != 0 {
		state.PlanVCPU = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["plan_memory_mb"]); v != 0 {
		state.PlanMemoryMB = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["plan_storage_gb"]); v != 0 {
		state.PlanStorageGB = types.Int64Value(int64(v))
	}
	if v, ok := result["region"].(string); ok && v != "" {
		state.Region = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SupabaseInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SupabaseInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SupabaseInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.PlanVCPU.IsNull() {
		body["plan_vcpu"] = plan.PlanVCPU.ValueInt64()
	}
	if !plan.PlanMemoryMB.IsNull() {
		body["plan_memory_mb"] = plan.PlanMemoryMB.ValueInt64()
	}
	if !plan.PlanStorageGB.IsNull() {
		body["plan_storage_gb"] = plan.PlanStorageGB.ValueInt64()
	}
	if !plan.Region.IsNull() {
		body["region"] = plan.Region.ValueString()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/supabase/instances/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Supabase instance", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SupabaseInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SupabaseInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/supabase/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Supabase instance", err.Error())
		return
	}
}

func (r *SupabaseInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
