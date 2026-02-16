package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*poolSnapshotTaskResource)(nil)
	_ resource.ResourceWithConfigure   = (*poolSnapshotTaskResource)(nil)
	_ resource.ResourceWithImportState = (*poolSnapshotTaskResource)(nil)
)

type poolSnapshotTaskResource struct {
	client *client.Client
}

type poolSnapshotTaskResourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Dataset       types.String `tfsdk:"dataset"`
	Recursive     types.Bool   `tfsdk:"recursive"`
	LifetimeValue types.Int64  `tfsdk:"lifetime_value"`
	LifetimeUnit  types.String `tfsdk:"lifetime_unit"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Exclude       types.List   `tfsdk:"exclude"`
	NamingSchema  types.String `tfsdk:"naming_schema"`
	AllowEmpty    types.Bool   `tfsdk:"allow_empty"`
	Schedule      types.Object `tfsdk:"schedule"`
}

type poolSnapshotTaskScheduleModel struct {
	Minute types.String `tfsdk:"minute"`
	Hour   types.String `tfsdk:"hour"`
	Dom    types.String `tfsdk:"dom"`
	Month  types.String `tfsdk:"month"`
	Dow    types.String `tfsdk:"dow"`
	Begin  types.String `tfsdk:"begin"`
	End    types.String `tfsdk:"end"`
}

var poolSnapshotTaskScheduleAttrTypes = map[string]attr.Type{
	"minute": types.StringType,
	"hour":   types.StringType,
	"dom":    types.StringType,
	"month":  types.StringType,
	"dow":    types.StringType,
	"begin":  types.StringType,
	"end":    types.StringType,
}

type poolSnapshotTaskCreateParams struct {
	Dataset       string                   `json:"dataset"`
	Recursive     bool                     `json:"recursive"`
	LifetimeValue int64                    `json:"lifetime_value"`
	LifetimeUnit  string                   `json:"lifetime_unit"`
	Enabled       bool                     `json:"enabled"`
	Exclude       []string                 `json:"exclude"`
	NamingSchema  string                   `json:"naming_schema"`
	AllowEmpty    bool                     `json:"allow_empty"`
	Schedule      poolSnapshotTaskSchedule `json:"schedule"`
}

type poolSnapshotTaskSchedule struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
	Begin  string `json:"begin"`
	End    string `json:"end"`
}

type poolSnapshotTaskResult struct {
	ID            int64                    `json:"id"`
	Dataset       string                   `json:"dataset"`
	Recursive     bool                     `json:"recursive"`
	LifetimeValue int64                    `json:"lifetime_value"`
	LifetimeUnit  string                   `json:"lifetime_unit"`
	Enabled       bool                     `json:"enabled"`
	Exclude       []string                 `json:"exclude"`
	NamingSchema  string                   `json:"naming_schema"`
	AllowEmpty    bool                     `json:"allow_empty"`
	Schedule      poolSnapshotTaskSchedule `json:"schedule"`
}

func NewPoolSnapshotTaskResource() resource.Resource {
	return &poolSnapshotTaskResource{}
}

func (r *poolSnapshotTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool_snapshot_task"
}

func (r *poolSnapshotTaskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS periodic snapshot task.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the snapshot task.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset": schema.StringAttribute{
				Description: "The dataset to snapshot (e.g. tank/data).",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to take recursive snapshots of child datasets.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"lifetime_value": schema.Int64Attribute{
				Description: "How long to keep snapshots (numeric part).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"lifetime_unit": schema.StringAttribute{
				Description: "Unit for snapshot lifetime. Valid values: HOUR, DAY, WEEK, MONTH, YEAR.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("WEEK"),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the snapshot task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"exclude": schema.ListAttribute{
				Description: "List of child datasets to exclude from recursive snapshots.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"naming_schema": schema.StringAttribute{
				Description: "Naming schema for snapshots. Uses strftime-style format.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("auto-%Y-%m-%d_%H-%M"),
			},
			"allow_empty": schema.BoolAttribute{
				Description: "Whether to create snapshots even when there are no changes.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"schedule": schema.SingleNestedAttribute{
				Description: "The snapshot schedule.",
				Optional:    true,
				Computed:    true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(poolSnapshotTaskScheduleAttrTypes, map[string]attr.Value{
						"minute": types.StringValue("00"),
						"hour":   types.StringValue("*"),
						"dom":    types.StringValue("*"),
						"month":  types.StringValue("*"),
						"dow":    types.StringValue("*"),
						"begin":  types.StringValue("00:00"),
						"end":    types.StringValue("23:59"),
					}),
				),
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute field (0-59, *, or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("00"),
					},
					"hour": schema.StringAttribute{
						Description: "Hour field (0-23, *, or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"dom": schema.StringAttribute{
						Description: "Day of month field (1-31, *, or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"month": schema.StringAttribute{
						Description: "Month field (1-12, *, or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"dow": schema.StringAttribute{
						Description: "Day of week field (0-6, *, or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"begin": schema.StringAttribute{
						Description: "Start time of the allowed window (HH:MM).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("00:00"),
					},
					"end": schema.StringAttribute{
						Description: "End time of the allowed window (HH:MM).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("23:59"),
					},
				},
			},
		},
	}
}

func (r *poolSnapshotTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *poolSnapshotTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poolSnapshotTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sched poolSnapshotTaskScheduleModel
	resp.Diagnostics.Append(plan.Schedule.As(ctx, &sched, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exclude []string
	if !plan.Exclude.IsNull() {
		resp.Diagnostics.Append(plan.Exclude.ElementsAs(ctx, &exclude, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		exclude = []string{}
	}

	params := poolSnapshotTaskCreateParams{
		Dataset:       plan.Dataset.ValueString(),
		Recursive:     plan.Recursive.ValueBool(),
		LifetimeValue: plan.LifetimeValue.ValueInt64(),
		LifetimeUnit:  plan.LifetimeUnit.ValueString(),
		Enabled:       plan.Enabled.ValueBool(),
		Exclude:       exclude,
		NamingSchema:  plan.NamingSchema.ValueString(),
		AllowEmpty:    plan.AllowEmpty.ValueBool(),
		Schedule: poolSnapshotTaskSchedule{
			Minute: sched.Minute.ValueString(),
			Hour:   sched.Hour.ValueString(),
			Dom:    sched.Dom.ValueString(),
			Month:  sched.Month.ValueString(),
			Dow:    sched.Dow.ValueString(),
			Begin:  sched.Begin.ValueString(),
			End:    sched.End.ValueString(),
		},
	}

	var result poolSnapshotTaskResult
	err := r.client.Call(ctx, "pool.snapshottask.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Snapshot Task", err.Error())
		return
	}

	resp.Diagnostics.Append(setPoolSnapshotTaskResourceState(&plan, &result)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolSnapshotTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state poolSnapshotTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result poolSnapshotTaskResult
	err := r.client.Call(ctx, "pool.snapshottask.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Snapshot Task", err.Error())
		return
	}

	resp.Diagnostics.Append(setPoolSnapshotTaskResourceState(&state, &result)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *poolSnapshotTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan poolSnapshotTaskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state poolSnapshotTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sched poolSnapshotTaskScheduleModel
	resp.Diagnostics.Append(plan.Schedule.As(ctx, &sched, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exclude []string
	if !plan.Exclude.IsNull() {
		resp.Diagnostics.Append(plan.Exclude.ElementsAs(ctx, &exclude, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		exclude = []string{}
	}

	params := poolSnapshotTaskCreateParams{
		Dataset:       plan.Dataset.ValueString(),
		Recursive:     plan.Recursive.ValueBool(),
		LifetimeValue: plan.LifetimeValue.ValueInt64(),
		LifetimeUnit:  plan.LifetimeUnit.ValueString(),
		Enabled:       plan.Enabled.ValueBool(),
		Exclude:       exclude,
		NamingSchema:  plan.NamingSchema.ValueString(),
		AllowEmpty:    plan.AllowEmpty.ValueBool(),
		Schedule: poolSnapshotTaskSchedule{
			Minute: sched.Minute.ValueString(),
			Hour:   sched.Hour.ValueString(),
			Dom:    sched.Dom.ValueString(),
			Month:  sched.Month.ValueString(),
			Dow:    sched.Dow.ValueString(),
			Begin:  sched.Begin.ValueString(),
			End:    sched.End.ValueString(),
		},
	}

	var result poolSnapshotTaskResult
	err := r.client.Call(ctx, "pool.snapshottask.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Snapshot Task", err.Error())
		return
	}

	resp.Diagnostics.Append(setPoolSnapshotTaskResourceState(&plan, &result)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolSnapshotTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state poolSnapshotTaskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "pool.snapshottask.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Snapshot Task", err.Error())
		return
	}
}

func (r *poolSnapshotTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Snapshot Task",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(id))...)
}

func setPoolSnapshotTaskResourceState(model *poolSnapshotTaskResourceModel, result *poolSnapshotTaskResult) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.Int64Value(result.ID)
	model.Dataset = types.StringValue(result.Dataset)
	model.Recursive = types.BoolValue(result.Recursive)
	model.LifetimeValue = types.Int64Value(result.LifetimeValue)
	model.LifetimeUnit = types.StringValue(result.LifetimeUnit)
	model.Enabled = types.BoolValue(result.Enabled)
	model.NamingSchema = types.StringValue(result.NamingSchema)
	model.AllowEmpty = types.BoolValue(result.AllowEmpty)

	if len(result.Exclude) > 0 {
		elements := make([]attr.Value, len(result.Exclude))
		for i, e := range result.Exclude {
			elements[i] = types.StringValue(e)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Exclude = list
	} else {
		model.Exclude = types.ListNull(types.StringType)
	}

	scheduleValue, d := types.ObjectValue(poolSnapshotTaskScheduleAttrTypes, map[string]attr.Value{
		"minute": types.StringValue(result.Schedule.Minute),
		"hour":   types.StringValue(result.Schedule.Hour),
		"dom":    types.StringValue(result.Schedule.Dom),
		"month":  types.StringValue(result.Schedule.Month),
		"dow":    types.StringValue(result.Schedule.Dow),
		"begin":  types.StringValue(result.Schedule.Begin),
		"end":    types.StringValue(result.Schedule.End),
	})
	diags.Append(d...)
	model.Schedule = scheduleValue

	return diags
}
