package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*serviceResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceResource)(nil)
	_ resource.ResourceWithImportState = (*serviceResource)(nil)
)

type serviceResource struct {
	client *client.Client
}

type serviceResourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Service types.String `tfsdk:"service"`
	Enable  types.Bool   `tfsdk:"enable"`
	Running types.Bool   `tfsdk:"running"`
	State   types.String `tfsdk:"state"`
	Pids    types.List   `tfsdk:"pids"`
}

type serviceResult struct {
	ID      int64   `json:"id"`
	Service string  `json:"service"`
	Enable  bool    `json:"enable"`
	State   string  `json:"state"`
	Pids    []int64 `json:"pids"`
}

func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS service. Services are pre-existing system entities that can be enabled/disabled and started/stopped.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the service.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "The name of the service (e.g. \"ssh\", \"smb\", \"nfs\").",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enable": schema.BoolAttribute{
				Description: "Whether the service should start on boot.",
				Required:    true,
			},
			"running": schema.BoolAttribute{
				Description: "Desired running state. If set, Terraform will start or stop the service accordingly. If not set, the actual running state is reflected without being managed.",
				Optional:    true,
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The actual state of the service (\"RUNNING\" or \"STOPPED\").",
				Computed:    true,
			},
			"pids": schema.ListAttribute{
				Description: "Process IDs of the running service.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Look up the service by name
	var results []serviceResult
	err := r.client.Call(ctx, "service.query", []any{
		[][]any{{"service", "=", plan.Service.ValueString()}},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Querying Service", err.Error())
		return
	}
	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Service Not Found",
			fmt.Sprintf("No service named %q exists on this TrueNAS system.", plan.Service.ValueString()),
		)
		return
	}

	svc := results[0]

	// Update enable setting
	err = r.client.Call(ctx, "service.update", []any{svc.ID, map[string]any{
		"enable": plan.Enable.ValueBool(),
	}}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Service", err.Error())
		return
	}

	// Control running state if user specified it
	if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		desiredRunning := plan.Running.ValueBool()
		currentlyRunning := svc.State == "RUNNING"
		if desiredRunning != currentlyRunning {
			action := "STOP"
			if desiredRunning {
				action = "START"
			}
			err = r.client.Call(ctx, "service.control", []any{action, plan.Service.ValueString(), map[string]any{}}, nil)
			if err != nil {
				resp.Diagnostics.AddError("Error Controlling Service", err.Error())
				return
			}
		}
	}

	// Read back current state
	var result serviceResult
	err = r.client.Call(ctx, "service.get_instance", []any{svc.ID}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Service After Create", err.Error())
		return
	}

	populateServiceState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result serviceResult
	err := r.client.Call(ctx, "service.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Service", err.Error())
		return
	}

	populateServiceState(ctx, &state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update enable setting
	err := r.client.Call(ctx, "service.update", []any{state.ID.ValueInt64(), map[string]any{
		"enable": plan.Enable.ValueBool(),
	}}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Service", err.Error())
		return
	}

	// Control running state if user specified it
	if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		desiredRunning := plan.Running.ValueBool()
		currentlyRunning := state.State.ValueString() == "RUNNING"
		if desiredRunning != currentlyRunning {
			action := "STOP"
			if desiredRunning {
				action = "START"
			}
			err = r.client.Call(ctx, "service.control", []any{action, plan.Service.ValueString(), map[string]any{}}, nil)
			if err != nil {
				resp.Diagnostics.AddError("Error Controlling Service", err.Error())
				return
			}
		}
	}

	// Read back current state
	var result serviceResult
	err = r.client.Call(ctx, "service.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Service After Update", err.Error())
		return
	}

	populateServiceState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Services are pre-existing system entities. On delete, we simply
	// remove the resource from Terraform state without modifying the
	// service to avoid accidentally disrupting access.
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by service name
	var results []serviceResult
	err := r.client.Call(ctx, "service.query", []any{
		[][]any{{"service", "=", req.ID}},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Querying Service for Import", err.Error())
		return
	}
	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Service Not Found",
			fmt.Sprintf("No service named %q exists on this TrueNAS system.", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), results[0].ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), results[0].Service)...)
}

func populateServiceState(ctx context.Context, model *serviceResourceModel, result *serviceResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Service = types.StringValue(result.Service)
	model.Enable = types.BoolValue(result.Enable)
	model.State = types.StringValue(result.State)
	model.Running = types.BoolValue(result.State == "RUNNING")

	pids, d := types.ListValueFrom(ctx, types.Int64Type, result.Pids)
	diags.Append(d...)
	model.Pids = pids
}
