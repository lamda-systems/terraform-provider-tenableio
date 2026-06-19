package provider

import (
	"context"
	"os"

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
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
	BaseURL   types.String `tfsdk:"base_url"`
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
		},
	}
}

func (p *TenableioProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TenableioProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessKey := os.Getenv("TENABLEIO_ACCESS_KEY")
	if !config.AccessKey.IsNull() {
		accessKey = config.AccessKey.ValueString()
	}

	secretKey := os.Getenv("TENABLEIO_SECRET_KEY")
	if !config.SecretKey.IsNull() {
		secretKey = config.SecretKey.ValueString()
	}

	baseURL := os.Getenv("TENABLEIO_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if accessKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Access Key",
			"The provider cannot create the Tenable.io API client because the access key is missing. "+
				"Set the access_key attribute in the provider configuration or the TENABLEIO_ACCESS_KEY environment variable.",
		)
	}

	if secretKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Secret Key",
			"The provider cannot create the Tenable.io API client because the secret key is missing. "+
				"Set the secret_key attribute in the provider configuration or the TENABLEIO_SECRET_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(accessKey, secretKey, baseURL, p.version)

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
