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
	_ datasource.DataSource                   = &ScanTemplateCustomDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ScanTemplateCustomDataSource{}
)

// ScanTemplateCustomDataSource looks up one user-defined scan template -- what
// the API calls a policy -- by name or id, typically one built in the Tenable.io
// UI, so that a tenableio_scan can be based on it.
//
// It reads only GET /policies. The list is documented field by field and
// carries everything this data source exposes; GET /policies/{id} returns a
// different object (the settings) whose exact shape is not yet confirmed.
type ScanTemplateCustomDataSource struct {
	client *client.Client
}

// ScanTemplateCustomModel is a user-defined scan template, shared by the lookup
// and by the list data source.
type ScanTemplateCustomModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	TemplateUUID         types.String `tfsdk:"template_uuid"`
	Owner                types.String `tfsdk:"owner"`
	OwnerID              types.Int64  `tfsdk:"owner_id"`
	Visibility           types.String `tfsdk:"visibility"`
	Shared               types.Bool   `tfsdk:"shared"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
}

func NewScanTemplateCustomDataSource() datasource.DataSource {
	return &ScanTemplateCustomDataSource{}
}

func (d *ScanTemplateCustomDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scan_template_custom"
}

func (d *ScanTemplateCustomDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a user-defined scan template -- one listed under Scan Templates > User Defined in " +
			"the Tenable.io UI, which the API calls a policy -- by its name or its id. Use its `id` as the " +
			"`policy_id` of a `tenableio_scan`. The lookup fails unless exactly one template matches.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The template's id. Exactly one of `id` and `name` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The template's name, matched exactly, including case. Exactly one of `id` and " +
					"`name` must be set. Tenable.io does not require names to be unique; when two templates share " +
					"one, look the template up by `id` instead.",
				Optional: true,
				Computed: true,
			},
			"description": schema.StringAttribute{
				Description: "The template's description. Null when Tenable.io reports none.",
				Computed:    true,
			},
			"template_uuid": schema.StringAttribute{
				Description: "The uuid of the Tenable-provided template this one is based on, from the policy " +
					"template list (see `tenableio_policy_template_tenable`).",
				Computed: true,
			},
			"owner":    schema.StringAttribute{Description: "Username of the template's owner.", Computed: true},
			"owner_id": schema.Int64Attribute{Description: "Id of the template's owner.", Computed: true},
			"visibility": schema.StringAttribute{
				Description: "`private` or `shared`.",
				Computed:    true,
			},
			"shared": schema.BoolAttribute{
				Description: "Whether the template is shared with users other than its owner.",
				Computed:    true,
			},
			"creation_date": schema.Int64Attribute{
				Description: "Unix timestamp when the template was created.",
				Computed:    true,
			},
			"last_modification_date": schema.Int64Attribute{
				Description: "Unix timestamp when the template was last modified.",
				Computed:    true,
			},
		},
	}
}

func (d *ScanTemplateCustomDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScanTemplateCustomDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config ScanTemplateCustomModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An unknown value may still resolve to null, so it settles nothing yet.
	if config.ID.IsUnknown() || config.Name.IsUnknown() {
		return
	}
	if config.ID.IsNull() == config.Name.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid Scan Template Lookup",
			"Set exactly one of id and name.")
	}
}

func (d *ScanTemplateCustomDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ScanTemplateCustomModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing User-Defined Scan Templates", err.Error())
		return
	}

	query := customTemplateQuery{id: config.ID.ValueInt64Pointer(), name: config.Name.ValueString()}
	matches := query.match(result.Policies)
	switch len(matches) {
	case 1:
	case 0:
		resp.Diagnostics.AddError("No Matching User-Defined Scan Template",
			fmt.Sprintf("No user-defined scan template visible to these API keys has %s.\n\n%s",
				query, describePolicies(result.Policies)))
		return
	default:
		resp.Diagnostics.AddError("Ambiguous User-Defined Scan Template",
			fmt.Sprintf("%d user-defined scan templates have %s. Look the template up by id instead, or "+
				"rename one of them in Tenable.io.\n\nMatching templates:\n%s",
				len(matches), query, describePolicyLines(matches)))
		return
	}

	state := scanTemplateCustomModel(matches[0])
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// customTemplateQuery is what a lookup filters on. Exactly one of id and name
// is set; ValidateConfig guarantees it.
type customTemplateQuery struct {
	id   *int64
	name string
}

func (q customTemplateQuery) match(policies []client.PolicySummary) []client.PolicySummary {
	var matches []client.PolicySummary
	for _, p := range policies {
		if q.id != nil && int64(p.ID) == *q.id || q.id == nil && p.Name == q.name {
			matches = append(matches, p)
		}
	}
	return matches
}

func (q customTemplateQuery) String() string {
	if q.id != nil {
		return fmt.Sprintf("id %d", *q.id)
	}
	return fmt.Sprintf("name %q", q.name)
}

func describePolicies(policies []client.PolicySummary) string {
	if len(policies) == 0 {
		return "These API keys can see no user-defined scan templates at all. Check that the keys belong to " +
			"a user who can view them, or that the template has been shared with that user."
	}
	return "Available templates:\n" + describePolicyLines(policies)
}

// describePolicyLines lists templates one per line, sorted by name, so an error
// shows the user every valid lookup key instead of sending them to the UI.
func describePolicyLines(policies []client.PolicySummary) string {
	lines := make([]string, len(policies))
	for i, p := range policies {
		lines[i] = fmt.Sprintf("  - name %q, id %d", p.Name, p.ID)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func scanTemplateCustomModel(p client.PolicySummary) ScanTemplateCustomModel {
	description := types.StringNull()
	if p.Description != nil {
		description = types.StringValue(*p.Description)
	}
	return ScanTemplateCustomModel{
		ID:                   types.Int64Value(int64(p.ID)),
		Name:                 types.StringValue(p.Name),
		Description:          description,
		TemplateUUID:         types.StringValue(p.TemplateUUID),
		Owner:                types.StringValue(p.Owner),
		OwnerID:              types.Int64Value(int64(p.OwnerID)),
		Visibility:           types.StringValue(p.Visibility),
		Shared:               types.BoolValue(p.Shared == 1),
		CreationDate:         types.Int64Value(int64(p.CreationDate)),
		LastModificationDate: types.Int64Value(int64(p.LastModificationDate)),
	}
}

// valueOrEmpty reads a "string or null" field where null and empty mean the
// same thing to the caller.
func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
