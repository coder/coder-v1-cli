package tcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"
)

// ContainerConfig describes the ContainerRunner configuration schema for initializing a testing environment
type ContainerConfig struct {
	Name       string
	Image      string
	BindMounts map[string]string
}

func mountArgs(m map[string]string) []string {
	args := make([]string, 0, len(m))
	for src, dest := range m {
		args = append(args, "--mount", fmt.Sprintf("type=bind,source=%s,target=%s", src, dest))
	}
	return args
}

func preflightChecks() error {
	_, err := exec.LookPath("docker")
	if err != nil {
		return xerrors.Errorf(`"docker" not found in $PATH`)
	}
	return nil
}

// ContainerRunner specifies a runtime container for performing command tests
type ContainerRunner struct {
	name string
}

// NewContainerRunner starts a new docker container for executing command tests
func NewContainerRunner(ctx context.Context, config *ContainerConfig) (*ContainerRunner, error) {
	if err := preflightChecks(); err != nil {
		return nil, err
	}

	args := []string{
		"run",
		"--name", config.Name,
		"--network", "host",
		"-it", "-d",
	}
	args = append(args, mountArgs(config.BindMounts)...)
	args = append(args, config.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, xerrors.Errorf(
			"start testing container %q, (%s): %w",
			config.Name, string(out), err)
	}

	return &ContainerRunner{
		name: config.Name,
	}, nil
}

// Close kills and removes the command execution testing container
func (r *ContainerRunner) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"sh", "-c", strings.Join([]string{
			"docker", "kill", r.name, "&&",
			"docker", "rm", r.name,
		}, " "))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return xerrors.Errorf(
			"stop testing container %q, (%s): %w",
			r.name, string(out), err)
	}
	return nil
}

// Run executes the given command in the runtime container with reasonable defaults.
// "command" is executed in a shell as an argument to "sh -c".
func (r *ContainerRunner) Run(ctx context.Context, command string) *Assertable {
	cmd := exec.CommandContext(ctx,
		"docker", "exec", "-i", r.name,
		"sh", "-c", command,
	)

	return &Assertable{
		cmd:   cmd,
		tname: command,
	}
}

// RunCmd lifts the given *exec.Cmd into the runtime container
func (r *ContainerRunner) RunCmd(cmd *exec.Cmd) *Assertable {
	path, _ := exec.LookPath("docker")
	cmd.Path = path
	command := strings.Join(cmd.Args, " ")
	cmd.Args = append([]string{"docker", "exec", "-i", r.name}, cmd.Args...)

	return &Assertable{
		cmd:   cmd,
		tname: command,
	}
}

// HostRunner executes command tests on the host, outside of a container
type HostRunner struct{}

// Run executes the given command on the host.
// "command" is executed in a shell as an argument to "sh -c".
func (r *HostRunner) Run(ctx context.Context, command string) *Assertable {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	return &Assertable{
		cmd:   cmd,
		tname: command,
	}
}

// RunCmd executes the given *exec.Cmd on the host
func (r *HostRunner) RunCmd(cmd *exec.Cmd) *Assertable {
	return &Assertable{
		cmd:   cmd,
		tname: strings.Join(cmd.Args, " "),
	}
}

// Close is a noop for HostRunner
func (r *HostRunner) Close() error {
	return nil
}

// Assertable describes an initialized command ready to be run and asserted against
type Assertable struct {
	cmd   *exec.Cmd
	tname string
}

// Assert runs the Assertable and
func (a *Assertable) Assert(t *testing.T, option ...Assertion) {
	slog.Helper()
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
		result CommandResult
	)
	if a.cmd == nil {
		slogtest.Fatal(t, "test failed to initialize: no command specified")
	}

	a.cmd.Stdout = &stdout
	a.cmd.Stderr = &stderr

	start := time.Now()
	err := a.cmd.Run()
	result.Duration = time.Since(start)

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		slogtest.Fatal(t, "command failed to run", slog.Error(err), slog.F("command", a.cmd))
	} else {
		result.ExitCode = 0
	}

	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	slogtest.Info(t, "command output",
		slog.F("command", a.cmd),
		slog.F("stdout", string(result.Stdout)),
		slog.F("stderr", string(result.Stderr)),
		slog.F("exit_code", result.ExitCode),
		slog.F("duration", result.Duration),
	)

	for _, assertion := range option {
		assertion(t, &result)
	}
}

