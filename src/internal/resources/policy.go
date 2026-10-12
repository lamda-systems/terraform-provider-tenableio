package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &PolicyResource{}
	_ resource.ResourceWithImportState = &PolicyResource{}
)

type PolicyResource struct {
	client *client.Client
}

type PolicyResourceModel struct {
	ID                   types.Int64  `tfsdk:"id"`
	TemplateUUID         types.String `tfsdk:"template_uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Visibility           types.String `tfsdk:"visibility"`
	Owner                types.String `tfsdk:"owner"`
	OwnerID              types.Int64  `tfsdk:"owner_id"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	LastModificationDate types.Int64  `tfsdk:"last_modification_date"`
}

// policyDeprecation is shown by Terraform on every plan that uses the resource,
// and logged on every operation for runs whose warnings nobody reads.
const policyDeprecation = "tenableio_policy is deprecated and will be removed in a future release. " +
	"Use tenableio_scan_template_custom, which manages the same user-defined scan templates under the name " +
	"the Tenable.io UI uses, and adds their settings and plugin families. Rename the resource and add a " +
	"moved block (from = tenableio_policy.NAME, to = tenableio_scan_template_custom.NAME); the template " +
	"keeps its id and nothing is recreated."

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:        "Manages a scan policy (template) in Tenable.io.",
		DeprecationMessage: policyDeprecation,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"template_uuid": schema.StringAttribute{
				Description: "The UUID of the base scan template for this policy. Changing it replaces the " +
					"policy: Tenable.io ignores it on update.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the policy.",
				Required:    true,
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"description": schema.StringAttribute{
				Description: "The description of the policy.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"visibility": schema.StringAttribute{
				Description: "The visibility of the policy: shared or private.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("private"),
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the policy.",
				Computed:    true,
			},
			"owner_id": schema.Int64Attribute{
				Description: "The ID of the policy owner.",
				Computed:    true,
			},
			"creation_date": schema.Int64Attribute{
				Description: "Unix timestamp when the policy was created.",
				Computed:    true,
			},
			"last_modification_date": schema.Int64Attribute{
				Description: "Unix timestamp when the policy was last modified.",
				Computed:    true,
			},
		},
	}
}

func (r *PolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, policyDeprecation)

	createReq := client.PolicyRequest{
		TemplateUUID: plan.TemplateUUID.ValueString(),
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Visibility:   plan.Visibility.ValueString(),
	}

	result, err := r.client.CreatePolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Policy", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(result.PolicyID))

	details, summary, err := r.fetch(ctx, result.PolicyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Policy After Create", err.Error())
		return
	}

	applyPolicyComputed(&plan, summary)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.verify(&resp.Diagnostics, &plan, details, summary)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, policyDeprecation)

	details, summary, err := r.fetch(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Policy", err.Error())
		return
	}

	mapPolicyState(&state, details, summary)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, policyDeprecation)

	updateReq := client.PolicyRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Visibility:  plan.Visibility.ValueString(),
	}

	policyID := int(state.ID.ValueInt64())
	if err := r.client.UpdatePolicy(ctx, policyID, updateReq); err != nil {
		resp.Diagnostics.AddError("Error Updating Policy", err.Error())
		return
	}

	details, summary, err := r.fetch(ctx, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Policy After Update", err.Error())
		return
	}

	plan.ID = state.ID
	applyPolicyComputed(&plan, summary)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.verify(&resp.Diagnostics, &plan, details, summary)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePolicy(ctx, int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error Deleting Policy", err.Error())
		return
	}
}

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy ID", fmt.Sprintf("Could not parse policy ID %q: %s", req.ID, err))
		return
	}

	details, summary, err := r.fetch(ctx, int(policyID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Policy", err.Error())
		return
	}

	state := PolicyResourceModel{
		ID:           types.Int64Value(policyID),
		TemplateUUID: types.StringNull(),
		Name:         types.StringValue(""),
		Description:  types.StringValue(""),
		Visibility:   types.StringValue("private"),
	}
	mapPolicyState(&state, details, summary)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fetch reads a policy's details, which carry its name and description, and
// its list entry, which carries everything else this resource exposes.
func (r *PolicyResource) fetch(ctx context.Context, policyID int) (*client.PolicyDetails, *client.PolicySummary, error) {
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

// mapPolicyState maps both responses into state, for Read and ImportState. A
// field neither response reports keeps its state value.
func mapPolicyState(state *PolicyResourceModel, details *client.PolicyDetails, summary *client.PolicySummary) {
	state.Name = readReportedString(state.Name, details.Setting("name"))
	state.Description = readReportedString(state.Description, details.Setting("description"))
	if details.TemplateUUID != nil {
		state.TemplateUUID = types.StringValue(*details.TemplateUUID)
	}
	if summary != nil {
		state.Visibility = types.StringValue(summary.Visibility)
		if state.TemplateUUID.IsNull() {
			state.TemplateUUID = types.StringValue(summary.TemplateUUID)
		}
	}
	applyPolicyComputed(state, summary)
}

// applyPolicyComputed settles the computed-only attributes from the list
// entry, or keeps what state has when the list does not carry the policy.
func applyPolicyComputed(m *PolicyResourceModel, summary *client.PolicySummary) {
	if summary == nil {
		// Still unknown only on create or update, where a value must be known
		// once the apply finishes: null, rather than an invented zero.
		if m.Owner.IsUnknown() {
			m.Owner = types.StringNull()
		}
		if m.OwnerID.IsUnknown() {
			m.OwnerID = types.Int64Null()
		}
		if m.CreationDate.IsUnknown() {
			m.CreationDate = types.Int64Null()
		}
		if m.LastModificationDate.IsUnknown() {
			m.LastModificationDate = types.Int64Null()
		}
		return
	}
	m.Owner = types.StringValue(summary.Owner)
	m.OwnerID = types.Int64Value(int64(summary.OwnerID))
	m.CreationDate = types.Int64Value(int64(summary.CreationDate))
	m.LastModificationDate = types.Int64Value(int64(summary.LastModificationDate))
}

// verify fails the apply when the stored name, description or visibility
// differs from the plan. Each is checked only where a response reports it.
func (r *PolicyResource) verify(diags *diag.Diagnostics, plan *PolicyResourceModel, details *client.PolicyDetails, summary *client.PolicySummary) {
	if name := details.Setting("name"); name != nil {
		requireEcho(diags, "tenableio_policy", "name", plan.Name.ValueString(), *name)
	}
	if description := details.Setting("description"); description != nil {
		requireEcho(diags, "tenableio_policy", "description", plan.Description.ValueString(), *description)
	}
	if summary != nil {
		requireEcho(diags, "tenableio_policy", "visibility", plan.Visibility.ValueString(), summary.Visibility)
	}
}
