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
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
)

type DatabaseResource struct {
	client *client.Client
}

type DatabaseResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Engine        types.String `tfsdk:"engine"`
	EngineVersion types.String `tfsdk:"engine_version"`
	InstanceClass types.String `tfsdk:"instance_class"`
	StorageGB     types.Int64  `tfsdk:"storage_gb"`
	VPCID         types.String `tfsdk:"vpc_id"`
	Status        types.String `tfsdk:"status"`
	Endpoint      types.String `tfsdk:"endpoint"`
}

func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor managed database instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the database.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the database instance.",
				Required:    true,
			},
			"engine": schema.StringAttribute{
				Description: "The database engine (e.g., postgres, mysql, mariadb).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"engine_version": schema.StringAttribute{
				Description: "The version of the database engine.",
				Required:    true,
			},
			"instance_class": schema.StringAttribute{
				Description: "The instance class for the database (e.g., db.small, db.medium).",
				Required:    true,
			},
			"storage_gb": schema.Int64Attribute{
				Description: "The allocated storage in gigabytes.",
				Required:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the database.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The connection endpoint for the database.",
				Computed:    true,
			},
		},
	}
}

func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"engine":         plan.Engine.ValueString(),
		"engine_version": plan.EngineVersion.ValueString(),
		"instance_class": plan.InstanceClass.ValueString(),
		"storage_gb":     plan.StorageGB.ValueInt64(),
		"vpc_id":         plan.VPCID.ValueString(),
	}

	result, err := r.client.Create(ctx, "/api/v1/databases", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/databases/%s", id), "active", 15*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for database", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.Endpoint = types.StringValue(safeString(final["endpoint"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/databases/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.Engine = types.StringValue(safeString(result["engine"]))
	state.EngineVersion = types.StringValue(safeString(result["engine_version"]))
	state.InstanceClass = types.StringValue(safeString(result["instance_class"]))
	state.StorageGB = types.Int64Value(int64(safeFloat64(result["storage_gb"])))
	state.VPCID = types.StringValue(safeString(result["vpc_id"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.Endpoint = types.StringValue(safeString(result["endpoint"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"engine_version": plan.EngineVersion.ValueString(),
		"instance_class": plan.InstanceClass.ValueString(),
		"storage_gb":     plan.StorageGB.ValueInt64(),
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/databases/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.Endpoint = types.StringValue(safeString(result["endpoint"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/databases/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting database", err.Error())
		return
	}
}

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
