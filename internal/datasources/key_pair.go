package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var _ datasource.DataSource = &KeyPairDataSource{}

type KeyPairDataSource struct{ client *client.Client }
type KeyPairDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

func NewKeyPairDataSource() datasource.DataSource { return &KeyPairDataSource{} }

func (d *KeyPairDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_pair"
}

func (d *KeyPairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Hazor key pair by ID.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Description: "KeyPair ID.", Required: true},
			"name":   schema.StringAttribute{Description: "KeyPair name.", Computed: true},
			"status": schema.StringAttribute{Description: "KeyPair status.", Computed: true},
		},
	}
}

func (d *KeyPairDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	d.client = req.ProviderData.(*client.Client)
}

func (d *KeyPairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KeyPairDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() { return }

	result, err := d.client.Read(ctx, fmt.Sprintf("/api/v1/key-pairs/%s", state.ID.ValueString()))
	if err != nil { resp.Diagnostics.AddError("Error reading key pair", err.Error()); return }
	if result == nil { resp.Diagnostics.AddError("Not found", state.ID.ValueString()); return }

	data := result["data"].(map[string]interface{})
	state.Name = types.StringValue(getString(data, "name"))
	state.Status = types.StringValue(getString(data, "status"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
