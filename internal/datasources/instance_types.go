package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
)

var _ datasource.DataSource = &InstanceTypesDataSource{}

type InstanceTypesDataSource struct{ client *client.Client }

type InstanceTypeItem struct {
	ID       types.String  `tfsdk:"id"`
	Name     types.String  `tfsdk:"name"`
	VCPUs    types.Int64   `tfsdk:"vcpus"`
	MemoryMB types.Int64   `tfsdk:"memory_mb"`
	Category types.String  `tfsdk:"category"`
	Price    types.Float64 `tfsdk:"price_per_hour"`
}

type InstanceTypesDataSourceModel struct {
	Category types.String       `tfsdk:"category"`
	Types    []InstanceTypeItem `tfsdk:"types"`
}

func NewInstanceTypesDataSource() datasource.DataSource { return &InstanceTypesDataSource{} }

func (d *InstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_types"
}

func (d *InstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List available instance types, optionally filtered by category.",
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{Description: "Filter by category (general, compute_optimized, memory_optimized).", Optional: true},
			"types": schema.ListNestedAttribute{
				Description: "List of instance types.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Description: "Instance type ID.", Computed: true},
						"name":           schema.StringAttribute{Description: "Display name.", Computed: true},
						"vcpus":          schema.Int64Attribute{Description: "Number of vCPUs.", Computed: true},
						"memory_mb":      schema.Int64Attribute{Description: "Memory in MB.", Computed: true},
						"category":       schema.StringAttribute{Description: "Category.", Computed: true},
						"price_per_hour": schema.Float64Attribute{Description: "Price per hour (USD).", Computed: true},
					},
				},
			},
		},
	}
}

func (d *InstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil { return }
	d.client = req.ProviderData.(*client.Client)
}

func (d *InstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config InstanceTypesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() { return }

	result, err := d.client.Read(ctx, "/api/v1/instance-types")
	if err != nil { resp.Diagnostics.AddError("Error reading instance types", err.Error()); return }

	var items []InstanceTypeItem
	if data, ok := result["data"].([]interface{}); ok {
		for _, raw := range data {
			item := raw.(map[string]interface{})
			cat := getString(item, "category")
			if !config.Category.IsNull() && cat != config.Category.ValueString() {
				continue
			}
			it := InstanceTypeItem{
				ID:       types.StringValue(getString(item, "id")),
				Name:     types.StringValue(getString(item, "name")),
				Category: types.StringValue(cat),
			}
			if v, ok := item["vcpu"].(float64); ok { it.VCPUs = types.Int64Value(int64(v)) }
			if v, ok := item["memory_mb"].(float64); ok { it.MemoryMB = types.Int64Value(int64(v)) }
			if v, ok := item["price_per_hour"].(string); ok {
				var f float64
				fmt.Sscanf(v, "%f", &f)
				it.Price = types.Float64Value(f)
			}
			items = append(items, it)
		}
	}

	state := InstanceTypesDataSourceModel{Category: config.Category, Types: items}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
