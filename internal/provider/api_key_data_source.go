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
	_ datasource.DataSource              = (*apiKeyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*apiKeyDataSource)(nil)
)

type apiKeyDataSource struct {
	client *client.Client
}

type apiKeyDataSourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	CreatedAt types.String `tfsdk:"created_at"`
	Revoked   types.Bool   `tfsdk:"revoked"`
}

func NewAPIKeyDataSource() datasource.DataSource {
	return &apiKeyDataSource{}
}

func (d *apiKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *apiKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an existing TrueNAS API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the API key. At least one of id or name must be provided.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the API key. At least one of id or name must be provided.",
				Optional:    true,
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username associated with the API key.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The expiration date of the API key.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the API key.",
				Computed:    true,
			},
			"revoked": schema.BoolAttribute{
				Description: "Whether the API key has been revoked.",
				Computed:    true,
			},
		},
	}
}

func (d *apiKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiKeyDataSourceModel
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

	var results []apiKeyResult
	err := d.client.Call(ctx, "api_key.query", []any{
		[]any{filter},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading API Key", err.Error())
		return
	}

	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"API Key Not Found",
			"No API key matching the given criteria was found.",
		)
		return
	}

	result := results[0]
	state := apiKeyDataSourceModel{
		ID:        types.Int64Value(result.ID),
		Name:      types.StringValue(result.Name),
		Username:  types.StringValue(result.Username),
		CreatedAt: types.StringValue(result.CreatedAt.Value),
		Revoked:   types.BoolValue(result.Revoked),
	}
	if result.ExpiresAt.Value != "" {
		state.ExpiresAt = types.StringValue(result.ExpiresAt.Value)
	} else {
		state.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
