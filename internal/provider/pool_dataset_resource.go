package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = (*poolDatasetResource)(nil)
	_ resource.ResourceWithConfigure   = (*poolDatasetResource)(nil)
	_ resource.ResourceWithImportState = (*poolDatasetResource)(nil)
)

type poolDatasetResource struct {
	client *client.Client
}

type poolDatasetResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Pool            types.String `tfsdk:"pool"`
	Comments        types.String `tfsdk:"comments"`
	Sync            types.String `tfsdk:"sync"`
	Compression     types.String `tfsdk:"compression"`
	Atime           types.String `tfsdk:"atime"`
	Exec            types.String `tfsdk:"exec"`
	Readonly        types.String `tfsdk:"readonly"`
	Deduplication   types.String `tfsdk:"deduplication"`
	Checksum        types.String `tfsdk:"checksum"`
	Copies          types.Int64  `tfsdk:"copies"`
	Snapdir         types.String `tfsdk:"snapdir"`
	Quota           types.Int64  `tfsdk:"quota"`
	Refquota        types.Int64  `tfsdk:"refquota"`
	Reservation     types.Int64  `tfsdk:"reservation"`
	Refreservation  types.Int64  `tfsdk:"refreservation"`
	Recordsize      types.String `tfsdk:"recordsize"`
	Aclmode         types.String `tfsdk:"aclmode"`
	Acltype         types.String `tfsdk:"acltype"`
	Casesensitivity types.String `tfsdk:"casesensitivity"`
	CreateAncestors types.Bool   `tfsdk:"create_ancestors"`
	Mountpoint      types.String `tfsdk:"mountpoint"`
	Encrypted       types.Bool   `tfsdk:"encrypted"`
}

type zfsProperty struct {
	Parsed json.RawMessage `json:"parsed"`
	Value  string          `json:"value"`
	Source string          `json:"source"`
}

func (p *zfsProperty) isLocal() bool {
	return p.Source == "LOCAL"
}

func (p *zfsProperty) stringValue() string {
	return p.Value
}

func (p *zfsProperty) int64Value() (int64, bool) {
	var n int64
	if err := json.Unmarshal(p.Parsed, &n); err == nil {
		return n, true
	}
	// Try parsing as float (API sometimes returns floats for int fields)
	var f float64
	if err := json.Unmarshal(p.Parsed, &f); err == nil {
		return int64(f), true
	}
	return 0, false
}

type poolDatasetUserProperties struct {
	Comments *zfsProperty `json:"comments,omitempty"`
}

type poolDatasetResult struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	Pool            string                    `json:"pool"`
	Type            string                    `json:"type"`
	Mountpoint      *string                   `json:"mountpoint"`
	Encrypted       bool                      `json:"encrypted"`
	UserProperties  poolDatasetUserProperties `json:"user_properties"`
	Sync            zfsProperty               `json:"sync"`
	Compression     zfsProperty               `json:"compression"`
	Atime           zfsProperty               `json:"atime"`
	Exec            zfsProperty               `json:"exec"`
	Readonly        zfsProperty               `json:"readonly"`
	Deduplication   zfsProperty               `json:"deduplication"`
	Checksum        zfsProperty               `json:"checksum"`
	Copies          zfsProperty               `json:"copies"`
	Snapdir         zfsProperty               `json:"snapdir"`
	Quota           zfsProperty               `json:"quota"`
	Refquota        zfsProperty               `json:"refquota"`
	Reservation     zfsProperty               `json:"reservation"`
	Refreservation  zfsProperty               `json:"refreservation"`
	Recordsize      zfsProperty               `json:"recordsize"`
	Aclmode         zfsProperty               `json:"aclmode"`
	Acltype         zfsProperty               `json:"acltype"`
	Casesensitivity zfsProperty               `json:"casesensitivity"`
}

func NewPoolDatasetResource() resource.Resource {
	return &poolDatasetResource{}
}

func (r *poolDatasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool_dataset"
}

