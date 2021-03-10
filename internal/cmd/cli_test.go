package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/pkg/clog"
)

func init() {
	tmpDir, err := ioutil.TempDir("", "coder-cli-config-dir")
	if err != nil {
		panic(err)
	}
	config.SetRoot(tmpDir)

	// TODO: might need to make this a command scoped option to make assertions against its output
	clog.SetOutput(ioutil.Discard)

	email := os.Getenv("CODER_EMAIL")
	password := os.Getenv("CODER_PASSWORD")
	rawURL := os.Getenv("CODER_URL")
	if email == "" || password == "" || rawURL == "" {
		panic("CODER_EMAIL, CODER_PASSWORD, and CODER_URL are required environment variables")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("invalid CODER_URL: " + err.Error())
	}
	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL:  u,
		Email:    email,
		Password: password,
	})
	if err != nil {
		panic("new client: " + err.Error())
	}
	if err := config.URL.Write(rawURL); err != nil {
		panic("write config url: " + err.Error())
	}
	if err := config.Session.Write(client.Token()); err != nil {
		panic("write config token: " + err.Error())
	}
}

type result struct {
	outBuffer *bytes.Buffer
	errBuffer *bytes.Buffer
	exitErr   error
}

func (r result) success(t *testing.T) {
	assert.Success(t, "execute command", r.exitErr)
}

func (r result) stdoutContains(t *testing.T, substring string) {
	if !strings.Contains(r.outBuffer.String(), substring) {
		slogtest.Fatal(t, "stdout contains substring", slog.F("substring", substring), slog.F("stdout", r.outBuffer.String()))
	}
}

func (r result) stdoutUnmarshals(t *testing.T, target interface{}) {
	err := json.Unmarshal(r.outBuffer.Bytes(), target)
	assert.Success(t, "unmarshal json", err)
}

func (r result) stdoutEmpty(t *testing.T) {
	assert.Equal(t, "stdout empty", "", r.outBuffer.String())
}

func (r result) stderrEmpty(t *testing.T) {
	assert.Equal(t, "stderr empty", "", r.errBuffer.String())
}

func (r result) stderrContains(t *testing.T, substring string) {
	if !strings.Contains(r.errBuffer.String(), substring) {
		slogtest.Fatal(t, "stderr contains substring", slog.F("substring", substring), slog.F("stderr", r.errBuffer.String()))
	}
}

func (r result) clogError(t *testing.T) clog.CLIError {
	var cliErr clog.CLIError
	if !xerrors.As(r.exitErr, &cliErr) {
		slogtest.Fatal(t, "expected clog error, none found", slog.Error(r.exitErr), slog.F("type", fmt.Sprintf("%T", r.exitErr)))
	}
	slogtest.Debug(t, "clog error", slog.F("message", cliErr.String()))
	return cliErr
}

func execute(t *testing.T, args []string, in io.Reader) result {
	cmd := Make()

	var outStream bytes.Buffer
	var errStream bytes.Buffer

	cmd.SetArgs(args)

	cmd.SetIn(in)
	cmd.SetOut(&outStream)
	cmd.SetErr(&errStream)

	err := cmd.Execute()

	slogtest.Debug(t, "execute command",
		slog.F("outBuffer", outStream.String()),
		slog.F("errBuffer", errStream.String()),
		slog.F("args", args),
		slog.F("execute_error", err),
	)
	return result{
		outBuffer: &outStream,
		errBuffer: &errStream,
		exitErr:   err,
	}
}
