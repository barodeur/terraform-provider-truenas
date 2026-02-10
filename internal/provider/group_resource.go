package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*groupResource)(nil)
	_ resource.ResourceWithConfigure   = (*groupResource)(nil)
	_ resource.ResourceWithImportState = (*groupResource)(nil)
)

type groupResource struct {
	client *client.Client
}

type groupResourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	GID             types.Int64  `tfsdk:"gid"`
	Name            types.String `tfsdk:"name"`
	Smb             types.Bool   `tfsdk:"smb"`
	AllowDuplicateGID types.Bool `tfsdk:"allow_duplicate_gid"`
	Builtin         types.Bool   `tfsdk:"builtin"`
}

type groupResult struct {
	ID      int64  `json:"id"`
	GID     int64  `json:"gid"`
	Group   string `json:"group"`
	Builtin bool   `json:"builtin"`
	Smb     bool   `json:"smb"`
}

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS local group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "The GID of the group. If not specified, TrueNAS assigns the next available GID. Cannot be changed after creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the group.",
				Required:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the group is available for SMB authentication.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_duplicate_gid": schema.BoolAttribute{
				Description: "Allow duplicate GID. Only used during creation.",
				Optional:    true,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system group.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.GID.IsNull() && !plan.GID.IsUnknown() {
		params["gid"] = plan.GID.ValueInt64()
	}
	if !plan.Smb.IsNull() && !plan.Smb.IsUnknown() {
		params["smb"] = plan.Smb.ValueBool()
	}
	if !plan.AllowDuplicateGID.IsNull() && !plan.AllowDuplicateGID.IsUnknown() && plan.AllowDuplicateGID.ValueBool() {
		params["allow_duplicate_gid"] = true
	}

	var groupID int64
	err := r.client.Call(ctx, "group.create", []any{params}, &groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Group", err.Error())
		return
	}

	var result groupResult
	err = r.client.Call(ctx, "group.get_instance", []any{groupID}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Group After Creation", err.Error())
		return
	}

	populateGroupState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result groupResult
	err := r.client.Call(ctx, "group.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Group", err.Error())
		return
	}

	populateGroupState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.Smb.IsNull() {
		params["smb"] = plan.Smb.ValueBool()
	}

	var updatedID int64
	err := r.client.Call(ctx, "group.update", []any{state.ID.ValueInt64(), params}, &updatedID)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Group", err.Error())
		return
	}

	var result groupResult
	err = r.client.Call(ctx, "group.get_instance", []any{updatedID}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Group After Update", err.Error())
		return
	}

	populateGroupState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "group.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Group", err.Error())
		return
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Group",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateGroupState(model *groupResourceModel, result *groupResult) {
	model.ID = types.Int64Value(result.ID)
	model.GID = types.Int64Value(result.GID)
	model.Name = types.StringValue(result.Group)
	model.Smb = types.BoolValue(result.Smb)
	model.Builtin = types.BoolValue(result.Builtin)
}
