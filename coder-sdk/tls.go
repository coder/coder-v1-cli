package coder

import (
	"context"
	"net/http"
	"net/url"
)

type TLSCertificateResponse struct {
	Success bool `json:"success"`
}

type SelfSignedCertificateRequest struct {
	Hosts []string `json:"hosts"`
}

func (c *DefaultClient) GenerateSelfSignedCertificate(ctx context.Context, req SelfSignedCertificateRequest) (*TLSCertificateResponse, error) {
	var resp TLSCertificateResponse
	err := c.requestBody(ctx, http.MethodPost, "/api/v0/tls/self-sign", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type UploadCertificateRequest struct {
	Certificate []byte `json:"cert"`
	PrivateKey  []byte `json:"key"`
}

func (c *DefaultClient) UploadCustomCertificate(ctx context.Context, req UploadCertificateRequest) (*TLSCertificateResponse, error) {
	var resp TLSCertificateResponse
	err := c.requestBody(ctx, http.MethodPut, "/api/v0/tls/custom", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type CredentialsSet []string

type ProviderInfo struct {
	Code                string           `json:"code"`
	Name                string           `json:"name"`
	RequiredCredentials []CredentialsSet `json:"required_credentials"`
}

type LetsEncryptProviders struct {
	Providers []ProviderInfo `json:"providers"`
}

func (c *DefaultClient) GetLetsEncryptProviders(ctx context.Context) (*LetsEncryptProviders, error) {
	var resp LetsEncryptProviders
	err := c.requestBody(ctx, http.MethodGet, "/api/v0/tls/acme-providers", nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type GenerateLetsEncryptCertRequest struct {
	Email       string            `json:"email"`
	Domains     []string          `json:"domains"`
	DNSProvider string            `json:"dns_provider"`
	Credentials map[string]string `json:"credentials"`
}

func (c *DefaultClient) GenerateLetsEncryptCert(ctx context.Context, req GenerateLetsEncryptCertRequest) (*TLSCertificateResponse, error) {
	var resp TLSCertificateResponse
	err := c.requestBody(ctx, http.MethodPost, "/api/v0/tls/acme", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type DisableTLSResponse struct {
	DeletedCertificate bool `json:"deleted_certificate"`
	CARevocationFailed bool `json:"ca_revocation_failed"`
}

func (c *DefaultClient) DisableTLS(ctx context.Context, forceDelete bool) (*DisableTLSResponse, error) {
	var (
		resp  DisableTLSResponse
		query = url.Values{}
	)

	if forceDelete {
		query.Set("force-delete", "true")
	}

	err := c.requestBody(ctx, http.MethodDelete, "/api/v0/tls", nil, &resp, withQueryParams(query))
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
