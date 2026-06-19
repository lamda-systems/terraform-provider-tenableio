package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &TagValuesDataSource{}

type TagValuesDataSource struct {
	client *client.Client
}

type TagValuesDataSourceModel struct {
	Values []TagValueItemModel `tfsdk:"values"`
}

type TagValueItemModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Value        types.String `tfsdk:"value"`
	Description  types.String `tfsdk:"description"`
	CategoryUUID types.String `tfsdk:"category_uuid"`
	CategoryName types.String `tfsdk:"category_name"`
	Type         types.String `tfsdk:"type"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewTagValuesDataSource() datasource.DataSource {
	return &TagValuesDataSource{}
}

func (d *TagValuesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_values"
}

func (d *TagValuesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of tag values from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"values": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":          schema.StringAttribute{Computed: true},
						"value":         schema.StringAttribute{Computed: true},
						"description":   schema.StringAttribute{Computed: true},
						"category_uuid": schema.StringAttribute{Computed: true},
						"category_name": schema.StringAttribute{Computed: true},
						"type":          schema.StringAttribute{Computed: true},
						"created_at":    schema.StringAttribute{Computed: true},
						"updated_at":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TagValuesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagValuesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListTagValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Tag Values", err.Error())
		return
	}

	values := make([]TagValueItemModel, len(result.Values))
	for i, v := range result.Values {
		values[i] = TagValueItemModel{
			UUID:         types.StringValue(v.UUID),
			Value:        types.StringValue(v.Value),
			Description:  types.StringValue(v.Description),
			CategoryUUID: types.StringValue(v.CategoryUUID),
			CategoryName: types.StringValue(v.CategoryName),
			Type:         types.StringValue(v.Type),
			CreatedAt:    types.StringValue(v.CreatedAt),
			UpdatedAt:    types.StringValue(v.UpdatedAt),
		}
	}

	state := TagValuesDataSourceModel{Values: values}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
