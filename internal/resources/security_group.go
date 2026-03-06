package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var (
	_ resource.Resource                = &SecurityGroupResource{}
	_ resource.ResourceWithImportState = &SecurityGroupResource{}
)

type SecurityGroupResource struct {
	client *client.Client
}

type SecurityGroupResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	VPCID        types.String `tfsdk:"vpc_id"`
	IngressRules types.List   `tfsdk:"ingress_rules"`
	EgressRules  types.List   `tfsdk:"egress_rules"`
}

var securityRuleAttrTypes = map[string]attr.Type{
	"protocol":  types.StringType,
	"from_port": types.Int64Type,
	"to_port":   types.Int64Type,
	"cidr":      types.StringType,
}

func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

func (r *SecurityGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ruleNestedSchema := schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"protocol": schema.StringAttribute{
					Description: "The protocol (tcp, udp, icmp, or -1 for all).",
					Required:    true,
				},
				"from_port": schema.Int64Attribute{
					Description: "The start of the port range.",
					Required:    true,
				},
				"to_port": schema.Int64Attribute{
					Description: "The end of the port range.",
					Required:    true,
				},
				"cidr": schema.StringAttribute{
					Description: "The CIDR block to allow/deny.",
					Required:    true,
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Hazor security group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the security group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the security group.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the security group.",
				Optional:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID this security group belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ingress_rules": func() schema.ListNestedAttribute {
				s := ruleNestedSchema
				s.Description = "Inbound rules for the security group."
				return s
			}(),
			"egress_rules": func() schema.ListNestedAttribute {
				s := ruleNestedSchema
				s.Description = "Outbound rules for the security group."
				return s
			}(),
		},
	}
}

func (r *SecurityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func rulesFromModel(ctx context.Context, rulesList types.List) ([]map[string]interface{}, error) {
	if rulesList.IsNull() || rulesList.IsUnknown() {
		return nil, nil
	}

	var rules []map[string]interface{}
	type ruleModel struct {
		Protocol types.String `tfsdk:"protocol"`
		FromPort types.Int64  `tfsdk:"from_port"`
		ToPort   types.Int64  `tfsdk:"to_port"`
		CIDR     types.String `tfsdk:"cidr"`
	}
	var ruleModels []ruleModel
	diags := rulesList.ElementsAs(ctx, &ruleModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to read rules")
	}

	for _, rm := range ruleModels {
		rules = append(rules, map[string]interface{}{
			"protocol":  rm.Protocol.ValueString(),
			"from_port": rm.FromPort.ValueInt64(),
			"to_port":   rm.ToPort.ValueInt64(),
			"cidr":      rm.CIDR.ValueString(),
		})
	}
	return rules, nil
}

func rulesToListValue(ctx context.Context, rawRules interface{}) types.List {
	rules, ok := rawRules.([]interface{})
	if !ok || len(rules) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: securityRuleAttrTypes})
	}

	var elements []attr.Value
	for _, r := range rules {
		rule, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		obj, _ := types.ObjectValue(securityRuleAttrTypes, map[string]attr.Value{
			"protocol":  types.StringValue(safeString(rule["protocol"])),
			"from_port": types.Int64Value(int64(safeFloat64(rule["from_port"]))),
			"to_port":   types.Int64Value(int64(safeFloat64(rule["to_port"]))),
			"cidr":      types.StringValue(safeString(rule["cidr"])),
		})
		elements = append(elements, obj)
	}

	list, _ := types.ListValue(types.ObjectType{AttrTypes: securityRuleAttrTypes}, elements)
	return list
}

func (r *SecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"vpc_id": plan.VPCID.ValueString(),
	}
	if !plan.Description.IsNull() {
		body["description"] = plan.Description.ValueString()
	}

	ingress, err := rulesFromModel(ctx, plan.IngressRules)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingress rules", err.Error())
		return
	}
	if ingress != nil {
		body["ingress_rules"] = ingress
	}

	egress, err := rulesFromModel(ctx, plan.EgressRules)
	if err != nil {
		resp.Diagnostics.AddError("Error reading egress rules", err.Error())
		return
	}
	if egress != nil {
		body["egress_rules"] = egress
	}

	result, err := r.client.Create(ctx, "/api/v1/security-groups", body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating security group", err.Error())
		return
	}

	plan.ID = types.StringValue(safeString(result["id"]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Read(ctx, fmt.Sprintf("/api/v1/security-groups/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading security group", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(safeString(result["name"]))
	state.VPCID = types.StringValue(safeString(result["vpc_id"]))

	if v, ok := result["description"].(string); ok && v != "" {
		state.Description = types.StringValue(v)
	}

	state.IngressRules = rulesToListValue(ctx, result["ingress_rules"])
	state.EgressRules = rulesToListValue(ctx, result["egress_rules"])

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() {
		body["description"] = plan.Description.ValueString()
	}

	ingress, err := rulesFromModel(ctx, plan.IngressRules)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingress rules", err.Error())
		return
	}
	if ingress != nil {
		body["ingress_rules"] = ingress
	}

	egress, err := rulesFromModel(ctx, plan.EgressRules)
	if err != nil {
		resp.Diagnostics.AddError("Error reading egress rules", err.Error())
		return
	}
	if egress != nil {
		body["egress_rules"] = egress
	}

	_, err = r.client.Update(ctx, fmt.Sprintf("/api/v1/security-groups/%s", state.ID.ValueString()), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating security group", err.Error())
		return
	}

	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/security-groups/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting security group", err.Error())
		return
	}
}

func (r *SecurityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
