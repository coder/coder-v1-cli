package tcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"
)

type RunContainer struct {
	name string
	ctx  context.Context
}

func NewRunContainer(ctx context.Context, image, name string) (*RunContainer, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "run",
		"--name", name,
		"-it", "-d",
		image,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, xerrors.Errorf(
			"failed to start testing container %q, (%s): %w",
			name, string(out), err)
	}

	return &RunContainer{
		name: name,
		ctx:  ctx,
	}, nil
}

func (r *RunContainer) Close() error {
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

type Assertable struct {
	cmd       string
	ctx       context.Context
	container *RunContainer
}

func (r *RunContainer) Run(ctx context.Context, cmd string) *Assertable {
	return &Assertable{
		cmd:       cmd,
		ctx:       ctx,
		container: r,
	}
}

func (a Assertable) Assert(t *testing.T, option ...Assertion) {
	var cmdResult CommandResult

	cmd := exec.CommandContext(a.ctx,
		"docker", "exec", a.container.name,
		"sh", "-c", a.cmd,
	)
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
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

	for ix, o := range option {
		name := fmt.Sprintf("assertion_#%v", ix)
		if named, ok := o.(Named); ok {
			name = named.Name()
		}
		t.Run(name, func(t *testing.T) {
			err := o.Valid(cmdResult)
			assert.Success(t, name, err)
		})
	}
}

type Assertion interface {
	Valid(r CommandResult) error
}

type Named interface {
	Name() string
}

type CommandResult struct {
	Stdout, Stderr []byte
	ExitCode       int
	Duration       time.Duration
}

type simpleFuncAssert struct {
	valid func(r CommandResult) error
	name  string
}

func (s simpleFuncAssert) Valid(r CommandResult) error {
	return s.valid(r)
}

func (s simpleFuncAssert) Name() string {
	return s.name
}

func Success() Assertion {
	return ExitCodeIs(0)
}

func ExitCodeIs(code int) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			if r.ExitCode != code {
				return xerrors.Errorf("exit code of %s expected, got %v", code, r.ExitCode)
			}
			return nil
		},
		name: fmt.Sprintf("exitcode"),
	}
}

func StdoutEmpty() Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			return empty("stdout", r.Stdout)
		},
		name: fmt.Sprintf("stdout-empty"),
	}
}

func StderrEmpty() Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			return empty("stderr", r.Stderr)
		},
		name: fmt.Sprintf("stderr-empty"),
	}
}

func StdoutMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			return matches("stdout", pattern, r.Stdout)
		},
		name: fmt.Sprintf("stdout-matches"),
	}
}

func StderrMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			return matches("stderr", pattern, r.Stderr)
		},
		name: fmt.Sprintf("stderr-matches"),
	}
}

func CombinedMatches(pattern string) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
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
		return xerrors.Errorf("expected to find pattern (%s) in %s, no match found", pattern, name)
	}
	return nil
}

func empty(name string, a []byte) error {
	if len(a) > 0 {
		return xerrors.Errorf("expected %s to be empty, got (%s)", name, string(a))
	}
	return nil
}

func DurationLessThan(dur time.Duration) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			if r.Duration > dur {
				return xerrors.Errorf("expected duration less than %s, took %s", dur.String(), r.Duration.String())
			}
			return nil
		},
		name: fmt.Sprintf("duration-lessthan"),
	}
}

func DurationGreaterThan(dur time.Duration) Assertion {
	return simpleFuncAssert{
		valid: func(r CommandResult) error {
			if r.Duration < dur {
				return xerrors.Errorf("expected duration greater than %s, took %s", dur.String(), r.Duration.String())
			}
			return nil
		},
		name: fmt.Sprintf("duration-greaterthan"),
	}
}
