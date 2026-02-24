package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ datasource.DataSource              = (*iscsiGlobalDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*iscsiGlobalDataSource)(nil)
)

type iscsiGlobalDataSource struct {
	client *client.Client
}

type iscsiGlobalDataSourceModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Basename           types.String `tfsdk:"basename"`
	ISNSServers        types.List   `tfsdk:"isns_servers"`
	ListenPort         types.Int64  `tfsdk:"listen_port"`
	PoolAvailThreshold types.Int64  `tfsdk:"pool_avail_threshold"`
	ALUA               types.Bool   `tfsdk:"alua"`
}

func NewISCSIGlobalDataSource() datasource.DataSource {
	return &iscsiGlobalDataSource{}
}

func (d *iscsiGlobalDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_global"
}

func (d *iscsiGlobalDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the TrueNAS global iSCSI configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The identifier (always 1).",
				Computed:    true,
			},
			"basename": schema.StringAttribute{
				Description: "The base name for iSCSI targets.",
				Computed:    true,
			},
			"isns_servers": schema.ListAttribute{
				Description: "List of iSNS server addresses.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"listen_port": schema.Int64Attribute{
				Description: "The TCP port iSCSI listens on.",
				Computed:    true,
			},
			"pool_avail_threshold": schema.Int64Attribute{
				Description: "Pool available space threshold percentage.",
				Computed:    true,
			},
			"alua": schema.BoolAttribute{
				Description: "Whether ALUA is enabled.",
				Computed:    true,
			},
		},
	}
}

func (d *iscsiGlobalDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *iscsiGlobalDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var result iscsiGlobalResult
	err := d.client.Call(ctx, "iscsi.global.config", nil, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading iSCSI Global Config", err.Error())
		return
	}

	state := iscsiGlobalDataSourceModel{
		ID:         types.Int64Value(result.ID),
		Basename:   types.StringValue(result.Basename),
		ListenPort: types.Int64Value(result.ListenPort),
		ALUA:       types.BoolValue(result.ALUA),
	}

	if result.PoolAvailThreshold != nil {
		state.PoolAvailThreshold = types.Int64Value(*result.PoolAvailThreshold)
	} else {
		state.PoolAvailThreshold = types.Int64Null()
	}

	if len(result.ISNSServers) > 0 {
		elements := make([]attr.Value, len(result.ISNSServers))
		for i, s := range result.ISNSServers {
			elements[i] = types.StringValue(s)
		}
		list, diags := types.ListValue(types.StringType, elements)
		resp.Diagnostics.Append(diags...)
		state.ISNSServers = list
	} else {
		state.ISNSServers = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
