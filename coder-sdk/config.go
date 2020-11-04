package coder

import (
	"context"
	"net/http"
)

// AuthProviderType is an enum of each valid auth provider.
type AuthProviderType string

// AuthProviderType enum.
const (
	AuthProviderBuiltIn AuthProviderType = "built-in"
	AuthProviderSAML    AuthProviderType = "saml"
	AuthProviderOIDC    AuthProviderType = "oidc"
)

// ConfigAuth describes the authentication configuration for a Coder Enterprise deployment.
type ConfigAuth struct {
	ProviderType *AuthProviderType `json:"provider_type"`
	OIDC         *ConfigOIDC       `json:"oidc"`
	SAML         *ConfigSAML       `json:"saml"`
}

// ConfigOIDC describes the OIDC configuration for single-signon support in Coder Enterprise.
type ConfigOIDC struct {
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	Issuer       *string `json:"issuer"`
}

// ConfigSAML describes the SAML configuration values.
type ConfigSAML struct {
	IdentityProviderMetadataURL *string `json:"idp_metadata_url"`
	SignatureAlgorithm          *string `json:"signature_algorithm"`
	NameIDFormat                *string `json:"name_id_format"`
	PrivateKey                  *string `json:"private_key"`
	PublicKeyCertificate        *string `json:"public_key_certificate"`
}

// ConfigOAuthBitbucketServer describes the Bitbucket integration configuration for a Coder Enterprise deployment.
type ConfigOAuthBitbucketServer struct {
	BaseURL string `json:"base_url" diff:"oauth.bitbucket_server.base_url"`
}

// ConfigOAuthGitHub describes the Github integration configuration for a Coder Enterprise deployment.
type ConfigOAuthGitHub struct {
	BaseURL      string `json:"base_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ConfigOAuthGitLab describes the GitLab integration configuration for a Coder Enterprise deployment.
type ConfigOAuthGitLab struct {
	BaseURL      string `json:"base_url"`
	ClientID     string `json:"client_id" `
	ClientSecret string `json:"client_secret"`
}

// ConfigOAuth describes the aggregate git integration configuration for a Coder Enterprise deployment.
type ConfigOAuth struct {
	BitbucketServer ConfigOAuthBitbucketServer `json:"bitbucket_server"`
	GitHub          ConfigOAuthGitHub          `json:"github"`
	GitLab          ConfigOAuthGitLab          `json:"gitlab"`
}

// SiteConfigAuth fetches the sitewide authentication configuration.
func (c Client) SiteConfigAuth(ctx context.Context) (*ConfigAuth, error) {
	var conf ConfigAuth
	if err := c.requestBody(ctx, http.MethodGet, "/api/auth/config", nil, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

// PutSiteConfigAuth sets the sitewide authentication configuration.
func (c Client) PutSiteConfigAuth(ctx context.Context, req ConfigAuth) error {
	return c.requestBody(ctx, http.MethodPut, "/api/auth/config", req, nil)
}

// SiteConfigOAuth fetches the sitewide git provider OAuth configuration.
func (c Client) SiteConfigOAuth(ctx context.Context) (*ConfigOAuth, error) {
	var conf ConfigOAuth
	if err := c.requestBody(ctx, http.MethodGet, "/api/oauth/config", nil, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

// PutSiteConfigOAuth sets the sitewide git provider OAuth configuration.
func (c Client) PutSiteConfigOAuth(ctx context.Context, req ConfigOAuth) error {
	return c.requestBody(ctx, http.MethodPut, "/api/oauth/config", req, nil)
}

type configSetupMode struct {
	SetupMode bool `json:"setup_mode"`
}

// SiteSetupModeEnabled fetches the current setup_mode state of a Coder Enterprise deployment.
func (c Client) SiteSetupModeEnabled(ctx context.Context) (bool, error) {
	var conf configSetupMode
	if err := c.requestBody(ctx, http.MethodGet, "/api/config/setup-mode", nil, &conf); err != nil {
		return false, nil
	}
	return conf.SetupMode, nil
}

// ExtensionMarketplaceType is an enum of the valid extension marketplace configurations.
type ExtensionMarketplaceType string

// ExtensionMarketplaceType enum.
const (
	ExtensionMarketplaceInternal ExtensionMarketplaceType = "internal"
	ExtensionMarketplaceCustom   ExtensionMarketplaceType = "custom"
	ExtensionMarketplacePublic   ExtensionMarketplaceType = "public"
)

// MarketplaceExtensionPublicURL is the URL of the coder.com public marketplace that serves open source Code OSS extensions.
const MarketplaceExtensionPublicURL = "https://extensions.coder.com/api"

// ConfigExtensionMarketplace describes the sitewide extension marketplace configuration.
type ConfigExtensionMarketplace struct {
	URL  string                   `json:"url"`
	Type ExtensionMarketplaceType `json:"type"`
}

// SiteConfigExtensionMarketplace fetches the extension marketplace configuration.
func (c Client) SiteConfigExtensionMarketplace(ctx context.Context) (*ConfigExtensionMarketplace, error) {
	var conf ConfigExtensionMarketplace
	if err := c.requestBody(ctx, http.MethodGet, "/api/extensions/config", nil, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

// PutSiteConfigExtensionMarketplace sets the extension marketplace configuration.
func (c Client) PutSiteConfigExtensionMarketplace(ctx context.Context, req ConfigExtensionMarketplace) error {
	return c.requestBody(ctx, http.MethodPut, "/api/extensions/config", req, nil)
}
