package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
)

var _ provider.Provider = (*truenasProvider)(nil)

type truenasProvider struct {
	cachedClient *client.Client
	cachedHost   string
	cachedAPIKey string
	cachedInsec  bool
}

type truenasProviderModel struct {
	Host     types.String `tfsdk:"host"`
	APIKey   types.String `tfsdk:"api_key"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &truenasProvider{}
	}
}

func (p *truenasProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "truenas"
}

func (p *truenasProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with TrueNAS Scale via its WebSocket API.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The WebSocket URL of the TrueNAS server (e.g. wss://truenas.local). If no scheme is provided, wss:// is assumed. Can also be set with the TRUENAS_HOST environment variable.",
				Required:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with TrueNAS. Can also be set with the TRUENAS_API_KEY environment variable.",
				Required:    true,
				Sensitive:   true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Defaults to false.",
				Optional:    true,
			},
		},
	}
}

func (p *truenasProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config truenasProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("TRUENAS_HOST")
	apiKey := os.Getenv("TRUENAS_API_KEY")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError(
			"Missing Host Configuration",
			"The provider cannot create the TrueNAS client because the host is not configured. "+
				"Set it in the provider configuration block or the TRUENAS_HOST environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key Configuration",
			"The provider cannot create the TrueNAS client because the API key is not configured. "+
				"Set it in the provider configuration block or the TRUENAS_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Default to wss:// if no scheme is provided
	if !strings.HasPrefix(host, "ws://") && !strings.HasPrefix(host, "wss://") {
		host = "wss://" + host
	}

	insecure := false
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	}

	// Reuse existing client if config hasn't changed
	if p.cachedClient != nil &&
		p.cachedHost == host &&
		p.cachedAPIKey == apiKey &&
		p.cachedInsec == insecure {
		resp.DataSourceData = p.cachedClient
		resp.ResourceData = p.cachedClient
		return
	}

	c, err := client.NewClient(ctx, host, apiKey, insecure)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create TrueNAS Client",
			"An unexpected error occurred when creating the TrueNAS client: "+err.Error(),
		)
		return
	}

	p.cachedClient = c
	p.cachedHost = host
	p.cachedAPIKey = apiKey
	p.cachedInsec = insecure

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *truenasProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAPIKeyResource,
		NewCronjobResource,
		NewPoolDatasetResource,
		NewGroupResource,
		NewUserResource,
		NewSMBShareResource,
		NewNFSShareResource,
	}
}

func (p *truenasProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAPIKeyDataSource,
		NewCronjobDataSource,
		NewPoolDataSource,
	}
}
