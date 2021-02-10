// Code generated by go-swagger; DO NOT EDIT.

package config

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new config API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for config API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	GetConfigEnvironments(params *GetConfigEnvironmentsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetConfigEnvironmentsOK, error)

	SetConfigEnvironments(params *SetConfigEnvironmentsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*SetConfigEnvironmentsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  GetConfigEnvironments gets environment related config
*/
func (a *Client) GetConfigEnvironments(params *GetConfigEnvironmentsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetConfigEnvironmentsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetConfigEnvironmentsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "get-config-environments",
		Method:             "GET",
		PathPattern:        "/v0/environments/config",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetConfigEnvironmentsReader{formats: a.formats},
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
	success, ok := result.(*GetConfigEnvironmentsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for get-config-environments: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  SetConfigEnvironments sets environment related config
*/
func (a *Client) SetConfigEnvironments(params *SetConfigEnvironmentsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*SetConfigEnvironmentsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewSetConfigEnvironmentsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "set-config-environments",
		Method:             "PUT",
		PathPattern:        "/v0/environments/config",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &SetConfigEnvironmentsReader{formats: a.formats},
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
	success, ok := result.(*SetConfigEnvironmentsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for set-config-environments: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
