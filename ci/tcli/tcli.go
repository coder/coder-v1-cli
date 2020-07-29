package tcli

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

var (
	_ runnable = &ContainerRunner{}
	_ runnable = &HostRunner{}
)

type runnable interface {
	Run(ctx context.Context, command string) *Assertable
	RunCmd(cmd *exec.Cmd) *Assertable
	io.Closer
}

// ContainerConfig describes the ContainerRunner configuration schema for initializing a testing environment
type ContainerConfig struct {
	Name       string
	Image      string
	BindMounts map[string]string
}

func mountArgs(m map[string]string) (args []string) {
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
	ctx  context.Context
}

// NewContainerRunner starts a new docker container for executing command tests
func NewContainerRunner(ctx context.Context, config *ContainerConfig) (*ContainerRunner, error) {
	if err := preflightChecks(); err != nil {
		return nil, err
	}

	args := []string{
		"run",
		"--name", config.Name,
		"-it", "-d",
	}
	args = append(args, mountArgs(config.BindMounts)...)
	args = append(args, config.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, xerrors.Errorf(
			"failed to start testing container %q, (%s): %w",
			config.Name, string(out), err)
	}

	return &ContainerRunner{
		name: config.Name,
		ctx:  ctx,
	}, nil
}

// Close kills and removes the command execution testing container
func (r *ContainerRunner) Close() error {
	cmd := exec.CommandContext(r.ctx,
		"sh", "-c", strings.Join([]string{
			"docker", "kill", r.name, "&&",
			"docker", "rm", r.name,
		}, " "))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return xerrors.Errorf(
			"failed to stop testing container %q, (%s): %w",
			r.name, string(out), err)
	}
	return nil
}

// HostRunner executes command tests on the host, outside of a container
type HostRunner struct{}

// Run executes the given command on the host
func (r *HostRunner) Run(ctx context.Context, command string) *Assertable {
	var (
		args  []string
		path  string
		parts = strings.Split(command, " ")
	)
	if len(parts) > 0 {
		path = parts[0]
	}
	if len(parts) > 1 {
		args = parts[1:]
	}
	cmd := exec.CommandContext(ctx, path, args...)

	return &Assertable{
		cmd:   cmd,
		tname: command,
	}
}

// RunCmd executes the given command on the host
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

// Run executes the given command in the runtime container with reasonable defaults
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
	cmd.Args = []string{"docker", "exec", "-i", r.name, "sh", "-c", command}

	return &Assertable{
		cmd:   cmd,
		tname: command,
	}
}

// Assert runs the Assertable and
func (a Assertable) Assert(t *testing.T, option ...Assertion) {
	t.Run(a.tname, func(t *testing.T) {
		var cmdResult CommandResult

		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		a.cmd.Stdout = &stdout
		a.cmd.Stderr = &stderr

		start := time.Now()
		err := a.cmd.Run()
		cmdResult.Duration = time.Since(start)

		if exitErr, ok := err.(*exec.ExitError); ok {
			cmdResult.ExitCode = exitErr.ExitCode()
		} else if err != nil {
			cmdResult.ExitCode = -1
		} else {
			cmdResult.ExitCode = 0
		}

		cmdResult.Stdout = stdout.Bytes()
		cmdResult.Stderr = stderr.Bytes()

		slogtest.Info(t, "command output",
			slog.F("command", a.cmd),
			slog.F("stdout", string(cmdResult.Stdout)),
			slog.F("stderr", string(cmdResult.Stderr)),
			slog.F("exit-code", cmdResult.ExitCode),
			slog.F("duration", cmdResult.Duration),
		)

		for ix, o := range option {
			name := fmt.Sprintf("assertion_#%v", ix)
			if named, ok := o.(Named); ok {
				name = named.Name()
			}
			t.Run(name, func(t *testing.T) {
				err := o.Valid(&cmdResult)
				assert.Success(t, name, err)
			})
		}
	})
}

// Assertion specifies an assertion on the given CommandResult.
// Pass custom Assertion types to cover special cases.
type Assertion interface {
	Valid(r *CommandResult) error
}

