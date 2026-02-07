package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

// truenasDate handles TrueNAS date fields that may be either a plain string
// or an object like {"$date": <epoch_milliseconds>}.
type truenasDate struct {
	Value string
}

func (d *truenasDate) UnmarshalJSON(data []byte) error {
	// Try plain string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		d.Value = s
		return nil
	}

	// Try {"$date": <epoch_ms>} object
	var obj struct {
		Date int64 `json:"$date"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.Date != 0 {
		d.Value = time.UnixMilli(obj.Date).UTC().Format(time.RFC3339)
		return nil
	}

	// Try null
	if string(data) == "null" {
		d.Value = ""
		return nil
	}

	return fmt.Errorf("unexpected date format: %s", string(data))
}

var (
	_ resource.Resource                = (*apiKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*apiKeyResource)(nil)
	_ resource.ResourceWithImportState = (*apiKeyResource)(nil)
)

type apiKeyResource struct {
	client *client.Client
}

type apiKeyResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	Key       types.String `tfsdk:"key"`
	CreatedAt types.String `tfsdk:"created_at"`
	Revoked   types.Bool   `tfsdk:"revoked"`
}

type apiKeyCreateParams struct {
	Name      string `json:"name"`
	Username  string `json:"username,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type apiKeyUpdateParams struct {
	Name      string `json:"name,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type apiKeyResult struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Username  string      `json:"username"`
	Key       string      `json:"key,omitempty"`
	ExpiresAt truenasDate `json:"expires_at"`
	CreatedAt truenasDate `json:"created_at"`
	Revoked   bool        `json:"revoked"`
}

func NewAPIKeyResource() resource.Resource {
	return &apiKeyResource{}
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The unique identifier of the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the API key.",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username associated with the API key. Defaults to the authenticated user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "The expiration date of the API key (ISO 8601 format). If not set, the key does not expire.",
				Optional:    true,
			},
			"key": schema.StringAttribute{
				Description: "The API key value. Only available after creation. Cannot be retrieved after initial creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"revoked": schema.BoolAttribute{
				Description: "Whether the API key has been revoked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := plan.Username.ValueString()
	if plan.Username.IsNull() || plan.Username.IsUnknown() || username == "" {
		// TrueNAS 25.10+ requires username; look up the authenticated user
		var me struct {
			Username string `json:"pw_name"`
		}
		if err := r.client.Call(ctx, "auth.me", nil, &me); err != nil {
			resp.Diagnostics.AddError("Error Looking Up Authenticated User", err.Error())
			return
		}
		username = me.Username
	}

	params := apiKeyCreateParams{
		Name:     plan.Name.ValueString(),
		Username: username,
	}
	if !plan.ExpiresAt.IsNull() {
		params.ExpiresAt = plan.ExpiresAt.ValueString()
	}

	var result apiKeyResult
	err := r.client.Call(ctx, "api_key.create", []any{params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating API Key", err.Error())
		return
	}

	plan.ID = types.Int64Value(result.ID)
	plan.Name = types.StringValue(result.Name)
	plan.Username = types.StringValue(result.Username)
	plan.Key = types.StringValue(result.Key)
	plan.CreatedAt = types.StringValue(result.CreatedAt.Value)
	plan.Revoked = types.BoolValue(result.Revoked)
	if result.ExpiresAt.Value != "" {
		plan.ExpiresAt = types.StringValue(result.ExpiresAt.Value)
	} else {
		plan.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var results []apiKeyResult
	err := r.client.Call(ctx, "api_key.query", []any{
		[]any{[]any{"id", "=", state.ID.ValueInt64()}},
	}, &results)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading API Key", err.Error())
		return
	}

	if len(results) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	result := results[0]
	// Preserve key from prior state since it cannot be re-fetched
	priorKey := state.Key

	state.ID = types.Int64Value(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Username = types.StringValue(result.Username)
	state.Key = priorKey
	state.CreatedAt = types.StringValue(result.CreatedAt.Value)
	state.Revoked = types.BoolValue(result.Revoked)
	if result.ExpiresAt.Value != "" {
		state.ExpiresAt = types.StringValue(result.ExpiresAt.Value)
	} else {
		state.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := apiKeyUpdateParams{
		Name: plan.Name.ValueString(),
	}
	if !plan.ExpiresAt.IsNull() {
		params.ExpiresAt = plan.ExpiresAt.ValueString()
	}

	var result apiKeyResult
	err := r.client.Call(ctx, "api_key.update", []any{state.ID.ValueInt64(), params}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating API Key", err.Error())
		return
	}

	// Preserve key from prior state since update without reset doesn't return it
	plan.ID = types.Int64Value(result.ID)
	plan.Name = types.StringValue(result.Name)
	plan.Username = types.StringValue(result.Username)
	plan.Key = state.Key
	plan.CreatedAt = types.StringValue(result.CreatedAt.Value)
	plan.Revoked = types.BoolValue(result.Revoked)
	if result.ExpiresAt.Value != "" {
		plan.ExpiresAt = types.StringValue(result.ExpiresAt.Value)
	} else {
		plan.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Call(ctx, "api_key.delete", []any{state.ID.ValueInt64()}, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting API Key", err.Error())
		return
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing API Key",
			fmt.Sprintf("Could not parse ID %q as an integer: %s", req.ID, err.Error()),
		)
		return
	}

	state := apiKeyResourceModel{
		ID:  types.Int64Value(id),
		Key: types.StringNull(),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
