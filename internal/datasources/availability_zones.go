package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var _ datasource.DataSource = &AvailabilityZonesDataSource{}

type AvailabilityZonesDataSource struct{ client *client.Client }
type AvailabilityZonesDataSourceModel struct {
	Names []types.String `tfsdk:"names"`
}

func NewAvailabilityZonesDataSource() datasource.DataSource { return &AvailabilityZonesDataSource{} }

func (d *AvailabilityZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_availability_zones"
}

func (d *AvailabilityZonesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List available availability zones.",
		Attributes: map[string]schema.Attribute{
			"names": schema.ListAttribute{Description: "List of AZ names.", Computed: true, ElementType: types.StringType},
		},
	}
}

func (d *AvailabilityZonesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	d.client = req.ProviderData.(*client.Client)
}

func (d *AvailabilityZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.Read(ctx, "/api/v1/regions/availability-zones")
	if err != nil { resp.Diagnostics.AddError("Error reading AZs", err.Error()); return }

	var names []types.String
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if az, ok := item.(map[string]interface{}); ok {
				if name, ok := az["name"].(string); ok {
					names = append(names, types.StringValue(name))
				}
			}
		}
	}

	state := AvailabilityZonesDataSourceModel{Names: names}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
