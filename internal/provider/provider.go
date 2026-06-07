package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/httpdss/terraform-provider-googleads/internal/googleads"
)

type googleAdsProvider struct{ version string }

type providerModel struct {
	DeveloperToken  types.String `tfsdk:"developer_token"`
	CustomerID      types.String `tfsdk:"customer_id"`
	LoginCustomerID types.String `tfsdk:"login_customer_id"`
	ClientID        types.String `tfsdk:"client_id"`
	ClientSecret    types.String `tfsdk:"client_secret"`
	RefreshToken    types.String `tfsdk:"refresh_token"`
	TokenFile       types.String `tfsdk:"token_file"`
	CredentialsFile types.String `tfsdk:"credentials_file"`
	APIVersion      types.String `tfsdk:"api_version"`
	ValidateOnly    types.Bool   `tfsdk:"validate_only"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &googleAdsProvider{version: version} }
}
func (p *googleAdsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "googleads"
	resp.Version = p.version
}
func (p *googleAdsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}
func (p *googleAdsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewCampaignBudgetResource, NewCampaignResource, NewAdGroupResource, NewAdGroupKeywordResource, NewAdGroupAdResource, NewCampaignCriterionResource}
}
func (p *googleAdsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Terraform provider for managing Google Ads campaign structures through the Google Ads API.", Attributes: map[string]schema.Attribute{
		"developer_token":   schema.StringAttribute{Optional: true, Sensitive: true, Description: "Google Ads developer token. Env: GOOGLEADS_DEVELOPER_TOKEN."},
		"customer_id":       schema.StringAttribute{Optional: true, Description: "Google Ads customer ID, with or without dashes. Env: GOOGLEADS_CUSTOMER_ID."},
		"login_customer_id": schema.StringAttribute{Optional: true, Description: "Optional manager account customer ID. Env: GOOGLEADS_LOGIN_CUSTOMER_ID."},
		"client_id":         schema.StringAttribute{Optional: true, Sensitive: true, Description: "OAuth client ID. Env: GOOGLEADS_CLIENT_ID."},
		"client_secret":     schema.StringAttribute{Optional: true, Sensitive: true, Description: "OAuth client secret. Env: GOOGLEADS_CLIENT_SECRET."},
		"refresh_token":     schema.StringAttribute{Optional: true, Sensitive: true, Description: "OAuth refresh token. Env: GOOGLEADS_REFRESH_TOKEN."},
		"token_file":        schema.StringAttribute{Optional: true, Description: "Path to OAuth token JSON. Env: GOOGLEADS_TOKEN_FILE."},
		"credentials_file":  schema.StringAttribute{Optional: true, Description: "Path to OAuth client credentials JSON. Env: GOOGLEADS_CREDENTIALS_FILE."},
		"api_version":       schema.StringAttribute{Optional: true, Description: "Google Ads API version, e.g. v23. Env: GOOGLEADS_API_VERSION. Defaults to v23."},
		"validate_only":     schema.BoolAttribute{Optional: true, Description: "Send mutate requests with validateOnly=true. Env: GOOGLEADS_VALIDATE_ONLY."},
	}}
}
func (p *googleAdsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg := googleads.Config{
		DeveloperToken: getString(data.DeveloperToken, "GOOGLEADS_DEVELOPER_TOKEN"), CustomerID: getString(data.CustomerID, "GOOGLEADS_CUSTOMER_ID"), LoginCustomerID: getString(data.LoginCustomerID, "GOOGLEADS_LOGIN_CUSTOMER_ID"), ClientID: getString(data.ClientID, "GOOGLEADS_CLIENT_ID"), ClientSecret: getString(data.ClientSecret, "GOOGLEADS_CLIENT_SECRET"), RefreshToken: getString(data.RefreshToken, "GOOGLEADS_REFRESH_TOKEN"), TokenFile: getString(data.TokenFile, "GOOGLEADS_TOKEN_FILE"), CredentialsFile: getString(data.CredentialsFile, "GOOGLEADS_CREDENTIALS_FILE"), APIVersion: getString(data.APIVersion, "GOOGLEADS_API_VERSION"), ValidateOnly: getBool(data.ValidateOnly, "GOOGLEADS_VALIDATE_ONLY"),
	}
	client, err := googleads.NewClient(ctx, cfg)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("developer_token"), "Unable to configure Google Ads client", err.Error())
		return
	}
	tflog.Info(ctx, "configured Google Ads provider", map[string]any{"customer_id": client.CustomerID(), "validate_only": client.ValidateOnly()})
	resp.DataSourceData = client
	resp.ResourceData = client
}
func getString(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return os.Getenv(env)
}
func getBool(v types.Bool, env string) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	return os.Getenv(env) == "true" || os.Getenv(env) == "1"
}

var _ provider.Provider = &googleAdsProvider{}
