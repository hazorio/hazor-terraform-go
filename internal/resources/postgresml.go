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
	_ resource.Resource                = &PostgresMLInstanceResource{}
	_ resource.ResourceWithImportState = &PostgresMLInstanceResource{}
)

type PostgresMLInstanceResource struct {
	client *client.Client
}

type PostgresMLInstanceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	PlanName  types.String `tfsdk:"plan_name"`
	VCPUCount types.Int64  `tfsdk:"vcpu_count"`
	MemoryMB  types.Int64  `tfsdk:"memory_mb"`
	StorageGB types.Int64  `tfsdk:"storage_gb"`
	Region    types.String `tfsdk:"region"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewPostgresMLInstanceResource() resource.Resource {
	return &PostgresMLInstanceResource{}
}

func (r *PostgresMLInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresml_instance"
}

func (r *PostgresMLInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor PostgresML instance with integrated machine learning.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the PostgresML instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the PostgresML instance.",
				Required:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: "The plan name (e.g., starter, professional, enterprise).",
				Optional:    true,
			},
			"vcpu_count": schema.Int64Attribute{
				Description: "The number of vCPUs.",
				Optional:    true,
			},
			"memory_mb": schema.Int64Attribute{
				Description: "The amount of memory in megabytes.",
				Optional:    true,
			},
			"storage_gb": schema.Int64Attribute{
				Description: "The amount of storage in gigabytes.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region for the PostgresML instance.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the PostgresML instance.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the PostgresML instance was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the PostgresML instance was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *PostgresMLInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PostgresMLInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PostgresMLInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.PlanName.IsNull() {
		body["plan_name"] = plan.PlanName.ValueString()
	}
	if !plan.VCPUCount.IsNull() {
		body["vcpu_count"] = plan.VCPUCount.ValueInt64()
	}
	if !plan.MemoryMB.IsNull() {
		body["memory_mb"] = plan.MemoryMB.ValueInt64()
	}
	if !plan.StorageGB.IsNull() {
		body["storage_gb"] = plan.StorageGB.ValueInt64()
	}
	if !plan.Region.IsNull() {
		body["region"] = plan.Region.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/postgresml/instances", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PostgresML instance", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/postgresml/instances/%s", id), "running", 15*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for PostgresML instance", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PostgresMLInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PostgresMLInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/postgresml/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgresML instance", err.Error())
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

	if v, ok := result["plan_name"].(string); ok && v != "" {
		state.PlanName = types.StringValue(v)
	}
	if v := safeFloat64(result["vcpu_count"]); v != 0 {
		state.VCPUCount = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["memory_mb"]); v != 0 {
		state.MemoryMB = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["storage_gb"]); v != 0 {
		state.StorageGB = types.Int64Value(int64(v))
	}
	if v, ok := result["region"].(string); ok && v != "" {
		state.Region = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PostgresMLInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PostgresMLInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PostgresMLInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.PlanName.IsNull() {
		body["plan_name"] = plan.PlanName.ValueString()
	}
	if !plan.VCPUCount.IsNull() {
		body["vcpu_count"] = plan.VCPUCount.ValueInt64()
	}
	if !plan.MemoryMB.IsNull() {
		body["memory_mb"] = plan.MemoryMB.ValueInt64()
	}
	if !plan.StorageGB.IsNull() {
		body["storage_gb"] = plan.StorageGB.ValueInt64()
	}
	if !plan.Region.IsNull() {
		body["region"] = plan.Region.ValueString()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/postgresml/instances/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating PostgresML instance", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PostgresMLInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PostgresMLInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/postgresml/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting PostgresML instance", err.Error())
		return
	}
}

func (r *PostgresMLInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
