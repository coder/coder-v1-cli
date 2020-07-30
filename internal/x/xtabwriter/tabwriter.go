package xtabwriter

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// NewWriter chooses reasonable defaults for a human readable output of tabular data
func NewWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
}

// StructValues tab delimits the values of a given struct
func StructValues(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		s.WriteString(fmt.Sprintf("%s\t", v.Field(i).Interface()))
	}
	return s.String()
}

// StructFieldNames tab delimits the field names of a given struct
func StructFieldNames(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		s.WriteString(fmt.Sprintf("%s\t", v.Type().Field(i).Name))
	}
	return s.String()
}
