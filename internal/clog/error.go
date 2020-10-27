package clog

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/xerrors"
)

// CLIMessage provides a human-readable message for CLI errors and messages.
type CLIMessage struct {
	Level  string
	Color  color.Attribute
	Header string
	Lines  []string
}

// CLIError wraps a CLIMessage and allows consumers to treat it as a normal error.
type CLIError struct {
	CLIMessage
	error
}

// String formats the CLI message for consumption by a human.
func (m CLIMessage) String() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("%s: %s\n",
		color.New(m.Color).Sprint(m.Level),
		color.New(color.Bold).Sprint(m.Header)),
	)
	for _, line := range m.Lines {
		str.WriteString(fmt.Sprintf("  %s %s\n", color.New(m.Color).Sprint("|"), line))
	}
	return str.String()
}

// Log logs the given error to stderr, defaulting to "fatal" if the error is not a CLIError.
// If the error is a CLIError, the plain error chain is ignored and the CLIError
// is logged on its own.
func Log(err error) {
	var cliErr CLIError
	if !xerrors.As(err, &cliErr) {
		cliErr = Fatal(err.Error())
	}
	fmt.Fprintln(os.Stderr, cliErr.String())
}

// LogInfo prints the given info message to stderr.
func LogInfo(header string, lines ...string) {
	fmt.Fprint(os.Stderr, CLIMessage{
		Level:  "info",
		Color:  color.FgBlue,
		Header: header,
		Lines:  lines,
	}.String())
}

// LogSuccess prints the given info message to stderr.
func LogSuccess(header string, lines ...string) {
	fmt.Fprint(os.Stderr, CLIMessage{
		Level:  "success",
		Color:  color.FgGreen,
		Header: header,
		Lines:  lines,
	}.String())
}

// LogWarn prints the given warn message to stderr.
func LogWarn(header string, lines ...string) {
	fmt.Fprint(os.Stderr, CLIMessage{
		Level:  "warning",
		Color:  color.FgYellow,
		Header: header,
		Lines:  lines,
	}.String())
}

// Warn creates an error with the level "warning".
func Warn(header string, lines ...string) CLIError {
	return CLIError{
		CLIMessage: CLIMessage{
			Color:  color.FgYellow,
			Level:  "warning",
			Header: header,
			Lines:  lines,
		},
		error: errors.New(header),
	}
}

// Error creates an error with the level "error".
func Error(header string, lines ...string) CLIError {
	return CLIError{
		CLIMessage: CLIMessage{
			Color:  color.FgRed,
			Level:  "error",
			Header: header,
			Lines:  lines,
		},
		error: errors.New(header),
	}
}

// Fatal creates an error with the level "fatal".
func Fatal(header string, lines ...string) CLIError {
	return CLIError{
		CLIMessage: CLIMessage{
			Color:  color.FgRed,
			Level:  "fatal",
			Header: header,
			Lines:  lines,
		},
		error: errors.New(header),
	}
}

// Bold provides a convenience wrapper around color.New for brevity when logging.
func Bold(a string) string {
	return color.New(color.Bold).Sprint(a)
}

// Tipf formats according to the given format specifier and prepends a bolded "tip: " header.
func Tipf(format string, a ...interface{}) string {
	return fmt.Sprintf("%s %s", Bold("tip:"), fmt.Sprintf(format, a...))
}

// Hintf formats according to the given format specifier and prepends a bolded "hint: " header.
func Hintf(format string, a ...interface{}) string {
	return fmt.Sprintf("%s %s", Bold("hint:"), fmt.Sprintf(format, a...))
}

// Causef formats according to the given format specifier and prepends a bolded "cause: " header.
func Causef(format string, a ...interface{}) string {
	return fmt.Sprintf("%s %s", Bold("cause:"), fmt.Sprintf(format, a...))
}

// BlankLine is an empty string meant to be used in CLIMessage and CLIError construction.
const BlankLine = ""
