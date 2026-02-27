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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*nvmetHostResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetHostResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetHostResource)(nil)
)

type nvmetHostResource struct {
	client *client.Client
}

type nvmetHostResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	HostNQN        types.String `tfsdk:"hostnqn"`
	DHCHAPKey      types.String `tfsdk:"dhchap_key"`
	DHCHAPCtrlKey  types.String `tfsdk:"dhchap_ctrl_key"`
	DHCHAPDHGroup  types.String `tfsdk:"dhchap_dhgroup"`
	DHCHAPHash     types.String `tfsdk:"dhchap_hash"`
}

type nvmetHostResult struct {
	ID             int64   `json:"id"`
	HostNQN        string  `json:"hostnqn"`
	DHCHAPKey      string  `json:"dhchap_key"`
	DHCHAPCtrlKey  string  `json:"dhchap_ctrl_key"`
	DHCHAPDHGroup  *string `json:"dhchap_dhgroup"`
	DHCHAPHash     string  `json:"dhchap_hash"`
}

func NewNVMeTHostResource() resource.Resource {
	return &nvmetHostResource{}
}

func (r *nvmetHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_host"
}

func (r *nvmetHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF host.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF host.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hostnqn": schema.StringAttribute{
				Description: "NQN of the connecting host (11–223 characters).",
				Required:    true,
			},
			"dhchap_key": schema.StringAttribute{
				Description: "Host authentication secret.",
				Optional:    true,
				Sensitive:   true,
			},
			"dhchap_ctrl_key": schema.StringAttribute{
				Description: "Bidirectional authentication secret.",
				Optional:    true,
				Sensitive:   true,
			},
			"dhchap_dhgroup": schema.StringAttribute{
				Description: "DH-CHAP Diffie-Hellman group (2048-BIT, 3072-BIT, 4096-BIT, 6144-BIT, 8192-BIT, or null).",
				Optional:    true,
			},
			"dhchap_hash": schema.StringAttribute{
				Description: "DH-CHAP hash algorithm (SHA-256, SHA-384, SHA-512). Defaults to SHA-256.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SHA-256"),
			},
		},
	}
}

func (r *nvmetHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetHostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"hostnqn": plan.HostNQN.ValueString(),
	}

	if !plan.DHCHAPKey.IsNull() && !plan.DHCHAPKey.IsUnknown() {
		params["dhchap_key"] = plan.DHCHAPKey.ValueString()
	}
	if !plan.DHCHAPCtrlKey.IsNull() && !plan.DHCHAPCtrlKey.IsUnknown() {
		params["dhchap_ctrl_key"] = plan.DHCHAPCtrlKey.ValueString()
	}
	if !plan.DHCHAPDHGroup.IsNull() && !plan.DHCHAPDHGroup.IsUnknown() {
		params["dhchap_dhgroup"] = plan.DHCHAPDHGroup.ValueString()
	}
	if !plan.DHCHAPHash.IsNull() && !plan.DHCHAPHash.IsUnknown() {
		params["dhchap_hash"] = plan.DHCHAPHash.ValueString()
	}

	var result nvmetHostResult
	err := r.client.Call(ctx, "nvmet.host.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Host", err.Error())
		return
	}

	populateNVMeTHostState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetHostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetHostResult
	err := r.client.Call(ctx, "nvmet.host.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Host", err.Error())
		return
	}

	populateNVMeTHostState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetHostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetHostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"hostnqn": plan.HostNQN.ValueString(),
	}

	if !plan.DHCHAPKey.IsNull() && !plan.DHCHAPKey.IsUnknown() {
		params["dhchap_key"] = plan.DHCHAPKey.ValueString()
	}
	if !plan.DHCHAPCtrlKey.IsNull() && !plan.DHCHAPCtrlKey.IsUnknown() {
		params["dhchap_ctrl_key"] = plan.DHCHAPCtrlKey.ValueString()
	}
	if !plan.DHCHAPDHGroup.IsNull() {
		params["dhchap_dhgroup"] = plan.DHCHAPDHGroup.ValueString()
	} else {
		params["dhchap_dhgroup"] = nil
	}
	if !plan.DHCHAPHash.IsNull() && !plan.DHCHAPHash.IsUnknown() {
		params["dhchap_hash"] = plan.DHCHAPHash.ValueString()
	}

	var result nvmetHostResult
	err := r.client.Call(ctx, "nvmet.host.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Host", err.Error())
		return
	}

	populateNVMeTHostState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetHostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.host.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Host", err.Error())
		return
	}
}

func (r *nvmetHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Host",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTHostState(model *nvmetHostResourceModel, result *nvmetHostResult) {
	model.ID = types.Int64Value(result.ID)
	model.HostNQN = types.StringValue(result.HostNQN)

	if result.DHCHAPKey != "" {
		model.DHCHAPKey = types.StringValue(result.DHCHAPKey)
	} else {
		model.DHCHAPKey = types.StringNull()
	}

	if result.DHCHAPCtrlKey != "" {
		model.DHCHAPCtrlKey = types.StringValue(result.DHCHAPCtrlKey)
	} else {
		model.DHCHAPCtrlKey = types.StringNull()
	}

	if result.DHCHAPDHGroup != nil && *result.DHCHAPDHGroup != "" {
		model.DHCHAPDHGroup = types.StringValue(*result.DHCHAPDHGroup)
	} else {
		model.DHCHAPDHGroup = types.StringNull()
	}

	model.DHCHAPHash = types.StringValue(result.DHCHAPHash)
}
