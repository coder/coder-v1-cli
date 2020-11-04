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
func (c Client) CreateAPIToken(ctx context.Context, userID string, req CreateAPITokenReq) (token string, _ error) {
	var resp createAPITokenResp
	err := c.requestBody(ctx, http.MethodPost, "/api/api-keys/"+userID, req, &resp)
	if err != nil {
		return "", err
	}
	return resp.Key, nil
}

// APITokens fetches all APITokens owned by the given user.
func (c Client) APITokens(ctx context.Context, userID string) ([]APIToken, error) {
	var tokens []APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/api-keys/"+userID, nil, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// APITokenByID fetches the metadata for a given APIToken.
func (c Client) APITokenByID(ctx context.Context, userID, tokenID string) (*APIToken, error) {
	var token APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/api-keys/"+userID+"/"+tokenID, nil, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteAPIToken deletes an APIToken.
func (c Client) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/api-keys/"+userID+"/"+tokenID, nil, nil)
}

// RegenerateAPIToken regenerates the given APIToken and returns the new value.
func (c Client) RegenerateAPIToken(ctx context.Context, userID, tokenID string) (token string, _ error) {
	var resp createAPITokenResp
	if err := c.requestBody(ctx, http.MethodPost, "/api/api-keys/"+userID+"/"+tokenID+"/regen", nil, &resp); err != nil {
		return "", err
	}
	return resp.Key, nil
}
