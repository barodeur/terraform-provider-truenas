package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*nvmetPortResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetPortResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetPortResource)(nil)
)

type nvmetPortResource struct {
	client *client.Client
}

type nvmetPortResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Index          types.Int64  `tfsdk:"index"`
	AddrTrtype     types.String `tfsdk:"addr_trtype"`
	AddrTraddr     types.String `tfsdk:"addr_traddr"`
	AddrTrsvcid    types.Int64  `tfsdk:"addr_trsvcid"`
	AddrAdrfam     types.String `tfsdk:"addr_adrfam"`
	InlineDataSize types.Int64  `tfsdk:"inline_data_size"`
	MaxQueueSize   types.Int64  `tfsdk:"max_queue_size"`
	PIEnable       types.Bool   `tfsdk:"pi_enable"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

type nvmetPortResult struct {
	ID             int64  `json:"id"`
	Index          int64  `json:"index"`
	AddrTrtype     string `json:"addr_trtype"`
	AddrTraddr     string `json:"addr_traddr"`
	AddrTrsvcid    *int64 `json:"addr_trsvcid"`
	AddrAdrfam     string `json:"addr_adrfam"`
	InlineDataSize *int64 `json:"inline_data_size"`
	MaxQueueSize   *int64 `json:"max_queue_size"`
	PIEnable       bool   `json:"pi_enable"`
	Enabled        bool   `json:"enabled"`
}

func NewNVMeTPortResource() resource.Resource {
	return &nvmetPortResource{}
}

func (r *nvmetPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_port"
}

func (r *nvmetPortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF port.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF port.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"index": schema.Int64Attribute{
				Description: "Internal port index.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"addr_trtype": schema.StringAttribute{
				Description: "Transport type (TCP, RDMA, FC).",
				Required:    true,
			},
			"addr_traddr": schema.StringAttribute{
				Description: "IP address or FC identifier.",
				Required:    true,
			},
			"addr_trsvcid": schema.Int64Attribute{
				Description: "Port number, 1024–65535 (TCP/RDMA only). Defaults to 4420.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"addr_adrfam": schema.StringAttribute{
				Description: "Address family (IPV4, IPV6, FC). Computed from addr_traddr.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inline_data_size": schema.Int64Attribute{
				Description: "Inline data size.",
				Optional:    true,
			},
			"max_queue_size": schema.Int64Attribute{
				Description: "Maximum queue size.",
				Optional:    true,
			},
			"pi_enable": schema.BoolAttribute{
				Description: "Enable protection information.",
				Optional:    true,
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the port is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *nvmetPortResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetPortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"addr_trtype": plan.AddrTrtype.ValueString(),
		"addr_traddr": plan.AddrTraddr.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
	}

	if !plan.AddrTrsvcid.IsNull() && !plan.AddrTrsvcid.IsUnknown() {
		params["addr_trsvcid"] = plan.AddrTrsvcid.ValueInt64()
	}
	if !plan.InlineDataSize.IsNull() && !plan.InlineDataSize.IsUnknown() {
		params["inline_data_size"] = plan.InlineDataSize.ValueInt64()
	}
	if !plan.MaxQueueSize.IsNull() && !plan.MaxQueueSize.IsUnknown() {
		params["max_queue_size"] = plan.MaxQueueSize.ValueInt64()
	}
	if !plan.PIEnable.IsNull() && !plan.PIEnable.IsUnknown() {
		params["pi_enable"] = plan.PIEnable.ValueBool()
	}

	var result nvmetPortResult
	err := r.client.Call(ctx, "nvmet.port.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Port", err.Error())
		return
	}

	populateNVMeTPortState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetPortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetPortResult
	err := r.client.Call(ctx, "nvmet.port.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Port", err.Error())
		return
	}

	populateNVMeTPortState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetPortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetPortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"addr_trtype": plan.AddrTrtype.ValueString(),
		"addr_traddr": plan.AddrTraddr.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
	}

	if !plan.AddrTrsvcid.IsNull() && !plan.AddrTrsvcid.IsUnknown() {
		params["addr_trsvcid"] = plan.AddrTrsvcid.ValueInt64()
	}
	if !plan.InlineDataSize.IsNull() {
		params["inline_data_size"] = plan.InlineDataSize.ValueInt64()
	} else {
		params["inline_data_size"] = nil
	}
	if !plan.MaxQueueSize.IsNull() {
		params["max_queue_size"] = plan.MaxQueueSize.ValueInt64()
	} else {
		params["max_queue_size"] = nil
	}
	if !plan.PIEnable.IsNull() && !plan.PIEnable.IsUnknown() {
		params["pi_enable"] = plan.PIEnable.ValueBool()
	}

	var result nvmetPortResult
	err := r.client.Call(ctx, "nvmet.port.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Port", err.Error())
		return
	}

	populateNVMeTPortState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetPortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.port.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Port", err.Error())
		return
	}
}

func (r *nvmetPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Port",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTPortState(model *nvmetPortResourceModel, result *nvmetPortResult) {
	model.ID = types.Int64Value(result.ID)
	model.Index = types.Int64Value(result.Index)
	model.AddrTrtype = types.StringValue(result.AddrTrtype)
	model.AddrTraddr = types.StringValue(result.AddrTraddr)

	if result.AddrTrsvcid != nil {
		model.AddrTrsvcid = types.Int64Value(*result.AddrTrsvcid)
	} else {
		model.AddrTrsvcid = types.Int64Null()
	}

	model.AddrAdrfam = types.StringValue(result.AddrAdrfam)

	if result.InlineDataSize != nil {
		model.InlineDataSize = types.Int64Value(*result.InlineDataSize)
	} else {
		model.InlineDataSize = types.Int64Null()
	}

	if result.MaxQueueSize != nil {
		model.MaxQueueSize = types.Int64Value(*result.MaxQueueSize)
	} else {
		model.MaxQueueSize = types.Int64Null()
	}

	model.PIEnable = types.BoolValue(result.PIEnable)
	model.Enabled = types.BoolValue(result.Enabled)
}
