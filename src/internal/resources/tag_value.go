package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &TagValueResource{}
	_ resource.ResourceWithImportState = &TagValueResource{}
)

type TagValueResource struct {
	client *client.Client
}

type TagValueResourceModel struct {
	UUID                types.String `tfsdk:"uuid"`
	CategoryUUID        types.String `tfsdk:"category_uuid"`
	CategoryName        types.String `tfsdk:"category_name"`
	CategoryDescription types.String `tfsdk:"category_description"`
	Value               types.String `tfsdk:"value"`
	Description         types.String `tfsdk:"description"`
	Type                types.String `tfsdk:"type"`
	CreatedAt           types.String `tfsdk:"created_at"`
	CreatedBy           types.String `tfsdk:"created_by"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	UpdatedBy           types.String `tfsdk:"updated_by"`
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
				Description: "The UUID of the tag category. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"category_name": schema.StringAttribute{
				Description: "The name of the tag category. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"category_description": schema.StringAttribute{
				Description: "Description for a new category (used only when category_name creates a new category). Cannot be changed after creation.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "The tag value.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the tag value.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"type": schema.StringAttribute{
				Description: "The type of the tag value (static or dynamic).",
				Computed:    true,
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

	r.mapToState(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Tag Value", err.Error())
		return
	}

	r.mapToState(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
