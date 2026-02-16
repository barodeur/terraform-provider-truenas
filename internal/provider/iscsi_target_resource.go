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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*iscsiTargetResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiTargetResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiTargetResource)(nil)
)

type iscsiTargetResource struct {
	client *client.Client
}

type iscsiTargetResourceModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Alias  types.String `tfsdk:"alias"`
	Mode   types.String `tfsdk:"mode"`
	Groups types.List   `tfsdk:"groups"`
}

var iscsiTargetGroupAttrTypes = map[string]attr.Type{
	"portal":     types.Int64Type,
	"initiator":  types.Int64Type,
	"authmethod": types.StringType,
	"auth":       types.Int64Type,
}

type iscsiTargetGroupResult struct {
	Portal     int64  `json:"portal"`
	Initiator  *int64 `json:"initiator"`
	Authmethod string `json:"authmethod"`
	Auth       *int64 `json:"auth"`
}

type iscsiTargetResult struct {
	ID     int64                    `json:"id"`
	Name   string                   `json:"name"`
	Alias  string                   `json:"alias"`
	Mode   string                   `json:"mode"`
	Groups []iscsiTargetGroupResult `json:"groups"`
}

func NewISCSITargetResource() resource.Resource {
	return &iscsiTargetResource{}
}

func (r *iscsiTargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_target"
}

func (r *iscsiTargetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS iSCSI target.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the target.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The base name of the target (appended to the global basename).",
				Required:    true,
			},
			"alias": schema.StringAttribute{
				Description: "An optional alias for the target.",
				Optional:    true,
			},
			"mode": schema.StringAttribute{
				Description: "Target mode: ISCSI, FC, or BOTH.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"groups": schema.ListNestedAttribute{
				Description: "Portal-initiator group associations.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"portal": schema.Int64Attribute{
							Description: "Portal ID for this group.",
							Required:    true,
						},
						"initiator": schema.Int64Attribute{
							Description: "Initiator group ID. Omit to allow all initiators.",
							Optional:    true,
						},
						"authmethod": schema.StringAttribute{
							Description: "Authentication method: NONE, CHAP, or CHAP_MUTUAL.",
							Optional:    true,
							Computed:    true,
						},
						"auth": schema.Int64Attribute{
							Description: "Auth group tag (references iscsi_auth tag).",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *iscsiTargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiTargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.Alias.IsNull() && !plan.Alias.IsUnknown() {
		params["alias"] = plan.Alias.ValueString()
	}
	if !plan.Mode.IsNull() && !plan.Mode.IsUnknown() {
		params["mode"] = plan.Mode.ValueString()
	}

	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		groups := iscsiTargetGroupsFromPlan(ctx, plan.Groups, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		params["groups"] = groups
	}

	var result iscsiTargetResult
	err := r.client.Call(ctx, "iscsi.target.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Target", err.Error())
		return
	}

	populateISCSITargetState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiTargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiTargetResult
	err := r.client.Call(ctx, "iscsi.target.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Target", err.Error())
		return
	}

	populateISCSITargetState(&state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiTargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiTargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.Alias.IsNull() {
		params["alias"] = plan.Alias.ValueString()
	} else {
		params["alias"] = ""
	}
	if !plan.Mode.IsNull() && !plan.Mode.IsUnknown() {
		params["mode"] = plan.Mode.ValueString()
	}

	if !plan.Groups.IsNull() {
		groups := iscsiTargetGroupsFromPlan(ctx, plan.Groups, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		params["groups"] = groups
	} else {
		params["groups"] = []map[string]any{}
	}

	var result iscsiTargetResult
	err := r.client.Call(ctx, "iscsi.target.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Target", err.Error())
		return
	}

	populateISCSITargetState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiTargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pass false, false: no force, no cascade
	err := r.client.Call(ctx, "iscsi.target.delete", []any{state.ID.ValueInt64(), false, false}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Target", err.Error())
		return
	}
}

func (r *iscsiTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Target",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func iscsiTargetGroupsFromPlan(ctx context.Context, listVal types.List, diags *diag.Diagnostics) []map[string]any {
	var groupObjs []types.Object
	diags.Append(listVal.ElementsAs(ctx, &groupObjs, false)...)
	if diags.HasError() {
		return nil
	}

	result := make([]map[string]any, len(groupObjs))
	for i, obj := range groupObjs {
		attrs := obj.Attributes()
		group := map[string]any{
			"portal": attrs["portal"].(types.Int64).ValueInt64(),
		}

		initiator := attrs["initiator"].(types.Int64)
		if !initiator.IsNull() && !initiator.IsUnknown() {
			group["initiator"] = initiator.ValueInt64()
		}

		authmethod := attrs["authmethod"].(types.String)
		if !authmethod.IsNull() && !authmethod.IsUnknown() {
			group["authmethod"] = authmethod.ValueString()
		}

		auth := attrs["auth"].(types.Int64)
		if !auth.IsNull() && !auth.IsUnknown() {
			group["auth"] = auth.ValueInt64()
		}

		result[i] = group
	}
	return result
}

func populateISCSITargetState(model *iscsiTargetResourceModel, result *iscsiTargetResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Name = types.StringValue(result.Name)
	model.Mode = types.StringValue(result.Mode)

	if result.Alias != "" {
		model.Alias = types.StringValue(result.Alias)
	} else {
		model.Alias = types.StringNull()
	}

	if len(result.Groups) > 0 {
		groupElements := make([]attr.Value, len(result.Groups))
		for i, g := range result.Groups {
			attrs := map[string]attr.Value{
				"portal":     types.Int64Value(g.Portal),
				"authmethod": types.StringValue(g.Authmethod),
			}

			if g.Initiator != nil && *g.Initiator != 0 {
				attrs["initiator"] = types.Int64Value(*g.Initiator)
			} else {
				attrs["initiator"] = types.Int64Null()
			}

			if g.Auth != nil && *g.Auth != 0 {
				attrs["auth"] = types.Int64Value(*g.Auth)
			} else {
				attrs["auth"] = types.Int64Null()
			}

			obj, d := types.ObjectValue(iscsiTargetGroupAttrTypes, attrs)
			diags.Append(d...)
			groupElements[i] = obj
		}

		list, d := types.ListValue(types.ObjectType{AttrTypes: iscsiTargetGroupAttrTypes}, groupElements)
		diags.Append(d...)
		model.Groups = list
	} else {
		model.Groups = types.ListNull(types.ObjectType{AttrTypes: iscsiTargetGroupAttrTypes})
	}
}
