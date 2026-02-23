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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*iscsiAuthResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiAuthResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiAuthResource)(nil)
)

type iscsiAuthResource struct {
	client *client.Client
}

type iscsiAuthResourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Tag           types.Int64  `tfsdk:"tag"`
	User          types.String `tfsdk:"user"`
	Secret        types.String `tfsdk:"secret"`
	Peeruser      types.String `tfsdk:"peeruser"`
	Peersecret    types.String `tfsdk:"peersecret"`
	DiscoveryAuth types.String `tfsdk:"discovery_auth"`
}

type iscsiAuthResult struct {
	ID            int64  `json:"id"`
	Tag           int64  `json:"tag"`
	User          string `json:"user"`
	Secret        string `json:"secret"`
	Peeruser      string `json:"peeruser"`
	Peersecret    string `json:"peersecret"`
	DiscoveryAuth string `json:"discovery_auth"`
}

func NewISCSIAuthResource() resource.Resource {
	return &iscsiAuthResource{}
}

func (r *iscsiAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_auth"
}

func (r *iscsiAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages TrueNAS iSCSI CHAP authentication credentials.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the auth entry.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tag": schema.Int64Attribute{
				Description: "The authentication group tag.",
				Required:    true,
			},
			"user": schema.StringAttribute{
				Description: "CHAP user name.",
				Required:    true,
			},
			"secret": schema.StringAttribute{
				Description: "CHAP secret (12-16 characters).",
				Required:    true,
				Sensitive:   true,
			},
			"peeruser": schema.StringAttribute{
				Description: "Mutual CHAP peer user name.",
				Optional:    true,
			},
			"peersecret": schema.StringAttribute{
				Description: "Mutual CHAP peer secret (12-16 characters).",
				Optional:    true,
				Sensitive:   true,
			},
			"discovery_auth": schema.StringAttribute{
				Description: "Discovery authentication method (NONE, CHAP, or CHAP_MUTUAL).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *iscsiAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiAuthResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"tag":    plan.Tag.ValueInt64(),
		"user":   plan.User.ValueString(),
		"secret": plan.Secret.ValueString(),
	}

	if !plan.Peeruser.IsNull() && !plan.Peeruser.IsUnknown() {
		params["peeruser"] = plan.Peeruser.ValueString()
	}
	if !plan.Peersecret.IsNull() && !plan.Peersecret.IsUnknown() {
		params["peersecret"] = plan.Peersecret.ValueString()
	}
	if !plan.DiscoveryAuth.IsNull() && !plan.DiscoveryAuth.IsUnknown() {
		params["discovery_auth"] = plan.DiscoveryAuth.ValueString()
	}

	var result iscsiAuthResult
	err := r.client.Call(ctx, "iscsi.auth.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Auth", err.Error())
		return
	}

	populateISCSIAuthState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve sensitive values from state since the API may mask them
	prevSecret := state.Secret
	prevPeersecret := state.Peersecret

	var result iscsiAuthResult
	err := r.client.Call(ctx, "iscsi.auth.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Auth", err.Error())
		return
	}

	populateISCSIAuthState(&state, &result)

	// Restore sensitive values if the API returned empty/masked values
	if result.Secret == "" && !prevSecret.IsNull() {
		state.Secret = prevSecret
	}
	if result.Peersecret == "" && !prevPeersecret.IsNull() {
		state.Peersecret = prevPeersecret
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiAuthResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"tag":    plan.Tag.ValueInt64(),
		"user":   plan.User.ValueString(),
		"secret": plan.Secret.ValueString(),
	}

	if !plan.Peeruser.IsNull() {
		params["peeruser"] = plan.Peeruser.ValueString()
	} else {
		params["peeruser"] = ""
	}
	if !plan.Peersecret.IsNull() {
		params["peersecret"] = plan.Peersecret.ValueString()
	} else {
		params["peersecret"] = ""
	}
	if !plan.DiscoveryAuth.IsNull() && !plan.DiscoveryAuth.IsUnknown() {
		params["discovery_auth"] = plan.DiscoveryAuth.ValueString()
	}

	var result iscsiAuthResult
	err := r.client.Call(ctx, "iscsi.auth.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Auth", err.Error())
		return
	}

	populateISCSIAuthState(&plan, &result)

	// Preserve sensitive values if the API returned empty/masked values
	if result.Secret == "" {
		plan.Secret = state.Secret
	}
	if result.Peersecret == "" && !state.Peersecret.IsNull() {
		plan.Peersecret = state.Peersecret
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "iscsi.auth.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Auth", err.Error())
		return
	}
}

func (r *iscsiAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Auth",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateISCSIAuthState(model *iscsiAuthResourceModel, result *iscsiAuthResult) {
	model.ID = types.Int64Value(result.ID)
	model.Tag = types.Int64Value(result.Tag)
	model.User = types.StringValue(result.User)

	if result.Secret != "" {
		model.Secret = types.StringValue(result.Secret)
	}

	if result.Peeruser != "" {
		model.Peeruser = types.StringValue(result.Peeruser)
	} else {
		model.Peeruser = types.StringNull()
	}

	if result.Peersecret != "" {
		model.Peersecret = types.StringValue(result.Peersecret)
	}

	model.DiscoveryAuth = types.StringValue(result.DiscoveryAuth)
}
