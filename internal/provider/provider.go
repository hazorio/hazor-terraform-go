package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/client"
	"github.com/hazor-cloud/terraform-provider-hazor/internal/resources"
)

var _ provider.Provider = &HazorProvider{}

// HazorProvider defines the provider implementation.
type HazorProvider struct {
	version string
}

// HazorProviderModel describes the provider data model.
type HazorProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HazorProvider{
			version: version,
		}
	}
}

func (p *HazorProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hazor"
	resp.Version = p.version
}

func (p *HazorProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Hazor provider allows you to manage Hazor cloud resources.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The Hazor API endpoint URL. Can also be set via the HAZOR_ENDPOINT environment variable.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with the Hazor API. Can also be set via the HAZOR_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *HazorProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HazorProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("HAZOR_ENDPOINT")
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}
	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Endpoint",
			"The provider requires an endpoint to be set. Set the 'endpoint' attribute in the provider block or the HAZOR_ENDPOINT environment variable.",
		)
		return
	}

	apiKey := os.Getenv("HAZOR_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires an API key to be set. Set the 'api_key' attribute in the provider block or the HAZOR_API_KEY environment variable.",
		)
		return
	}

	c := client.NewClient(endpoint, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *HazorProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewInstanceResource,
		resources.NewVPCResource,
		resources.NewSubnetResource,
		resources.NewVolumeResource,
		resources.NewSecurityGroupResource,
		resources.NewElasticIPResource,
		resources.NewNATGatewayResource,
		resources.NewKeyPairResource,
		resources.NewLoadBalancerResource,
		resources.NewDatabaseResource,
		resources.NewSnapshotResource,
		resources.NewBucketResource,
		resources.NewDNSZoneResource,
		resources.NewDNSRecordResource,
		resources.NewRedisInstanceResource,
		resources.NewCDNDistributionResource,
		resources.NewK8sClusterResource,
		resources.NewFunctionResource,
		resources.NewNoSQLInstanceResource,
		resources.NewStreamingClusterResource,
		resources.NewPostgresMLInstanceResource,
		resources.NewContainerRegistryResource,
		resources.NewRunnerResource,
		resources.NewBunAppResource,
		resources.NewSupabaseInstanceResource,
	}
}

func (p *HazorProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
