package xvalidate

import (
	"bytes"
	"fmt"

	"github.com/spf13/pflag"

	"go.coder.com/flog"
)

// cerrors contains a list of errors.
type cerrors struct {
	cerrors []error
}

func (e cerrors) writeTo(buf *bytes.Buffer) {
	for i, err := range e.cerrors {
		if err == nil {
			continue
		}
		buf.WriteString(err.Error())
		// don't newline after last error
		if i != len(e.cerrors)-1 {
			buf.WriteRune('\n')
		}
	}
}

func (e cerrors) Error() string {
	buf := &bytes.Buffer{}
	e.writeTo(buf)
	return buf.String()
}

// stripNils removes nil errors from the slice.
func stripNils(errs []error) []error {
	// We can't range since errs may be resized
	// during the loop.
	for i := 0; i < len(errs); i++ {
		err := errs[i]
		if err == nil {
			// shift down
			copy(errs[i:], errs[i+1:])
			// pop off last element
			errs = errs[:len(errs)-1]
		}
	}
	return errs
}

// flatten expands all parts of cerrors onto errs.
func flatten(errs []error) []error {
	nerrs := make([]error, 0, len(errs))
	for _, err := range errs {
		errs, ok := err.(cerrors)
		if !ok {
			nerrs = append(nerrs, err)
			continue
		}
		nerrs = append(nerrs, errs.cerrors...)
	}
	return nerrs
}

// combineErrors combines multiple errors into one
func combineErrors(errs ...error) error {
	errs = stripNils(errs)
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		// Don't return if all of the errors of nil.
		for _, err := range errs {
			if err != nil {
				return cerrors{cerrors: flatten(errs)}
			}
		}
		return nil
	}
}

// Validator is a command capable of validating its flags
type Validator interface {
	Validate(fl *pflag.FlagSet) []error
}

// Validate performs validation and exits with a nonzero status code if validation fails.
// The proper errors are printed to stderr.
func Validate(fl *pflag.FlagSet, v Validator) {
	errs := v.Validate(fl)

	err := combineErrors(errs...)
	if err != nil {
		fl.Usage()
		fmt.Println("")
		flog.Fatal("failed to validate this command\n%v", err)
	}
}
