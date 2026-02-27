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
	_ resource.Resource                = (*nvmetSubsysResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetSubsysResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetSubsysResource)(nil)
)

type nvmetSubsysResource struct {
	client *client.Client
}

type nvmetSubsysResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	SubNQN       types.String `tfsdk:"subnqn"`
	Serial       types.String `tfsdk:"serial"`
	AllowAnyHost types.Bool   `tfsdk:"allow_any_host"`
	PIEnable     types.Bool   `tfsdk:"pi_enable"`
	QIDMax       types.Int64  `tfsdk:"qid_max"`
	IEEEOUI      types.String `tfsdk:"ieee_oui"`
	ANA          types.Bool   `tfsdk:"ana"`
}

type nvmetSubsysResult struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	SubNQN       string  `json:"subnqn"`
	Serial       string  `json:"serial"`
	AllowAnyHost bool    `json:"allow_any_host"`
	PIEnable     bool    `json:"pi_enable"`
	QIDMax       *int64  `json:"qid_max"`
	IEEEOUI      *string `json:"ieee_oui"`
	ANA          *bool   `json:"ana"`
}

func NewNVMeTSubsysResource() resource.Resource {
	return &nvmetSubsysResource{}
}

func (r *nvmetSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_subsys"
}

func (r *nvmetSubsysResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF subsystem.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF subsystem.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the subsystem.",
				Required:    true,
			},
			"subnqn": schema.StringAttribute{
				Description: "The subsystem NQN (11–223 characters). Auto-generated from basenqn if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"serial": schema.StringAttribute{
				Description: "The subsystem serial number.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_any_host": schema.BoolAttribute{
				Description: "Whether any host can connect. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"pi_enable": schema.BoolAttribute{
				Description: "Enable protection information.",
				Optional:    true,
				Computed:    true,
			},
			"qid_max": schema.Int64Attribute{
				Description: "Maximum queue IDs.",
				Optional:    true,
			},
			"ieee_oui": schema.StringAttribute{
				Description: "IEEE OUI identifier.",
				Optional:    true,
			},
			"ana": schema.BoolAttribute{
				Description: "Asymmetric Namespace Access (overrides global if set).",
				Optional:    true,
			},
		},
	}
}

func (r *nvmetSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":           plan.Name.ValueString(),
		"allow_any_host": plan.AllowAnyHost.ValueBool(),
	}

	if !plan.SubNQN.IsNull() && !plan.SubNQN.IsUnknown() {
		params["subnqn"] = plan.SubNQN.ValueString()
	}
	if !plan.PIEnable.IsNull() && !plan.PIEnable.IsUnknown() {
		params["pi_enable"] = plan.PIEnable.ValueBool()
	}
	if !plan.QIDMax.IsNull() && !plan.QIDMax.IsUnknown() {
		params["qid_max"] = plan.QIDMax.ValueInt64()
	}
	if !plan.IEEEOUI.IsNull() && !plan.IEEEOUI.IsUnknown() {
		params["ieee_oui"] = plan.IEEEOUI.ValueString()
	}
	if !plan.ANA.IsNull() && !plan.ANA.IsUnknown() {
		params["ana"] = plan.ANA.ValueBool()
	}

	var result nvmetSubsysResult
	err := r.client.Call(ctx, "nvmet.subsys.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Subsystem", err.Error())
		return
	}

	populateNVMeTSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetSubsysResult
	err := r.client.Call(ctx, "nvmet.subsys.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Subsystem", err.Error())
		return
	}

	populateNVMeTSubsysState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetSubsysResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":           plan.Name.ValueString(),
		"allow_any_host": plan.AllowAnyHost.ValueBool(),
	}

	if !plan.PIEnable.IsNull() && !plan.PIEnable.IsUnknown() {
		params["pi_enable"] = plan.PIEnable.ValueBool()
	}
	if !plan.QIDMax.IsNull() {
		params["qid_max"] = plan.QIDMax.ValueInt64()
	} else {
		params["qid_max"] = nil
	}
	if !plan.IEEEOUI.IsNull() {
		params["ieee_oui"] = plan.IEEEOUI.ValueString()
	} else {
		params["ieee_oui"] = ""
	}
	if !plan.ANA.IsNull() {
		params["ana"] = plan.ANA.ValueBool()
	} else {
		params["ana"] = nil
	}

	var result nvmetSubsysResult
	err := r.client.Call(ctx, "nvmet.subsys.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Subsystem", err.Error())
		return
	}

	populateNVMeTSubsysState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetSubsysResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.subsys.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Subsystem", err.Error())
		return
	}
}

func (r *nvmetSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Subsystem",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTSubsysState(model *nvmetSubsysResourceModel, result *nvmetSubsysResult) {
	model.ID = types.Int64Value(result.ID)
	model.Name = types.StringValue(result.Name)
	model.SubNQN = types.StringValue(result.SubNQN)
	model.Serial = types.StringValue(result.Serial)
	model.AllowAnyHost = types.BoolValue(result.AllowAnyHost)
	model.PIEnable = types.BoolValue(result.PIEnable)

	if result.QIDMax != nil {
		model.QIDMax = types.Int64Value(*result.QIDMax)
	} else {
		model.QIDMax = types.Int64Null()
	}

	if result.IEEEOUI != nil && *result.IEEEOUI != "" {
		model.IEEEOUI = types.StringValue(*result.IEEEOUI)
	} else {
		model.IEEEOUI = types.StringNull()
	}

	if result.ANA != nil {
		model.ANA = types.BoolValue(*result.ANA)
	} else {
		model.ANA = types.BoolNull()
	}
}
