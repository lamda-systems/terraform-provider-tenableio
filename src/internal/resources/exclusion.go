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
	_ resource.Resource                = &ExclusionResource{}
	_ resource.ResourceWithImportState = &ExclusionResource{}
)

type ExclusionResource struct {
	client *client.Client
}

type ExclusionScheduleModel struct {
	Enabled   types.Bool   `tfsdk:"enabled"`
	Starttime types.String `tfsdk:"starttime"`
	Endtime   types.String `tfsdk:"endtime"`
	Timezone  types.String `tfsdk:"timezone"`
	RRules    types.String `tfsdk:"rrules"`
}

type ExclusionResourceModel struct {
	ID                   types.Int64             `tfsdk:"id"`
	Name                 types.String            `tfsdk:"name"`
	Description          types.String            `tfsdk:"description"`
	Members              types.String            `tfsdk:"members"`
	NetworkID            types.String            `tfsdk:"network_id"`
	Schedule             *ExclusionScheduleModel `tfsdk:"schedule"`
	CreationDate         types.Int64             `tfsdk:"creation_date"`
	LastModificationDate types.Int64             `tfsdk:"last_modification_date"`
}

func NewExclusionResource() resource.Resource {
	return &ExclusionResource{}
}

func (r *ExclusionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exclusion"
}

func (r *ExclusionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scan exclusion in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the exclusion.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the exclusion.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the exclusion.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"members": schema.StringAttribute{
				Description: "Comma-separated list of targets to exclude (IPs, hostnames, CIDR ranges).",
				Required:    true,
			},
			"network_id": schema.StringAttribute{
				Description: "The UUID of the network the exclusion applies to.",
				Optional:    true,
			},
			"schedule": schema.SingleNestedAttribute{
				Description: "Schedule for when the exclusion is active.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether the schedule is enabled.",
						Required:    true,
					},
					"starttime": schema.StringAttribute{
						Description: "Start time (format: YYYY-MM-DD HH:MM:SS).",
						Optional:    true,
					},
					"endtime": schema.StringAttribute{
						Description: "End time (format: YYYY-MM-DD HH:MM:SS).",
						Optional:    true,
					},
					"timezone": schema.StringAttribute{
						Description: "Timezone for the schedule.",
						Optional:    true,
					},
					"rrules": schema.StringAttribute{
						Description: "Recurrence rules (iCal RRULE format).",
						Optional:    true,
					},
				},
			},
			"creation_date": schema.Int64Attribute{
				Description: "Unix timestamp when the exclusion was created.",
				Computed:    true,
			},
			"last_modification_date": schema.Int64Attribute{
				Description: "Unix timestamp when the exclusion was last modified.",
				Computed:    true,
			},
		},
	}
}

