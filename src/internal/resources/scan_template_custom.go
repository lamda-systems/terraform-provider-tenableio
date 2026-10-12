package resources

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

const scanTemplateCustomType = "tenableio_scan_template_custom"

var (
	_ resource.Resource                   = &ScanTemplateCustomResource{}
	_ resource.ResourceWithImportState    = &ScanTemplateCustomResource{}
	_ resource.ResourceWithModifyPlan     = &ScanTemplateCustomResource{}
	_ resource.ResourceWithValidateConfig = &ScanTemplateCustomResource{}
	_ resource.ResourceWithMoveState      = &ScanTemplateCustomResource{}
)

// complianceTemplates are the Tenable-provided templates whose policies are
// built around audits. They are the compliance flavour of a user-defined
// template, which this resource does not manage: a policy based on one cannot
// be created without at least one audit.
var complianceTemplates = map[string]bool{
	"compliance": true, "agent_compliance": true, "scap": true, "agent_scap": true,
}

// ScanTemplateCustomResource manages a user-defined scan template -- a policy,
// as the API calls it.
//
// The settings, their read-back and their verification are driven by the
// scanSettings table (see scan_template_blocks.go). What lives here is the
// lifecycle and the parts that are not settings: the name and description,
// the plugin families, and the metadata only the policy list reports.
type ScanTemplateCustomResource struct {
	client *client.Client
}

type ScanTemplateCustomResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	TemplateUUID   types.String `tfsdk:"template_uuid"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Discovery      types.Object `tfsdk:"discovery"`
	Assessment     types.Object `tfsdk:"assessment"`
	Report         types.Object `tfsdk:"report"`
	Advanced       types.Object `tfsdk:"advanced"`
	Agent          types.Object `tfsdk:"agent"`
	PluginFamilies types.Set    `tfsdk:"plugin_family"`

	Owner                types.String `tfsdk:"owner"`
	OwnerID              types.Int64  `tfsdk:"owner_id"`
	Visibility           types.String `tfsdk:"visibility"`
	Shared               types.Bool   `tfsdk:"shared"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
}

func (m *ScanTemplateCustomResourceModel) settingBlocks() map[settingBlock]*types.Object {
	return map[settingBlock]*types.Object{
		blockDiscovery:  &m.Discovery,
		blockAssessment: &m.Assessment,
		blockReport:     &m.Report,
		blockAdvanced:   &m.Advanced,
		blockAgent:      &m.Agent,
	}
}

type PluginFamilyModel struct {
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

var pluginFamilyType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":   types.StringType,
	"status": types.StringType,
}}

func NewScanTemplateCustomResource() resource.Resource {
	return &ScanTemplateCustomResource{}
}

func (r *ScanTemplateCustomResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scan_template_custom"
}

