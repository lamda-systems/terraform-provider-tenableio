package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &TemplatesTenableDataSource{}

// TemplatesTenableDataSource lists every template in one Tenable-provided
// template list. Like TemplateTenableDataSource it backs one data source per
// list.
type TemplatesTenableDataSource struct {
	client       *client.Client
	templateType client.TemplateType
}

type TemplatesTenableDataSourceModel struct {
	Templates []TemplateTenableModel `tfsdk:"templates"`
}

func NewScanTemplatesTenableDataSource() datasource.DataSource {
	return &TemplatesTenableDataSource{templateType: client.TemplateTypeScan}
}

func NewPolicyTemplatesTenableDataSource() datasource.DataSource {
	return &TemplatesTenableDataSource{templateType: client.TemplateTypePolicy}
}

func (d *TemplatesTenableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_templates_tenable", req.ProviderTypeName, d.templateType)
}

func (d *TemplatesTenableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Lists the Tenable-provided templates in the `%s` template list. "+
			"To pick one template, `tenableio_%s_template_tenable` looks it up by name or title.",
			d.templateType, d.templateType),
		Attributes: map[string]schema.Attribute{
			"templates": schema.ListNestedAttribute{
				Description: "The templates, in the order Tenable.io returns them.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":              schema.StringAttribute{Computed: true},
						"name":              schema.StringAttribute{Computed: true},
						"title":             schema.StringAttribute{Computed: true},
						"description":       schema.StringAttribute{Computed: true},
						"order":             schema.Int64Attribute{Computed: true},
						"unsupported":       schema.BoolAttribute{Computed: true},
						"cloud_only":        schema.BoolAttribute{Computed: true},
						"subscription_only": schema.BoolAttribute{Computed: true},
						"manager_only":      schema.BoolAttribute{Computed: true},
						"is_agent":          schema.BoolAttribute{Computed: true},
						"is_was":            schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TemplatesTenableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TemplatesTenableDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListTemplates(ctx, d.templateType)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Tenable-Provided Templates", err.Error())
		return
	}

	templates := make([]TemplateTenableModel, len(result.Templates))
	for i, t := range result.Templates {
		templates[i] = templateModel(t)
	}

	state := TemplatesTenableDataSourceModel{Templates: templates}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