// Assertion specifies an assertion on the given CommandResult.
// Pass custom Assertion functions to cover special cases.
type Assertion func(t *testing.T, r *CommandResult)

// CommandResult contains the aggregated result of a command execution
type CommandResult struct {
	Stdout, Stderr []byte
	ExitCode       int
	Duration       time.Duration
}

// Success asserts that the command exited with an exit code of 0
func Success() Assertion {
	slog.Helper()
	return ExitCodeIs(0)
}

// Error asserts that the command exited with a nonzero exit code
func Error() Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		assert.True(t, "exit code is nonzero", r.ExitCode != 0)
	}
}

// ExitCodeIs asserts that the command exited with the given code
func ExitCodeIs(code int) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		assert.Equal(t, "exit code is as expected", code, r.ExitCode)
	}
}

// StdoutEmpty asserts that the command did not write any data to Stdout
func StdoutEmpty() Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		empty(t, "stdout", r.Stdout)
	}
}

// GetResult offers an escape hatch from tcli
// The pointer passed as "result" will be assigned to the command's *CommandResult
func GetResult(result **CommandResult) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		*result = r
	}
}

// StderrEmpty asserts that the command did not write any data to Stderr
func StderrEmpty() Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		empty(t, "stderr", r.Stderr)
	}
}

// StdoutMatches asserts that Stdout contains a substring which matches the given regexp
func StdoutMatches(pattern string) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		matches(t, "stdout", pattern, r.Stdout)
	}
}

// StderrMatches asserts that Stderr contains a substring which matches the given regexp
func StderrMatches(pattern string) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		matches(t, "stderr", pattern, r.Stderr)
	}
}

func matches(t *testing.T, name, pattern string, target []byte) {
	slog.Helper()
	fields := []slog.Field{
		slog.F("pattern", pattern),
		slog.F("target", string(target)),
		slog.F("sink", name),
	}

	ok, err := regexp.Match(pattern, target)
	if err != nil {
		slogtest.Fatal(t, "attempt regexp match", append(fields, slog.Error(err))...)
	}
	if !ok {
		slogtest.Fatal(t, "expected to find pattern, no match found", fields...)
	}
}

func empty(t *testing.T, name string, a []byte) {
	slog.Helper()
	if len(a) > 0 {
		slogtest.Fatal(t, "expected "+name+" to be empty", slog.F("got", string(a)))
	}
}

// DurationLessThan asserts that the command completed in less than the given duration
func DurationLessThan(dur time.Duration) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		if r.Duration > dur {
			slogtest.Fatal(t, "duration longer than expected",
				slog.F("expected_less_than", dur.String),
				slog.F("actual", r.Duration.String()),
			)
		}
	}
}

// DurationGreaterThan asserts that the command completed in greater than the given duration
func DurationGreaterThan(dur time.Duration) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		if r.Duration < dur {
			slogtest.Fatal(t, "duration shorter than expected",
				slog.F("expected_greater_than", dur.String),
				slog.F("actual", r.Duration.String()),
			)
		}
	}
}

// StdoutJSONUnmarshal attempts to unmarshal stdout into the given target
func StdoutJSONUnmarshal(target interface{}) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		err := json.Unmarshal(r.Stdout, target)
		assert.Success(t, "stdout json unmarshals", err)
	}
}

// StderrJSONUnmarshal attempts to unmarshal stderr into the given target
func StderrJSONUnmarshal(target interface{}) Assertion {
	return func(t *testing.T, r *CommandResult) {
		slog.Helper()
		err := json.Unmarshal(r.Stdout, target)
		assert.Success(t, "stderr json unmarshals", err)
	}
}
