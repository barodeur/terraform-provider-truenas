package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*iscsiGlobalResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiGlobalResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiGlobalResource)(nil)
)

type iscsiGlobalResource struct {
	client *client.Client
}

type iscsiGlobalResourceModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Basename           types.String `tfsdk:"basename"`
	ISNSServers        types.List   `tfsdk:"isns_servers"`
	ListenPort         types.Int64  `tfsdk:"listen_port"`
	PoolAvailThreshold types.Int64  `tfsdk:"pool_avail_threshold"`
	ALUA               types.Bool   `tfsdk:"alua"`
}

type iscsiGlobalResult struct {
	ID                 int64    `json:"id"`
	Basename           string   `json:"basename"`
	ISNSServers        []string `json:"isns_servers"`
	ListenPort         int64    `json:"listen_port"`
	PoolAvailThreshold *int64   `json:"pool_avail_threshold"`
	ALUA               bool     `json:"alua"`
}

func NewISCSIGlobalResource() resource.Resource {
	return &iscsiGlobalResource{}
}

func (r *iscsiGlobalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_global"
}

func (r *iscsiGlobalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TrueNAS global iSCSI configuration. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The identifier (always 1 for singleton config).",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"basename": schema.StringAttribute{
				Description: "The base name for iSCSI targets (e.g. iqn.2005-10.org.freenas.ctl).",
				Required:    true,
			},
			"isns_servers": schema.ListAttribute{
				Description: "List of iSNS server addresses.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"listen_port": schema.Int64Attribute{
				Description: "The TCP port iSCSI listens on. Defaults to 3260.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pool_avail_threshold": schema.Int64Attribute{
				Description: "Pool available space threshold percentage for alerts.",
				Optional:    true,
			},
			"alua": schema.BoolAttribute{
				Description: "Enable Asymmetric Logical Unit Access.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *iscsiGlobalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := iscsiGlobalParamsFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiGlobalResult
	err := r.client.Call(ctx, "iscsi.global.update", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Configuring iSCSI Global", err.Error())
		return
	}

	populateISCSIGlobalState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiGlobalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiGlobalResult
	err := r.client.Call(ctx, "iscsi.global.config", nil, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Global Config", err.Error())
		return
	}

	populateISCSIGlobalState(&state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := iscsiGlobalParamsFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiGlobalResult
	err := r.client.Call(ctx, "iscsi.global.update", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Global Config", err.Error())
		return
	}

	populateISCSIGlobalState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiGlobalResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Reset to defaults — singleton cannot be truly deleted
	defaults := map[string]any{
		"basename":     "iqn.2005-10.org.freenas.ctl",
		"isns_servers": []string{},
		"listen_port":  3260,
		"alua":         false,
	}

	err := r.client.Call(ctx, "iscsi.global.update", []any{defaults}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Resetting iSCSI Global Config", err.Error())
		return
	}
}

func (r *iscsiGlobalResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Singleton: read config directly
	var result iscsiGlobalResult
	err := r.client.Call(ctx, "iscsi.global.config", nil, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Importing iSCSI Global Config", err.Error())
		return
	}

	var model iscsiGlobalResourceModel
	populateISCSIGlobalState(&model, &result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func iscsiGlobalParamsFromModel(ctx context.Context, model *iscsiGlobalResourceModel, diags *diag.Diagnostics) map[string]any {
	params := map[string]any{
		"basename": model.Basename.ValueString(),
	}

	if !model.ISNSServers.IsNull() && !model.ISNSServers.IsUnknown() {
		var servers []string
		diags.Append(model.ISNSServers.ElementsAs(ctx, &servers, false)...)
		params["isns_servers"] = servers
	} else {
		params["isns_servers"] = []string{}
	}

	if !model.ListenPort.IsNull() && !model.ListenPort.IsUnknown() {
		params["listen_port"] = model.ListenPort.ValueInt64()
	}

	if !model.PoolAvailThreshold.IsNull() && !model.PoolAvailThreshold.IsUnknown() {
		params["pool_avail_threshold"] = model.PoolAvailThreshold.ValueInt64()
	}

	if !model.ALUA.IsNull() && !model.ALUA.IsUnknown() {
		params["alua"] = model.ALUA.ValueBool()
	}

	return params
}

func populateISCSIGlobalState(model *iscsiGlobalResourceModel, result *iscsiGlobalResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Basename = types.StringValue(result.Basename)
	model.ListenPort = types.Int64Value(result.ListenPort)
	model.ALUA = types.BoolValue(result.ALUA)

	if result.PoolAvailThreshold != nil {
		model.PoolAvailThreshold = types.Int64Value(*result.PoolAvailThreshold)
	} else {
		model.PoolAvailThreshold = types.Int64Null()
	}

	if len(result.ISNSServers) > 0 {
		elements := make([]attr.Value, len(result.ISNSServers))
		for i, s := range result.ISNSServers {
			elements[i] = types.StringValue(s)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.ISNSServers = list
	} else {
		model.ISNSServers = types.ListNull(types.StringType)
	}
}