func (r *ExclusionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExclusionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExclusionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.ExclusionCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Members:     plan.Members.ValueString(),
	}

	if !plan.NetworkID.IsNull() {
		createReq.NetworkID = plan.NetworkID.ValueString()
	}

	if plan.Schedule != nil {
		createReq.Schedule = &client.ExclusionSchedule{
			Enabled: plan.Schedule.Enabled.ValueBool(),
		}
		if !plan.Schedule.Starttime.IsNull() {
			createReq.Schedule.Starttime = plan.Schedule.Starttime.ValueString()
		}
		if !plan.Schedule.Endtime.IsNull() {
			createReq.Schedule.Endtime = plan.Schedule.Endtime.ValueString()
		}
		if !plan.Schedule.Timezone.IsNull() {
			createReq.Schedule.Timezone = plan.Schedule.Timezone.ValueString()
		}
		if !plan.Schedule.RRules.IsNull() {
			createReq.Schedule.RRules = plan.Schedule.RRules.ValueString()
		}
	}

	result, err := r.client.CreateExclusion(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Exclusion", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(result.ID))
	plan.CreationDate = types.Int64Value(int64(result.CreationDate))
	plan.LastModificationDate = types.Int64Value(int64(result.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExclusionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ExclusionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetExclusion(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Exclusion", err.Error())
		return
	}

	state.Name = types.StringValue(result.Name)
	state.Description = types.StringValue(result.Description)
	state.Members = types.StringValue(result.Members)
	state.CreationDate = types.Int64Value(int64(result.CreationDate))
	state.LastModificationDate = types.Int64Value(int64(result.LastModificationDate))

	state.NetworkID = readOptionalString(state.NetworkID, result.NetworkID)

	if state.Schedule != nil {
		state.Schedule = &ExclusionScheduleModel{
			Enabled:   types.BoolValue(result.Schedule.Enabled),
			Starttime: readOptionalString(state.Schedule.Starttime, result.Schedule.Starttime),
			Endtime:   readOptionalString(state.Schedule.Endtime, result.Schedule.Endtime),
			Timezone:  readOptionalString(state.Schedule.Timezone, result.Schedule.Timezone),
			RRules:    readOptionalString(state.Schedule.RRules, result.Schedule.RRules),
		}
	} else if result.Schedule.Enabled {
		state.Schedule = &ExclusionScheduleModel{
			Enabled:   types.BoolValue(result.Schedule.Enabled),
			Starttime: types.StringValue(result.Schedule.Starttime),
			Endtime:   types.StringValue(result.Schedule.Endtime),
			Timezone:  types.StringValue(result.Schedule.Timezone),
			RRules:    types.StringValue(result.Schedule.RRules),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ExclusionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExclusionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ExclusionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.ExclusionUpdateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Members:     plan.Members.ValueString(),
	}

	if !plan.NetworkID.IsNull() {
		updateReq.NetworkID = plan.NetworkID.ValueString()
	}

	if plan.Schedule != nil {
		updateReq.Schedule = &client.ExclusionSchedule{
			Enabled: plan.Schedule.Enabled.ValueBool(),
		}
		if !plan.Schedule.Starttime.IsNull() {
			updateReq.Schedule.Starttime = plan.Schedule.Starttime.ValueString()
		}
		if !plan.Schedule.Endtime.IsNull() {
			updateReq.Schedule.Endtime = plan.Schedule.Endtime.ValueString()
		}
		if !plan.Schedule.Timezone.IsNull() {
			updateReq.Schedule.Timezone = plan.Schedule.Timezone.ValueString()
		}
		if !plan.Schedule.RRules.IsNull() {
			updateReq.Schedule.RRules = plan.Schedule.RRules.ValueString()
		}
	}

	exclusionID := int(state.ID.ValueInt64())
	result, err := r.client.UpdateExclusion(ctx, exclusionID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Exclusion", err.Error())
		return
	}

	plan.ID = state.ID
	plan.CreationDate = types.Int64Value(int64(result.CreationDate))
	plan.LastModificationDate = types.Int64Value(int64(result.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ExclusionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteExclusion(ctx, int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error Deleting Exclusion", err.Error())
		return
	}
}

func (r *ExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	exclusionID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Exclusion ID", fmt.Sprintf("Could not parse exclusion ID %q: %s", req.ID, err))
		return
	}

	result, err := r.client.GetExclusion(ctx, int(exclusionID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Exclusion", err.Error())
		return
	}

	state := ExclusionResourceModel{
		ID:                   types.Int64Value(exclusionID),
		Name:                 types.StringValue(result.Name),
		Description:          types.StringValue(result.Description),
		Members:              types.StringValue(result.Members),
		CreationDate:         types.Int64Value(int64(result.CreationDate)),
		LastModificationDate: types.Int64Value(int64(result.LastModificationDate)),
	}

	state.NetworkID = readOptionalString(state.NetworkID, result.NetworkID)

	if result.Schedule.Enabled || result.Schedule.Starttime != "" {
		state.Schedule = &ExclusionScheduleModel{
			Enabled:   types.BoolValue(result.Schedule.Enabled),
			Starttime: types.StringValue(result.Schedule.Starttime),
			Endtime:   types.StringValue(result.Schedule.Endtime),
			Timezone:  types.StringValue(result.Schedule.Timezone),
			RRules:    types.StringValue(result.Schedule.RRules),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
