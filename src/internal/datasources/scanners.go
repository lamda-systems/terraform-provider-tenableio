package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &ScannersDataSource{}

type ScannersDataSource struct {
	client *client.Client
}

type ScannersDataSourceModel struct {
	Scanners []ScannerItemModel `tfsdk:"scanners"`
}

type ScannerItemModel struct {
	ID               types.Int64  `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	Status           types.String `tfsdk:"status"`
	ScanCount        types.Int64  `tfsdk:"scan_count"`
	Platform         types.String `tfsdk:"platform"`
	EngineVersion    types.String `tfsdk:"engine_version"`
	Owner            types.String `tfsdk:"owner"`
	Pool             types.Bool   `tfsdk:"pool"`
	NetworkName      types.String `tfsdk:"network_name"`
	CreationDate     types.Int64  `tfsdk:"creation_date"`
	LastModifiedDate types.Int64  `tfsdk:"last_modification_date"`
}

func NewScannersDataSource() datasource.DataSource {
	return &ScannersDataSource{}
}

func (d *ScannersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scanners"
}

func (d *ScannersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of scanners from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"scanners": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.Int64Attribute{Computed: true},
						"uuid":                   schema.StringAttribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"type":                   schema.StringAttribute{Computed: true},
						"status":                 schema.StringAttribute{Computed: true},
						"scan_count":             schema.Int64Attribute{Computed: true},
						"platform":               schema.StringAttribute{Computed: true},
						"engine_version":         schema.StringAttribute{Computed: true},
						"owner":                  schema.StringAttribute{Computed: true},
						"pool":                   schema.BoolAttribute{Computed: true},
						"network_name":           schema.StringAttribute{Computed: true},
						"creation_date":          schema.Int64Attribute{Computed: true},
						"last_modification_date": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ScannersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScannersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListScanners(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Scanners", err.Error())
		return
	}

	scanners := make([]ScannerItemModel, len(result.Scanners))
	for i, s := range result.Scanners {
		scanners[i] = ScannerItemModel{
			ID:               types.Int64Value(int64(s.ID)),
			UUID:             types.StringValue(s.UUID),
			Name:             types.StringValue(s.Name),
			Type:             types.StringValue(s.Type),
			Status:           types.StringValue(s.Status),
			ScanCount:        types.Int64Value(int64(s.ScanCount)),
			Platform:         types.StringValue(s.Platform),
			EngineVersion:    types.StringValue(s.EngineVersion),
			Owner:            types.StringValue(s.Owner),
			Pool:             types.BoolValue(s.Pool),
			NetworkName:      types.StringValue(s.NetworkName),
			CreationDate:     types.Int64Value(int64(s.CreationDate)),
			LastModifiedDate: types.Int64Value(int64(s.LastModifiedDate)),
		}
	}

	state := ScannersDataSourceModel{Scanners: scanners}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
