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
	_ datasource.DataSource              = (*poolDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*poolDataSource)(nil)
)

type poolDataSource struct {
	client *client.Client
}

type poolDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	Status      types.String `tfsdk:"status"`
	Healthy     types.Bool   `tfsdk:"healthy"`
	IsDecrypted types.Bool   `tfsdk:"is_decrypted"`
}

type poolResult struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	Healthy     bool   `json:"healthy"`
	IsDecrypted bool   `json:"is_decrypted"`
}

func NewPoolDataSource() datasource.DataSource {
	return &poolDataSource{}
}

func (d *poolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *poolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an existing TrueNAS ZFS pool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the pool. At least one of id or name must be provided.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the pool (e.g. \"tank\"). At least one of id or name must be provided.",
				Optional:    true,
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The mount path of the pool (e.g. /mnt/tank).",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The pool status (e.g. ONLINE, DEGRADED, FAULTED, OFFLINE).",
				Computed:    true,
			},
			"healthy": schema.BoolAttribute{
				Description: "Whether the pool is healthy.",
				Computed:    true,
			},
			"is_decrypted": schema.BoolAttribute{
				Description: "Whether the pool is decrypted.",
				Computed:    true,
			},
		},
	}
}

func (d *poolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *poolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config poolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ID.IsNull() && config.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"At least one of id or name must be provided.",
		)
		return
	}

	var filter []any
	if !config.ID.IsNull() {
		filter = []any{"id", "=", config.ID.ValueInt64()}
	} else {
		filter = []any{"name", "=", config.Name.ValueString()}
	}

	var results []poolResult
	err := d.client.Call(ctx, "pool.query", []any{
		[]any{filter},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Pool", err.Error())
		return
	}

	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Pool Not Found",
			"No pool matching the given criteria was found.",
		)
		return
	}

	result := results[0]
	state := poolDataSourceModel{
		ID:          types.Int64Value(result.ID),
		Name:        types.StringValue(result.Name),
		Path:        types.StringValue(result.Path),
		Status:      types.StringValue(result.Status),
		Healthy:     types.BoolValue(result.Healthy),
		IsDecrypted: types.BoolValue(result.IsDecrypted),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
