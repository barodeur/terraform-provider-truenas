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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*iscsiInitiatorResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiInitiatorResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiInitiatorResource)(nil)
)

type iscsiInitiatorResource struct {
	client *client.Client
}

type iscsiInitiatorResourceModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Initiators types.List   `tfsdk:"initiators"`
	Comment    types.String `tfsdk:"comment"`
}

type iscsiInitiatorResult struct {
	ID         int64    `json:"id"`
	Initiators []string `json:"initiators"`
	Comment    string   `json:"comment"`
}

func NewISCSIInitiatorResource() resource.Resource {
	return &iscsiInitiatorResource{}
}

func (r *iscsiInitiatorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_initiator"
}

func (r *iscsiInitiatorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS iSCSI authorized initiator group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the initiator group.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"initiators": schema.ListAttribute{
				Description: "List of initiator IQN names. Empty or null means allow all initiators.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"comment": schema.StringAttribute{
				Description: "Description of the initiator group.",
				Optional:    true,
			},
		},
	}
}

func (r *iscsiInitiatorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiInitiatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiInitiatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	if !plan.Initiators.IsNull() && !plan.Initiators.IsUnknown() {
		var initiators []string
		resp.Diagnostics.Append(plan.Initiators.ElementsAs(ctx, &initiators, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["initiators"] = initiators
	}

	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		params["comment"] = plan.Comment.ValueString()
	}

	var result iscsiInitiatorResult
	err := r.client.Call(ctx, "iscsi.initiator.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Initiator", err.Error())
		return
	}

	populateISCSIInitiatorState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiInitiatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiInitiatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiInitiatorResult
	err := r.client.Call(ctx, "iscsi.initiator.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Initiator", err.Error())
		return
	}

	populateISCSIInitiatorState(ctx, &state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiInitiatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiInitiatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiInitiatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	if !plan.Initiators.IsNull() && !plan.Initiators.IsUnknown() {
		var initiators []string
		resp.Diagnostics.Append(plan.Initiators.ElementsAs(ctx, &initiators, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["initiators"] = initiators
	} else if plan.Initiators.IsNull() {
		params["initiators"] = []string{}
	}

	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	} else {
		params["comment"] = ""
	}

	var result iscsiInitiatorResult
	err := r.client.Call(ctx, "iscsi.initiator.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Initiator", err.Error())
		return
	}

	populateISCSIInitiatorState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiInitiatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiInitiatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "iscsi.initiator.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Initiator", err.Error())
		return
	}
}

func (r *iscsiInitiatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Initiator",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateISCSIInitiatorState(_ context.Context, model *iscsiInitiatorResourceModel, result *iscsiInitiatorResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)

	if result.Comment != "" {
		model.Comment = types.StringValue(result.Comment)
	} else {
		model.Comment = types.StringNull()
	}

	if len(result.Initiators) > 0 {
		elements := make([]attr.Value, len(result.Initiators))
		for i, v := range result.Initiators {
			elements[i] = types.StringValue(v)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Initiators = list
	} else {
		model.Initiators = types.ListNull(types.StringType)
	}
}
