package coder

import (
	"context"
	"net/http"
	"time"
)

type APIToken struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Application bool      `json:"application"`
	UserID      string    `json:"user_id"`
	LastUsed    time.Time `json:"last_used"`
}

type CreateAPITokenReq struct {
	Name string `json:"name"`
}

type createAPITokenResp struct {
	Key string `json:"key"`
}

func (c Client) CreateAPIToken(ctx context.Context, userID string, req CreateAPITokenReq) (token string, _ error) {
	var resp createAPITokenResp
	err :=  c.requestBody(ctx, http.MethodPost, "/api/api-keys/"+userID, req, &resp)
	if err != nil {
		return "", err
	}
	return resp.Key, nil
}

func (c Client) APITokens(ctx context.Context, userID string) ([]APIToken, error) {
	var tokens []APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/api-keys/"+userID, nil, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (c Client) APITokenByID(ctx context.Context, userID, tokenID string) (*APIToken, error) {
	var token APIToken
	if err := c.requestBody(ctx, http.MethodGet, "/api/api-keys/"+userID+"/"+tokenID, nil, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (c Client) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/api-keys/"+userID+"/"+tokenID, nil, nil)
}

func (c Client) RegenerateAPIToken(ctx context.Context, userID, tokenID string) (token string, _ error) {
	var resp createAPITokenResp
	if err := c.requestBody(ctx, http.MethodPost, "/api/api-keys/"+userID+"/"+tokenID+"/regen", nil, &resp); err != nil {
		return "", err
	}
	return resp.Key, nil
}
