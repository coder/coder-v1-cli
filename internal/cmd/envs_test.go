package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/pkg/clog"
)

func init() {
	tmpDir, err := ioutil.TempDir("", "coder-cli-config-dir")
	if err != nil {
		panic(err)
	}
	config.SetRoot(tmpDir)
}

func TestEnvsCommand(t *testing.T) {
	res := execute(t, []string{"envs", "ls"}, nil)
	assert.Error(t, "execute without auth", res.ExitErr)

	err := assertClogErr(t, res.ExitErr)
	assert.True(t, "login hint in error", strings.Contains(err.String(), "did you run \"coder login"))
}

type result struct {
	OutBuffer *bytes.Buffer
	ErrBuffer *bytes.Buffer
	ExitErr   error
}

func execute(t *testing.T, args []string, in io.Reader) result {
	cmd := Make()

	outStream := bytes.NewBuffer(nil)
	errStream := bytes.NewBuffer(nil)

	cmd.SetArgs(args)

	cmd.SetIn(in)
	cmd.SetOut(outStream)
	cmd.SetErr(errStream)

	err := cmd.Execute()

	slogtest.Debug(t, "execute command",
		slog.F("outBuffer", outStream.String()),
		slog.F("errBuffer", errStream.String()),
		slog.F("args", args),
		slog.F("execute_error", err),
	)
	return result{
		OutBuffer: outStream,
		ErrBuffer: errStream,
		ExitErr:   err,
	}
}

func assertClogErr(t *testing.T, err error) clog.CLIError {
	var cliErr clog.CLIError
	if !xerrors.As(err, &cliErr) {
		slogtest.Fatal(t, "expected clog error, none found", slog.Error(err), slog.F("type", fmt.Sprintf("%T", err)))
	}
	slogtest.Debug(t, "clog error", slog.F("message", cliErr.String()))
	return cliErr
}
