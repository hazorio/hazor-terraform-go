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
	_ resource.Resource                = &K8sClusterResource{}
	_ resource.ResourceWithImportState = &K8sClusterResource{}
)

type K8sClusterResource struct {
	client *client.Client
}

type K8sClusterResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Version   types.String `tfsdk:"version"`
	PlanID    types.String `tfsdk:"plan_id"`
	VPCID     types.String `tfsdk:"vpc_id"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewK8sClusterResource() resource.Resource {
	return &K8sClusterResource{}
}

func (r *K8sClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k8s_cluster"
}

func (r *K8sClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor Kubernetes cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Kubernetes cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Kubernetes cluster.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The Kubernetes version (e.g., 1.28, 1.29).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1.28"),
			},
			"plan_id": schema.StringAttribute{
				Description: "The plan ID determining cluster size and resources.",
				Optional:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the Kubernetes cluster.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the Kubernetes cluster.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the Kubernetes cluster was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the Kubernetes cluster was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *K8sClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *K8sClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan K8sClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"version": plan.Version.ValueString(),
	}
	if !plan.PlanID.IsNull() {
		body["plan_id"] = plan.PlanID.ValueString()
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}

	result, err := r.client.Create(ctx, "/api/v1/kubernetes/clusters", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Kubernetes cluster", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/kubernetes/clusters/%s", id), "running", 20*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Kubernetes cluster", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *K8sClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state K8sClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/kubernetes/clusters/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Kubernetes cluster", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.Version = types.StringValue(safeString(result["version"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	if v, ok := result["plan_id"].(string); ok && v != "" {
		state.PlanID = types.StringValue(v)
	}
	if v, ok := result["vpc_id"].(string); ok && v != "" {
		state.VPCID = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *K8sClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan K8sClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state K8sClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"version": plan.Version.ValueString(),
	}
	if !plan.PlanID.IsNull() {
		body["plan_id"] = plan.PlanID.ValueString()
	}
	if !plan.VPCID.IsNull() {
		body["vpc_id"] = plan.VPCID.ValueString()
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/kubernetes/clusters/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Kubernetes cluster", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *K8sClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state K8sClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/kubernetes/clusters/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Kubernetes cluster", err.Error())
		return
	}
}

func (r *K8sClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
