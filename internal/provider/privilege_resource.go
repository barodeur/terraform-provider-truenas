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
	_ resource.Resource                = (*privilegeResource)(nil)
	_ resource.ResourceWithConfigure   = (*privilegeResource)(nil)
	_ resource.ResourceWithImportState = (*privilegeResource)(nil)
)

type privilegeResource struct {
	client *client.Client
}

type privilegeResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	BuiltinName types.String `tfsdk:"builtin_name"`
	LocalGroups types.List   `tfsdk:"local_groups"`
	DSGroups    types.List   `tfsdk:"ds_groups"`
	Roles       types.List   `tfsdk:"roles"`
	WebShell    types.Bool   `tfsdk:"web_shell"`
}

type privilegeGroupObject struct {
	ID  int64 `json:"id"`
	GID int64 `json:"gid"`
}

type privilegeResult struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	BuiltinName *string                `json:"builtin_name"`
	LocalGroups []privilegeGroupObject `json:"local_groups"`
	DSGroups    []privilegeGroupObject `json:"ds_groups"`
	Roles       []string               `json:"roles"`
	WebShell    bool                   `json:"web_shell"`
}

func NewPrivilegeResource() resource.Resource {
	return &privilegeResource{}
}

func (r *privilegeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege"
}

func (r *privilegeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS privilege (RBAC role assignment).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the privilege.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the privilege.",
				Required:    true,
			},
			"builtin_name": schema.StringAttribute{
				Description: "The built-in name of the privilege, if it is a system-defined privilege.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"local_groups": schema.ListAttribute{
				Description: "List of local group GIDs assigned to this privilege.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"ds_groups": schema.ListAttribute{
				Description: "List of directory service group GIDs assigned to this privilege.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"roles": schema.ListAttribute{
				Description: "List of role names assigned to this privilege (e.g. READONLY_ADMIN).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"web_shell": schema.BoolAttribute{
				Description: "Whether members of this privilege can access the web shell.",
				Required:    true,
			},
		},
	}
}

func (r *privilegeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privilegeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privilegeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":      plan.Name.ValueString(),
		"web_shell": plan.WebShell.ValueBool(),
	}

	if !plan.LocalGroups.IsNull() && !plan.LocalGroups.IsUnknown() {
		var ids []int64
		resp.Diagnostics.Append(plan.LocalGroups.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["local_groups"] = ids
	}
	if !plan.DSGroups.IsNull() && !plan.DSGroups.IsUnknown() {
		var ids []int64
		resp.Diagnostics.Append(plan.DSGroups.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["ds_groups"] = ids
	}
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["roles"] = roles
	}

	var result privilegeResult
	err := r.client.Call(ctx, "privilege.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Privilege", err.Error())
		return
	}

	populatePrivilegeState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privilegeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result privilegeResult
	err := r.client.Call(ctx, "privilege.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Privilege", err.Error())
		return
	}

	populatePrivilegeState(ctx, &state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privilegeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privilegeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state privilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":      plan.Name.ValueString(),
		"web_shell": plan.WebShell.ValueBool(),
	}

	if !plan.LocalGroups.IsNull() && !plan.LocalGroups.IsUnknown() {
		var ids []int64
		resp.Diagnostics.Append(plan.LocalGroups.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["local_groups"] = ids
	}
	if !plan.DSGroups.IsNull() && !plan.DSGroups.IsUnknown() {
		var ids []int64
		resp.Diagnostics.Append(plan.DSGroups.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["ds_groups"] = ids
	}
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["roles"] = roles
	}

	var result privilegeResult
	err := r.client.Call(ctx, "privilege.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Privilege", err.Error())
		return
	}

	populatePrivilegeState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privilegeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "privilege.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Privilege", err.Error())
		return
	}
}

func (r *privilegeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Privilege",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populatePrivilegeState(ctx context.Context, model *privilegeResourceModel, result *privilegeResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Name = types.StringValue(result.Name)

	if result.BuiltinName != nil && *result.BuiltinName != "" {
		model.BuiltinName = types.StringValue(*result.BuiltinName)
	} else {
		model.BuiltinName = types.StringNull()
	}

	if len(result.LocalGroups) > 0 {
		elements := make([]attr.Value, len(result.LocalGroups))
		for i, g := range result.LocalGroups {
			elements[i] = types.Int64Value(g.GID)
		}
		list, d := types.ListValue(types.Int64Type, elements)
		diags.Append(d...)
		model.LocalGroups = list
	} else {
		model.LocalGroups = types.ListNull(types.Int64Type)
	}

	if len(result.DSGroups) > 0 {
		elements := make([]attr.Value, len(result.DSGroups))
		for i, g := range result.DSGroups {
			elements[i] = types.Int64Value(g.GID)
		}
		list, d := types.ListValue(types.Int64Type, elements)
		diags.Append(d...)
		model.DSGroups = list
	} else {
		model.DSGroups = types.ListNull(types.Int64Type)
	}

	if len(result.Roles) > 0 {
		elements := make([]attr.Value, len(result.Roles))
		for i, r := range result.Roles {
			elements[i] = types.StringValue(r)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Roles = list
	} else {
		model.Roles = types.ListNull(types.StringType)
	}

	model.WebShell = types.BoolValue(result.WebShell)
}
