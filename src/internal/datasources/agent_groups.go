package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tenable/terraform-provider-tenableio/internal/client"
)

var _ datasource.DataSource = &AgentGroupsDataSource{}

type AgentGroupsDataSource struct {
	client *client.Client
}

type AgentGroupsDataSourceModel struct {
	Groups []AgentGroupItemModel `tfsdk:"groups"`
}

type AgentGroupItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	OwnerID     types.Int64  `tfsdk:"owner_id"`
	Owner       types.String `tfsdk:"owner"`
	Shared      types.Int64  `tfsdk:"shared"`
	AgentsCount types.Int64  `tfsdk:"agents_count"`
}

func NewAgentGroupsDataSource() datasource.DataSource {
	return &AgentGroupsDataSource{}
}

func (d *AgentGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_groups"
}

func (d *AgentGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of agent groups from Tenable.io.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.Int64Attribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"owner_id":     schema.Int64Attribute{Computed: true},
						"owner":        schema.StringAttribute{Computed: true},
						"shared":       schema.Int64Attribute{Computed: true},
						"agents_count": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *AgentGroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AgentGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.ListAgentGroups(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Agent Groups", err.Error())
		return
	}

	groups := make([]AgentGroupItemModel, len(result.Groups))
	for i, g := range result.Groups {
		groups[i] = AgentGroupItemModel{
			ID:          types.Int64Value(int64(g.ID)),
			Name:        types.StringValue(g.Name),
			OwnerID:     types.Int64Value(int64(g.OwnerID)),
			Owner:       types.StringValue(g.Owner),
			Shared:      types.Int64Value(int64(g.Shared)),
			AgentsCount: types.Int64Value(int64(g.AgentsCount)),
		}
	}

	state := AgentGroupsDataSourceModel{Groups: groups}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
