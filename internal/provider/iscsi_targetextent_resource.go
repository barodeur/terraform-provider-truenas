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
	_ resource.Resource                = (*iscsiTargetextentResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiTargetextentResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiTargetextentResource)(nil)
)

type iscsiTargetextentResource struct {
	client *client.Client
}

type iscsiTargetextentResourceModel struct {
	ID     types.Int64 `tfsdk:"id"`
	Target types.Int64 `tfsdk:"target"`
	Extent types.Int64 `tfsdk:"extent"`
	LunID  types.Int64 `tfsdk:"lunid"`
}

type iscsiTargetextentResult struct {
	ID     int64 `json:"id"`
	Target int64 `json:"target"`
	Extent int64 `json:"extent"`
	LunID  int64 `json:"lunid"`
}

func NewISCSITargetextentResource() resource.Resource {
	return &iscsiTargetextentResource{}
}

func (r *iscsiTargetextentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_targetextent"
}

func (r *iscsiTargetextentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS iSCSI target-to-extent association.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the target-extent association.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"target": schema.Int64Attribute{
				Description: "The target ID.",
				Required:    true,
			},
			"extent": schema.Int64Attribute{
				Description: "The extent ID.",
				Required:    true,
			},
			"lunid": schema.Int64Attribute{
				Description: "The LUN ID. Auto-assigned if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *iscsiTargetextentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiTargetextentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiTargetextentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"target": plan.Target.ValueInt64(),
		"extent": plan.Extent.ValueInt64(),
	}

	if !plan.LunID.IsNull() && !plan.LunID.IsUnknown() {
		params["lunid"] = plan.LunID.ValueInt64()
	}

	var result iscsiTargetextentResult
	err := r.client.Call(ctx, "iscsi.targetextent.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Target-Extent", err.Error())
		return
	}

	populateISCSITargetextentState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiTargetextentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiTargetextentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiTargetextentResult
	err := r.client.Call(ctx, "iscsi.targetextent.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Target-Extent", err.Error())
		return
	}

	populateISCSITargetextentState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiTargetextentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiTargetextentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiTargetextentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"target": plan.Target.ValueInt64(),
		"extent": plan.Extent.ValueInt64(),
	}

	if !plan.LunID.IsNull() && !plan.LunID.IsUnknown() {
		params["lunid"] = plan.LunID.ValueInt64()
	}

	var result iscsiTargetextentResult
	err := r.client.Call(ctx, "iscsi.targetextent.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Target-Extent", err.Error())
		return
	}

	populateISCSITargetextentState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiTargetextentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiTargetextentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "iscsi.targetextent.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Target-Extent", err.Error())
		return
	}
}

func (r *iscsiTargetextentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Target-Extent",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateISCSITargetextentState(model *iscsiTargetextentResourceModel, result *iscsiTargetextentResult) {
	model.ID = types.Int64Value(result.ID)
	model.Target = types.Int64Value(result.Target)
	model.Extent = types.Int64Value(result.Extent)
	model.LunID = types.Int64Value(result.LunID)
}
