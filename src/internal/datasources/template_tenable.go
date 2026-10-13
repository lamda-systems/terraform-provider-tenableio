package datasources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ datasource.DataSource                   = &TemplateTenableDataSource{}
	_ datasource.DataSourceWithValidateConfig = &TemplateTenableDataSource{}
)

// TemplateTenableDataSource looks up one Tenable-provided template by name or
// title, so configuration never has to carry a 52-character uuid. It backs two
// data sources, one per template list, because a scan and a user-defined scan
// template draw their uuid from different lists (see client.TemplateType).
type TemplateTenableDataSource struct {
	client       *client.Client
	templateType client.TemplateType
}

// TemplateTenableModel is a Tenable-provided template, shared by the lookup
// and by the list data sources.
type TemplateTenableModel struct {
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Title            types.String `tfsdk:"title"`
	Description      types.String `tfsdk:"description"`
	Order            types.Int64  `tfsdk:"order"`
	Unsupported      types.Bool   `tfsdk:"unsupported"`
	CloudOnly        types.Bool   `tfsdk:"cloud_only"`
	SubscriptionOnly types.Bool   `tfsdk:"subscription_only"`
	ManagerOnly      types.Bool   `tfsdk:"manager_only"`
	IsAgent          types.Bool   `tfsdk:"is_agent"`
	IsWAS            types.Bool   `tfsdk:"is_was"`
}

func NewScanTemplateTenableDataSource() datasource.DataSource {
	return &TemplateTenableDataSource{templateType: client.TemplateTypeScan}
}

func NewPolicyTemplateTenableDataSource() datasource.DataSource {
	return &TemplateTenableDataSource{templateType: client.TemplateTypePolicy}
}

func (d *TemplateTenableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_template_tenable", req.ProviderTypeName, d.templateType)
}

func (d *TemplateTenableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: templateLookupDescription(d.templateType),
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The template's short system name, such as `advanced` or `agent_basic`. " +
					"Exactly one of `name` and `title` must be set. Matched case-insensitively.",
				Optional: true,
				Computed: true,
			},
			"title": schema.StringAttribute{
				Description: "The template's name as the Tenable.io UI shows it, such as `Advanced Network Scan`. " +
					"Exactly one of `name` and `title` must be set. Matched case-insensitively. Some titles are " +
					"shared by a network and an agent template; set `is_agent` to choose between them.",
				Optional: true,
				Computed: true,
			},
			"is_agent": schema.BoolAttribute{
				Description: "Whether the template is for agent scans. Set it to narrow a lookup whose title " +
					"matches both a network and an agent template. Tenable.io reports null for non-agent " +
					"templates; that is exposed as `false`.",
				Optional: true,
				Computed: true,
			},
			"uuid":        schema.StringAttribute{Description: "The template uuid.", Computed: true},
			"description": schema.StringAttribute{Description: "What the template is for.", Computed: true},
			"order": schema.Int64Attribute{
				Description: "Display order in the Tenable.io UI. Null when Tenable.io reports none.",
				Computed:    true,
			},
			"unsupported":       schema.BoolAttribute{Description: "Whether the template is unsupported.", Computed: true},
			"cloud_only":        schema.BoolAttribute{Description: "Whether the template is available only in the cloud.", Computed: true},
			"subscription_only": schema.BoolAttribute{Description: "Whether the template is available only to subscribers.", Computed: true},
			"manager_only":      schema.BoolAttribute{Description: "Whether only a manager can use the template.", Computed: true},
			"is_was": schema.BoolAttribute{
				Description: "Whether the template is only for Tenable Web App Scanning. Null from Tenable.io is exposed as `false`.",
				Computed:    true,
			},
		},
	}
}

func templateLookupDescription(templateType client.TemplateType) string {
	use := "Use its `uuid` as the `template_uuid` of a `tenableio_scan`."
	if templateType == client.TemplateTypePolicy {
		use = "Use its `uuid` as the `template_uuid` of a `tenableio_scan_template_custom`."
	}
	return fmt.Sprintf("Looks up a Tenable-provided template from the `%s` template list by its name or its "+
		"title, instead of by uuid. %s The lookup fails unless exactly one template matches.", templateType, use)
}

