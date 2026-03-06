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
	_ resource.Resource                = &NATGatewayResource{}
	_ resource.ResourceWithImportState = &NATGatewayResource{}
)

type NATGatewayResource struct {
	client *client.Client
}

type NATGatewayResourceModel struct {
	ID       types.String `tfsdk:"id"`
	VPCID    types.String `tfsdk:"vpc_id"`
	SubnetID types.String `tfsdk:"subnet_id"`
	Status   types.String `tfsdk:"status"`
}

func NewNATGatewayResource() resource.Resource {
	return &NATGatewayResource{}
}

func (r *NATGatewayResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_gateway"
}

func (r *NATGatewayResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hazor NAT Gateway.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the NAT gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the NAT gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Description: "The subnet ID for the NAT gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the NAT gateway.",
				Computed:    true,
			},
		},
	}
}

func (r *NATGatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NATGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NATGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"vpc_id":    plan.VPCID.ValueString(),
		"subnet_id": plan.SubnetID.ValueString(),
	}

	result, err := r.client.Create(ctx, "/api/v1/nat-gateways", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating NAT gateway", err.Error())
		return
	}

	id := safeString(result["id"])
	final, err := r.client.WaitForStatus(ctx, fmt.Sprintf("/api/v1/nat-gateways/%s", id), "active", 5*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for NAT gateway", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Status = types.StringValue(safeString(final["status"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NATGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NATGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/nat-gateways/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading NAT gateway", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.VPCID = types.StringValue(safeString(result["vpc_id"]))
	state.SubnetID = types.StringValue(safeString(result["subnet_id"]))
	state.Status = types.StringValue(safeString(result["status"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NATGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// NAT Gateways are immutable; all attributes require replacement.
	resp.Diagnostics.AddError("Update not supported", "NAT gateways cannot be updated in-place. All changes require replacement.")
}

func (r *NATGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NATGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/nat-gateways/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting NAT gateway", err.Error())
		return
	}
}

func (r *NATGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
