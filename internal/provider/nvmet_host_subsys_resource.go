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
	_ resource.Resource                = (*nvmetHostSubsysResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetHostSubsysResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetHostSubsysResource)(nil)
)

type nvmetHostSubsysResource struct {
	client *client.Client
}

type nvmetHostSubsysResourceModel struct {
	ID       types.Int64 `tfsdk:"id"`
	HostID   types.Int64 `tfsdk:"host_id"`
	SubsysID types.Int64 `tfsdk:"subsys_id"`
}

type nvmetHostSubsysResultRef struct {
	ID int64 `json:"id"`
}

type nvmetHostSubsysResult struct {
	ID     int64                    `json:"id"`
	Host   nvmetHostSubsysResultRef `json:"host"`
	Subsys nvmetHostSubsysResultRef `json:"subsys"`
}

func NewNVMeTHostSubsysResource() resource.Resource {
	return &nvmetHostSubsysResource{}
}

func (r *nvmetHostSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_host_subsys"
}

func (r *nvmetHostSubsysResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF host-to-subsystem association.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the host-subsystem association.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.Int64Attribute{
				Description: "The host ID.",
				Required:    true,
			},
			"subsys_id": schema.Int64Attribute{
				Description: "The subsystem ID.",
				Required:    true,
			},
		},
	}
}

func (r *nvmetHostSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetHostSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetHostSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"host_id":   plan.HostID.ValueInt64(),
		"subsys_id": plan.SubsysID.ValueInt64(),
	}

	var result nvmetHostSubsysResult
	err := r.client.Call(ctx, "nvmet.host_subsys.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Host-Subsystem Association", err.Error())
		return
	}

	populateNVMeTHostSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetHostSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetHostSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetHostSubsysResult
	err := r.client.Call(ctx, "nvmet.host_subsys.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Host-Subsystem Association", err.Error())
		return
	}

	populateNVMeTHostSubsysState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetHostSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetHostSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetHostSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"host_id":   plan.HostID.ValueInt64(),
		"subsys_id": plan.SubsysID.ValueInt64(),
	}

	var result nvmetHostSubsysResult
	err := r.client.Call(ctx, "nvmet.host_subsys.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Host-Subsystem Association", err.Error())
		return
	}

	populateNVMeTHostSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetHostSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetHostSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.host_subsys.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Host-Subsystem Association", err.Error())
		return
	}
}

func (r *nvmetHostSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Host-Subsystem Association",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTHostSubsysState(model *nvmetHostSubsysResourceModel, result *nvmetHostSubsysResult) {
	model.ID = types.Int64Value(result.ID)
	model.HostID = types.Int64Value(result.Host.ID)
	model.SubsysID = types.Int64Value(result.Subsys.ID)
}
