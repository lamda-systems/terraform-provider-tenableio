package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &ScansDataSource{}

type ScansDataSource struct {
	client *client.Client
}

type ScansDataSourceModel struct {
	FolderID types.Int64      `tfsdk:"folder_id"`
	Scans    []ScanItemModel  `tfsdk:"scans"`
}

type ScanItemModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	FolderID             types.Int64  `tfsdk:"folder_id"`
	Type                 types.String `tfsdk:"type"`
	Status               types.String `tfsdk:"status"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
}

func NewScansDataSource() datasource.DataSource {
	return &ScansDataSource{}
}

func (d *ScansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scans"
}

func (d *ScansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of scan configurations from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"folder_id": schema.Int64Attribute{
				Description: "Filter scans by folder ID.",
				Optional:    true,
			},
			"scans": schema.ListNestedAttribute{
				Description: "List of scans.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.Int64Attribute{Computed: true},
						"uuid":                   schema.StringAttribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"description":            schema.StringAttribute{Computed: true},
						"folder_id":              schema.Int64Attribute{Computed: true},
						"type":                   schema.StringAttribute{Computed: true},
						"status":                 schema.StringAttribute{Computed: true},
						"enabled":                schema.BoolAttribute{Computed: true},
						"creation_date":          schema.Int64Attribute{Computed: true},
						"last_modification_date": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ScansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ScansDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var folderID *int
	if !config.FolderID.IsNull() {
		v := int(config.FolderID.ValueInt64())
		folderID = &v
	}

	result, err := d.client.ListScans(ctx, folderID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Scans", err.Error())
		return
	}

	scans := make([]ScanItemModel, len(result.Scans))
	for i, s := range result.Scans {
		scans[i] = ScanItemModel{
			ID:                   types.Int64Value(int64(s.ID)),
			UUID:                 types.StringValue(s.UUID),
			Name:                 types.StringValue(s.Name),
			Description:          types.StringValue(s.Description),
			FolderID:             types.Int64Value(int64(s.FolderID)),
			Type:                 types.StringValue(s.Type),
			Status:               types.StringValue(s.Status),
			Enabled:              types.BoolValue(s.Enabled),
			CreationDate:         types.Int64Value(int64(s.CreationDate)),
			LastModificationDate: types.Int64Value(int64(s.LastModificationDate)),
		}
	}

	config.Scans = scans
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
