package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &AgentGroupResource{}
	_ resource.ResourceWithImportState = &AgentGroupResource{}
)

type AgentGroupResource struct {
	client *client.Client
}

type AgentGroupResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	OwnerID     types.Int64  `tfsdk:"owner_id"`
	Owner       types.String `tfsdk:"owner"`
	Shared      types.Int64  `tfsdk:"shared"`
	AgentsCount types.Int64  `tfsdk:"agents_count"`
}

func NewAgentGroupResource() resource.Resource {
	return &AgentGroupResource{}
}

func (r *AgentGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_group"
}

func (r *AgentGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an agent group in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the agent group.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the agent group. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_id": schema.Int64Attribute{
				Description: "The ID of the agent group owner.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the agent group.",
				Computed:    true,
			},
			"shared": schema.Int64Attribute{
				Description: "Whether the agent group is shared.",
				Computed:    true,
			},
			"agents_count": schema.Int64Attribute{
				Description: "Number of agents in the group.",
				Computed:    true,
			},
		},
	}
}

func (r *AgentGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateAgentGroup(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Agent Group", err.Error())
		return
	}

	r.mapToState(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetAgentGroup(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Agent Group", err.Error())
		return
	}

	r.mapToState(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentGroupResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Agent groups cannot be renamed. Changing the name forces replacement.",
	)
}

func (r *AgentGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAgentGroup(ctx, int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error Deleting Agent Group", err.Error())
		return
	}
}

func (r *AgentGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Agent Group ID", fmt.Sprintf("Could not parse agent group ID %q: %s", req.ID, err))
		return
	}

	result, err := r.client.GetAgentGroup(ctx, int(groupID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Agent Group", err.Error())
		return
	}

	var state AgentGroupResourceModel
	r.mapToState(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentGroupResource) mapToState(ag *client.AgentGroup, state *AgentGroupResourceModel) {
	state.ID = types.Int64Value(int64(ag.ID))
	state.Name = types.StringValue(ag.Name)
	state.OwnerID = types.Int64Value(int64(ag.OwnerID))
	state.Owner = types.StringValue(ag.Owner)
	state.Shared = types.Int64Value(int64(ag.Shared))
	state.AgentsCount = types.Int64Value(int64(ag.AgentsCount))
}
