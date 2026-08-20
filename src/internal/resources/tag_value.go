package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &TagValueResource{}
	_ resource.ResourceWithImportState = &TagValueResource{}
)

type TagValueResource struct {
	client *client.Client
}

type TagValueResourceModel struct {
	UUID                types.String          `tfsdk:"uuid"`
	CategoryUUID        types.String          `tfsdk:"category_uuid"`
	CategoryName        types.String          `tfsdk:"category_name"`
	CategoryDescription types.String          `tfsdk:"category_description"`
	Value               types.String          `tfsdk:"value"`
	Description         types.String          `tfsdk:"description"`
	Type                types.String          `tfsdk:"type"`
	Filters             *TagValueFiltersModel `tfsdk:"filters"`
	CreatedAt           types.String          `tfsdk:"created_at"`
	CreatedBy           types.String          `tfsdk:"created_by"`
	UpdatedAt           types.String          `tfsdk:"updated_at"`
	UpdatedBy           types.String          `tfsdk:"updated_by"`
}

type TagValueFiltersModel struct {
	Asset TagValueAssetRulesModel `tfsdk:"asset"`
}

type TagValueAssetRulesModel struct {
	And []TagValueRuleModel `tfsdk:"and"`
	Or  []TagValueRuleModel `tfsdk:"or"`
}

type TagValueRuleModel struct {
	Property string   `tfsdk:"property"`
	Operator string   `tfsdk:"operator"`
	Values   []string `tfsdk:"values"`
}

func NewTagValueResource() resource.Resource {
	return &TagValueResource{}
}

func (r *TagValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_value"
}