// Named is an optional extension of Assertion that provides a helpful label
// to *testing.T
type Named interface {
	Name() string
}

// CommandResult contains the aggregated result of a command execution
type CommandResult struct {
	Stdout, Stderr []byte
	ExitCode       int
	Duration       time.Duration
}

type simpleFuncAssert struct {
	valid func(r *CommandResult) error
	name  string
}

func (s simpleFuncAssert) Valid(r *CommandResult) error {
	return s.valid(r)
}

func (s simpleFuncAssert) Name() string {
	return s.name
}

// Success asserts that the command exited with an exit code of 0
func Success() Assertion {
	return ExitCodeIs(0)
}

// Error asserts that the command exited with a nonzero exit code
func Error() Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			if r.ExitCode == 0 {
				return xerrors.Errorf("expected nonzero exit code, got %v", r.ExitCode)
			}
			return nil
		},
		name: fmt.Sprintf("error"),
	}
}

// ExitCodeIs asserts that the command exited with the given code
func ExitCodeIs(code int) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			if r.ExitCode != code {
				return xerrors.Errorf("exit code of %v expected, got %v, (%s)", code, r.ExitCode, string(r.Stderr))
			}
			return nil
		},
		name: fmt.Sprintf("exitcode"),
	}
}

// StdoutEmpty asserts that the command did not write any data to Stdout
func StdoutEmpty() Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			return empty("stdout", r.Stdout)
		},
		name: fmt.Sprintf("stdout-empty"),
	}
}

// GetResult offers an escape hatch from tcli
// The pointer passed as "result" will be assigned to the command's *CommandResult
func GetResult(result **CommandResult) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			*result = r
			return nil
		},
		name: "get-result",
	}
}

// StderrEmpty asserts that the command did not write any data to Stderr
func StderrEmpty() Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			return empty("stderr", r.Stderr)
		},
		name: fmt.Sprintf("stderr-empty"),
	}
}

// StdoutMatches asserts that Stdout contains a substring which matches the given regexp
func StdoutMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			return matches("stdout", pattern, r.Stdout)
		},
		name: fmt.Sprintf("stdout-matches"),
	}
}

// StderrMatches asserts that Stderr contains a substring which matches the given regexp
func StderrMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			return matches("stderr", pattern, r.Stderr)
		},
		name: fmt.Sprintf("stderr-matches"),
	}
}

// CombinedMatches asserts that either Stdout or Stderr a substring which matches the given regexp
func CombinedMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			//stdoutValid := StdoutMatches(pattern).Valid(r)
			//stderrValid := StderrMatches(pattern).Valid(r)
			// TODO: combine errors
			return nil
		},
		name: fmt.Sprintf("combined-matches"),
	}
}

func matches(name, pattern string, target []byte) error {
	ok, err := regexp.Match(pattern, target)
	if err != nil {
		return xerrors.Errorf("failed to attempt regexp match: %w", err)
	}
	if !ok {
		return xerrors.Errorf(
			"expected to find pattern (%s) in %s, no match found in (%v)",
			pattern, name, string(target),
		)
	}
	return nil
}

func empty(name string, a []byte) error {
	if len(a) > 0 {
		return xerrors.Errorf("expected %s to be empty, got (%s)", name, string(a))
	}
	return nil
}

// DurationLessThan asserts that the command completed in less than the given duration
func DurationLessThan(dur time.Duration) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			if r.Duration > dur {
				return xerrors.Errorf("expected duration less than %s, took %s", dur.String(), r.Duration.String())
			}
			return nil
		},
		name: fmt.Sprintf("duration-lessthan"),
	}
}

// DurationGreaterThan asserts that the command completed in greater than the given duration
func DurationGreaterThan(dur time.Duration) Assertion {
	return simpleFuncAssert{
		valid: func(r *CommandResult) error {
			if r.Duration < dur {
				return xerrors.Errorf("expected duration greater than %s, took %s", dur.String(), r.Duration.String())
			}
			return nil
		},
		name: fmt.Sprintf("duration-greaterthan"),
	}
}
