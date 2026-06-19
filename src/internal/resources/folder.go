package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lamda-systems/terraform-provider-tenableio/internal/client"
)

var (
	_ resource.Resource                = &FolderResource{}
	_ resource.ResourceWithImportState = &FolderResource{}
)

type FolderResource struct {
	client *client.Client
}

type FolderResourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func NewFolderResource() resource.Resource {
	return &FolderResource{}
}

func (r *FolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (r *FolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a scan folder in Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier for the folder.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the folder.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the folder (custom, main, trash).",
				Computed:    true,
			},
		},
	}
}

func (r *FolderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateFolder(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Folder", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(result.ID))
	plan.Type = types.StringValue("custom")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.client.GetFolder(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Folder", err.Error())
		return
	}

	state.Name = types.StringValue(folder.Name)
	state.Type = types.StringValue(folder.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FolderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.EditFolder(ctx, int(state.ID.ValueInt64()), plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Updating Folder", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Type = state.Type

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FolderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteFolder(ctx, int(state.ID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error Deleting Folder", err.Error())
		return
	}
}

func (r *FolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	folderID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Folder ID", fmt.Sprintf("Could not parse folder ID %q: %s", req.ID, err))
		return
	}

	folder, err := r.client.GetFolder(ctx, int(folderID))
	if err != nil {
		resp.Diagnostics.AddError("Error Importing Folder", err.Error())
		return
	}

	state := FolderResourceModel{
		ID:   types.Int64Value(folderID),
		Name: types.StringValue(folder.Name),
		Type: types.StringValue(folder.Type),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
