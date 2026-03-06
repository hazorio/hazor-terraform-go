package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &NoSQLInstanceResource{}
	_ resource.ResourceWithImportState = &NoSQLInstanceResource{}
)

type NoSQLInstanceResource struct {
	client *client.Client
}

type NoSQLInstanceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Engine    types.String `tfsdk:"engine"`
	VCPUs     types.Int64  `tfsdk:"vcpus"`
	MemoryMB  types.Int64  `tfsdk:"memory_mb"`
	StorageGB types.Int64  `tfsdk:"storage_gb"`
	VPCID     types.String `tfsdk:"vpc_id"`
	SubnetID  types.String `tfsdk:"subnet_id"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewNoSQLInstanceResource() resource.Resource {
	return &NoSQLInstanceResource{}
}

func (r *NoSQLInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nosql_instance"
}

func (r *NoSQLInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor NoSQL database instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the NoSQL instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the NoSQL instance.",
				Required:    true,
			},
			"engine": schema.StringAttribute{
				Description: "The NoSQL engine (e.g., mongodb, cassandra, scylladb).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("scylladb"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vcpus": schema.Int64Attribute{
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
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the NoSQL instance.",
				Optional:    true,
			},
			"subnet_id": schema.StringAttribute{
				Description: "The subnet ID for the NoSQL instance.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the NoSQL instance.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the NoSQL instance was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the NoSQL instance was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *NoSQLInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NoSQLInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NoSQLInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"engine": plan.Engine.ValueString(),
	}
	if !plan.VCPUs.IsNull() {
		body["vcpus"] = plan.VCPUs.ValueInt64()
	}
	if !plan.MemoryMB.IsNull() {
		body["memory_mb"] = plan.MemoryMB.ValueInt64()
	}
	if !plan.StorageGB.IsNull() {
		body["storage_gb"] = plan.StorageGB.ValueInt64()
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}
	if !plan.SubnetID.IsNull() {
		body["subnet_id"] = plan.SubnetID.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/nosql/instances", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating NoSQL instance", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/nosql/instances/%s", id), "running", 15*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for NoSQL instance", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NoSQLInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NoSQLInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/nosql/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading NoSQL instance", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.Engine = types.StringValue(safeString(result["engine"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	if v := safeFloat64(result["vcpus"]); v != 0 {
		state.VCPUs = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["memory_mb"]); v != 0 {
		state.MemoryMB = types.Int64Value(int64(v))
	}
	if v := safeFloat64(result["storage_gb"]); v != 0 {
		state.StorageGB = types.Int64Value(int64(v))
	}
	if v, ok := result["vpc_id"].(string); ok && v != "" {
		state.VPCID = types.StringValue(v)
	}
	if v, ok := result["subnet_id"].(string); ok && v != "" {
		state.SubnetID = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NoSQLInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NoSQLInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NoSQLInstanceResourceModel
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
	if !plan.StorageGB.IsNull() {
		body["storage_gb"] = plan.StorageGB.ValueInt64()
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}
	if !plan.SubnetID.IsNull() {
		body["subnet_id"] = plan.SubnetID.ValueString()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/nosql/instances/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating NoSQL instance", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NoSQLInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NoSQLInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/nosql/instances/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting NoSQL instance", err.Error())
		return
	}
}

func (r *NoSQLInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
