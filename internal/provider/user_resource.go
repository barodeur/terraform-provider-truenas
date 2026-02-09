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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*userResource)(nil)
	_ resource.ResourceWithConfigure   = (*userResource)(nil)
	_ resource.ResourceWithImportState = (*userResource)(nil)
)

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	UID              types.Int64  `tfsdk:"uid"`
	Username         types.String `tfsdk:"username"`
	FullName         types.String `tfsdk:"full_name"`
	Email            types.String `tfsdk:"email"`
	Password         types.String `tfsdk:"password"`
	PasswordDisabled types.Bool   `tfsdk:"password_disabled"`
	Group            types.Int64  `tfsdk:"group"`
	GroupCreate      types.Bool   `tfsdk:"group_create"`
	Groups           types.List   `tfsdk:"groups"`
	Home             types.String `tfsdk:"home"`
	HomeCreate       types.Bool   `tfsdk:"home_create"`
	Shell            types.String `tfsdk:"shell"`
	Sshpubkey        types.String `tfsdk:"sshpubkey"`
	Smb              types.Bool   `tfsdk:"smb"`
	Locked           types.Bool   `tfsdk:"locked"`
	Builtin          types.Bool   `tfsdk:"builtin"`
}

type userGroupRef struct {
	ID int64 `json:"id"`
}

