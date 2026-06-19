package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &FoldersDataSource{}

type FoldersDataSource struct {
	client *client.Client
}

type FoldersDataSourceModel struct {
	Folders []FolderItemModel `tfsdk:"folders"`
}

type FolderItemModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Custom types.Int64  `tfsdk:"custom"`
}

func NewFoldersDataSource() datasource.DataSource {
	return &FoldersDataSource{}
}

func (d *FoldersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folders"
}

func (d *FoldersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of scan folders from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"folders": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":     schema.Int64Attribute{Computed: true},
						"name":   schema.StringAttribute{Computed: true},
						"type":   schema.StringAttribute{Computed: true},
						"custom": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *FoldersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FoldersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListFolders(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Folders", err.Error())
		return
	}

	folders := make([]FolderItemModel, len(result.Folders))
	for i, f := range result.Folders {
		folders[i] = FolderItemModel{
			ID:     types.Int64Value(int64(f.ID)),
			Name:   types.StringValue(f.Name),
			Type:   types.StringValue(f.Type),
			Custom: types.Int64Value(int64(f.Custom)),
		}
	}

	state := FoldersDataSourceModel{Folders: folders}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
