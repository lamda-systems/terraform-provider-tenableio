package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &NetworksDataSource{}

type NetworksDataSource struct {
	client *client.Client
}

type NetworksDataSourceModel struct {
	Networks []NetworkItemModel `tfsdk:"networks"`
}

type NetworkItemModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	IsDefault         types.Bool   `tfsdk:"is_default"`
	AssetsTTLDays     types.Int64  `tfsdk:"assets_ttl_days"`
	ScannerCount      types.Int64  `tfsdk:"scanner_count"`
	CreatedBy         types.String `tfsdk:"created_by"`
	CreatedInSeconds  types.Int64  `tfsdk:"created_in_seconds"`
	ModifiedInSeconds types.Int64  `tfsdk:"modified_in_seconds"`
}

func NewNetworksDataSource() datasource.DataSource {
	return &NetworksDataSource{}
}

func (d *NetworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *NetworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of networks from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"networks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":               schema.StringAttribute{Computed: true},
						"name":               schema.StringAttribute{Computed: true},
						"description":        schema.StringAttribute{Computed: true},
						"is_default":         schema.BoolAttribute{Computed: true},
						"assets_ttl_days":    schema.Int64Attribute{Computed: true},
						"scanner_count":      schema.Int64Attribute{Computed: true},
						"created_by":         schema.StringAttribute{Computed: true},
						"created_in_seconds": schema.Int64Attribute{Computed: true},
						"modified_in_seconds": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *NetworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListNetworks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Networks", err.Error())
		return
	}

	networks := make([]NetworkItemModel, len(result.Networks))
	for i, n := range result.Networks {
		networks[i] = NetworkItemModel{
			UUID:              types.StringValue(n.UUID),
			Name:              types.StringValue(n.Name),
			Description:       types.StringValue(n.Description),
			IsDefault:         types.BoolValue(n.IsDefault),
			AssetsTTLDays:     types.Int64Value(int64(n.AssetsTTLDays)),
			ScannerCount:      types.Int64Value(int64(n.ScannerCount)),
			CreatedBy:         types.StringValue(n.CreatedBy),
			CreatedInSeconds:  types.Int64Value(int64(n.CreatedInSeconds)),
			ModifiedInSeconds: types.Int64Value(int64(n.ModifiedInSeconds)),
		}
	}

	state := NetworksDataSourceModel{Networks: networks}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
