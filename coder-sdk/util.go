package coder

import (
	"encoding/json"
	"strconv"
	"time"
)

// String gives a string pointer.
func String(s string) *string {
	return &s
}

// Duration is a time.Duration wrapper that marshals to millisecond precision.
// While it looses precision, most javascript applications expect durations to be in milliseconds.
type Duration time.Duration

// MarshalJSON marshals the duration to millisecond precision.
func (d Duration) MarshalJSON() ([]byte, error) {
	du := time.Duration(d)
	return json.Marshal(du.Milliseconds())
}

// UnmarshalJSON unmarshals a millisecond-precision integer to
// a time.Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}

	*d = Duration(time.Duration(i) * time.Millisecond)
	return nil
}

func (d Duration) String() string { return time.Duration(d).String() }
