// Code generated by go-swagger; DO NOT EDIT.

package api_keys

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new api keys API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for api keys API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	DeleteAPIKey(params *DeleteAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAPIKeyOK, error)

	GenerateApplicationAPIKey(params *GenerateApplicationAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GenerateApplicationAPIKeyCreated, error)

	GetAPIKeys(params *GetAPIKeysParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAPIKeysOK, error)

	ListAPIKeys(params *ListAPIKeysParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAPIKeysOK, error)

	RegenAPIKey(params *RegenAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RegenAPIKeyOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  DeleteAPIKey deletes an API key
*/
func (a *Client) DeleteAPIKey(params *DeleteAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAPIKeyOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteAPIKeyParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "delete-api-key",
		Method:             "DELETE",
		PathPattern:        "/v0/api-keys/{user_id}/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &DeleteAPIKeyReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeleteAPIKeyOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for delete-api-key: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GenerateApplicationAPIKey generates an application API key

  An application API key is a long-lived key intended for
non-browser applications.
*/
func (a *Client) GenerateApplicationAPIKey(params *GenerateApplicationAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GenerateApplicationAPIKeyCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGenerateApplicationAPIKeyParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "generate-application-api-key",
		Method:             "POST",
		PathPattern:        "/v0/api-keys/{user_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GenerateApplicationAPIKeyReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GenerateApplicationAPIKeyCreated)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for generate-application-api-key: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetAPIKeys gets an API key
*/
func (a *Client) GetAPIKeys(params *GetAPIKeysParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAPIKeysOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAPIKeysParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "get-api-keys",
		Method:             "GET",
		PathPattern:        "/v0/api-keys/{user_id}/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetAPIKeysReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetAPIKeysOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for get-api-keys: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  ListAPIKeys lists a user s API keys
*/
func (a *Client) ListAPIKeys(params *ListAPIKeysParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAPIKeysOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAPIKeysParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "list-api-keys",
		Method:             "GET",
		PathPattern:        "/v0/api-keys/{user_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ListAPIKeysReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAPIKeysOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for list-api-keys: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  RegenAPIKey regenerates an API key
*/
func (a *Client) RegenAPIKey(params *RegenAPIKeyParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RegenAPIKeyOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewRegenAPIKeyParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "regen-api-key",
		Method:             "POST",
		PathPattern:        "/v0/api-keys/{user_id}/{id}/regen",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &RegenAPIKeyReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*RegenAPIKeyOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for regen-api-key: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
