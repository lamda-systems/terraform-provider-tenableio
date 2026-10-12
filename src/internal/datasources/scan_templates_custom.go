package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &ScanTemplatesCustomDataSource{}

// ScanTemplatesCustomDataSource lists every user-defined scan template the API
// keys can see. It replaces tenableio_policies.
type ScanTemplatesCustomDataSource struct {
	client *client.Client
}

type ScanTemplatesCustomDataSourceModel struct {
	Templates []ScanTemplateCustomModel `tfsdk:"templates"`
}

func NewScanTemplatesCustomDataSource() datasource.DataSource {
	return &ScanTemplatesCustomDataSource{}
}

func (d *ScanTemplatesCustomDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scan_templates_custom"
}

func (d *ScanTemplatesCustomDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the user-defined scan templates -- Scan Templates > User Defined in the Tenable.io " +
			"UI, which the API calls policies -- that the API keys can see. To pick one, " +
			"`tenableio_scan_template_custom` looks it up by name or id.",
		Attributes: map[string]schema.Attribute{
			"templates": schema.ListNestedAttribute{
				Description: "The templates, in the order Tenable.io returns them.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.Int64Attribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"description":            schema.StringAttribute{Computed: true},
						"template_uuid":          schema.StringAttribute{Computed: true},
						"owner":                  schema.StringAttribute{Computed: true},
						"owner_id":               schema.Int64Attribute{Computed: true},
						"visibility":             schema.StringAttribute{Computed: true},
						"shared":                 schema.BoolAttribute{Computed: true},
						"creation_date":          schema.Int64Attribute{Computed: true},
						"last_modification_date": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ScanTemplatesCustomDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScanTemplatesCustomDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing User-Defined Scan Templates", err.Error())
		return
	}

	templates := make([]ScanTemplateCustomModel, len(result.Policies))
	for i, p := range result.Policies {
		templates[i] = scanTemplateCustomModel(p)
	}

	state := ScanTemplatesCustomDataSourceModel{Templates: templates}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
