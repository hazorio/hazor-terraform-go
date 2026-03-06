package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &FunctionResource{}
	_ resource.ResourceWithImportState = &FunctionResource{}
)

type FunctionResource struct {
	client *client.Client
}

type FunctionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Runtime     types.String `tfsdk:"runtime"`
	Handler     types.String `tfsdk:"handler"`
	MemoryMB    types.Int64  `tfsdk:"memory_mb"`
	VPCID       types.String `tfsdk:"vpc_id"`
	SubnetID    types.String `tfsdk:"subnet_id"`
	Environment types.Map    `tfsdk:"environment"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewFunctionResource() resource.Resource {
	return &FunctionResource{}
}

func (r *FunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *FunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor serverless function.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the function.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the function.",
				Required:    true,
			},
			"runtime": schema.StringAttribute{
				Description: "The function runtime (e.g., nodejs20, python3.12, go1.22).",
				Required:    true,
			},
			"handler": schema.StringAttribute{
				Description: "The function handler (e.g., index.handler).",
				Required:    true,
			},
			"memory_mb": schema.Int64Attribute{
				Description: "The amount of memory in megabytes allocated to the function.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(128),
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the function.",
				Optional:    true,
			},
			"subnet_id": schema.StringAttribute{
				Description: "The subnet ID for the function.",
				Optional:    true,
			},
			"environment": schema.MapAttribute{
				Description: "Environment variables for the function.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the function.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the function was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the function was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *FunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":      plan.Name.ValueString(),
		"runtime":   plan.Runtime.ValueString(),
		"handler":   plan.Handler.ValueString(),
		"memory_mb": plan.MemoryMB.ValueInt64(),
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}
	if !plan.SubnetID.IsNull() {
		body["subnet_id"] = plan.SubnetID.ValueString()
	}
	if !plan.Environment.IsNull() {
		envMap := make(map[string]string)
		resp.Diagnostics.Append(plan.Environment.ElementsAs(ctx, &envMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["environment"] = envMap
	}

	result, err := r.client.Create(ctx, "/api/v1/functions", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating function", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/functions/%s", id), "active", 5*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for function", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/functions/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading function", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.Runtime = types.StringValue(safeString(result["runtime"]))
	state.Handler = types.StringValue(safeString(result["handler"]))
	state.MemoryMB = types.Int64Value(int64(safeFloat64(result["memory_mb"])))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	if v, ok := result["vpc_id"].(string); ok && v != "" {
		state.VPCID = types.StringValue(v)
	}
	if v, ok := result["subnet_id"].(string); ok && v != "" {
		state.SubnetID = types.StringValue(v)
	}
	if envRaw, ok := result["environment"].(map[string]interface{}); ok {
		envMap := make(map[string]string)
		for k, v := range envRaw {
			envMap[k] = safeString(v)
		}
		mapVal, diags := types.MapValueFrom(ctx, types.StringType, envMap)
		resp.Diagnostics.Append(diags...)
		state.Environment = mapVal
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FunctionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":      plan.Name.ValueString(),
		"runtime":   plan.Runtime.ValueString(),
		"handler":   plan.Handler.ValueString(),
		"memory_mb": plan.MemoryMB.ValueInt64(),
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}
	if !plan.SubnetID.IsNull() {
		body["subnet_id"] = plan.SubnetID.ValueString()
	}
	if !plan.Environment.IsNull() {
		envMap := make(map[string]string)
		resp.Diagnostics.Append(plan.Environment.ElementsAs(ctx, &envMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["environment"] = envMap
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/functions/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating function", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FunctionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/functions/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting function", err.Error())
		return
	}
}

func (r *FunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
