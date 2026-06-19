package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
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

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scan policy (template) in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"template_uuid": schema.StringAttribute{
				Description: "The UUID of the base scan template for this policy.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the policy.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the policy.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
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

	createReq := client.PolicyCreateRequest{
		UUID: plan.TemplateUUID.ValueString(),
		Settings: client.PolicySettings{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Visibility:  plan.Visibility.ValueString(),
		},
	}

	result, err := r.client.CreatePolicy(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Policy", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(result.PolicyID))

	detail, err := r.client.GetPolicy(ctx, result.PolicyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Policy After Create", err.Error())
		return
	}

	plan.Owner = types.StringValue(detail.Owner)
	plan.OwnerID = types.Int64Value(int64(detail.OwnerID))
	plan.CreationDate = types.Int64Value(int64(detail.CreationDate))
	plan.LastModificationDate = types.Int64Value(int64(detail.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	detail, err := r.client.GetPolicy(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Policy", err.Error())
		return
	}

	state.Name = types.StringValue(detail.Name)
	state.Description = types.StringValue(detail.Description)
	state.Visibility = types.StringValue(detail.Visibility)
	state.Owner = types.StringValue(detail.Owner)
	state.OwnerID = types.Int64Value(int64(detail.OwnerID))
	state.CreationDate = types.Int64Value(int64(detail.CreationDate))
	state.LastModificationDate = types.Int64Value(int64(detail.LastModificationDate))
	if detail.TemplateUUID != "" {
		state.TemplateUUID = types.StringValue(detail.TemplateUUID)
	}

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

	updateReq := client.PolicyUpdateRequest{
		Settings: client.PolicySettings{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Visibility:  plan.Visibility.ValueString(),
		},
	}
	if plan.TemplateUUID.ValueString() != state.TemplateUUID.ValueString() {
		updateReq.UUID = plan.TemplateUUID.ValueString()
	}

	policyID := int(state.ID.ValueInt64())
	if err := r.client.UpdatePolicy(ctx, policyID, updateReq); err != nil {
		resp.Diagnostics.AddError("Error Updating Policy", err.Error())
		return
	}

	detail, err := r.client.GetPolicy(ctx, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Policy After Update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Owner = types.StringValue(detail.Owner)
	plan.OwnerID = types.Int64Value(int64(detail.OwnerID))
	plan.CreationDate = types.Int64Value(int64(detail.CreationDate))
	plan.LastModificationDate = types.Int64Value(int64(detail.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

	detail, err := r.client.GetPolicy(ctx, int(policyID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Policy", err.Error())
		return
	}

	state := PolicyResourceModel{
		ID:                   types.Int64Value(policyID),
		TemplateUUID:         types.StringValue(detail.TemplateUUID),
		Name:                 types.StringValue(detail.Name),
		Description:          types.StringValue(detail.Description),
		Visibility:           types.StringValue(detail.Visibility),
		Owner:                types.StringValue(detail.Owner),
		OwnerID:              types.Int64Value(int64(detail.OwnerID)),
		CreationDate:         types.Int64Value(int64(detail.CreationDate)),
		LastModificationDate: types.Int64Value(int64(detail.LastModificationDate)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
