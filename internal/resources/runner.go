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
	_ resource.Resource                = &RunnerResource{}
	_ resource.ResourceWithImportState = &RunnerResource{}
)

type RunnerResource struct {
	client *client.Client
}

type RunnerResourceModel struct {
	ID        types.String `tfsdk:"id"`
	OrgID     types.String `tfsdk:"org_id"`
	Labels    types.List   `tfsdk:"labels"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewRunnerResource() resource.Resource {
	return &RunnerResource{}
}

func (r *RunnerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner"
}

func (r *RunnerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor CI/CD runner.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the runner.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID the runner belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.ListAttribute{
				Description: "Labels assigned to the runner for job matching.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the runner.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the runner was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the runner was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *RunnerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RunnerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"org_id": plan.OrgID.ValueString(),
	}

	if !plan.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	result, err := r.client.Create(ctx, "/api/v1/runners", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating runner", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/runners/%s", id), "active", 10*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for runner", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))
	plan.CreatedAt = types.StringValue(safeString(final["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(final["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RunnerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/runners/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading runner", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.OrgID = types.StringValue(safeString(result["org_id"]))
	state.Status = types.StringValue(safeString(result["status"]))
	state.CreatedAt = types.StringValue(safeString(result["created_at"]))
	state.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	if labelsRaw, ok := result["labels"].([]interface{}); ok {
		var labels []string
		for _, l := range labelsRaw {
			labels = append(labels, safeString(l))
		}
		listVal, diags := types.ListValueFrom(ctx, types.StringType, labels)
		resp.Diagnostics.Append(diags...)
		state.Labels = listVal
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RunnerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RunnerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}

	if !plan.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	result, err := r.client.Update(ctx, fmt.Sprintf("/api/v1/runners/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating runner", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Status = types.StringValue(safeString(result["status"]))
	plan.CreatedAt = types.StringValue(safeString(result["created_at"]))
	plan.UpdatedAt = types.StringValue(safeString(result["updated_at"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RunnerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/runners/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting runner", err.Error())
		return
	}
}

func (r *RunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
