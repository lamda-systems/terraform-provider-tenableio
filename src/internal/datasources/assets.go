package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &AssetsDataSource{}

type AssetsDataSource struct {
	client *client.Client
}

type AssetsDataSourceModel struct {
	DateRange types.Int64         `tfsdk:"date_range"`
	Assets    []AssetListItemModel `tfsdk:"assets"`
}

type AssetListItemModel struct {
	ID              types.String `tfsdk:"id"`
	HasAgent        types.Bool   `tfsdk:"has_agent"`
	HasPluginResult types.Bool   `tfsdk:"has_plugin_results"`
	FQDN            types.List   `tfsdk:"fqdn"`
	IPv4            types.List   `tfsdk:"ipv4"`
	IPv6            types.List   `tfsdk:"ipv6"`
	MacAddress      types.List   `tfsdk:"mac_address"`
	NetbiosName     types.List   `tfsdk:"netbios_name"`
	OperatingSystem types.List   `tfsdk:"operating_system"`
	AgentName       types.List   `tfsdk:"agent_name"`
	FirstSeen       types.String `tfsdk:"first_seen"`
	LastSeen        types.String `tfsdk:"last_seen"`
}

func NewAssetsDataSource() datasource.DataSource {
	return &AssetsDataSource{}
}

func (d *AssetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assets"
}

func (d *AssetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of assets from Tenable.io workbenches.",
		Attributes: map[string]schema.Attribute{
			"date_range": schema.Int64Attribute{
				Description: "Number of days of data to retrieve. Defaults to 30.",
				Optional:    true,
			},
			"assets": schema.ListNestedAttribute{
				Description: "List of assets.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The UUID of the asset.",
							Computed:    true,
						},
						"has_agent": schema.BoolAttribute{
							Description: "Whether the asset has a Tenable agent.",
							Computed:    true,
						},
						"has_plugin_results": schema.BoolAttribute{
							Description: "Whether the asset has plugin results.",
							Computed:    true,
						},
						"fqdn": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"ipv4": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"ipv6": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"mac_address": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"netbios_name": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"operating_system": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"agent_name": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"first_seen": schema.StringAttribute{
							Computed: true,
						},
						"last_seen": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *AssetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AssetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AssetsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dateRange := 30
	if !config.DateRange.IsNull() {
		dateRange = int(config.DateRange.ValueInt64())
	}

	result, err := d.client.ListAssets(ctx, dateRange)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Assets", err.Error())
		return
	}

	assets := make([]AssetListItemModel, len(result.Assets))
	for i, a := range result.Assets {
		assets[i] = AssetListItemModel{
			ID:              types.StringValue(a.ID),
			HasAgent:        types.BoolValue(a.HasAgent),
			HasPluginResult: types.BoolValue(a.HasPluginResult),
			FQDN:            stringSliceToList(a.FQDN),
			IPv4:            stringSliceToList(a.IPv4),
			IPv6:            stringSliceToList(a.IPv6),
			MacAddress:      stringSliceToList(a.MacAddress),
			NetbiosName:     stringSliceToList(a.NetbiosName),
			OperatingSystem: stringSliceToList(a.OperatingSystem),
			AgentName:       stringSliceToList(a.AgentName),
			FirstSeen:       types.StringValue(a.FirstSeen),
			LastSeen:        types.StringValue(a.LastSeen),
		}
	}

	config.Assets = assets
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
