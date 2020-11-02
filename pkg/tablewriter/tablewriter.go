package tablewriter

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

const structFieldTagKey = "table"

// StructValues tab delimits the values of a given struct.
//
// Tag a field `table:"-"` to hide it from output.
func StructValues(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		if shouldHideField(v.Type().Field(i)) {
			continue
		}
		fmt.Fprintf(s, "%v\t", v.Field(i).Interface())
	}
	return s.String()
}

// StructFieldNames tab delimits the field names of a given struct.
//
// Tag a field `table:"-"` to hide it from output.
func StructFieldNames(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if shouldHideField(field) {
			continue
		}
		fmt.Fprintf(s, "%s\t", fieldName(field))
	}
	return s.String()
}

// WriteTable writes the given list elements to stdout in a human readable
// tabular format. Headers abide by the `tab` struct tag.
//
// `table:"-"` omits the field and no tag defaults to the Go identifier.
func WriteTable(length int, each func(i int) interface{}) error {
	if length < 1 {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	defer func() { _ = w.Flush() }() // Best effort.
	for ix := 0; ix < length; ix++ {
		item := each(ix)
		if ix == 0 {
			if _, err := fmt.Fprintln(w, StructFieldNames(item)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, StructValues(item)); err != nil {
			return err
		}
	}
	return nil
}

func fieldName(f reflect.StructField) string {
	custom, ok := f.Tag.Lookup(structFieldTagKey)
	if ok {
		return custom
	}
	return f.Name
}

func shouldHideField(f reflect.StructField) bool {
	return f.Tag.Get(structFieldTagKey) == "-"
}
