package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &ScanResource{}
	_ resource.ResourceWithImportState = &ScanResource{}
)

type ScanResource struct {
	client *client.Client
}

type ScanResourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	TemplateUUID    types.String `tfsdk:"template_uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	PolicyID        types.Int64  `tfsdk:"policy_id"`
	FolderID        types.Int64  `tfsdk:"folder_id"`
	ScannerID       types.Int64  `tfsdk:"scanner_id"`
	TextTargets     types.String `tfsdk:"text_targets"`
	TagTargets      types.List   `tfsdk:"tag_targets"`
	FileTargets     types.String `tfsdk:"file_targets"`
	Launch          types.String `tfsdk:"launch"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Starttime       types.String `tfsdk:"starttime"`
	RRules          types.String `tfsdk:"rrules"`
	Timezone        types.String `tfsdk:"timezone"`
	Emails          types.String `tfsdk:"emails"`
	ScanTimeWindow  types.Int64  `tfsdk:"scan_time_window"`
	Status          types.String `tfsdk:"status"`
	CreationDate    types.Int64  `tfsdk:"creation_date"`
	LastModifiedDate types.Int64 `tfsdk:"last_modification_date"`
}

func NewScanResource() resource.Resource {
	return &ScanResource{}
}

func (r *ScanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scan"
}

func (r *ScanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scan configuration in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the scan.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "The UUID of the scan (assigned by Tenable.io).",
				Computed:    true,
			},
			"template_uuid": schema.StringAttribute{
				Description: "The UUID of the scan template to use.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the scan.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the scan.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"policy_id": schema.Int64Attribute{
				Description: "The ID of the policy to use for the scan.",
				Optional:    true,
			},
			"folder_id": schema.Int64Attribute{
				Description: "The ID of the folder to store the scan in.",
				Optional:    true,
			},
			"scanner_id": schema.Int64Attribute{
				Description: "The ID of the scanner to use.",
				Optional:    true,
			},
			"text_targets": schema.StringAttribute{
				Description: "Comma-separated list of targets to scan (IPs, hostnames, CIDR ranges).",
				Optional:    true,
			},
			"tag_targets": schema.ListAttribute{
				Description: "List of tag UUIDs identifying assets to scan.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"file_targets": schema.StringAttribute{
				Description: "Name of an uploaded file containing scan targets.",
				Optional:    true,
			},
			"launch": schema.StringAttribute{
				Description: "Launch schedule type: ON_DEMAND, DAILY, WEEKLY, MONTHLY, YEARLY.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the scan schedule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"starttime": schema.StringAttribute{
				Description: "The start time for the scan schedule (format: YYYYMMDDTHHmmss).",
				Optional:    true,
			},
			"rrules": schema.StringAttribute{
				Description: "Recurrence rules for the scan schedule (iCal RRULE format).",
				Optional:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "The timezone for the scan schedule.",
				Optional:    true,
			},
			"emails": schema.StringAttribute{
				Description: "Comma-separated list of email addresses to notify on scan completion.",
				Optional:    true,
			},
			"scan_time_window": schema.Int64Attribute{
				Description: "Maximum time window in minutes for the scan to run.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the scan.",
				Computed:    true,
			},
			"creation_date": schema.Int64Attribute{
				Description: "Unix timestamp when the scan was created.",
				Computed:    true,
			},
			"last_modification_date": schema.Int64Attribute{
				Description: "Unix timestamp when the scan was last modified.",
				Computed:    true,
			},
		},
	}
}

