package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var _ datasource.DataSource = &SubnetDataSource{}

type SubnetDataSource struct{ client *client.Client }
type SubnetDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

func NewSubnetDataSource() datasource.DataSource { return &SubnetDataSource{} }

func (d *SubnetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (d *SubnetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hazor subnet by ID.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Description: "Subnet ID.", Required: true},
			"name":   schema.StringAttribute{Description: "Subnet name.", Computed: true},
			"status": schema.StringAttribute{Description: "Subnet status.", Computed: true},
		},
	}
}

func (d *SubnetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	d.client = req.ProviderData.(*client.Client)
}

func (d *SubnetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state SubnetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := d.client.Read(ctx, fmt.Sprintf("/api/v1/subnets/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading subnet", err.Error()); return }
	if result == nil { resp.Diagnostics.AddError("Not found", state.ID.ValueString()); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.Status = types.StringValue(getString(data, "status"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
