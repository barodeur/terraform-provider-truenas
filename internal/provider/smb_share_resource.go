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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*smbShareResource)(nil)
	_ resource.ResourceWithConfigure   = (*smbShareResource)(nil)
	_ resource.ResourceWithImportState = (*smbShareResource)(nil)
)

type smbShareResource struct {
	client *client.Client
}

type smbShareResourceModel struct {
	ID                        types.Int64  `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Path                      types.String `tfsdk:"path"`
	Comment                   types.String `tfsdk:"comment"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	Purpose                   types.String `tfsdk:"purpose"`
	Readonly                  types.Bool   `tfsdk:"readonly"`
	Browsable                 types.Bool   `tfsdk:"browsable"`
	AccessBasedShareEnum      types.Bool   `tfsdk:"access_based_share_enumeration"`
	Hostsallow                types.List   `tfsdk:"hostsallow"`
	Hostsdeny                 types.List   `tfsdk:"hostsdeny"`
	Locked                    types.Bool   `tfsdk:"locked"`
}

type smbShareResultOptions struct {
	AaplNameMangling bool     `json:"aapl_name_mangling"`
	Hostsallow       []string `json:"hostsallow"`
	Hostsdeny        []string `json:"hostsdeny"`
}

type smbShareResult struct {
	ID                        int64                 `json:"id"`
	Name                      string                `json:"name"`
	Path                      string                `json:"path"`
	Comment                   string                `json:"comment"`
	Enabled                   bool                  `json:"enabled"`
	Purpose                   string                `json:"purpose"`
	Readonly                  bool                  `json:"readonly"`
	Browsable                 bool                  `json:"browsable"`
	AccessBasedShareEnum      bool                  `json:"access_based_share_enumeration"`
	Locked                    bool                  `json:"locked"`
	Options                   smbShareResultOptions `json:"options"`
}

func NewSMBShareResource() resource.Resource {
	return &smbShareResource{}
}

func (r *smbShareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smb_share"
}

func (r *smbShareResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS SMB share.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the SMB share.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The share name.",
				Required:    true,
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
			"purpose": schema.StringAttribute{
				Description: "Share preset/purpose (e.g. DEFAULT_SHARE, TIMEMACHINE_SHARE, MULTIPROTOCOL_SHARE).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"readonly": schema.BoolAttribute{
				Description: "Whether the share is read-only.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"browsable": schema.BoolAttribute{
				Description: "Whether the share is visible in network browse lists.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"access_based_share_enumeration": schema.BoolAttribute{
				Description: "Enable Access Based Enumeration (hide files/folders the user has no access to).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"hostsallow": schema.ListAttribute{
				Description: "List of allowed hosts/networks.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"hostsdeny": schema.ListAttribute{
				Description: "List of denied hosts/networks.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the share is locked (e.g. because the underlying dataset is encrypted and locked).",
				Computed:    true,
			},
		},
	}
}

func (r *smbShareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *smbShareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan smbShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":    plan.Name.ValueString(),
		"path":    plan.Path.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		params["comment"] = plan.Comment.ValueString()
	}
	if !plan.Purpose.IsNull() && !plan.Purpose.IsUnknown() {
		params["purpose"] = plan.Purpose.ValueString()
	}
	if !plan.Readonly.IsNull() && !plan.Readonly.IsUnknown() {
		params["readonly"] = plan.Readonly.ValueBool()
	}
	if !plan.Browsable.IsNull() && !plan.Browsable.IsUnknown() {
		params["browsable"] = plan.Browsable.ValueBool()
	}
	if !plan.AccessBasedShareEnum.IsNull() && !plan.AccessBasedShareEnum.IsUnknown() {
		params["access_based_share_enumeration"] = plan.AccessBasedShareEnum.ValueBool()
	}

	var result smbShareResult
	err := r.client.Call(ctx, "sharing.smb.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating SMB Share", err.Error())
		return
	}

	populateSMBShareState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smbShareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state smbShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result smbShareResult
	err := r.client.Call(ctx, "sharing.smb.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading SMB Share", err.Error())
		return
	}

	populateSMBShareState(ctx, &state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *smbShareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan smbShareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state smbShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":    plan.Name.ValueString(),
		"path":    plan.Path.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	} else {
		params["comment"] = ""
	}
	if !plan.Readonly.IsNull() {
		params["readonly"] = plan.Readonly.ValueBool()
	}
	if !plan.Browsable.IsNull() {
		params["browsable"] = plan.Browsable.ValueBool()
	}
	if !plan.AccessBasedShareEnum.IsNull() {
		params["access_based_share_enumeration"] = plan.AccessBasedShareEnum.ValueBool()
	}

	var result smbShareResult
	err := r.client.Call(ctx, "sharing.smb.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating SMB Share", err.Error())
		return
	}

	populateSMBShareState(ctx, &plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smbShareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state smbShareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "sharing.smb.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting SMB Share", err.Error())
		return
	}
}

func (r *smbShareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing SMB Share",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateSMBShareState(ctx context.Context, model *smbShareResourceModel, result *smbShareResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Name = types.StringValue(result.Name)
	model.Path = types.StringValue(result.Path)

	if result.Comment != "" {
		model.Comment = types.StringValue(result.Comment)
	} else {
		model.Comment = types.StringNull()
	}

	model.Enabled = types.BoolValue(result.Enabled)
	model.Purpose = types.StringValue(result.Purpose)
	model.Readonly = types.BoolValue(result.Readonly)
	model.Browsable = types.BoolValue(result.Browsable)
	model.AccessBasedShareEnum = types.BoolValue(result.AccessBasedShareEnum)

	if len(result.Options.Hostsallow) > 0 {
		elements := make([]attr.Value, len(result.Options.Hostsallow))
		for i, h := range result.Options.Hostsallow {
			elements[i] = types.StringValue(h)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Hostsallow = list
	} else {
		model.Hostsallow = types.ListNull(types.StringType)
	}

	if len(result.Options.Hostsdeny) > 0 {
		elements := make([]attr.Value, len(result.Options.Hostsdeny))
		for i, h := range result.Options.Hostsdeny {
			elements[i] = types.StringValue(h)
		}
		list, d := types.ListValue(types.StringType, elements)
		diags.Append(d...)
		model.Hostsdeny = list
	} else {
		model.Hostsdeny = types.ListNull(types.StringType)
	}

	model.Locked = types.BoolValue(result.Locked)
}
