package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/datasources"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/resources"
)

var _ provider.Provider = &TenableioProvider{}

type TenableioProvider struct {
	version string
}

type TenableioProviderModel struct {
	AccessKey       types.String `tfsdk:"access_key"`
	SecretKey       types.String `tfsdk:"secret_key"`
	BaseURL         types.String `tfsdk:"base_url"`
	ProxyAuthHeader types.String `tfsdk:"proxy_auth_header"`
	ProxyAuthValue  types.String `tfsdk:"proxy_auth_value"`
	Prefix          types.String `tfsdk:"prefix"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TenableioProvider{
			version: version,
		}
	}
}

func (p *TenableioProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tenableio"
	resp.Version = p.version
}

func (p *TenableioProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Tenable.io Vulnerability Management.",
		Attributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Description: "Tenable.io API access key. Can also be set via TENABLEIO_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "Tenable.io API secret key. Can also be set via TENABLEIO_SECRET_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Tenable.io API base URL. Defaults to https://cloud.tenable.com. Can also be set via TENABLEIO_BASE_URL environment variable.",
				Optional:    true,
			},
			"proxy_auth_header": schema.StringAttribute{
				Description: "Name of an additional HTTP header to send with every API request, typically used to authenticate against a forward proxy. Can also be set via TENABLEIO_PROXY_AUTH_HEADER environment variable.",
				Optional:    true,
			},
			"proxy_auth_value": schema.StringAttribute{
				Description: "Value for the proxy_auth_header HTTP header. Can also be set via TENABLEIO_PROXY_AUTH_VALUE environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"prefix": schema.StringAttribute{
				Description: "Prefix for the environment variables this provider instance reads, so each aliased provider can use its own credentials and proxy token. " +
					"Setting it to `TENABLEIO_EU`, for example, reads TENABLEIO_EU_ACCESS_KEY, TENABLEIO_EU_SECRET_KEY, TENABLEIO_EU_BASE_URL, " +
					"TENABLEIO_EU_PROXY_AUTH_HEADER and TENABLEIO_EU_PROXY_AUTH_VALUE. " +
					"Each variable falls back independently to its unprefixed TENABLEIO_ equivalent when unset, and attributes set in the provider block always win.",
				Optional: true,
			},
		},
	}
}

func (p *TenableioProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TenableioProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, diags := resolveSettings(config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(
		settings.accessKey,
		settings.secretKey,
		settings.baseURL,
		settings.proxyAuthHeader,
		settings.proxyAuthValue,
		p.version,
	)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *TenableioProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewScanResource,
		resources.NewPolicyResource,
		resources.NewFolderResource,
		resources.NewExclusionResource,
		resources.NewNetworkResource,
		resources.NewTagCategoryResource,
		resources.NewTagValueResource,
		resources.NewAgentGroupResource,
	}
}

func (p *TenableioProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewScansDataSource,
		datasources.NewPoliciesDataSource,
		datasources.NewAssetDataSource,
		datasources.NewAssetsDataSource,
		datasources.NewFoldersDataSource,
		datasources.NewExclusionsDataSource,
		datasources.NewNetworksDataSource,
		datasources.NewScannersDataSource,
		datasources.NewAgentGroupsDataSource,
		datasources.NewTagCategoriesDataSource,
		datasources.NewTagValuesDataSource,
	}
}
