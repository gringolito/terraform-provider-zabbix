package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ZabbixProvider satisfies various provider interfaces.
var _ provider.Provider = &ZabbixProvider{}
var _ provider.ProviderWithActions = &ZabbixProvider{}

// ZabbixProvider defines the provider implementation.
type ZabbixProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ZabbixProviderModel describes the provider data model.
type ZabbixProviderModel struct {
	ZabbixURL types.String `tfsdk:"zabbix_url"`
	APIToken  types.String `tfsdk:"api_token"`
}

func (p *ZabbixProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zabbix"
	resp.Version = p.version
}

func (p *ZabbixProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Zabbix provider manages Zabbix resources via the JSON-RPC API. " +
			"Credentials can be supplied via provider attributes or environment variables.",
		Attributes: map[string]schema.Attribute{
			"zabbix_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Zabbix frontend (e.g. `https://zabbix.example.com`). " +
					"May also be set via the `ZABBIX_URL` environment variable.",
				Optional:   true,
				Validators: []validator.String{URLValidator{}},
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "Zabbix API token for authentication. " +
					"May also be set via the `ZABBIX_TOKEN` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *ZabbixProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ZabbixProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zabbixURL := data.ZabbixURL.ValueString()
	if data.ZabbixURL.IsNull() || zabbixURL == "" {
		zabbixURL = os.Getenv("ZABBIX_URL")
	}
	if zabbixURL == "" {
		resp.Diagnostics.AddError(
			"Missing Zabbix URL",
			"The provider requires zabbix_url to be set, either as a provider attribute or via the ZABBIX_URL environment variable.",
		)
	}

	apiToken := data.APIToken.ValueString()
	if data.APIToken.IsNull() || apiToken == "" {
		apiToken = os.Getenv("ZABBIX_TOKEN")
	}
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"The provider requires api_token to be set, either as a provider attribute or via the ZABBIX_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(ctx, zabbixURL, apiToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to connect to Zabbix",
			fmt.Sprintf("Failed to initialise the Zabbix client: %s", err),
		)
		return
	}

	switch c.Tier() {
	case client.Unsupported:
		resp.Diagnostics.AddError(
			"Unsupported Zabbix version",
			fmt.Sprintf("Zabbix %s is not supported by this provider. Supported versions: 7.0.x (Targeted), 7.2.x / 7.4.x (Tolerated).", c.APIVersion()),
		)
		return
	case client.Tolerated:
		tflog.Warn(ctx, "Tolerated Zabbix version detected — best-effort support only", map[string]any{
			"version": c.APIVersion(),
		})
		resp.Diagnostics.AddWarning(
			"Tolerated Zabbix version",
			fmt.Sprintf("Zabbix %s is outside the Targeted range (7.0.x). The provider will attempt to work correctly but compatibility is not guaranteed.", c.APIVersion()),
		)
	}

	resp.ResourceData = c
	resp.DataSourceData = c
	resp.ActionData = c
}

func (p *ZabbixProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHostGroupResource,
		NewTemplateGroupResource,
		NewTemplateResource,
		NewTemplateLinkResource,
		NewHostTemplateLinkResource,
		NewUserGroupResource,
		NewMediaTypeEmailResource,
		NewMediaTypeSMSResource,
		NewMediaTypeScriptResource,
		NewMediaTypeWebhookResource,
		NewRoleResource,
		NewHostResource,
		NewHostInterfaceResource,
		NewUserDirectoryLDAPResource,
		NewUserDirectorySAMLResource,
	}
}

func (p *ZabbixProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHostGroupDataSource,
		NewTemplateGroupDataSource,
		NewTemplateDataSource,
		NewUserGroupDataSource,
		NewMediaTypeEmailDataSource,
		NewMediaTypeSMSDataSource,
		NewMediaTypeScriptDataSource,
		NewMediaTypeWebhookDataSource,
		NewRoleDataSource,
		NewHostDataSource,
		NewHostInterfaceDataSource,
		NewUserDirectoryLDAPDataSource,
		NewUserDirectorySAMLDataSource,
	}
}

func (p *ZabbixProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{
		NewTemplateImportAction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ZabbixProvider{
			version: version,
		}
	}
}