func (r *poolDatasetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS ZFS dataset (filesystem type).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the dataset (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Full dataset path including pool, e.g. \"tank/data\". Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pool": schema.StringAttribute{
				Description: "The pool name, extracted from the dataset path.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comments": schema.StringAttribute{
				Description: "User-provided comments for the dataset. Null means inherited from parent.",
				Optional:    true,
			},
			"sync": schema.StringAttribute{
				Description: "Sync mode: STANDARD, ALWAYS, or DISABLED. Null means inherited from parent.",
				Optional:    true,
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm: OFF, LZ4, GZIP, ZSTD, etc. Null means inherited from parent.",
				Optional:    true,
			},
			"atime": schema.StringAttribute{
				Description: "Access time updates: ON or OFF. Null means inherited from parent.",
				Optional:    true,
			},
			"exec": schema.StringAttribute{
				Description: "Allow execution of binaries: ON or OFF. Null means inherited from parent.",
				Optional:    true,
			},
			"readonly": schema.StringAttribute{
				Description: "Read-only mode: ON or OFF. Null means inherited from parent.",
				Optional:    true,
			},
			"deduplication": schema.StringAttribute{
				Description: "Deduplication: ON, VERIFY, or OFF. Null means inherited from parent.",
				Optional:    true,
			},
			"checksum": schema.StringAttribute{
				Description: "Checksum algorithm: ON, FLETCHER2, FLETCHER4, SHA256, SHA512, SKEIN, BLAKE3. Null means inherited from parent.",
				Optional:    true,
			},
			"copies": schema.Int64Attribute{
				Description: "Number of data copies: 1, 2, or 3. Null means inherited from parent.",
				Optional:    true,
			},
			"snapdir": schema.StringAttribute{
				Description: "Snapshot directory visibility: VISIBLE or HIDDEN. Null means inherited from parent.",
				Optional:    true,
			},
			"quota": schema.Int64Attribute{
				Description: "Quota in bytes (minimum 1 GiB, or 0 to disable). Null means inherited from parent.",
				Optional:    true,
			},
			"refquota": schema.Int64Attribute{
				Description: "Reference quota in bytes (minimum 1 GiB, or 0 to disable). Null means inherited from parent.",
				Optional:    true,
			},
			"reservation": schema.Int64Attribute{
				Description: "Reservation in bytes. Null means inherited from parent.",
				Optional:    true,
			},
			"refreservation": schema.Int64Attribute{
				Description: "Reference reservation in bytes. Null means inherited from parent.",
				Optional:    true,
			},
			"recordsize": schema.StringAttribute{
				Description: "Record size, e.g. \"128K\", \"1M\". Null means inherited from parent.",
				Optional:    true,
			},
			"aclmode": schema.StringAttribute{
				Description: "ACL mode: PASSTHROUGH, RESTRICTED, or DISCARD. Null means inherited from parent.",
				Optional:    true,
			},
			"acltype": schema.StringAttribute{
				Description: "ACL type: OFF, NFSV4, or POSIX. Cannot be changed after creation. Null means inherited from parent.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"casesensitivity": schema.StringAttribute{
				Description: "Case sensitivity: SENSITIVE or INSENSITIVE. Cannot be changed after creation. Null means inherited from parent.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"create_ancestors": schema.BoolAttribute{
				Description: "Create ancestor datasets if they don't exist. Only used during creation, not stored in state.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"mountpoint": schema.StringAttribute{
				Description: "The mount point of the dataset.",
				Computed:    true,
			},
			"encrypted": schema.BoolAttribute{
				Description: "Whether the dataset is encrypted.",
				Computed:    true,
			},
		},
	}
}