func (d *TemplateTenableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TemplateTenableDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config TemplateTenableModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An unknown value may still resolve to null, so it settles nothing yet.
	if config.Name.IsUnknown() || config.Title.IsUnknown() {
		return
	}
	if config.Name.IsNull() == config.Title.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid Template Lookup",
			"Set exactly one of name and title.")
	}
}

func (d *TemplateTenableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TemplateTenableModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListTemplates(ctx, d.templateType)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Tenable-Provided Templates", err.Error())
		return
	}

	query := templateQuery{
		name:    config.Name.ValueString(),
		title:   config.Title.ValueString(),
		isAgent: config.IsAgent.ValueBoolPointer(),
	}
	matches := query.match(result.Templates)
	switch len(matches) {
	case 1:
	case 0:
		resp.Diagnostics.AddError("No Matching Tenable-Provided Template",
			fmt.Sprintf("No template in the %s list matches %s.\n\nAvailable templates:\n%s",
				d.templateType, query, describeTemplates(result.Templates)))
		return
	default:
		resp.Diagnostics.AddError("Ambiguous Tenable-Provided Template",
			fmt.Sprintf("%d templates in the %s list match %s. Look the template up by name, or set "+
				"is_agent to choose between them.\n\nMatching templates:\n%s",
				len(matches), d.templateType, query, describeTemplates(matches)))
		return
	}

	state := templateModel(matches[0])
	// Terraform requires a configured value to come back unchanged, and the
	// match is case-insensitive, so keep the lookup keys as written.
	if !config.Name.IsNull() {
		state.Name = config.Name
	}
	if !config.Title.IsNull() {
		state.Title = config.Title
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// templateQuery is what a lookup filters on. Exactly one of name and title is
// set; ValidateConfig guarantees it.
type templateQuery struct {
	name    string
	title   string
	isAgent *bool
}

func (q templateQuery) match(templates []client.Template) []client.Template {
	var matches []client.Template
	for _, t := range templates {
		if q.name != "" && !strings.EqualFold(t.Name, q.name) {
			continue
		}
		if q.title != "" && !strings.EqualFold(t.Title, q.title) {
			continue
		}
		if q.isAgent != nil && isTrue(t.IsAgent) != *q.isAgent {
			continue
		}
		matches = append(matches, t)
	}
	return matches
}

func (q templateQuery) String() string {
	s := fmt.Sprintf("title %q", q.title)
	if q.name != "" {
		s = fmt.Sprintf("name %q", q.name)
	}
	if q.isAgent != nil {
		s += fmt.Sprintf(" with is_agent = %t", *q.isAgent)
	}
	return s
}

// describeTemplates lists templates one per line, sorted by name, so an error
// shows the user every valid lookup key instead of sending them to the API.
func describeTemplates(templates []client.Template) string {
	lines := make([]string, len(templates))
	for i, t := range templates {
		line := fmt.Sprintf("  - name %q, title %q", t.Name, t.Title)
		if isTrue(t.IsAgent) {
			line += ", is_agent = true"
		}
		lines[i] = line
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func templateModel(t client.Template) TemplateTenableModel {
	order := types.Int64Null()
	if t.Order != nil {
		order = types.Int64Value(int64(*t.Order))
	}
	return TemplateTenableModel{
		UUID:             types.StringValue(t.UUID),
		Name:             types.StringValue(t.Name),
		Title:            types.StringValue(t.Title),
		Description:      types.StringValue(t.Description),
		Order:            order,
		Unsupported:      types.BoolValue(t.Unsupported),
		CloudOnly:        types.BoolValue(t.CloudOnly),
		SubscriptionOnly: types.BoolValue(t.SubscriptionOnly),
		ManagerOnly:      types.BoolValue(t.ManagerOnly),
		IsAgent:          types.BoolValue(isTrue(t.IsAgent)),
		IsWAS:            types.BoolValue(isTrue(t.IsWAS)),
	}
}

// isTrue reads a "boolean or null" flag, where null means no.
func isTrue(b *bool) bool {
	return b != nil && *b
}