func (r *TagValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a tag value in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Description: "The UUID of the tag value.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"category_uuid": schema.StringAttribute{
				Description: "The UUID of the tag category, usually a reference to a tenableio_tag_category resource. " +
					"Preferred over category_name: the reference orders the two resources correctly and survives category renames. " +
					"Set either this or category_name. Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"category_name": schema.StringAttribute{
				Description: "The name of the tag category. Tenable.io creates the category when no category of that name exists, " +
					"which is convenient for ad-hoc use but risky otherwise: a bare name creates no dependency on a tenableio_tag_category " +
					"resource (so the category may be created twice), tag values sharing a not-yet-existing name can race each other, and " +
					"renaming the category in Tenable.io forces every value pinned to it by name to be replaced. Prefer category_uuid. " +
					"Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{NoSurroundingWhitespace()},
			},
			"category_description": schema.StringAttribute{
				Description: "Description for a new category (used only when category_name creates a new category). Cannot be changed after creation.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{NoSurroundingWhitespace()},
			},
			"value": schema.StringAttribute{
				Description: "The tag value.",
				Required:    true,
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"description": schema.StringAttribute{
				Description: "The description of the tag value.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"type": schema.StringAttribute{
				Description: "The type of the tag value (static or dynamic).",
				Computed:    true,
			},
			"filters": schema.SingleNestedAttribute{
				Description: "Asset-matching rules that make this a dynamic tag: Tenable.io automatically applies the tag to every asset the rules match. " +
					"Omit for a static tag. Use the tenableio_tag_asset_filters data source to discover valid rule properties and operators. " +
					"Removing filters from an existing dynamic tag forces a new resource. " +
					"Rule changes made outside Terraform while the tag stays dynamic are not detected.",
				Optional: true,
				PlanModifiers: []planmodifier.Object{
					filtersRemovalRequiresReplace{},
				},
				Attributes: map[string]schema.Attribute{
					"asset": schema.SingleNestedAttribute{
						Description: "Rules matched against asset attributes. At least one of `and` or `or` must be set.",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"and": schema.ListNestedAttribute{
								Description:  "Rules that must all match for the tag to apply.",
								Optional:     true,
								NestedObject: tagRuleNestedObject(),
							},
							"or": schema.ListNestedAttribute{
								Description:  "Rules of which any one matching applies the tag.",
								Optional:     true,
								NestedObject: tagRuleNestedObject(),
							},
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_by": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func tagRuleNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"property": schema.StringAttribute{
				Description: "The asset attribute or tag to match, e.g. `asset_class`, `ipv4`, `operating_system`.",
				Required:    true,
			},
			"operator": schema.StringAttribute{
				Description: "The operator to apply, e.g. `equals`. Supported operators depend on the property.",
				Required:    true,
			},
			"values": schema.ListAttribute{
				Description: "The value(s) to match against.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// filtersRemovalRequiresReplace forces a new resource when filters are removed
// from a dynamic tag. The API does not document whether omitting filters on
// update preserves or clears the rules, so recreating is the only reliable way
// back to a static tag.
type filtersRemovalRequiresReplace struct{}

func (filtersRemovalRequiresReplace) Description(_ context.Context) string {
	return "Removing filters from a dynamic tag forces a new resource."
}

func (m filtersRemovalRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (filtersRemovalRequiresReplace) PlanModifyObject(_ context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.StateValue.IsNull() && req.ConfigValue.IsNull() {
		resp.RequiresReplace = true
	}
}

func (m *TagValueFiltersModel) toRequest() *client.TagValueFilters {
	if m == nil {
		return nil
	}
	return &client.TagValueFilters{
		Asset: client.TagAssetRules{
			And: toClientRules(m.Asset.And),
			Or:  toClientRules(m.Asset.Or),
		},
	}
}

func toClientRules(rules []TagValueRuleModel) []client.TagRule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]client.TagRule, len(rules))
	for i, r := range rules {
		out[i] = client.TagRule{Property: r.Property, Operator: r.Operator, Values: r.Values}
	}
	return out
}

func filtersFromResponse(rules *client.TagAssetRules) *TagValueFiltersModel {
	if rules == nil || (len(rules.And) == 0 && len(rules.Or) == 0) {
		return nil
	}
	return &TagValueFiltersModel{
		Asset: TagValueAssetRulesModel{
			And: fromClientRules(rules.And),
			Or:  fromClientRules(rules.Or),
		},
	}
}

func fromClientRules(rules []client.TagRule) []TagValueRuleModel {
	if len(rules) == 0 {
		return nil
	}
	out := make([]TagValueRuleModel, len(rules))
	for i, r := range rules {
		out[i] = TagValueRuleModel{Property: r.Property, Operator: r.Operator, Values: r.Values}
	}
	return out
}

func (r *TagValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TagValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.TagValueCreateRequest{
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueString(),
		Filters:     plan.Filters.toRequest(),
	}

	if !plan.CategoryUUID.IsNull() {
		createReq.CategoryUUID = plan.CategoryUUID.ValueString()
	}
	if !plan.CategoryName.IsNull() {
		createReq.CategoryName = plan.CategoryName.ValueString()
	}
	if !plan.CategoryDescription.IsNull() {
		createReq.CategoryDescription = plan.CategoryDescription.ValueString()
	}

	result, err := r.client.CreateTagValue(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Tag Value", err.Error())
		return
	}

	r.applyComputed(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	requireEcho(&resp.Diagnostics, "tenableio_tag_value", "value", plan.Value.ValueString(), result.Value)
	requireEcho(&resp.Diagnostics, "tenableio_tag_value", "description", plan.Description.ValueString(), result.Description)
}

func (r *TagValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TagValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetTagValue(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tag Value", err.Error())
		return
	}

	r.mapToState(result, &state)
	reconcileFilters(result, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TagValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TagValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateTagValue(ctx, state.UUID.ValueString(), client.TagValueUpdateRequest{
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueString(),
		Filters:     plan.Filters.toRequest(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Tag Value", err.Error())
		return
	}

	r.applyComputed(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	requireEcho(&resp.Diagnostics, "tenableio_tag_value", "value", plan.Value.ValueString(), result.Value)
	requireEcho(&resp.Diagnostics, "tenableio_tag_value", "description", plan.Description.ValueString(), result.Description)
}

func (r *TagValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TagValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTagValue(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Tag Value", err.Error())
		return
	}
}

func (r *TagValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	result, err := r.client.GetTagValue(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Tag Value", err.Error())
		return
	}

	var state TagValueResourceModel
	r.mapToState(result, &state)
	reconcileFilters(result, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// reconcileFilters settles what filters end up in state after a read or
// import. The API echoes filters in a different shape than it accepts (a
// JSON-formatted string whose rules use "field" keys and short operator
// codes), so configured rules cannot be compared verbatim against the
// response. The configured rules stay authoritative while the tag remains
// dynamic; only coarse drift is handled: a tag turned static out-of-band
// clears the rules (so the next plan re-adds them), and a tag with no rules
// in state (import, or made dynamic out-of-band) adopts the response rules.
func reconcileFilters(tv *client.TagValue, state *TagValueResourceModel, diags *diag.Diagnostics) {
	if tv.Type == "static" {
		state.Filters = nil
		return
	}
	if state.Filters != nil {
		return
	}
	rules, err := tv.Filters.ParseAssetRules()
	if err != nil {
		diags.AddWarning(
			"Unparseable Tag Filters",
			fmt.Sprintf("The tag value %q is dynamic but its filters could not be parsed and were left out of state: %s", tv.UUID, err),
		)
		return
	}
	state.Filters = filtersFromResponse(rules)
}

func (r *TagValueResource) mapToState(tv *client.TagValue, state *TagValueResourceModel) {
	state.UUID = types.StringValue(tv.UUID)
	state.CategoryUUID = types.StringValue(tv.CategoryUUID)
	state.CategoryName = types.StringValue(tv.CategoryName)
	state.Value = types.StringValue(tv.Value)
	state.Description = types.StringValue(tv.Description)
	state.Type = types.StringValue(tv.Type)
	state.CreatedAt = types.StringValue(tv.CreatedAt)
	state.CreatedBy = types.StringValue(tv.CreatedBy)
	state.UpdatedAt = types.StringValue(tv.UpdatedAt)
	state.UpdatedBy = types.StringValue(tv.UpdatedBy)
}

// applyComputed copies back only the attributes Terraform could not know at
// plan time. Attributes that came from configuration are deliberately left as
// planned.
//
// Terraform requires the value applied to equal the value planned for every
// attribute that was known during planning. Writing the API's echo over a known
// plan value therefore turns any server-side normalisation -- trimming, case
// folding, a silently substituted default -- into an unrecoverable "Provider
// produced inconsistent result after apply". Divergence is not swallowed: the
// next Read maps the whole object and surfaces it as an ordinary diff the user
// can act on.
//
// Read and ImportState use mapToState instead, which maps everything, because
// there the API is the source of truth.
func (r *TagValueResource) applyComputed(tv *client.TagValue, state *TagValueResourceModel) {
	state.UUID = types.StringValue(tv.UUID)
	state.Type = types.StringValue(tv.Type)
	state.CreatedAt = types.StringValue(tv.CreatedAt)
	state.CreatedBy = types.StringValue(tv.CreatedBy)
	state.UpdatedAt = types.StringValue(tv.UpdatedAt)
	state.UpdatedBy = types.StringValue(tv.UpdatedBy)

	// category_uuid and category_name are Optional+Computed with no default, so
	// whichever one the configuration left out is unknown at plan time and has
	// to be filled from the response. The one that was configured stays exactly
	// as planned -- notably, a category_name that Tenable.io normalises must not
	// be written back here.
	if state.CategoryUUID.IsUnknown() {
		state.CategoryUUID = types.StringValue(tv.CategoryUUID)
	}
	if state.CategoryName.IsUnknown() {
		state.CategoryName = types.StringValue(tv.CategoryName)
	}
}