func (r *ScanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := client.ScanSettings{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	if !plan.PolicyID.IsNull() {
		settings.PolicyID = int(plan.PolicyID.ValueInt64())
	}
	if !plan.FolderID.IsNull() {
		settings.FolderID = int(plan.FolderID.ValueInt64())
	}
	if !plan.ScannerID.IsNull() {
		settings.ScannerID = int(plan.ScannerID.ValueInt64())
	}
	if !plan.TextTargets.IsNull() {
		settings.TextTargets = plan.TextTargets.ValueString()
	}
	if !plan.FileTargets.IsNull() {
		settings.FileTargets = plan.FileTargets.ValueString()
	}
	if !plan.Launch.IsNull() {
		settings.Launch = plan.Launch.ValueString()
	}
	if !plan.Starttime.IsNull() {
		settings.Starttime = plan.Starttime.ValueString()
	}
	if !plan.RRules.IsNull() {
		settings.RRules = plan.RRules.ValueString()
	}
	if !plan.Timezone.IsNull() {
		settings.Timezone = plan.Timezone.ValueString()
	}
	if !plan.Emails.IsNull() {
		settings.Emails = plan.Emails.ValueString()
	}
	if !plan.ScanTimeWindow.IsNull() {
		settings.ScanTimeWindow = int(plan.ScanTimeWindow.ValueInt64())
	}

	if !plan.TagTargets.IsNull() {
		var tags []string
		resp.Diagnostics.Append(plan.TagTargets.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		settings.TagTargets = tags
	}

	createReq := client.ScanCreateRequest{
		UUID:     plan.TemplateUUID.ValueString(),
		Settings: settings,
	}

	result, err := r.client.CreateScan(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Scan", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(result.Scan.ID))
	plan.UUID = types.StringValue(result.Scan.UUID)
	plan.Status = types.StringValue(result.Scan.Status)
	plan.CreationDate = types.Int64Value(int64(result.Scan.CreationDate))
	plan.LastModifiedDate = types.Int64Value(int64(result.Scan.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetScan(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Scan", err.Error())
		return
	}

	info := result.Info
	state.UUID = types.StringValue(info.UUID)
	state.Name = types.StringValue(info.Name)
	state.Description = types.StringValue(info.Description)
	state.Status = types.StringValue(info.Status)
	state.Enabled = types.BoolValue(info.Enabled)
	state.CreationDate = types.Int64Value(int64(info.CreationDate))
	state.LastModifiedDate = types.Int64Value(int64(info.LastModificationDate))

	state.FolderID = readOptionalInt64(state.FolderID, info.FolderID)
	state.ScannerID = readOptionalInt64(state.ScannerID, info.ScannerID)
	state.PolicyID = readOptionalInt64(state.PolicyID, info.PolicyID)
	state.ScanTimeWindow = readOptionalInt64(state.ScanTimeWindow, info.ScanTimeWindow)
	state.TextTargets = readOptionalString(state.TextTargets, info.Targets)
	state.RRules = readOptionalString(state.RRules, info.RRules)
	state.Starttime = readOptionalString(state.Starttime, info.Starttime)
	state.Timezone = readOptionalString(state.Timezone, info.Timezone)
	state.Launch = readOptionalString(state.Launch, info.Launch)
	state.Emails = readOptionalString(state.Emails, info.Emails)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ScanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := client.ScanSettings{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	if !plan.PolicyID.IsNull() {
		settings.PolicyID = int(plan.PolicyID.ValueInt64())
	}
	if !plan.FolderID.IsNull() {
		settings.FolderID = int(plan.FolderID.ValueInt64())
	}
	if !plan.ScannerID.IsNull() {
		settings.ScannerID = int(plan.ScannerID.ValueInt64())
	}
	if !plan.TextTargets.IsNull() {
		settings.TextTargets = plan.TextTargets.ValueString()
	}
	if !plan.FileTargets.IsNull() {
		settings.FileTargets = plan.FileTargets.ValueString()
	}
	if !plan.Launch.IsNull() {
		settings.Launch = plan.Launch.ValueString()
	}
	if !plan.Starttime.IsNull() {
		settings.Starttime = plan.Starttime.ValueString()
	}
	if !plan.RRules.IsNull() {
		settings.RRules = plan.RRules.ValueString()
	}
	if !plan.Timezone.IsNull() {
		settings.Timezone = plan.Timezone.ValueString()
	}
	if !plan.Emails.IsNull() {
		settings.Emails = plan.Emails.ValueString()
	}
	if !plan.ScanTimeWindow.IsNull() {
		settings.ScanTimeWindow = int(plan.ScanTimeWindow.ValueInt64())
	}

	if !plan.TagTargets.IsNull() {
		var tags []string
		resp.Diagnostics.Append(plan.TagTargets.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		settings.TagTargets = tags
	}

	updateReq := client.ScanUpdateRequest{Settings: settings}
	if !plan.TemplateUUID.IsNull() {
		updateReq.UUID = plan.TemplateUUID.ValueString()
	}

	scanID := int(state.ID.ValueInt64())
	if err := r.client.UpdateScan(ctx, scanID, updateReq); err != nil {
		resp.Diagnostics.AddError("Error Updating Scan", err.Error())
		return
	}

	result, err := r.client.GetScan(ctx, scanID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Scan After Update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.UUID = types.StringValue(result.Info.UUID)
	plan.Status = types.StringValue(result.Info.Status)
	plan.CreationDate = types.Int64Value(int64(result.Info.CreationDate))
	plan.LastModifiedDate = types.Int64Value(int64(result.Info.LastModificationDate))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteScan(ctx, int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error Deleting Scan", err.Error())
		return
	}
}

func (r *ScanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	scanID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Scan ID", fmt.Sprintf("Could not parse scan ID %q: %s", req.ID, err))
		return
	}

	result, err := r.client.GetScan(ctx, int(scanID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Scan", err.Error())
		return
	}

	info := result.Info
	// Import starts with zero-value (null) types, so readOptional* will only
	// populate fields that have non-zero API values — correct for import.
	state := ScanResourceModel{
		ID:               types.Int64Value(scanID),
		UUID:             types.StringValue(info.UUID),
		TemplateUUID:     types.StringValue(info.TemplateUUID),
		Name:             types.StringValue(info.Name),
		Description:      types.StringValue(info.Description),
		Enabled:          types.BoolValue(info.Enabled),
		Status:           types.StringValue(info.Status),
		CreationDate:     types.Int64Value(int64(info.CreationDate)),
		LastModifiedDate: types.Int64Value(int64(info.LastModificationDate)),
	}

	state.FolderID = readOptionalInt64(state.FolderID, info.FolderID)
	state.ScannerID = readOptionalInt64(state.ScannerID, info.ScannerID)
	state.PolicyID = readOptionalInt64(state.PolicyID, info.PolicyID)
	state.ScanTimeWindow = readOptionalInt64(state.ScanTimeWindow, info.ScanTimeWindow)
	state.TextTargets = readOptionalString(state.TextTargets, info.Targets)
	state.RRules = readOptionalString(state.RRules, info.RRules)
	state.Starttime = readOptionalString(state.Starttime, info.Starttime)
	state.Timezone = readOptionalString(state.Timezone, info.Timezone)
	state.Launch = readOptionalString(state.Launch, info.Launch)
	state.Emails = readOptionalString(state.Emails, info.Emails)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
