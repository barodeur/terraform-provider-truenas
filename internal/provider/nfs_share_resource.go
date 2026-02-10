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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*nfsShareResource)(nil)
	_ resource.ResourceWithConfigure   = (*nfsShareResource)(nil)
	_ resource.ResourceWithImportState = (*nfsShareResource)(nil)
)

type nfsShareResource struct {
	client *client.Client
}

type nfsShareResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Path         types.String `tfsdk:"path"`
	Comment      types.String `tfsdk:"comment"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Networks     types.List   `tfsdk:"networks"`
	Hosts        types.List   `tfsdk:"hosts"`
	MaprootUser  types.String `tfsdk:"maproot_user"`
	MaprootGroup types.String `tfsdk:"maproot_group"`
	MapallUser   types.String `tfsdk:"mapall_user"`
	MapallGroup  types.String `tfsdk:"mapall_group"`
	Locked       types.Bool   `tfsdk:"locked"`
}

type nfsShareResult struct {
	ID           int64    `json:"id"`
	Path         string   `json:"path"`
	Comment      string   `json:"comment"`
	Enabled      bool     `json:"enabled"`
	Networks     []string `json:"networks"`
	Hosts        []string `json:"hosts"`
	MaprootUser  *string  `json:"maproot_user"`
	MaprootGroup *string  `json:"maproot_group"`
	MapallUser   *string  `json:"mapall_user"`
	MapallGroup  *string  `json:"mapall_group"`
	Locked       bool     `json:"locked"`
}

func NewNFSShareResource() resource.Resource {
	return &nfsShareResource{}
}

func (r *nfsShareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nfs_share"
}

func (r *nfsShareResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NFS share.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NFS share.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The filesystem path to share (e.g. /mnt/tank/data).",
				Required:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Description of the share.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"networks": schema.ListAttribute{
				Description: "List of allowed networks in CIDR notation (e.g. 192.168.1.0/24).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"hosts": schema.ListAttribute{
				Description: "List of allowed hostnames or IP addresses.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"maproot_user": schema.StringAttribute{
				Description: "Map root requests to this user.",
				Optional:    true,
			},
			"maproot_group": schema.StringAttribute{
				Description: "Map root requests to this group.",
				Optional:    true,
			},
			"mapall_user": schema.StringAttribute{
				Description: "Map all requests to this user.",
				Optional:    true,
			},
			"mapall_group": schema.StringAttribute{
				Description: "Map all requests to this group.",
				Optional:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the share is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *nfsShareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nfsShareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nfsShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"path":    plan.Path.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	}
	if !plan.Networks.IsNull() {
		var networks []string
		resp.Diagnostics.Append(plan.Networks.ElementsAs(ctx, &networks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["networks"] = networks
	}
	if !plan.Hosts.IsNull() {
		var hosts []string
		resp.Diagnostics.Append(plan.Hosts.ElementsAs(ctx, &hosts, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["hosts"] = hosts
	}
	if !plan.MaprootUser.IsNull() {
		params["maproot_user"] = plan.MaprootUser.ValueString()
	}
	if !plan.MaprootGroup.IsNull() {
		params["maproot_group"] = plan.MaprootGroup.ValueString()
	}
	if !plan.MapallUser.IsNull() {
		params["mapall_user"] = plan.MapallUser.ValueString()
	}
	if !plan.MapallGroup.IsNull() {
		params["mapall_group"] = plan.MapallGroup.ValueString()
	}

	var result nfsShareResult
	err := r.client.Call(ctx, "sharing.nfs.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NFS Share", err.Error())
		return
	}

	populateNFSShareState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nfsShareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nfsShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nfsShareResult
	err := r.client.Call(ctx, "sharing.nfs.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NFS Share", err.Error())
		return
	}

	populateNFSShareState(ctx, &state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nfsShareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nfsShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nfsShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"path":    plan.Path.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	} else {
		params["comment"] = ""
	}
	if !plan.Networks.IsNull() {
		var networks []string
		resp.Diagnostics.Append(plan.Networks.ElementsAs(ctx, &networks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["networks"] = networks
	} else {
		params["networks"] = []string{}
	}
	if !plan.Hosts.IsNull() {
		var hosts []string
		resp.Diagnostics.Append(plan.Hosts.ElementsAs(ctx, &hosts, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params["hosts"] = hosts
	} else {
		params["hosts"] = []string{}
	}
	if !plan.MaprootUser.IsNull() {
		params["maproot_user"] = plan.MaprootUser.ValueString()
	} else {
		params["maproot_user"] = ""
	}
	if !plan.MaprootGroup.IsNull() {
		params["maproot_group"] = plan.MaprootGroup.ValueString()
	} else {
		params["maproot_group"] = ""
	}
	if !plan.MapallUser.IsNull() {
		params["mapall_user"] = plan.MapallUser.ValueString()
	} else {
		params["mapall_user"] = ""
	}
	if !plan.MapallGroup.IsNull() {
		params["mapall_group"] = plan.MapallGroup.ValueString()
	} else {
		params["mapall_group"] = ""
	}

	var result nfsShareResult
	err := r.client.Call(ctx, "sharing.nfs.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NFS Share", err.Error())
		return
	}

	populateNFSShareState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nfsShareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nfsShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "sharing.nfs.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting NFS Share", err.Error())
		return
	}
}

func (r *nfsShareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NFS Share",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNFSShareState(ctx context.Context, model *nfsShareResourceModel, result *nfsShareResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Path = types.StringValue(result.Path)

	if result.Comment != "" {
		model.Comment = types.StringValue(result.Comment)
	} else {
		model.Comment = types.StringNull()
	}

	model.Enabled = types.BoolValue(result.Enabled)

	if len(result.Networks) > 0 {
		elements := make([]attr.Value, len(result.Networks))
		for i, n := range result.Networks {
			elements[i] = types.StringValue(n)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Networks = list
	} else {
		model.Networks = types.ListNull(types.StringType)
	}

	if len(result.Hosts) > 0 {
		elements := make([]attr.Value, len(result.Hosts))
		for i, h := range result.Hosts {
			elements[i] = types.StringValue(h)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Hosts = list
	} else {
		model.Hosts = types.ListNull(types.StringType)
	}

	if result.MaprootUser != nil && *result.MaprootUser != "" {
		model.MaprootUser = types.StringValue(*result.MaprootUser)
	} else {
		model.MaprootUser = types.StringNull()
	}
	if result.MaprootGroup != nil && *result.MaprootGroup != "" {
		model.MaprootGroup = types.StringValue(*result.MaprootGroup)
	} else {
		model.MaprootGroup = types.StringNull()
	}
	if result.MapallUser != nil && *result.MapallUser != "" {
		model.MapallUser = types.StringValue(*result.MapallUser)
	} else {
		model.MapallUser = types.StringNull()
	}
	if result.MapallGroup != nil && *result.MapallGroup != "" {
		model.MapallGroup = types.StringValue(*result.MapallGroup)
	} else {
		model.MapallGroup = types.StringNull()
	}

	model.Locked = types.BoolValue(result.Locked)
}