func (r *poolDatasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *poolDatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poolDatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"name": plan.Name.ValueString(),
		"type": "FILESYSTEM",
	}

	setStringParam(params, "comments", plan.Comments)
	setStringParam(params, "sync", plan.Sync)
	setStringParam(params, "compression", plan.Compression)
	setStringParam(params, "atime", plan.Atime)
	setStringParam(params, "exec", plan.Exec)
	setStringParam(params, "readonly", plan.Readonly)
	setStringParam(params, "deduplication", plan.Deduplication)
	setStringParam(params, "checksum", plan.Checksum)
	setInt64Param(params, "copies", plan.Copies)
	setStringParam(params, "snapdir", plan.Snapdir)
	setInt64Param(params, "quota", plan.Quota)
	setInt64Param(params, "refquota", plan.Refquota)
	setInt64Param(params, "reservation", plan.Reservation)
	setInt64Param(params, "refreservation", plan.Refreservation)
	setStringParam(params, "recordsize", plan.Recordsize)
	setStringParam(params, "aclmode", plan.Aclmode)
	setStringParam(params, "acltype", plan.Acltype)
	setStringParam(params, "casesensitivity", plan.Casesensitivity)

	if !plan.CreateAncestors.IsNull() && plan.CreateAncestors.ValueBool() {
		params["create_ancestors"] = true
	}

	var result poolDatasetResult
	err := r.client.Call(ctx, "pool.dataset.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Pool Dataset", err.Error())
		return
	}

	populateDatasetState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolDatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state poolDatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result poolDatasetResult
	err := r.client.Call(ctx, "pool.dataset.get_instance", []any{state.ID.ValueString()}, &result)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Pool Dataset", err.Error())
		return
	}

	populateDatasetState(&state, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *poolDatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan poolDatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state poolDatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{}

	// comments is a ZFS user property — handle explicit set and explicit unset (inherit)
	if !plan.Comments.IsNull() {
		params["comments"] = plan.Comments.ValueString()
	} else if !state.Comments.IsNull() {
		// Transition from a previously set comment to null: clear/unset so it can inherit
		params["comments"] = nil
	}
	setStringParamOrInherit(params, "sync", plan.Sync)
	setStringParamOrInherit(params, "compression", plan.Compression)
	setStringParamOrInherit(params, "atime", plan.Atime)
	setStringParamOrInherit(params, "exec", plan.Exec)
	setStringParamOrInherit(params, "readonly", plan.Readonly)
	setStringParamOrInherit(params, "deduplication", plan.Deduplication)
	setStringParamOrInherit(params, "checksum", plan.Checksum)
	// copies accepts int or "INHERIT"
	if plan.Copies.IsNull() {
		params["copies"] = "INHERIT"
	} else {
		params["copies"] = plan.Copies.ValueInt64()
	}
	setStringParamOrInherit(params, "snapdir", plan.Snapdir)
	// quota, refquota, reservation, refreservation: omit when null to leave unchanged.
	// The API accepts nil for quota/refquota but sets them to 0 (LOCAL), not inherited.
	setInt64Param(params, "quota", plan.Quota)
	setInt64Param(params, "refquota", plan.Refquota)
	setInt64Param(params, "reservation", plan.Reservation)
	setInt64Param(params, "refreservation", plan.Refreservation)
	setStringParamOrInherit(params, "recordsize", plan.Recordsize)
	setStringParamOrInherit(params, "aclmode", plan.Aclmode)

	var result poolDatasetResult
	err := r.client.Call(ctx, "pool.dataset.update", []any{state.ID.ValueString(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Pool Dataset", err.Error())
		return
	}

	populateDatasetState(&plan, &result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolDatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state poolDatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteOpts := map[string]any{
		"recursive": false,
		"force":     false,
	}

	err := r.client.Call(ctx, "pool.dataset.delete", []any{state.ID.ValueString(), deleteOpts}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Pool Dataset", err.Error())
		return
	}
}

func (r *poolDatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// setStringParam sets a string field in params if the Terraform value is non-null,
// or sets INHERIT if null. Works for both create and update.
func setStringParam(params map[string]any, key string, val types.String) {
	if val.IsNull() {
		params[key] = "INHERIT"
	} else {
		params[key] = val.ValueString()
	}
}

// setInt64Param sets an int field in params only if the Terraform value is non-null.
// Null int fields are omitted entirely — the API does not accept "INHERIT" for integers.
func setInt64Param(params map[string]any, key string, val types.Int64) {
	if !val.IsNull() {
		params[key] = val.ValueInt64()
	}
}

// setStringParamOrInherit is an alias for setStringParam (same behavior for update).
func setStringParamOrInherit(params map[string]any, key string, val types.String) {
	setStringParam(params, key, val)
}

// populateDatasetState updates the Terraform resource model from a TrueNAS API result.
// For ZFS properties, only LOCAL-sourced values are stored; inherited/default values become null.
func populateDatasetState(model *poolDatasetResourceModel, result *poolDatasetResult) {
	model.ID = types.StringValue(result.ID)
	model.Name = types.StringValue(result.Name)
	model.Pool = types.StringValue(result.Pool)

	if result.Mountpoint != nil {
		model.Mountpoint = types.StringValue(*result.Mountpoint)
	} else {
		model.Mountpoint = types.StringNull()
	}
	model.Encrypted = types.BoolValue(result.Encrypted)

	if result.UserProperties.Comments != nil {
		model.Comments = readStringProperty(result.UserProperties.Comments)
	} else {
		model.Comments = types.StringNull()
	}
	model.Sync = readStringProperty(&result.Sync)
	model.Compression = readStringProperty(&result.Compression)
	model.Atime = readStringProperty(&result.Atime)
	model.Exec = readStringProperty(&result.Exec)
	model.Readonly = readStringProperty(&result.Readonly)
	model.Deduplication = readStringProperty(&result.Deduplication)
	model.Checksum = readStringProperty(&result.Checksum)
	model.Copies = readInt64Property(&result.Copies)
	model.Snapdir = readStringProperty(&result.Snapdir)
	model.Quota = readInt64Property(&result.Quota)
	model.Refquota = readInt64Property(&result.Refquota)
	model.Reservation = readInt64Property(&result.Reservation)
	model.Refreservation = readInt64Property(&result.Refreservation)
	model.Recordsize = readStringProperty(&result.Recordsize)
	model.Aclmode = readStringProperty(&result.Aclmode)
	model.Acltype = readStringProperty(&result.Acltype)
	model.Casesensitivity = readStringProperty(&result.Casesensitivity)
}

func readStringProperty(prop *zfsProperty) types.String {
	if prop.isLocal() {
		return types.StringValue(prop.stringValue())
	}
	return types.StringNull()
}

func readInt64Property(prop *zfsProperty) types.Int64 {
	if prop.isLocal() {
		if n, ok := prop.int64Value(); ok {
			return types.Int64Value(n)
		}
	}
	return types.Int64Null()
}
