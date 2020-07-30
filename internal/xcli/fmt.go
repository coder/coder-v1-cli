package xcli

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// HumanReadableWriter chooses reasonable defaults for a human readable output of tabular data
func HumanReadableWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
}

// TabDelimitedStructValues tab delimits the values of a given struct
func TabDelimitedStructValues(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		s.WriteString(fmt.Sprintf("%s\t", v.Field(i).Interface()))
	}
	return s.String()
}

// TabDelimitedStructHeaders tab delimits the field names of a given struct
func TabDelimitedStructHeaders(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		s.WriteString(fmt.Sprintf("%s\t", v.Type().Field(i).Name))
	}
	return s.String()
}
