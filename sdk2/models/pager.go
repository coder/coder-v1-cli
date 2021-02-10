// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Pager pager
//
// swagger:model Pager
type Pager struct {

	// cursor
	Cursor *Cursor `json:"cursor,omitempty"`

	// next
	Next string `json:"next,omitempty"`

	// previous
	Previous string `json:"previous,omitempty"`

	// total
	Total int64 `json:"total,omitempty"`
}

// Validate validates this pager
func (m *Pager) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCursor(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Pager) validateCursor(formats strfmt.Registry) error {
	if swag.IsZero(m.Cursor) { // not required
		return nil
	}

	if m.Cursor != nil {
		if err := m.Cursor.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cursor")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this pager based on the context it is used
func (m *Pager) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCursor(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Pager) contextValidateCursor(ctx context.Context, formats strfmt.Registry) error {

	if m.Cursor != nil {
		if err := m.Cursor.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cursor")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Pager) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Pager) UnmarshalBinary(b []byte) error {
	var res Pager
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
