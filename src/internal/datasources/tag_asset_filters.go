package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &TagAssetFiltersDataSource{}

type TagAssetFiltersDataSource struct {
	client *client.Client
}

type TagAssetFiltersDataSourceModel struct {
	Filters []TagAssetFilterModel `tfsdk:"filters"`
}

type TagAssetFilterModel struct {
	Name         types.String               `tfsdk:"name"`
	ReadableName types.String               `tfsdk:"readable_name"`
	Operators    []types.String             `tfsdk:"operators"`
	Control      TagAssetFilterControlModel `tfsdk:"control"`
}

type TagAssetFilterControlModel struct {
	Type          types.String                   `tfsdk:"type"`
	Regex         types.String                   `tfsdk:"regex"`
	ReadableRegex types.String                   `tfsdk:"readable_regex"`
	List          []TagAssetFilterListEntryModel `tfsdk:"list"`
}

type TagAssetFilterListEntryModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func NewTagAssetFiltersDataSource() datasource.DataSource {
	return &TagAssetFiltersDataSource{}
}

func (d *TagAssetFiltersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_asset_filters"
}

func (d *TagAssetFiltersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the asset attributes that can be used as rule properties in dynamic tags, " +
			"with the operators each one supports. Use it to discover valid `property` and `operator` " +
			"values for the `filters` of a tenableio_tag_value.",
		Attributes: map[string]schema.Attribute{
			"filters": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The asset attribute or tag identifier, used as the rule property.",
							Computed:    true,
						},
						"readable_name": schema.StringAttribute{
							Description: "The display name shown in the Tenable.io UI.",
							Computed:    true,
						},
						"operators": schema.ListAttribute{
							Description: "The operators this attribute supports.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"control": schema.SingleNestedAttribute{
							Description: "How a value for this attribute is collected and validated.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "The control type (entry, dropdown, dropdown_multi).",
									Computed:    true,
								},
								"regex": schema.StringAttribute{
									Description: "Validation pattern for entry controls.",
									Computed:    true,
								},
								"readable_regex": schema.StringAttribute{
									Description: "Human-readable example of a valid value.",
									Computed:    true,
								},
								"list": schema.ListNestedAttribute{
									Description: "Selectable options for dropdown controls.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"name":  schema.StringAttribute{Computed: true},
											"value": schema.StringAttribute{Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *TagAssetFiltersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagAssetFiltersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListAssetTagFilters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Asset Tag Filters", err.Error())
		return
	}

	filters := make([]TagAssetFilterModel, len(result.Filters))
	for i, f := range result.Filters {
		operators := make([]types.String, len(f.Operators))
		for j, op := range f.Operators {
			operators[j] = types.StringValue(op)
		}

		list := make([]TagAssetFilterListEntryModel, len(f.Control.List))
		for j, e := range f.Control.List {
			list[j] = TagAssetFilterListEntryModel{
				Name:  types.StringValue(e.Name),
				Value: types.StringValue(e.Value),
			}
		}

		filters[i] = TagAssetFilterModel{
			Name:         types.StringValue(f.Name),
			ReadableName: types.StringValue(f.ReadableName),
			Operators:    operators,
			Control: TagAssetFilterControlModel{
				Type:          types.StringValue(f.Control.Type),
				Regex:         types.StringValue(f.Control.Regex),
				ReadableRegex: types.StringValue(f.Control.ReadableRegex),
				List:          list,
			},
		}
	}

	state := TagAssetFiltersDataSourceModel{Filters: filters}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
