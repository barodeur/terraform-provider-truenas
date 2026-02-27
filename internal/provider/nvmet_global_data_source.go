package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ datasource.DataSource              = (*nvmetGlobalDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*nvmetGlobalDataSource)(nil)
)

type nvmetGlobalDataSource struct {
	client *client.Client
}

type nvmetGlobalDataSourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Basenqn       types.String `tfsdk:"basenqn"`
	Kernel        types.Bool   `tfsdk:"kernel"`
	ANA           types.Bool   `tfsdk:"ana"`
	RDMA          types.Bool   `tfsdk:"rdma"`
	XportReferral types.Bool   `tfsdk:"xport_referral"`
}

func NewNVMeTGlobalDataSource() datasource.DataSource {
	return &nvmetGlobalDataSource{}
}

func (d *nvmetGlobalDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_global"
}

func (d *nvmetGlobalDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the TrueNAS NVMe-oF global configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF global configuration.",
				Computed:    true,
			},
			"basenqn": schema.StringAttribute{
				Description: "NQN prefix used for subsystem creation.",
				Computed:    true,
			},
			"kernel": schema.BoolAttribute{
				Description: "NVMe-oF backend selection.",
				Computed:    true,
			},
			"ana": schema.BoolAttribute{
				Description: "Asymmetric Namespace Access.",
				Computed:    true,
			},
			"rdma": schema.BoolAttribute{
				Description: "RDMA enabled (Enterprise only).",
				Computed:    true,
			},
			"xport_referral": schema.BoolAttribute{
				Description: "Cross-port referral generation.",
				Computed:    true,
			},
		},
	}
}

func (d *nvmetGlobalDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nvmetGlobalDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var result nvmetGlobalResult
	err := d.client.Call(ctx, "nvmet.global.config", nil, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading NVMe-oF Global Config", err.Error())
		return
	}

	state := nvmetGlobalDataSourceModel{
		ID:            types.Int64Value(result.ID),
		Basenqn:       types.StringValue(result.Basenqn),
		Kernel:        types.BoolValue(result.Kernel),
		ANA:           types.BoolValue(result.ANA),
		RDMA:          types.BoolValue(result.RDMA),
		XportReferral: types.BoolValue(result.XportReferral),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
