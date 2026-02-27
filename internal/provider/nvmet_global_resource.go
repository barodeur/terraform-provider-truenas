package provider

import (
	"context"
	"fmt"
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
	_ resource.Resource                = (*nvmetGlobalResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetGlobalResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetGlobalResource)(nil)
)

type nvmetGlobalResource struct {
	client *client.Client
}

type nvmetGlobalResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Basenqn      types.String `tfsdk:"basenqn"`
	Kernel       types.Bool   `tfsdk:"kernel"`
	ANA          types.Bool   `tfsdk:"ana"`
	RDMA         types.Bool   `tfsdk:"rdma"`
	XportReferral types.Bool  `tfsdk:"xport_referral"`
}

type nvmetGlobalResult struct {
	ID           int64  `json:"id"`
	Basenqn      string `json:"basenqn"`
	Kernel       bool   `json:"kernel"`
	ANA          bool   `json:"ana"`
	RDMA         bool   `json:"rdma"`
	XportReferral bool  `json:"xport_referral"`
}

func NewNVMeTGlobalResource() resource.Resource {
	return &nvmetGlobalResource{}
}

func (r *nvmetGlobalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_global"
}

func (r *nvmetGlobalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TrueNAS NVMe-oF global configuration. This is a singleton resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF global configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"basenqn": schema.StringAttribute{
				Description: "NQN prefix used for subsystem creation (11–223 characters).",
				Optional:    true,
				Computed:    true,
			},
			"kernel": schema.BoolAttribute{
				Description: "NVMe-oF backend selection.",
				Optional:    true,
				Computed:    true,
			},
			"ana": schema.BoolAttribute{
				Description: "Asymmetric Namespace Access.",
				Optional:    true,
				Computed:    true,
			},
			"rdma": schema.BoolAttribute{
				Description: "RDMA enabled (Enterprise only).",
				Optional:    true,
				Computed:    true,
			},
			"xport_referral": schema.BoolAttribute{
				Description: "Cross-port referral generation.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *nvmetGlobalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	if !plan.Basenqn.IsNull() && !plan.Basenqn.IsUnknown() {
		params["basenqn"] = plan.Basenqn.ValueString()
	}
	if !plan.Kernel.IsNull() && !plan.Kernel.IsUnknown() {
		params["kernel"] = plan.Kernel.ValueBool()
	}
	if !plan.ANA.IsNull() && !plan.ANA.IsUnknown() {
		params["ana"] = plan.ANA.ValueBool()
	}
	if !plan.RDMA.IsNull() && !plan.RDMA.IsUnknown() {
		params["rdma"] = plan.RDMA.ValueBool()
	}
	if !plan.XportReferral.IsNull() && !plan.XportReferral.IsUnknown() {
		params["xport_referral"] = plan.XportReferral.ValueBool()
	}

	var result nvmetGlobalResult
	err := r.client.Call(ctx, "nvmet.global.update", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Global Config", err.Error())
		return
	}

	populateNVMeTGlobalState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetGlobalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetGlobalResult
	err := r.client.Call(ctx, "nvmet.global.config", nil, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading NVMe-oF Global Config", err.Error())
		return
	}

	populateNVMeTGlobalState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	if !plan.Basenqn.IsNull() && !plan.Basenqn.IsUnknown() {
		params["basenqn"] = plan.Basenqn.ValueString()
	}
	if !plan.Kernel.IsNull() && !plan.Kernel.IsUnknown() {
		params["kernel"] = plan.Kernel.ValueBool()
	}
	if !plan.ANA.IsNull() && !plan.ANA.IsUnknown() {
		params["ana"] = plan.ANA.ValueBool()
	}
	if !plan.RDMA.IsNull() && !plan.RDMA.IsUnknown() {
		params["rdma"] = plan.RDMA.ValueBool()
	}
	if !plan.XportReferral.IsNull() && !plan.XportReferral.IsUnknown() {
		params["xport_referral"] = plan.XportReferral.ValueBool()
	}

	var result nvmetGlobalResult
	err := r.client.Call(ctx, "nvmet.global.update", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Global Config", err.Error())
		return
	}

	populateNVMeTGlobalState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetGlobalResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton resource — delete is a no-op.
}

func (r *nvmetGlobalResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var result nvmetGlobalResult
	err := r.client.Call(ctx, "nvmet.global.config", nil, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Importing NVMe-oF Global Config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), result.ID)...)
}

func populateNVMeTGlobalState(model *nvmetGlobalResourceModel, result *nvmetGlobalResult) {
	model.ID = types.Int64Value(result.ID)
	model.Basenqn = types.StringValue(result.Basenqn)
	model.Kernel = types.BoolValue(result.Kernel)
	model.ANA = types.BoolValue(result.ANA)
	model.RDMA = types.BoolValue(result.RDMA)
	model.XportReferral = types.BoolValue(result.XportReferral)
}
