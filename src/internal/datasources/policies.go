package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &PoliciesDataSource{}

type PoliciesDataSource struct {
	client *client.Client
}

type PoliciesDataSourceModel struct {
	Policies []PolicyItemModel `tfsdk:"policies"`
}

type PolicyItemModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Owner                types.String `tfsdk:"owner"`
	OwnerID              types.Int64  `tfsdk:"owner_id"`
	Visibility           types.String `tfsdk:"visibility"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
	TemplateUUID         types.String `tfsdk:"template_uuid"`
}

// policiesDeprecation is shown by Terraform on every plan that uses the data
// source, and logged on every read for runs whose warnings nobody reads.
const policiesDeprecation = "tenableio_policies is deprecated and will be removed in a future release. " +
	"Use tenableio_scan_templates_custom, which lists the same user-defined scan templates under the name " +
	"the Tenable.io UI uses, or tenableio_scan_template_custom to look one up by name or id."

func NewPoliciesDataSource() datasource.DataSource {
	return &PoliciesDataSource{}
}

func (d *PoliciesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policies"
}

func (d *PoliciesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:        "Retrieves a list of scan policies from Tenable.io.",
		DeprecationMessage: policiesDeprecation,
		Attributes: map[string]schema.Attribute{
			"policies": schema.ListNestedAttribute{
				Description: "List of policies.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.Int64Attribute{Computed: true},
						"uuid":                   schema.StringAttribute{Computed: true},
						"name":                   schema.StringAttribute{Computed: true},
						"description":            schema.StringAttribute{Computed: true},
						"owner":                  schema.StringAttribute{Computed: true},
						"owner_id":               schema.Int64Attribute{Computed: true},
						"visibility":             schema.StringAttribute{Computed: true},
						"creation_date":          schema.Int64Attribute{Computed: true},
						"last_modification_date": schema.Int64Attribute{Computed: true},
						"template_uuid":          schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *PoliciesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PoliciesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, policiesDeprecation)

	result, err := d.client.ListPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Policies", err.Error())
		return
	}

	policies := make([]PolicyItemModel, len(result.Policies))
	for i, p := range result.Policies {
		policies[i] = PolicyItemModel{
			ID:                   types.Int64Value(int64(p.ID)),
			UUID:                 types.StringValue(p.UUID),
			Name:                 types.StringValue(p.Name),
			Description:          types.StringValue(valueOrEmpty(p.Description)),
			Owner:                types.StringValue(p.Owner),
			OwnerID:              types.Int64Value(int64(p.OwnerID)),
			Visibility:           types.StringValue(p.Visibility),
			CreationDate:         types.Int64Value(int64(p.CreationDate)),
			LastModificationDate: types.Int64Value(int64(p.LastModificationDate)),
			TemplateUUID:         types.StringValue(p.TemplateUUID),
		}
	}

	config.Policies = policies
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
