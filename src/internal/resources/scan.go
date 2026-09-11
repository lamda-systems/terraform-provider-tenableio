package resources

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &ScanResource{}
	_ resource.ResourceWithImportState = &ScanResource{}
)

type ScanResource struct {
	client *client.Client
}

type ScanResourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	TemplateUUID     types.String `tfsdk:"template_uuid"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PolicyID         types.Int64  `tfsdk:"policy_id"`
	FolderID         types.Int64  `tfsdk:"folder_id"`
	ScannerID        types.Int64  `tfsdk:"scanner_id"`
	TextTargets      types.String `tfsdk:"text_targets"`
	TagTargets       types.List   `tfsdk:"tag_targets"`
	FileTargets      types.String `tfsdk:"file_targets"`
	Launch           types.String `tfsdk:"launch"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Starttime        types.String `tfsdk:"starttime"`
	RRules           types.String `tfsdk:"rrules"`
	Timezone         types.String `tfsdk:"timezone"`
	Emails           types.String `tfsdk:"emails"`
	ScanTimeWindow   types.Int64  `tfsdk:"scan_time_window"`
	Status           types.String `tfsdk:"status"`
	CreationDate     types.Int64  `tfsdk:"creation_date"`
	LastModifiedDate types.Int64  `tfsdk:"last_modification_date"`
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
				Validators:  []validator.String{NoSurroundingWhitespace()},
			},
			"description": schema.StringAttribute{
				Description: "The description of the scan.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators:  []validator.String{NoSurroundingWhitespace()},
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
				Description: "Launch schedule type: ON_DEMAND, DAILY, WEEKLY, MONTHLY, YEARLY. " +
					"Defaults to whatever Tenable.io assigns when omitted.",
				Optional: true,
				// Computed as well as Optional because Tenable.io fills this in
				// when the request omits it. Without Computed, the value the API
				// chose would sit in state against a null in configuration and
				// produce a diff on every plan, forever. No Default is declared
				// here on purpose: that would assert which value Tenable.io picks
				// rather than accepting the one it reports.
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	// A response with no scan id is not a created scan, whatever its status
	// code. Every later call addresses the scan by that id, so recording 0
	// would point the next read, update or destroy at /scans/0. Stop here and
	// name the real problem: a body that is valid JSON but not shaped like
	// {"scan": {...}} deserialises into an entirely zero-valued scan, and the
	// echo checks below would then blame the name and description for being
	// empty when it is the whole object that is missing.
	if result.Scan.ID == 0 {
		resp.Diagnostics.AddError(
			"Unreadable Response From Tenable.io",
			fmt.Sprintf(
				"POST %s/scans reported success, but the provider found no scan id in the response "+
					"body. The object it returned carried these keys: %s.\n\n"+
					"Nothing has been recorded in Terraform state, because there is no id to record "+
					"it under. The scan may still exist in Tenable.io: check the target folder and "+
					"delete it by hand before applying again, or the next apply creates a second "+
					"one.\n\n"+
					"A body that is valid JSON but not shaped like {\"scan\": {...}} -- an API "+
					"gateway envelope, or a proxy answering on Tenable's behalf -- arrives here as "+
					"an empty scan. Check that the URL above is the tenant you expect.",
				r.client.BaseURL, keyList(result.Scan.PresentKeys),
			),
		)
		return
	}

	plan.ID = types.Int64Value(int64(result.Scan.ID))

	// Every Computed attribute is unknown at this point and has to be settled
	// with a known value, but the create response is not documented to carry
	// status or launch at all. Starting from null rather than "" or 0 records
	// "Tenable.io did not say" instead of asserting an epoch timestamp or an
	// empty status it never reported.
	plan.UUID = readReportedString(types.StringNull(), result.Scan.UUID)
	plan.Status = readReportedString(types.StringNull(), result.Scan.Status)
	plan.CreationDate = readReportedInt64(types.Int64Null(), result.Scan.CreationDate)
	plan.LastModifiedDate = readReportedInt64(types.Int64Null(), result.Scan.LastModificationDate)

	// launch is Optional+Computed with no default, so it is unknown whenever the
	// configuration leaves it out. Settle it from the response; a configured
	// value is already known and must be left exactly as planned.
	if plan.Launch.IsUnknown() {
		plan.Launch = readReportedString(types.StringNull(), result.Scan.Launch)
	}

	source := echoSource(http.MethodPost, r.client.BaseURL+"/scans", result.Scan.PresentKeys, r.client.ProxyForURL())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	requireEcho(&resp.Diagnostics, "tenableio_scan", "name", plan.Name.ValueString(), result.Scan.Name, source)
	// POST /scans documents description on the response, but only a response
	// that actually carries the key says anything about what was stored. A
	// missing key is not an empty description -- see ScanInfo.Description.
	if echoed := result.Scan.Description; echoed != nil {
		requireEcho(&resp.Diagnostics, "tenableio_scan", "description", plan.Description.ValueString(), *echoed, source)
	}
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
	state.Name = types.StringValue(info.Name)

	// Nothing else here is guaranteed to be in the response: GET /scans/{id}
	// describes a scan result, not the settings that were submitted, and a field
	// it leaves out means "unknown", so state keeps what it holds. Writing the
	// zero value instead is what produced the drift loop -- an absent
	// scan_time_window read back as 0 against a configured 180, re-proposed on
	// every plan -- and, for description, an apply that could never succeed.
	//
	// The cost is accepted: an out-of-band edit to a field this endpoint does
	// not report goes undetected.
	state.UUID = readReportedString(state.UUID, info.UUID)
	state.Description = readReportedString(state.Description, info.Description)
	state.Status = readReportedString(state.Status, info.Status)
	state.Enabled = readReportedBool(state.Enabled, info.Enabled)
	state.CreationDate = readReportedInt64(state.CreationDate, info.CreationDate)
	state.LastModifiedDate = readReportedInt64(state.LastModifiedDate, info.LastModificationDate)

	state.FolderID = readReportedInt64(state.FolderID, info.FolderID)
	state.ScannerID = readReportedInt64(state.ScannerID, info.ScannerID)
	state.PolicyID = readReportedInt64(state.PolicyID, info.PolicyID)
	state.ScanTimeWindow = readReportedInt64(state.ScanTimeWindow, info.ScanTimeWindow)
	state.TextTargets = readReportedString(state.TextTargets, info.Targets)
	state.RRules = readReportedString(state.RRules, info.RRules)
	state.Starttime = readReportedString(state.Starttime, info.Starttime)
	state.Timezone = readReportedString(state.Timezone, info.Timezone)
	state.Launch = readReportedString(state.Launch, info.Launch)
	state.Emails = readReportedString(state.Emails, info.Emails)

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
	updated, err := r.client.UpdateScan(ctx, scanID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Scan", err.Error())
		return
	}

	result, err := r.client.GetScan(ctx, scanID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Scan After Update", err.Error())
		return
	}

	plan.ID = state.ID

	// The Computed attributes are unknown again on an update and must be settled
	// from the response -- except that this response does not report the
	// timestamps, so the prior state value is the best known answer. A stale
	// last_modification_date is invisible to Terraform; a zero one is a visible
	// lie, and an unknown left unset fails the apply outright.
	plan.UUID = readReportedString(state.UUID, result.Info.UUID)
	plan.Status = readReportedString(state.Status, result.Info.Status)
	plan.CreationDate = readReportedInt64(state.CreationDate, result.Info.CreationDate)
	plan.LastModifiedDate = readReportedInt64(state.LastModifiedDate, result.Info.LastModificationDate)

	if plan.Launch.IsUnknown() {
		plan.Launch = readReportedString(state.Launch, result.Info.Launch)
	}

	scanURL := fmt.Sprintf("%s/scans/%d", r.client.BaseURL, scanID)
	proxy := r.client.ProxyForURL()
	detailsSource := echoSource(http.MethodGet, scanURL, result.Info.PresentKeys, proxy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	requireEcho(&resp.Diagnostics, "tenableio_scan", "name", plan.Name.ValueString(), result.Info.Name, detailsSource)
	// The PUT echo is the only response that reports a description, and not
	// every tenant sends a body. When neither the echo nor the details response
	// carries the field the description is simply unverifiable: state keeps the
	// planned value, which is what was sent.
	if echoed, fromUpdate := echoedScanDescription(updated, result); echoed != nil {
		source := detailsSource
		if fromUpdate {
			source = echoSource(http.MethodPut, scanURL, updated.Scan.PresentKeys, proxy)
		}
		requireEcho(&resp.Diagnostics, "tenableio_scan", "description", plan.Description.ValueString(), *echoed, source)
	}
}

// echoedScanDescription returns the description Tenable.io reported back for an
// update, or nil when no response carried one. The second result says whether it
// came from the update echo rather than the details response, so a diagnostic
// can name the right one.
func echoedScanDescription(updated *client.ScanUpdateResponse, details *client.ScanDetailsResponse) (*string, bool) {
	if updated != nil && updated.Scan != nil && updated.Scan.Description != nil {
		return updated.Scan.Description, true
	}
	if details != nil {
		return details.Info.Description, false
	}
	return nil, false
}

// keyList renders the keys a response carried for a diagnostic.
func keyList(keys []string) string {
	if len(keys) == 0 {
		return "none at all"
	}
	return strings.Join(keys, ", ")
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
	// Import starts from null throughout, so readReported* populates only what
	// the response actually carries and everything it omits stays null. An
	// imported scan therefore has no description and no timestamps when the
	// details endpoint does not report them: the first plan proposes writing the
	// configured description back, which is a no-op against the stored value
	// rather than drift.
	state := ScanResourceModel{
		ID:               types.Int64Value(scanID),
		Name:             types.StringValue(info.Name),
		UUID:             readReportedString(types.StringNull(), info.UUID),
		TemplateUUID:     readReportedString(types.StringNull(), info.TemplateUUID),
		Description:      readReportedString(types.StringValue(""), info.Description),
		Enabled:          readReportedBool(types.BoolValue(true), info.Enabled),
		Status:           readReportedString(types.StringNull(), info.Status),
		CreationDate:     readReportedInt64(types.Int64Null(), info.CreationDate),
		LastModifiedDate: readReportedInt64(types.Int64Null(), info.LastModificationDate),
	}

	state.FolderID = readReportedInt64(state.FolderID, info.FolderID)
	state.ScannerID = readReportedInt64(state.ScannerID, info.ScannerID)
	state.PolicyID = readReportedInt64(state.PolicyID, info.PolicyID)
	state.ScanTimeWindow = readReportedInt64(state.ScanTimeWindow, info.ScanTimeWindow)
	state.TextTargets = readReportedString(state.TextTargets, info.Targets)
	state.RRules = readReportedString(state.RRules, info.RRules)
	state.Starttime = readReportedString(state.Starttime, info.Starttime)
	state.Timezone = readReportedString(state.Timezone, info.Timezone)
	state.Launch = readReportedString(state.Launch, info.Launch)
	state.Emails = readReportedString(state.Emails, info.Emails)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
