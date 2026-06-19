package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &ExclusionsDataSource{}

type ExclusionsDataSource struct {
	client *client.Client
}

type ExclusionsDataSourceModel struct {
	Exclusions []ExclusionItemModel `tfsdk:"exclusions"`
}

type ExclusionItemModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Members              types.String `tfsdk:"members"`
	NetworkID            types.String `tfsdk:"network_id"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
}

func NewExclusionsDataSource() datasource.DataSource {
	return &ExclusionsDataSource{}
}

func (d *ExclusionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exclusions"
}

func (d *ExclusionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of scan exclusions from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"exclusions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.Int64Attribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"description":            schema.StringAttribute{Computed: true},
						"members":                schema.StringAttribute{Computed: true},
						"network_id":             schema.StringAttribute{Computed: true},
						"creation_date":          schema.Int64Attribute{Computed: true},
						"last_modification_date": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ExclusionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ExclusionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListExclusions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Exclusions", err.Error())
		return
	}

	exclusions := make([]ExclusionItemModel, len(result.Exclusions))
	for i, e := range result.Exclusions {
		exclusions[i] = ExclusionItemModel{
			ID:                   types.Int64Value(int64(e.ID)),
			Name:                 types.StringValue(e.Name),
			Description:          types.StringValue(e.Description),
			Members:              types.StringValue(e.Members),
			NetworkID:            types.StringValue(e.NetworkID),
			CreationDate:         types.Int64Value(int64(e.CreationDate)),
			LastModificationDate: types.Int64Value(int64(e.LastModificationDate)),
		}
	}

	state := ExclusionsDataSourceModel{Exclusions: exclusions}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
