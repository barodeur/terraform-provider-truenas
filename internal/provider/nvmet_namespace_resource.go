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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*nvmetNamespaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*nvmetNamespaceResource)(nil)
	_ resource.ResourceWithImportState = (*nvmetNamespaceResource)(nil)
)

type nvmetNamespaceResource struct {
	client *client.Client
}

type nvmetNamespaceResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	NSID        types.Int64  `tfsdk:"nsid"`
	SubsysID    types.Int64  `tfsdk:"subsys_id"`
	DeviceType  types.String `tfsdk:"device_type"`
	DevicePath  types.String `tfsdk:"device_path"`
	Filesize    types.Int64  `tfsdk:"filesize"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	DeviceUUID  types.String `tfsdk:"device_uuid"`
	DeviceNGUID types.String `tfsdk:"device_nguid"`
	Locked      types.Bool   `tfsdk:"locked"`
}

type nvmetNamespaceResultSubsys struct {
	ID int64 `json:"id"`
}

type nvmetNamespaceResult struct {
	ID          int64                     `json:"id"`
	NSID        int64                     `json:"nsid"`
	Subsys      nvmetNamespaceResultSubsys `json:"subsys"`
	DeviceType  string                    `json:"device_type"`
	DevicePath  string                    `json:"device_path"`
	Filesize    *int64                    `json:"filesize"`
	Enabled     bool                      `json:"enabled"`
	DeviceUUID  string                    `json:"device_uuid"`
	DeviceNGUID string                    `json:"device_nguid"`
	Locked      bool                      `json:"locked"`
}

func NewNVMeTNamespaceResource() resource.Resource {
	return &nvmetNamespaceResource{}
}

func (r *nvmetNamespaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_namespace"
}

func (r *nvmetNamespaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS NVMe-oF namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the NVMe-oF namespace.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nsid": schema.Int64Attribute{
				Description: "Namespace ID (1 to 4294967294). Auto-assigned if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"subsys_id": schema.Int64Attribute{
				Description: "The parent subsystem ID.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"device_type": schema.StringAttribute{
				Description: "Device type (ZVOL or FILE).",
				Required:    true,
			},
			"device_path": schema.StringAttribute{
				Description: "Path to the zvol or file.",
				Required:    true,
			},
			"filesize": schema.Int64Attribute{
				Description: "Size in bytes when device_type is FILE.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the namespace is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"device_uuid": schema.StringAttribute{
				Description: "The device UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_nguid": schema.StringAttribute{
				Description: "The device NGUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the namespace is locked.",
				Computed:    true,
			},
		},
	}
}

func (r *nvmetNamespaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nvmetNamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nvmetNamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"subsys_id":   plan.SubsysID.ValueInt64(),
		"device_type": plan.DeviceType.ValueString(),
		"device_path": plan.DevicePath.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
	}

	if !plan.NSID.IsNull() && !plan.NSID.IsUnknown() {
		params["nsid"] = plan.NSID.ValueInt64()
	}
	if !plan.Filesize.IsNull() && !plan.Filesize.IsUnknown() {
		params["filesize"] = plan.Filesize.ValueInt64()
	}

	var result nvmetNamespaceResult
	err := r.client.Call(ctx, "nvmet.namespace.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NVMe-oF Namespace", err.Error())
		return
	}

	populateNVMeTNamespaceState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetNamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nvmetNamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result nvmetNamespaceResult
	err := r.client.Call(ctx, "nvmet.namespace.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading NVMe-oF Namespace", err.Error())
		return
	}

	populateNVMeTNamespaceState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *nvmetNamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nvmetNamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state nvmetNamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"subsys_id":   plan.SubsysID.ValueInt64(),
		"device_type": plan.DeviceType.ValueString(),
		"device_path": plan.DevicePath.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
	}

	if !plan.Filesize.IsNull() {
		params["filesize"] = plan.Filesize.ValueInt64()
	} else {
		params["filesize"] = nil
	}

	var result nvmetNamespaceResult
	err := r.client.Call(ctx, "nvmet.namespace.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NVMe-oF Namespace", err.Error())
		return
	}

	populateNVMeTNamespaceState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nvmetNamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nvmetNamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "nvmet.namespace.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NVMe-oF Namespace", err.Error())
		return
	}
}

func (r *nvmetNamespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing NVMe-oF Namespace",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func populateNVMeTNamespaceState(model *nvmetNamespaceResourceModel, result *nvmetNamespaceResult) {
	model.ID = types.Int64Value(result.ID)
	model.NSID = types.Int64Value(result.NSID)
	model.SubsysID = types.Int64Value(result.Subsys.ID)
	model.DeviceType = types.StringValue(result.DeviceType)
	model.DevicePath = types.StringValue(result.DevicePath)

	if result.Filesize != nil && *result.Filesize != 0 {
		model.Filesize = types.Int64Value(*result.Filesize)
	} else {
		model.Filesize = types.Int64Null()
	}

	model.Enabled = types.BoolValue(result.Enabled)
	model.DeviceUUID = types.StringValue(result.DeviceUUID)
	model.DeviceNGUID = types.StringValue(result.DeviceNGUID)
	model.Locked = types.BoolValue(result.Locked)
}
