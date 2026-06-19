package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &AssetDataSource{}

type AssetDataSource struct {
	client *client.Client
}

type AssetDataSourceModel struct {
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
	HostName        types.List   `tfsdk:"hostname"`
	FirstSeen       types.String `tfsdk:"first_seen"`
	LastSeen        types.String `tfsdk:"last_seen"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	SystemType      types.String `tfsdk:"system_type"`
}

func NewAssetDataSource() datasource.DataSource {
	return &AssetDataSource{}
}

func (d *AssetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_asset"
}

func (d *AssetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves details for a single asset from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The UUID of the asset.",
				Required:    true,
			},
			"has_agent": schema.BoolAttribute{
				Description: "Whether the asset has a Tenable agent installed.",
				Computed:    true,
			},
			"has_plugin_results": schema.BoolAttribute{
				Description: "Whether the asset has plugin results.",
				Computed:    true,
			},
			"fqdn": schema.ListAttribute{
				Description: "Fully qualified domain names associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv4": schema.ListAttribute{
				Description: "IPv4 addresses associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipv6": schema.ListAttribute{
				Description: "IPv6 addresses associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"mac_address": schema.ListAttribute{
				Description: "MAC addresses associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"netbios_name": schema.ListAttribute{
				Description: "NetBIOS names associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"operating_system": schema.ListAttribute{
				Description: "Operating systems detected on the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"agent_name": schema.ListAttribute{
				Description: "Tenable agent names associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"hostname": schema.ListAttribute{
				Description: "Hostnames associated with the asset.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"first_seen": schema.StringAttribute{
				Description: "When the asset was first seen.",
				Computed:    true,
			},
			"last_seen": schema.StringAttribute{
				Description: "When the asset was last seen.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "When the asset record was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "When the asset record was last updated.",
				Computed:    true,
			},
			"system_type": schema.StringAttribute{
				Description: "The system type of the asset.",
				Computed:    true,
			},
		},
	}
}

func (d *AssetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AssetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AssetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetAsset(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Asset", err.Error())
		return
	}

	info := result.Info
	config.HasAgent = types.BoolValue(info.HasAgent)
	config.HasPluginResult = types.BoolValue(info.HasPluginResults)
	config.FirstSeen = types.StringValue(info.FirstSeen)
	config.LastSeen = types.StringValue(info.LastSeen)
	config.CreatedAt = types.StringValue(info.CreatedAt)
	config.UpdatedAt = types.StringValue(info.UpdatedAt)
	config.SystemType = types.StringValue(info.SystemType)

	config.FQDN = stringSliceToList(info.FQDN)
	config.IPv4 = stringSliceToList(info.IPv4)
	config.IPv6 = stringSliceToList(info.IPv6)
	config.MacAddress = stringSliceToList(info.MacAddress)
	config.NetbiosName = stringSliceToList(info.NetbiosName)
	config.OperatingSystem = stringSliceToList(info.OperatingSystem)
	config.AgentName = stringSliceToList(info.AgentName)
	config.HostName = stringSliceToList(info.HostName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func stringSliceToList(s []string) types.List {
	if len(s) == 0 {
		l, _ := types.ListValueFrom(context.Background(), types.StringType, []string{})
		return l
	}
	l, _ := types.ListValueFrom(context.Background(), types.StringType, s)
	return l
}
