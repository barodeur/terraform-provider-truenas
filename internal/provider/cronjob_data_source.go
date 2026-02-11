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
	_ datasource.DataSource              = (*cronjobDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cronjobDataSource)(nil)
)

type cronjobDataSource struct {
	client *client.Client
}

type cronjobDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Command     types.String `tfsdk:"command"`
	User        types.String `tfsdk:"user"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Stdout      types.Bool   `tfsdk:"stdout"`
	Stderr      types.Bool   `tfsdk:"stderr"`
	Schedule    types.Object `tfsdk:"schedule"`
}

func NewCronjobDataSource() datasource.DataSource {
	return &cronjobDataSource{}
}

func (d *cronjobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (d *cronjobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an existing TrueNAS cron job.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the cron job.",
				Required:    true,
			},
			"command": schema.StringAttribute{
				Description: "The shell command to execute.",
				Computed:    true,
			},
			"user": schema.StringAttribute{
				Description: "The system user the command runs as.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A human-readable description of the cron job.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the cron job is enabled.",
				Computed:    true,
			},
			"stdout": schema.BoolAttribute{
				Description: "Whether to hide standard output.",
				Computed:    true,
			},
			"stderr": schema.BoolAttribute{
				Description: "Whether to hide standard error.",
				Computed:    true,
			},
			"schedule": schema.SingleNestedAttribute{
				Description: "The cron schedule.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute field (0-59, *, or cron expression).",
						Computed:    true,
					},
					"hour": schema.StringAttribute{
						Description: "Hour field (0-23, *, or cron expression).",
						Computed:    true,
					},
					"dom": schema.StringAttribute{
						Description: "Day of month field (1-31, *, or cron expression).",
						Computed:    true,
					},
					"month": schema.StringAttribute{
						Description: "Month field (1-12, *, or cron expression).",
						Computed:    true,
					},
					"dow": schema.StringAttribute{
						Description: "Day of week field (0-6, *, or cron expression).",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *cronjobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *cronjobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config cronjobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var results []cronjobResult
	err := d.client.Call(ctx, "cronjob.query", []any{
		[]any{[]any{"id", "=", config.ID.ValueInt64()}},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cron Job", err.Error())
		return
	}

	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Cron Job Not Found",
			fmt.Sprintf("No cron job with ID %d was found.", config.ID.ValueInt64()),
		)
		return
	}

	result := results[0]
	state := cronjobDataSourceModel{
		ID:      types.Int64Value(result.ID),
		Command: types.StringValue(result.Command),
		User:    types.StringValue(result.User),
		Enabled: types.BoolValue(result.Enabled),
		Stdout:  types.BoolValue(result.Stdout),
		Stderr:  types.BoolValue(result.Stderr),
		Schedule: types.ObjectValueMust(cronjobScheduleAttrTypes, map[string]attr.Value{
			"minute": types.StringValue(result.Schedule.Minute),
			"hour":   types.StringValue(result.Schedule.Hour),
			"dom":    types.StringValue(result.Schedule.Dom),
			"month":  types.StringValue(result.Schedule.Month),
			"dow":    types.StringValue(result.Schedule.Dow),
		}),
	}
	if result.Description != "" {
		state.Description = types.StringValue(result.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
