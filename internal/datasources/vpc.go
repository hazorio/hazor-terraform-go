package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var _ datasource.DataSource = &VPCDataSource{}

type VPCDataSource struct{ client *client.Client }
type VPCDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CidrBlock types.String `tfsdk:"cidr_block"`
	Status    types.String `tfsdk:"status"`
}

func NewVPCDataSource() datasource.DataSource { return &VPCDataSource{} }

func (d *VPCDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (d *VPCDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hazor VPC by ID.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Description: "VPC ID.", Required: true},
			"name":       schema.StringAttribute{Description: "VPC name.", Computed: true},
			"cidr_block": schema.StringAttribute{Description: "CIDR block.", Computed: true},
			"status":     schema.StringAttribute{Description: "VPC status.", Computed: true},
		},
	}
}

func (d *VPCDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	d.client = req.ProviderData.(*client.Client)
}

func (d *VPCDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VPCDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := d.client.Read(ctx, fmt.Sprintf("/api/v1/vpcs/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading VPC", err.Error()); return }
	if result == nil { resp.Diagnostics.AddError("VPC not found", state.ID.ValueString()); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.CidrBlock = types.StringValue(getString(data, "cidr_block"))
	state.Status = types.StringValue(getString(data, "status"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok { return v }
	return ""
}
