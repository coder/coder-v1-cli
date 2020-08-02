package xtabwriter

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

const structFieldTagKey = "tab"

// NewWriter chooses reasonable defaults for a human readable output of tabular data.
func NewWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
}

// StructValues tab delimits the values of a given struct.
//
// Tag a field `tab:"-"` to hide it from output.
func StructValues(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		if shouldHideField(v.Type().Field(i)) {
			continue
		}
		s.WriteString(fmt.Sprintf("%s\t", v.Field(i).Interface()))
	}
	return s.String()
}

// StructFieldNames tab delimits the field names of a given struct.
//
// Tag a field `tab:"-"` to hide it from output.
func StructFieldNames(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if shouldHideField(field) {
			continue
		}
		s.WriteString(fmt.Sprintf("%s\t", field.Name))
	}
	return s.String()
}

func shouldHideField(f reflect.StructField) bool {
	return f.Tag.Get(structFieldTagKey) == "-"
}
