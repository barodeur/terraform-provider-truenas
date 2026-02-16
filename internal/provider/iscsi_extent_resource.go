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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*iscsiExtentResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiExtentResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiExtentResource)(nil)
)

type iscsiExtentResource struct {
	client *client.Client
}

type iscsiExtentResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Disk           types.String `tfsdk:"disk"`
	Path           types.String `tfsdk:"path"`
	Serial         types.String `tfsdk:"serial"`
	Filesize       types.Int64  `tfsdk:"filesize"`
	Blocksize      types.Int64  `tfsdk:"blocksize"`
	Pblocksize     types.Bool   `tfsdk:"pblocksize"`
	AvailThreshold types.Int64  `tfsdk:"avail_threshold"`
	Comment        types.String `tfsdk:"comment"`
	InsecureTPC    types.Bool   `tfsdk:"insecure_tpc"`
	Xen            types.Bool   `tfsdk:"xen"`
	RPM            types.String `tfsdk:"rpm"`
	RO             types.Bool   `tfsdk:"ro"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	NAA            types.String `tfsdk:"naa"`
}

type iscsiExtentResult struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Disk           string `json:"disk"`
	Path           string `json:"path"`
	Serial         string `json:"serial"`
	Filesize       int64  `json:"filesize"`
	Blocksize      int64  `json:"blocksize"`
	Pblocksize     bool   `json:"pblocksize"`
	AvailThreshold *int64 `json:"avail_threshold"`
	Comment        string `json:"comment"`
	InsecureTPC    bool   `json:"insecure_tpc"`
	Xen            bool   `json:"xen"`
	RPM            string `json:"rpm"`
	RO             bool   `json:"ro"`
	Enabled        bool   `json:"enabled"`
	NAA            string `json:"naa"`
}

func NewISCSIExtentResource() resource.Resource {
	return &iscsiExtentResource{}
}

func (r *iscsiExtentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_extent"
}

func (r *iscsiExtentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS iSCSI extent (storage unit).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the extent.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the extent.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The extent type: DISK or FILE.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disk": schema.StringAttribute{
				Description: "The zvol path for DISK type extents (e.g. zvol/tank/iscsi/lun0).",
				Optional:    true,
			},
			"path": schema.StringAttribute{
				Description: "The file path for FILE type extents.",
				Optional:    true,
			},
			"serial": schema.StringAttribute{
				Description: "Serial number for the extent. Auto-generated if not specified.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"filesize": schema.Int64Attribute{
				Description: "Size of the file extent in bytes (only for FILE type).",
				Optional:    true,
			},
			"blocksize": schema.Int64Attribute{
				Description: "Logical block size (512, 1024, 2048, or 4096).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pblocksize": schema.BoolAttribute{
				Description: "Use physical block size reporting.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"avail_threshold": schema.Int64Attribute{
				Description: "Pool available space threshold percentage (1-99) for warnings.",
				Optional:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Description of the extent.",
				Optional:    true,
			},
			"insecure_tpc": schema.BoolAttribute{
				Description: "Allow Third Party Copy (TPC) commands.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"xen": schema.BoolAttribute{
				Description: "Enable Xen compatibility mode.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"rpm": schema.StringAttribute{
				Description: "RPM speed reported to initiators (SSD, UNKNOWN, 5400, 7200, 10000, 15000).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ro": schema.BoolAttribute{
				Description: "Whether the extent is read-only.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the extent is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"naa": schema.StringAttribute{
				Description: "NAA identifier assigned by TrueNAS.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *iscsiExtentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiExtentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiExtentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":    plan.Name.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		params["type"] = plan.Type.ValueString()
	}
	if !plan.Disk.IsNull() && !plan.Disk.IsUnknown() {
		params["disk"] = plan.Disk.ValueString()
	}
	if !plan.Path.IsNull() && !plan.Path.IsUnknown() {
		params["path"] = plan.Path.ValueString()
	}
	if !plan.Serial.IsNull() && !plan.Serial.IsUnknown() {
		params["serial"] = plan.Serial.ValueString()
	}
	if !plan.Filesize.IsNull() && !plan.Filesize.IsUnknown() {
		params["filesize"] = plan.Filesize.ValueInt64()
	}
	if !plan.Blocksize.IsNull() && !plan.Blocksize.IsUnknown() {
		params["blocksize"] = plan.Blocksize.ValueInt64()
	}
	if !plan.Pblocksize.IsNull() && !plan.Pblocksize.IsUnknown() {
		params["pblocksize"] = plan.Pblocksize.ValueBool()
	}
	if !plan.AvailThreshold.IsNull() && !plan.AvailThreshold.IsUnknown() {
		params["avail_threshold"] = plan.AvailThreshold.ValueInt64()
	}
	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		params["comment"] = plan.Comment.ValueString()
	}
	if !plan.InsecureTPC.IsNull() && !plan.InsecureTPC.IsUnknown() {
		params["insecure_tpc"] = plan.InsecureTPC.ValueBool()
	}
	if !plan.Xen.IsNull() && !plan.Xen.IsUnknown() {
		params["xen"] = plan.Xen.ValueBool()
	}
	if !plan.RPM.IsNull() && !plan.RPM.IsUnknown() {
		params["rpm"] = plan.RPM.ValueString()
	}
	if !plan.RO.IsNull() && !plan.RO.IsUnknown() {
		params["ro"] = plan.RO.ValueBool()
	}

	var result iscsiExtentResult
	err := r.client.Call(ctx, "iscsi.extent.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Extent", err.Error())
		return
	}

	populateISCSIExtentState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiExtentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiExtentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiExtentResult
	err := r.client.Call(ctx, "iscsi.extent.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Extent", err.Error())
		return
	}

	populateISCSIExtentState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiExtentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiExtentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiExtentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name":    plan.Name.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		params["type"] = plan.Type.ValueString()
	}
	if !plan.Disk.IsNull() {
		params["disk"] = plan.Disk.ValueString()
	} else {
		params["disk"] = ""
	}
	if !plan.Path.IsNull() {
		params["path"] = plan.Path.ValueString()
	} else {
		params["path"] = ""
	}
	if !plan.Serial.IsNull() && !plan.Serial.IsUnknown() {
		params["serial"] = plan.Serial.ValueString()
	}
	if !plan.Filesize.IsNull() {
		params["filesize"] = plan.Filesize.ValueInt64()
	}
	if !plan.Blocksize.IsNull() && !plan.Blocksize.IsUnknown() {
		params["blocksize"] = plan.Blocksize.ValueInt64()
	}
	if !plan.Pblocksize.IsNull() && !plan.Pblocksize.IsUnknown() {
		params["pblocksize"] = plan.Pblocksize.ValueBool()
	}
	if !plan.AvailThreshold.IsNull() {
		params["avail_threshold"] = plan.AvailThreshold.ValueInt64()
	}
	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	} else {
		params["comment"] = ""
	}
	if !plan.InsecureTPC.IsNull() && !plan.InsecureTPC.IsUnknown() {
		params["insecure_tpc"] = plan.InsecureTPC.ValueBool()
	}
	if !plan.Xen.IsNull() && !plan.Xen.IsUnknown() {
		params["xen"] = plan.Xen.ValueBool()
	}
	if !plan.RPM.IsNull() && !plan.RPM.IsUnknown() {
		params["rpm"] = plan.RPM.ValueString()
	}
	if !plan.RO.IsNull() && !plan.RO.IsUnknown() {
		params["ro"] = plan.RO.ValueBool()
	}

	var result iscsiExtentResult
	err := r.client.Call(ctx, "iscsi.extent.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Extent", err.Error())
		return
	}

	populateISCSIExtentState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiExtentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiExtentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pass false, false to never remove underlying storage
	err := r.client.Call(ctx, "iscsi.extent.delete", []any{state.ID.ValueInt64(), false, false}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Extent", err.Error())
		return
	}
}

func (r *iscsiExtentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Extent",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateISCSIExtentState(model *iscsiExtentResourceModel, result *iscsiExtentResult) {
	model.ID = types.Int64Value(result.ID)
	model.Name = types.StringValue(result.Name)
	model.Type = types.StringValue(result.Type)
	model.Enabled = types.BoolValue(result.Enabled)
	model.NAA = types.StringValue(result.NAA)
	model.Serial = types.StringValue(result.Serial)
	model.Blocksize = types.Int64Value(result.Blocksize)
	model.Pblocksize = types.BoolValue(result.Pblocksize)
	model.InsecureTPC = types.BoolValue(result.InsecureTPC)
	model.Xen = types.BoolValue(result.Xen)
	model.RPM = types.StringValue(result.RPM)
	model.RO = types.BoolValue(result.RO)

	if result.Disk != "" {
		model.Disk = types.StringValue(result.Disk)
	} else {
		model.Disk = types.StringNull()
	}

	if result.Path != "" {
		model.Path = types.StringValue(result.Path)
	} else {
		model.Path = types.StringNull()
	}

	if result.AvailThreshold != nil {
		model.AvailThreshold = types.Int64Value(*result.AvailThreshold)
	} else {
		model.AvailThreshold = types.Int64Null()
	}

	if result.Comment != "" {
		model.Comment = types.StringValue(result.Comment)
	} else {
		model.Comment = types.StringNull()
	}

	if result.Filesize != 0 {
		model.Filesize = types.Int64Value(result.Filesize)
	} else {
		model.Filesize = types.Int64Null()
	}
}
