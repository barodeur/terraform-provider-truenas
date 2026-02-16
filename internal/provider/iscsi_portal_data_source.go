package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ datasource.DataSource              = (*iscsiPortalDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*iscsiPortalDataSource)(nil)
)

type iscsiPortalDataSource struct {
	client *client.Client
}

type iscsiPortalDataSourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Listen  types.List   `tfsdk:"listen"`
	Comment types.String `tfsdk:"comment"`
	Tag     types.Int64  `tfsdk:"tag"`
}

func NewISCSIPortalDataSource() datasource.DataSource {
	return &iscsiPortalDataSource{}
}

func (d *iscsiPortalDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_portal"
}

func (d *iscsiPortalDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an existing TrueNAS iSCSI portal.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the portal.",
				Required:    true,
			},
			"listen": schema.ListNestedAttribute{
				Description: "List of IP addresses the portal listens on.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Description: "IP address.",
							Computed:    true,
						},
					},
				},
			},
			"comment": schema.StringAttribute{
				Description: "Description of the portal.",
				Computed:    true,
			},
			"tag": schema.Int64Attribute{
				Description: "The portal group tag.",
				Computed:    true,
			},
		},
	}
}

func (d *iscsiPortalDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *iscsiPortalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config iscsiPortalDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiPortalResult
	err := d.client.Call(ctx, "iscsi.portal.get_instance", []any{config.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.Diagnostics.AddError(
				"iSCSI Portal Not Found",
				fmt.Sprintf("No iSCSI portal with ID %d was found.", config.ID.ValueInt64()),
			)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Portal", err.Error())
		return
	}

	state := iscsiPortalDataSourceModel{
		ID:  types.Int64Value(result.ID),
		Tag: types.Int64Value(result.Tag),
	}

	if result.Comment != "" {
		state.Comment = types.StringValue(result.Comment)
	} else {
		state.Comment = types.StringNull()
	}

	listenElements := make([]attr.Value, len(result.Listen))
	for i, l := range result.Listen {
		obj, diags := types.ObjectValue(iscsiPortalListenAttrTypes, map[string]attr.Value{
			"ip": types.StringValue(l.IP),
		})
		resp.Diagnostics.Append(diags...)
		listenElements[i] = obj
	}
	if resp.Diagnostics.HasError() {
		return
	}

	listenList, diags := types.ListValue(types.ObjectType{AttrTypes: iscsiPortalListenAttrTypes}, listenElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Listen = listenList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
