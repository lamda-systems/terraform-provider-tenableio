package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &TagCategoriesDataSource{}

type TagCategoriesDataSource struct {
	client *client.Client
}

type TagCategoriesDataSourceModel struct {
	Categories []TagCategoryItemModel `tfsdk:"categories"`
}

type TagCategoryItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewTagCategoriesDataSource() datasource.DataSource {
	return &TagCategoriesDataSource{}
}

func (d *TagCategoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_categories"
}

func (d *TagCategoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of tag categories from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"categories": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":        schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"created_at":  schema.StringAttribute{Computed: true},
						"updated_at":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TagCategoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListTagCategories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Tag Categories", err.Error())
		return
	}

	categories := make([]TagCategoryItemModel, len(result.Categories))
	for i, c := range result.Categories {
		categories[i] = TagCategoryItemModel{
			UUID:        types.StringValue(c.UUID),
			Name:        types.StringValue(c.Name),
			Description: types.StringValue(c.Description),
			CreatedAt:   types.StringValue(c.CreatedAt),
			UpdatedAt:   types.StringValue(c.UpdatedAt),
		}
	}

	state := TagCategoriesDataSourceModel{Categories: categories}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