type userResult struct {
	ID               int64        `json:"id"`
	UID              int64        `json:"uid"`
	Username         string       `json:"username"`
	FullName         string       `json:"full_name"`
	Email            *string      `json:"email"`
	PasswordDisabled bool         `json:"password_disabled"`
	Group            userGroupRef `json:"group"`
	Groups           []int64      `json:"groups"`
	Home             string       `json:"home"`
	Shell            string       `json:"shell"`
	Sshpubkey        *string      `json:"sshpubkey"`
	Smb              bool         `json:"smb"`
	Locked           bool         `json:"locked"`
	Builtin          bool         `json:"builtin"`
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS local user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.Int64Attribute{
				Description: "The UID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The username.",
				Required:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the user.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email address of the user.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The user password. Write-only — cannot be read back from TrueNAS.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_disabled": schema.BoolAttribute{
				Description: "Whether password login is disabled for this user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"group": schema.Int64Attribute{
				Description: "The primary group ID. If not specified and group_create is true, a group with the username is created.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"group_create": schema.BoolAttribute{
				Description: "Create a new primary group with the same name as the user. Only used during creation.",
				Optional:    true,
			},
			"groups": schema.ListAttribute{
				Description: "List of auxiliary group IDs.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"home": schema.StringAttribute{
				Description: "The home directory path.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"home_create": schema.BoolAttribute{
				Description: "Create the home directory if it doesn't exist. Only used during creation.",
				Optional:    true,
			},
			"shell": schema.StringAttribute{
				Description: "The user's login shell (e.g. /usr/bin/bash, /usr/sbin/nologin).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sshpubkey": schema.StringAttribute{
				Description: "SSH public key for the user.",
				Optional:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the user is available for SMB authentication.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the user account is locked.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"username":  plan.Username.ValueString(),
		"full_name": plan.FullName.ValueString(),
	}

	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		params["email"] = plan.Email.ValueString()
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		params["password"] = plan.Password.ValueString()
	}
	if !plan.PasswordDisabled.IsNull() && !plan.PasswordDisabled.IsUnknown() {
		params["password_disabled"] = plan.PasswordDisabled.ValueBool()
	}
	if !plan.Group.IsNull() && !plan.Group.IsUnknown() {
		params["group"] = plan.Group.ValueInt64()
	}
	if !plan.GroupCreate.IsNull() && !plan.GroupCreate.IsUnknown() {
		params["group_create"] = plan.GroupCreate.ValueBool()
	} else if plan.Group.IsNull() || plan.Group.IsUnknown() {
		// Default: create a group if no explicit group is set
		params["group_create"] = true
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		var groupIDs []int64
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groupIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["groups"] = groupIDs
	}
	if !plan.Home.IsNull() && !plan.Home.IsUnknown() {
		params["home"] = plan.Home.ValueString()
	}
	if !plan.HomeCreate.IsNull() && !plan.HomeCreate.IsUnknown() && plan.HomeCreate.ValueBool() {
		params["home_create"] = true
	}
	if !plan.Shell.IsNull() && !plan.Shell.IsUnknown() {
		params["shell"] = plan.Shell.ValueString()
	}
	if !plan.Sshpubkey.IsNull() && !plan.Sshpubkey.IsUnknown() {
		params["sshpubkey"] = plan.Sshpubkey.ValueString()
	}
	if !plan.Smb.IsNull() && !plan.Smb.IsUnknown() {
		params["smb"] = plan.Smb.ValueBool()
	}
	if !plan.Locked.IsNull() && !plan.Locked.IsUnknown() {
		params["locked"] = plan.Locked.ValueBool()
	}

	var result userResult
	err := r.client.Call(ctx, "user.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating User", err.Error())
		return
	}

	populateUserState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result userResult
	err := r.client.Call(ctx, "user.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading User", err.Error())
		return
	}

	// Preserve password from prior state since it can't be read back
	priorPassword := state.Password

	populateUserState(ctx, &state, &result, &resp.Diagnostics)
	state.Password = priorPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"username":  plan.Username.ValueString(),
		"full_name": plan.FullName.ValueString(),
	}

	if !plan.Email.IsNull() {
		params["email"] = plan.Email.ValueString()
	}
	if !plan.Password.IsNull() && !plan.Password.Equal(state.Password) {
		params["password"] = plan.Password.ValueString()
	}
	if !plan.PasswordDisabled.IsNull() {
		params["password_disabled"] = plan.PasswordDisabled.ValueBool()
	}
	if !plan.Group.IsNull() {
		params["group"] = plan.Group.ValueInt64()
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		var groupIDs []int64
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groupIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["groups"] = groupIDs
	}
	if !plan.Home.IsNull() {
		params["home"] = plan.Home.ValueString()
	}
	if !plan.Shell.IsNull() {
		params["shell"] = plan.Shell.ValueString()
	}
	if plan.Sshpubkey.IsNull() {
		params["sshpubkey"] = ""
	} else {
		params["sshpubkey"] = plan.Sshpubkey.ValueString()
	}
	if !plan.Smb.IsNull() {
		params["smb"] = plan.Smb.ValueBool()
	}
	if !plan.Locked.IsNull() {
		params["locked"] = plan.Locked.ValueBool()
	}

	var result userResult
	err := r.client.Call(ctx, "user.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating User", err.Error())
		return
	}

	// Preserve password from plan since it can't be read back
	priorPassword := plan.Password

	populateUserState(ctx, &plan, &result, &resp.Diagnostics)
	plan.Password = priorPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteOpts := map[string]any{
		"delete_group": true,
	}

	err := r.client.Call(ctx, "user.delete", []any{state.ID.ValueInt64(), deleteOpts}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting User", err.Error())
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing User",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateUserState(ctx context.Context, model *userResourceModel, result *userResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.UID = types.Int64Value(result.UID)
	model.Username = types.StringValue(result.Username)
	model.FullName = types.StringValue(result.FullName)

	if result.Email != nil && *result.Email != "" {
		model.Email = types.StringValue(*result.Email)
	} else {
		model.Email = types.StringNull()
	}

	model.PasswordDisabled = types.BoolValue(result.PasswordDisabled)
	model.Group = types.Int64Value(result.Group.ID)

	if len(result.Groups) > 0 {
		elements := make([]attr.Value, len(result.Groups))
		for i, gid := range result.Groups {
			elements[i] = types.Int64Value(gid)
		}
		groupsList, d := types.ListValue(types.Int64Type, elements)
		diags.Append(d...)
		model.Groups = groupsList
	} else {
		model.Groups = types.ListNull(types.Int64Type)
	}

	model.Home = types.StringValue(result.Home)
	model.Shell = types.StringValue(result.Shell)

	if result.Sshpubkey != nil && *result.Sshpubkey != "" {
		model.Sshpubkey = types.StringValue(*result.Sshpubkey)
	} else {
		model.Sshpubkey = types.StringNull()
	}

	model.Smb = types.BoolValue(result.Smb)
	model.Locked = types.BoolValue(result.Locked)
	model.Builtin = types.BoolValue(result.Builtin)
}
