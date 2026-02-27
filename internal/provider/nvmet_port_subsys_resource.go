package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*nvmetPortSubsysResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetPortSubsysResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetPortSubsysResource)(nil)
)

type nvmetPortSubsysResource struct {
	client *client.Client
}

type nvmetPortSubsysResourceModel struct {
	ID       types.Int64 `tfsdk:"id"`
	PortID   types.Int64 `tfsdk:"port_id"`
	SubsysID types.Int64 `tfsdk:"subsys_id"`
}

type nvmetPortSubsysResultRef struct {
	ID int64 `json:"id"`
}

type nvmetPortSubsysResult struct {
	ID     int64                    `json:"id"`
	Port   nvmetPortSubsysResultRef `json:"port"`
	Subsys nvmetPortSubsysResultRef `json:"subsys"`
}

func NewNVMeTPortSubsysResource() resource.Resource {
	return &nvmetPortSubsysResource{}
}

func (r *nvmetPortSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_port_subsys"
}

func (r *nvmetPortSubsysResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF port-to-subsystem association.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the port-subsystem association.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"port_id": schema.Int64Attribute{
				Description: "The port ID.",
				Required:    true,
			},
			"subsys_id": schema.Int64Attribute{
				Description: "The subsystem ID.",
				Required:    true,
			},
		},
	}
}

func (r *nvmetPortSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetPortSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetPortSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"port_id":   plan.PortID.ValueInt64(),
		"subsys_id": plan.SubsysID.ValueInt64(),
	}

	var result nvmetPortSubsysResult
	err := r.client.Call(ctx, "nvmet.port_subsys.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Port-Subsystem Association", err.Error())
		return
	}

	populateNVMeTPortSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetPortSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetPortSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetPortSubsysResult
	err := r.client.Call(ctx, "nvmet.port_subsys.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Port-Subsystem Association", err.Error())
		return
	}

	populateNVMeTPortSubsysState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetPortSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetPortSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetPortSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"port_id":   plan.PortID.ValueInt64(),
		"subsys_id": plan.SubsysID.ValueInt64(),
	}

	var result nvmetPortSubsysResult
	err := r.client.Call(ctx, "nvmet.port_subsys.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Port-Subsystem Association", err.Error())
		return
	}

	populateNVMeTPortSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetPortSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetPortSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.port_subsys.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Port-Subsystem Association", err.Error())
		return
	}
}

func (r *nvmetPortSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Port-Subsystem Association",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTPortSubsysState(model *nvmetPortSubsysResourceModel, result *nvmetPortSubsysResult) {
	model.ID = types.Int64Value(result.ID)
	model.PortID = types.Int64Value(result.Port.ID)
	model.SubsysID = types.Int64Value(result.Subsys.ID)
}
