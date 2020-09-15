package xjson

import (
	"encoding/json"
	"strconv"
	"time"
)

// MSDuration is a time.MSDuration that marshals to millisecond precision.
// While is looses precision, most javascript applications expect durations to be in milliseconds.
type MSDuration time.Duration

// MarshalJSON marshals the duration to millisecond precision.
func (d MSDuration) MarshalJSON() ([]byte, error) {
	du := time.Duration(d)
	return json.Marshal(du.Milliseconds())
}

// UnmarshalJSON unmarshals a millisecond-precision integer to
// a time.Duration.
func (d *MSDuration) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}

	*d = MSDuration(time.Duration(i) * time.Millisecond)
	return nil
}

func (d MSDuration) String() string { return time.Duration(d).String() }