func (r *ScanTemplateCustomResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	blocks := settingBlocksSchema()
	blocks["plugin_family"] = schema.SetNestedBlock{
		Description: "Enables or disables a plugin family. Only templates that offer plugin selection, such as " +
			"Advanced Network Scan and Advanced Agent Scan, accept these. Families left out keep the template's status.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description: "The plugin family name, exactly as Tenable.io shows it, for example `Backdoors`.",
					Required:    true,
					Validators:  []validator.String{NoSurroundingWhitespace()},
				},
				"status": schema.StringAttribute{
					Description: "`enabled` or `disabled`.",
					Required:    true,
					Validators:  []validator.String{OneOf("enabled", "disabled")},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a user-defined scan template -- Scan Templates > User Defined in the Tenable.io UI, " +
			"which the API calls a policy. Settings left out of the configuration keep the defaults of the " +
			"Tenable-provided template it is based on, and are neither tracked nor changed. Removing a setting " +
			"or a plugin family from the configuration replaces the template, because Tenable.io keeps any " +
			"setting an update leaves out; set `create_before_destroy` if scans use it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The template's id. Use it as the `policy_id` of a `tenableio_scan`.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"template_uuid": schema.StringAttribute{
				Description: "The uuid of the Tenable-provided template to base this one on, from the policy " +
					"template list (`tenableio_policy_template_tenable`). Changing it replaces the template.",
				Required:   true,
				Validators: []validator.String{NoSurroundingWhitespace()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The template's name.",
				Required:    true,
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"description": schema.StringAttribute{
				Description: "The template's description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"owner": schema.StringAttribute{
				Description:   "Username of the template's owner.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"owner_id": schema.Int64Attribute{
				Description:   "Id of the template's owner.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"visibility": schema.StringAttribute{
				Description:   "`private` or `shared`, as Tenable.io reports it.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"shared": schema.BoolAttribute{
				Description:   "Whether the template is shared with users other than its owner.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"creation_date": schema.Int64Attribute{
				Description:   "Unix timestamp when the template was created.",
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"last_modification_date": schema.Int64Attribute{
				Description: "Unix timestamp when the template was last modified.",
				Computed:    true,
			},
		},
		Blocks: blocks,
	}
}

func (r *ScanTemplateCustomResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *ScanTemplateCustomResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateModes(&resp.Diagnostics, config.settingBlocks())
	validateDependencies(&resp.Diagnostics, config.settingBlocks())

	families, diags := pluginFamilies(ctx, config.PluginFamilies)
	resp.Diagnostics.Append(diags...)
	seen := map[string]bool{}
	for _, f := range families {
		if f.Name.IsUnknown() || f.Name.IsNull() {
			continue
		}
		if seen[f.Name.ValueString()] {
			resp.Diagnostics.AddAttributeError(path.Root("plugin_family"), "Duplicate Plugin Family",
				fmt.Sprintf("The plugin family %q is configured more than once.", f.Name.ValueString()))
		}
		seen[f.Name.ValueString()] = true
	}
}

// ModifyPlan forces replacement where an update cannot apply the change, and
// checks the configuration against the template it is based on before any
// API call that would change something.
func (r *ScanTemplateCustomResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !req.State.Raw.IsNull() {
		var state ScanTemplateCustomResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.RequiresReplace.Append(removedSettings(state.settingBlocks(), plan.settingBlocks())...)
		if removed := removedPluginFamilies(ctx, state.PluginFamilies, plan.PluginFamilies); removed {
			resp.RequiresReplace.Append(path.Root("plugin_family"))
		}
	}

	r.checkTemplate(ctx, &plan, &resp.Diagnostics)
}

// checkTemplate rejects a template uuid from the wrong list, a compliance
// template, and settings documented only for the other kind of template --
// all of which Tenable.io would otherwise reject or silently discard.
func (r *ScanTemplateCustomResource) checkTemplate(ctx context.Context, plan *ScanTemplateCustomResourceModel, diags *diag.Diagnostics) {
	if r.client == nil || plan.TemplateUUID.IsNull() || plan.TemplateUUID.IsUnknown() {
		return
	}
	uuid := plan.TemplateUUID.ValueString()

	list, err := r.client.ListTemplates(ctx, client.TemplateTypePolicy)
	if err != nil {
		diags.AddError("Error Listing Tenable-Provided Templates", err.Error())
		return
	}
	var template *client.Template
	for i := range list.Templates {
		if list.Templates[i].UUID == uuid {
			template = &list.Templates[i]
		}
	}

	if template == nil {
		hint := "Look the template up with tenableio_policy_template_tenable."
		if scans, err := r.client.ListTemplates(ctx, client.TemplateTypeScan); err == nil {
			for _, t := range scans.Templates {
				if t.UUID == uuid {
					hint = fmt.Sprintf("It is the uuid of %q in the scan template list, which is for scans. "+
						"Look the template up with tenableio_policy_template_tenable instead of "+
						"tenableio_scan_template_tenable.", t.Title)
				}
			}
		}
		diags.AddAttributeError(path.Root("template_uuid"), "Not a Policy Template",
			fmt.Sprintf("%q is not in Tenable.io's policy template list, which is where a user-defined scan "+
				"template must take its base from. %s", uuid, hint))
		return
	}

	if complianceTemplates[template.Name] {
		diags.AddAttributeError(path.Root("template_uuid"), "Compliance Template Not Supported",
			fmt.Sprintf("%q (%s) is a compliance template. Templates based on it need audits, which %s does "+
				"not configure.", template.Title, template.Name, scanTemplateCustomType))
		return
	}

	isAgent := template.IsAgent != nil && *template.IsAgent
	kind := "a network template"
	if isAgent {
		kind = "an agent template"
	}
	for _, s := range familyMismatches(configuredSettings(plan.settingBlocks()), isAgent) {
		diags.AddAttributeError(s.path(), "Setting Not Available for This Template",
			fmt.Sprintf("%s is documented only for %s templates, and %q is %s. Tenable.io discards a setting "+
				"the template does not define, so this one would never take effect.",
				s.label(), map[bool]string{true: "network", false: "agent"}[isAgent], template.Title, kind))
	}
}

func (r *ScanTemplateCustomResource) request(ctx context.Context, plan *ScanTemplateCustomResourceModel) (client.PolicyRequest, []configuredSetting, diag.Diagnostics) {
	configured := configuredSettings(plan.settingBlocks())
	req := client.PolicyRequest{
		TemplateUUID: plan.TemplateUUID.ValueString(),
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Settings:     settingsRequest(configured),
	}
	families, diags := pluginFamilies(ctx, plan.PluginFamilies)
	if len(families) > 0 {
		req.Plugins = make(map[string]client.PluginFamily, len(families))
		for _, f := range families {
			req.Plugins[f.Name.ValueString()] = client.PluginFamily{Status: f.Status.ValueString()}
		}
	}
	return req, configured, diags
}

func (r *ScanTemplateCustomResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, configured, diags := r.request(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreatePolicy(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Scan Template", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(created.PolicyID))

	details, summary, err := r.fetch(ctx, created.PolicyID)
	applyScanTemplateComputed(&plan, summary, nil)
	// Recorded even when the read-back failed, so the template is tainted and
	// replaced on the next apply rather than leaked.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Scan Template After Create", err.Error())
		return
	}
	r.verify(ctx, &resp.Diagnostics, &plan, configured, details)
}

func (r *ScanTemplateCustomResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, diags := r.read(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// read refreshes state from the API. The details response is the source of
// truth for what the template does; the list is the only source of its
// metadata. Settings and plugin families are refreshed only where state tracks
// them, and kept where the response does not report them.
func (r *ScanTemplateCustomResource) read(ctx context.Context, state *ScanTemplateCustomResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	policyID := int(state.ID.ValueInt64())

	details, summary, err := r.fetch(ctx, policyID)
	if err != nil {
		if client.IsNotFound(err) {
			return false, diags
		}
		diags.AddError("Error Reading Scan Template", err.Error())
		return false, diags
	}

	state.Name = readReportedString(state.Name, details.Setting("name"))
	state.Description = readReportedString(state.Description, details.Setting("description"))
	switch {
	case details.TemplateUUID != nil:
		state.TemplateUUID = types.StringValue(*details.TemplateUUID)
	case summary != nil:
		state.TemplateUUID = types.StringValue(summary.TemplateUUID)
	}

	diags.Append(readSettingBlocks(state.settingBlocks(), details)...)
	families, d := readPluginFamilies(ctx, state.PluginFamilies, details)
	diags.Append(d...)
	state.PluginFamilies = families

	prior := *state
	applyScanTemplateComputed(state, summary, &prior)
	return true, diags
}

func (r *ScanTemplateCustomResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, configured, diags := r.request(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID := int(state.ID.ValueInt64())
	if err := r.client.UpdatePolicy(ctx, policyID, body); err != nil {
		resp.Diagnostics.AddError("Error Updating Scan Template", err.Error())
		return
	}

	details, summary, err := r.fetch(ctx, policyID)
	plan.ID = state.ID
	applyScanTemplateComputed(&plan, summary, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Scan Template After Update", err.Error())
		return
	}
	r.verify(ctx, &resp.Diagnostics, &plan, configured, details)
}

func (r *ScanTemplateCustomResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScanTemplateCustomResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePolicy(ctx, int(state.ID.ValueInt64())); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Scan Template", err.Error())
	}
}

// ImportState imports a template by id. Settings and plugin families start
// untracked: nothing in the response says which of them the configuration
// means to manage, and adopting all of them would make every one the
// configuration leaves out a removal, and so a replacement.
func (r *ScanTemplateCustomResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Scan Template ID",
			fmt.Sprintf("Could not parse scan template ID %q: %s", req.ID, err))
		return
	}

	state := ScanTemplateCustomResourceModel{
		ID: types.Int64Value(policyID),
		// Known, so that the reported name and description are adopted.
		Name:           types.StringValue(""),
		Description:    types.StringValue(""),
		TemplateUUID:   types.StringNull(),
		PluginFamilies: types.SetNull(pluginFamilyType),
	}
	for block, value := range state.settingBlocks() {
		*value = types.ObjectNull(settingBlockAttrTypes(block))
	}

	found, diags := r.read(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Scan Template Not Found",
			fmt.Sprintf("No user-defined scan template with id %d exists, or these API keys cannot see it.", policyID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// MoveState accepts a `moved` block from the deprecated tenableio_policy, which
// manages the same objects. The policy keeps its id, so nothing is recreated;
// settings and plugin families start untracked, as after an import, and the
// next plan adds whatever the configuration sets.
func (r *ScanTemplateCustomResource) MoveState(ctx context.Context) []resource.StateMover {
	var source resource.SchemaResponse
	NewPolicyResource().Schema(ctx, resource.SchemaRequest{}, &source)

	return []resource.StateMover{{
		SourceSchema: &source.Schema,
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "tenableio_policy" {
				return
			}
			var old PolicyResourceModel
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &old)...)
			if resp.Diagnostics.HasError() {
				return
			}
			moved := ScanTemplateCustomResourceModel{
				ID:                   old.ID,
				TemplateUUID:         old.TemplateUUID,
				Name:                 old.Name,
				Description:          old.Description,
				PluginFamilies:       types.SetNull(pluginFamilyType),
				Owner:                old.Owner,
				OwnerID:              old.OwnerID,
				Visibility:           old.Visibility,
				Shared:               types.BoolNull(),
				CreationDate:         old.CreationDate,
				LastModificationDate: old.LastModificationDate,
			}
			for block, value := range moved.settingBlocks() {
				*value = types.ObjectNull(settingBlockAttrTypes(block))
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
		},
	}}
}

// fetch reads both halves of a template: the details, and its list entry. The
// list entry is nil when the list does not carry the template.
func (r *ScanTemplateCustomResource) fetch(ctx context.Context, policyID int) (*client.PolicyDetails, *client.PolicySummary, error) {
	details, err := r.client.GetPolicyDetails(ctx, policyID)
	if err != nil {
		return nil, nil, err
	}
	summary, err := r.client.FindPolicy(ctx, policyID)
	if err != nil {
		return nil, nil, err
	}
	return details, summary, nil
}

// verify compares what the configuration asked for with what the details
// response reports, after state has been recorded. A difference fails the
// apply; a setting the response does not report at all is listed in a warning,
// because it cannot be verified either way.
func (r *ScanTemplateCustomResource) verify(ctx context.Context, diags *diag.Diagnostics, plan *ScanTemplateCustomResourceModel,
	configured []configuredSetting, details *client.PolicyDetails,
) {
	source := echoSource(http.MethodGet, fmt.Sprintf("%s/policies/%d", r.client.BaseURL, plan.ID.ValueInt64()),
		details.PresentKeys, r.client.ProxyForURL())

	if name := details.Setting("name"); name != nil {
		requireEcho(diags, scanTemplateCustomType, "name", plan.Name.ValueString(), *name, source)
	}
	if description := details.Setting("description"); description != nil {
		requireEcho(diags, scanTemplateCustomType, "description", plan.Description.ValueString(), *description, source)
	}

	unreported := verifySettings(diags, configured, details, source)

	families, d := pluginFamilies(ctx, plan.PluginFamilies)
	diags.Append(d...)
	for _, f := range families {
		name := f.Name.ValueString()
		reported, ok := details.Plugins[name]
		if !ok {
			unreported = append(unreported, fmt.Sprintf("plugin_family %q", name))
			continue
		}
		requireEcho(diags, scanTemplateCustomType, fmt.Sprintf("the status of plugin family %q", name),
			f.Status.ValueString(), reported.Status, source)
	}

	if len(unreported) > 0 {
		diags.AddWarning("Scan Template Settings Not Reported Back",
			fmt.Sprintf("Tenable.io accepted the scan template %q, but its details response does not report: %s.\n\n"+
				"The response leaves out settings that depend on a toggle that is off -- the web crawler "+
				"settings while assessment.scan_webapps is off, for example -- and Tenable.io discards settings "+
				"the template does not define. Terraform keeps the configured values, which it cannot verify, and "+
				"will not notice them being changed outside Terraform.\n\n%s",
				plan.Name.ValueString(), sortedList(unreported), source))
	}
}

// applyScanTemplateComputed settles the computed-only attributes from the list
// entry. When the list does not carry the template, prior state is kept where
// there is one, and on create everything is null rather than invented.
func applyScanTemplateComputed(m *ScanTemplateCustomResourceModel, summary *client.PolicySummary, prior *ScanTemplateCustomResourceModel) {
	switch {
	case summary != nil:
		m.Owner = types.StringValue(summary.Owner)
		m.OwnerID = types.Int64Value(int64(summary.OwnerID))
		m.Visibility = types.StringValue(summary.Visibility)
		m.Shared = types.BoolValue(summary.Shared == 1)
		m.CreationDate = types.Int64Value(int64(summary.CreationDate))
		m.LastModificationDate = types.Int64Value(int64(summary.LastModificationDate))
	case prior != nil:
		m.Owner = prior.Owner
		m.OwnerID = prior.OwnerID
		m.Visibility = prior.Visibility
		m.Shared = prior.Shared
		m.CreationDate = prior.CreationDate
		m.LastModificationDate = prior.LastModificationDate
	default:
		m.Owner = types.StringNull()
		m.OwnerID = types.Int64Null()
		m.Visibility = types.StringNull()
		m.Shared = types.BoolNull()
		m.CreationDate = types.Int64Null()
		m.LastModificationDate = types.Int64Null()
	}
}

func pluginFamilies(ctx context.Context, set types.Set) ([]PluginFamilyModel, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var families []PluginFamilyModel
	diags := set.ElementsAs(ctx, &families, false)
	return families, diags
}

// readPluginFamilies refreshes the status of each family state tracks. A
// family the response does not report keeps its state value.
func readPluginFamilies(ctx context.Context, current types.Set, details *client.PolicyDetails) (types.Set, diag.Diagnostics) {
	families, diags := pluginFamilies(ctx, current)
	if diags.HasError() || current.IsNull() || current.IsUnknown() {
		return current, diags
	}
	for i, f := range families {
		if reported, ok := details.Plugins[f.Name.ValueString()]; ok {
			families[i].Status = types.StringValue(reported.Status)
		}
	}
	refreshed, d := types.SetValueFrom(ctx, pluginFamilyType, families)
	diags.Append(d...)
	return refreshed, diags
}

// removedPluginFamilies reports whether the plan drops a family state tracks.
// Tenable.io keeps the status of a family an update leaves out, so dropping
// one cannot be applied in place.
func removedPluginFamilies(ctx context.Context, state, plan types.Set) bool {
	before, _ := pluginFamilies(ctx, state)
	if plan.IsUnknown() {
		return false
	}
	after, _ := pluginFamilies(ctx, plan)
	kept := map[string]bool{}
	for _, f := range after {
		kept[f.Name.ValueString()] = true
	}
	for _, f := range before {
		if !kept[f.Name.ValueString()] {
			return true
		}
	}
	return false
}
