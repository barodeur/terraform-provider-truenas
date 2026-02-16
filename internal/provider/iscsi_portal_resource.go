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
	_ resource.Resource                = (*iscsiPortalResource)(nil)
	_ resource.ResourceWithConfigure   = (*iscsiPortalResource)(nil)
	_ resource.ResourceWithImportState = (*iscsiPortalResource)(nil)
)

type iscsiPortalResource struct {
	client *client.Client
}

type iscsiPortalResourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Listen  types.List   `tfsdk:"listen"`
	Comment types.String `tfsdk:"comment"`
	Tag     types.Int64  `tfsdk:"tag"`
}

var iscsiPortalListenAttrTypes = map[string]attr.Type{
	"ip": types.StringType,
}

type iscsiPortalListenResult struct {
	IP string `json:"ip"`
}

type iscsiPortalResult struct {
	ID      int64                     `json:"id"`
	Listen  []iscsiPortalListenResult `json:"listen"`
	Comment string                    `json:"comment"`
	Tag     int64                     `json:"tag"`
}

func NewISCSIPortalResource() resource.Resource {
	return &iscsiPortalResource{}
}

func (r *iscsiPortalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_portal"
}

func (r *iscsiPortalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS iSCSI portal.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the iSCSI portal.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"listen": schema.ListNestedAttribute{
				Description: "List of IP addresses the portal listens on.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Description: "IP address to listen on (e.g. 0.0.0.0 for all interfaces).",
							Required:    true,
						},
					},
				},
			},
			"comment": schema.StringAttribute{
				Description: "Description of the portal.",
				Optional:    true,
			},
			"tag": schema.Int64Attribute{
				Description: "The portal group tag, auto-assigned by TrueNAS.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *iscsiPortalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iscsiPortalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iscsiPortalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	listen := iscsiPortalListenFromPlan(ctx, plan.Listen, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	params["listen"] = listen

	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		params["comment"] = plan.Comment.ValueString()
	}

	var result iscsiPortalResult
	err := r.client.Call(ctx, "iscsi.portal.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Portal", err.Error())
		return
	}

	populateISCSIPortalState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiPortalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iscsiPortalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result iscsiPortalResult
	err := r.client.Call(ctx, "iscsi.portal.get_instance", []any{state.ID.ValueInt64()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Portal", err.Error())
		return
	}

	populateISCSIPortalState(&state, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iscsiPortalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iscsiPortalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state iscsiPortalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	listen := iscsiPortalListenFromPlan(ctx, plan.Listen, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	params["listen"] = listen

	if !plan.Comment.IsNull() {
		params["comment"] = plan.Comment.ValueString()
	} else {
		params["comment"] = ""
	}

	var result iscsiPortalResult
	err := r.client.Call(ctx, "iscsi.portal.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Portal", err.Error())
		return
	}

	populateISCSIPortalState(&plan, &result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iscsiPortalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iscsiPortalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "iscsi.portal.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting iSCSI Portal", err.Error())
		return
	}
}

func (r *iscsiPortalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing iSCSI Portal",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func iscsiPortalListenFromPlan(ctx context.Context, listVal types.List, diags *diag.Diagnostics) []map[string]any {
	var listenObjs []types.Object
	diags.Append(listVal.ElementsAs(ctx, &listenObjs, false)...)
	if diags.HasError() {
		return nil
	}

	result := make([]map[string]any, len(listenObjs))
	for i, obj := range listenObjs {
		ip := obj.Attributes()["ip"].(types.String).ValueString()
		result[i] = map[string]any{"ip": ip}
	}
	return result
}

func populateISCSIPortalState(model *iscsiPortalResourceModel, result *iscsiPortalResult, diags *diag.Diagnostics) {
	model.ID = types.Int64Value(result.ID)
	model.Tag = types.Int64Value(result.Tag)

	if result.Comment != "" {
		model.Comment = types.StringValue(result.Comment)
	} else {
		model.Comment = types.StringNull()
	}

	listenElements := make([]attr.Value, len(result.Listen))
	for i, l := range result.Listen {
		obj, d := types.ObjectValue(iscsiPortalListenAttrTypes, map[string]attr.Value{
			"ip": types.StringValue(l.IP),
		})
		diags.Append(d...)
		listenElements[i] = obj
	}

	listenList, d := types.ListValue(types.ObjectType{AttrTypes: iscsiPortalListenAttrTypes}, listenElements)
	diags.Append(d...)
	model.Listen = listenList
}
