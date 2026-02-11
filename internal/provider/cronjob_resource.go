package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*cronjobResource)(nil)
	_ resource.ResourceWithConfigure   = (*cronjobResource)(nil)
	_ resource.ResourceWithImportState = (*cronjobResource)(nil)
)

type cronjobResource struct {
	client *client.Client
}

type cronjobResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Command     types.String `tfsdk:"command"`
	User        types.String `tfsdk:"user"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Stdout      types.Bool   `tfsdk:"stdout"`
	Stderr      types.Bool   `tfsdk:"stderr"`
	Schedule    types.Object `tfsdk:"schedule"`
}

type cronjobScheduleModel struct {
	Minute types.String `tfsdk:"minute"`
	Hour   types.String `tfsdk:"hour"`
	Dom    types.String `tfsdk:"dom"`
	Month  types.String `tfsdk:"month"`
	Dow    types.String `tfsdk:"dow"`
}

var cronjobScheduleAttrTypes = map[string]attr.Type{
	"minute": types.StringType,
	"hour":   types.StringType,
	"dom":    types.StringType,
	"month":  types.StringType,
	"dow":    types.StringType,
}

type cronjobCreateParams struct {
	Command     string          `json:"command"`
	User        string          `json:"user"`
	Description string          `json:"description,omitempty"`
	Enabled     bool            `json:"enabled"`
	Stdout      bool            `json:"stdout"`
	Stderr      bool            `json:"stderr"`
	Schedule    cronjobSchedule `json:"schedule"`
}

type cronjobSchedule struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}

type cronjobResult struct {
	ID          int64           `json:"id"`
	Command     string          `json:"command"`
	User        string          `json:"user"`
	Description string          `json:"description"`
	Enabled     bool            `json:"enabled"`
	Stdout      bool            `json:"stdout"`
	Stderr      bool            `json:"stderr"`
	Schedule    cronjobSchedule `json:"schedule"`
}

func NewCronjobResource() resource.Resource {
	return &cronjobResource{}
}

func (r *cronjobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (r *cronjobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS cron job.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the cron job.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"command": schema.StringAttribute{
				Description: "The shell command to execute.",
				Required:    true,
			},
			"user": schema.StringAttribute{
				Description: "The system user to run the command as.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A human-readable description of the cron job.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the cron job is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"stdout": schema.BoolAttribute{
				Description: "Whether to hide standard output.",
				Optional:    true,
				Computed:    true,
			},
			"stderr": schema.BoolAttribute{
				Description: "Whether to hide standard error.",
				Optional:    true,
				Computed:    true,
			},
			"schedule": schema.SingleNestedAttribute{
				Description: "The cron schedule.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute field (0-59, *, or cron expression).",
						Required:    true,
					},
					"hour": schema.StringAttribute{
						Description: "Hour field (0-23, *, or cron expression).",
						Required:    true,
					},
					"dom": schema.StringAttribute{
						Description: "Day of month field (1-31, *, or cron expression).",
						Required:    true,
					},
					"month": schema.StringAttribute{
						Description: "Month field (1-12, *, or cron expression).",
						Required:    true,
					},
					"dow": schema.StringAttribute{
						Description: "Day of week field (0-6, *, or cron expression).",
						Required:    true,
					},
				},
			},
		},
	}
}

func (r *cronjobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *cronjobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cronjobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sched cronjobScheduleModel
	resp.Diagnostics.Append(plan.Schedule.As(ctx, &sched, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := cronjobCreateParams{
		Command: plan.Command.ValueString(),
		User:    plan.User.ValueString(),
		Schedule: cronjobSchedule{
			Minute: sched.Minute.ValueString(),
			Hour:   sched.Hour.ValueString(),
			Dom:    sched.Dom.ValueString(),
			Month:  sched.Month.ValueString(),
			Dow:    sched.Dow.ValueString(),
		},
	}
	if !plan.Description.IsNull() {
		params.Description = plan.Description.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		params.Enabled = plan.Enabled.ValueBool()
	} else {
		params.Enabled = true
	}
	if !plan.Stdout.IsNull() && !plan.Stdout.IsUnknown() {
		params.Stdout = plan.Stdout.ValueBool()
	} else {
		params.Stdout = true
	}
	if !plan.Stderr.IsNull() && !plan.Stderr.IsUnknown() {
		params.Stderr = plan.Stderr.ValueBool()
	}

	var result cronjobResult
	err := r.client.Call(ctx, "cronjob.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cron Job", err.Error())
		return
	}

	setCronjobResourceState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cronjobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cronjobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var results []cronjobResult
	err := r.client.Call(ctx, "cronjob.query", []any{
		[]any{[]any{"id", "=", state.ID.ValueInt64()}},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Cron Job", err.Error())
		return
	}

	if len(results) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	setCronjobResourceState(&state, &results[0])
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cronjobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cronjobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state cronjobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sched cronjobScheduleModel
	resp.Diagnostics.Append(plan.Schedule.As(ctx, &sched, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := cronjobCreateParams{
		Command: plan.Command.ValueString(),
		User:    plan.User.ValueString(),
		Schedule: cronjobSchedule{
			Minute: sched.Minute.ValueString(),
			Hour:   sched.Hour.ValueString(),
			Dom:    sched.Dom.ValueString(),
			Month:  sched.Month.ValueString(),
			Dow:    sched.Dow.ValueString(),
		},
	}
	if !plan.Description.IsNull() {
		params.Description = plan.Description.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		params.Enabled = plan.Enabled.ValueBool()
	} else {
		params.Enabled = true
	}
	if !plan.Stdout.IsNull() && !plan.Stdout.IsUnknown() {
		params.Stdout = plan.Stdout.ValueBool()
	} else {
		params.Stdout = true
	}
	if !plan.Stderr.IsNull() && !plan.Stderr.IsUnknown() {
		params.Stderr = plan.Stderr.ValueBool()
	}

	var result cronjobResult
	err := r.client.Call(ctx, "cronjob.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Cron Job", err.Error())
		return
	}

	setCronjobResourceState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cronjobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cronjobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "cronjob.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Cron Job", err.Error())
		return
	}
}

func (r *cronjobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Cron Job",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(id))...)
}

func setCronjobResourceState(model *cronjobResourceModel, result *cronjobResult) {
	model.ID = types.Int64Value(result.ID)
	model.Command = types.StringValue(result.Command)
	model.User = types.StringValue(result.User)
	if result.Description != "" {
		model.Description = types.StringValue(result.Description)
	} else {
		model.Description = types.StringNull()
	}
	model.Enabled = types.BoolValue(result.Enabled)
	model.Stdout = types.BoolValue(result.Stdout)
	model.Stderr = types.BoolValue(result.Stderr)
	model.Schedule = types.ObjectValueMust(cronjobScheduleAttrTypes, map[string]attr.Value{
		"minute": types.StringValue(result.Schedule.Minute),
		"hour":   types.StringValue(result.Schedule.Hour),
		"dom":    types.StringValue(result.Schedule.Dom),
		"month":  types.StringValue(result.Schedule.Month),
		"dow":    types.StringValue(result.Schedule.Dow),
	})
}
