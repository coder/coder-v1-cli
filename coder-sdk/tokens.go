package coder

import (
	"context"
	"net/http"
	"time"
)

// APIToken describes a Coder Enterprise APIToken resource for use in API requests.
type APIToken struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Application bool      `json:"application"`
	UserID      string    `json:"user_id"`
	LastUsed    time.Time `json:"last_used"`
}

// CreateAPITokenReq defines the paramemters for creating a new APIToken.
type CreateAPITokenReq struct {
	Name string `json:"name"`
}

type createAPITokenResp struct {
	Key string `json:"key"`
}

// CreateAPIToken creates a new APIToken for making authenticated requests to Coder Enterprise.
func (c *DefaultClient) CreateAPIToken(ctx context.Context, userID string, req CreateAPITokenReq) (token string, _ error) {
	var resp createAPITokenResp
	err := c.requestBody(ctx, http.MethodPost, "/api/v0/api-keys/"+userID, req, &resp)
	if err != nil {
		return "", err
	}
	return resp.Key, nil
}

// APITokens fetches all APITokens owned by the given user.
func (c *DefaultClient) APITokens(ctx context.Context, userID string) ([]APIToken, error) {
	var tokens []APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/api-keys/"+userID, nil, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// APITokenByID fetches the metadata for a given APIToken.
func (c *DefaultClient) APITokenByID(ctx context.Context, userID, tokenID string) (*APIToken, error) {
	var token APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/api-keys/"+userID+"/"+tokenID, nil, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteAPIToken deletes an APIToken.
func (c *DefaultClient) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/api-keys/"+userID+"/"+tokenID, nil, nil)
}

// RegenerateAPIToken regenerates the given APIToken and returns the new value.
func (c *DefaultClient) RegenerateAPIToken(ctx context.Context, userID, tokenID string) (token string, _ error) {
	var resp createAPITokenResp
	if err := c.requestBody(ctx, http.MethodPost, "/api/v0/api-keys/"+userID+"/"+tokenID+"/regen", nil, &resp); err != nil {
		return "", err
	}
	return resp.Key, nil
}
