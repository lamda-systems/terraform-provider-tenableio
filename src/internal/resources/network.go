package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &NetworkResource{}
	_ resource.ResourceWithImportState = &NetworkResource{}
)

type NetworkResource struct {
	client *client.Client
}

type NetworkResourceModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	AssetsTTLDays     types.Int64  `tfsdk:"assets_ttl_days"`
	IsDefault         types.Bool   `tfsdk:"is_default"`
	CreatedBy         types.String `tfsdk:"created_by"`
	CreatedInSeconds  types.Int64  `tfsdk:"created_in_seconds"`
	ModifiedInSeconds types.Int64  `tfsdk:"modified_in_seconds"`
	ScannerCount      types.Int64  `tfsdk:"scanner_count"`
}

func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

func (r *NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a network in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Description: "The UUID of the network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the network.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"assets_ttl_days": schema.Int64Attribute{
				Description: "The number of days to keep assets (14-365).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(180),
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether this is the default network.",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the network.",
				Computed:    true,
			},
			"created_in_seconds": schema.Int64Attribute{
				Description: "Unix timestamp when the network was created.",
				Computed:    true,
			},
			"modified_in_seconds": schema.Int64Attribute{
				Description: "Unix timestamp when the network was last modified.",
				Computed:    true,
			},
			"scanner_count": schema.Int64Attribute{
				Description: "Number of scanners assigned to this network.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.NetworkCreateRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		AssetsTTLDays: int(plan.AssetsTTLDays.ValueInt64()),
	}

	result, err := r.client.CreateNetwork(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Network", err.Error())
		return
	}

	r.mapNetworkToState(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetNetwork(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Network", err.Error())
		return
	}

	r.mapNetworkToState(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.NetworkUpdateRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		AssetsTTLDays: int(plan.AssetsTTLDays.ValueInt64()),
	}

	result, err := r.client.UpdateNetwork(ctx, state.UUID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Network", err.Error())
		return
	}

	r.mapNetworkToState(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNetwork(ctx, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Network", err.Error())
		return
	}
}

func (r *NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	result, err := r.client.GetNetwork(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Network", err.Error())
		return
	}

	var state NetworkResourceModel
	r.mapNetworkToState(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) mapNetworkToState(n *client.Network, state *NetworkResourceModel) {
	state.UUID = types.StringValue(n.UUID)
	state.Name = types.StringValue(n.Name)
	state.Description = types.StringValue(n.Description)
	state.AssetsTTLDays = types.Int64Value(int64(n.AssetsTTLDays))
	state.IsDefault = types.BoolValue(n.IsDefault)
	state.CreatedBy = types.StringValue(n.CreatedBy)
	state.CreatedInSeconds = types.Int64Value(int64(n.CreatedInSeconds))
	state.ModifiedInSeconds = types.Int64Value(int64(n.ModifiedInSeconds))
	state.ScannerCount = types.Int64Value(int64(n.ScannerCount))
}
