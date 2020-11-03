package coder

import (
	"context"
	"net/http"
)

type AuthProviderType string

// AuthProviderType enum.
const (
	AuthProviderBuiltIn AuthProviderType = "built-in"
	AuthProviderSAML    AuthProviderType = "saml"
	AuthProviderOIDC    AuthProviderType = "oidc"
)

type ConfigAuth struct {
	ProviderType *AuthProviderType `json:"provider_type"`
	OIDC         *ConfigOIDC       `json:"oidc"`
	SAML         *ConfigSAML       `json:"saml"`
}

type ConfigOIDC struct {
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	Issuer       *string `json:"issuer"`
}

type ConfigSAML struct {
	IdentityProviderMetadataURL *string `json:"idp_metadata_url"`
	SignatureAlgorithm          *string `json:"signature_algorithm"`
	NameIDFormat                *string `json:"name_id_format"`
	PrivateKey                  *string `json:"private_key"`
	PublicKeyCertificate        *string `json:"public_key_certificate"`
}

type ConfigOAuthBitbucketServer struct {
	BaseURL string `json:"base_url" diff:"oauth.bitbucket_server.base_url"`
}

type ConfigOAuthGitHub struct {
	BaseURL      string `json:"base_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type ConfigOAuthGitLab struct {
	BaseURL      string `json:"base_url"`
	ClientID     string `json:"client_id" `
	ClientSecret string `json:"client_secret"`
}

type ConfigOAuth struct {
	BitbucketServer ConfigOAuthBitbucketServer `json:"bitbucket_server"`
	GitHub          ConfigOAuthGitHub          `json:"github"`
	GitLab          ConfigOAuthGitLab          `json:"gitlab"`
}

func (c Client) SiteConfigAuth(ctx context.Context) (*ConfigAuth, error) {
	var conf ConfigAuth
	if err := c.requestBody(ctx, http.MethodGet, "/api/auth/config", nil, &c); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (c Client) PutSiteConfigAuth(ctx context.Context, req ConfigAuth) error {
	return c.requestBody(ctx, http.MethodPut, "/api/auth/config", req, nil)
}

func (c Client) SiteConfigOAuth(ctx context.Context) (*ConfigOAuth, error) {
	var conf ConfigOAuth
	if err := c.requestBody(ctx, http.MethodGet, "/api/oauth/config", nil, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (c Client) PutSiteConfigOAuth(ctx context.Context, req ConfigOAuth) error {
	return c.requestBody(ctx, http.MethodPut, "/api/oauth/config", req, nil)
}

type configSetupMode struct {
	SetupMode bool `json:"setup_mode"`
}

func (c Client) SiteSetupModeEnabled(ctx context.Context) (bool, error) {
	var conf configSetupMode
	if err := c.requestBody(ctx, http.MethodGet, "/api/config/setup-mode", nil, &conf); err != nil {
		return false, nil
	}
	return conf.SetupMode, nil
}

type ExtensionMarketplaceType string

// ExtensionMarketplaceType enum.
const (
	ExtensionMarketplaceInternal ExtensionMarketplaceType = "internal"
	ExtensionMarketplaceCustom   ExtensionMarketplaceType = "custom"
	ExtensionMarketplacePublic   ExtensionMarketplaceType = "public"
)

const MarketplaceExtensionPublicURL = "https://extensions.coder.com/api"

type ConfigExtensionMarketplace struct {
	URL  string                   `json:"url"`
	Type ExtensionMarketplaceType `json:"type"`
}

func (c Client) SiteConfigExtensionMarketplace(ctx context.Context) (*ConfigExtensionMarketplace, error) {
	var conf ConfigExtensionMarketplace
	if err := c.requestBody(ctx, http.MethodGet, "/api/extensions/config", nil, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (c Client) PutSiteConfigExtensionMarketplace(ctx context.Context, req ConfigExtensionMarketplace) error {
	return c.requestBody(ctx, http.MethodGet, "/api/extensions/config", req, nil)
}
