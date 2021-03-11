package clog

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"
)

func TestError(t *testing.T) {
	t.Run("oneline", func(t *testing.T) {
		var mockErr error = Error("fake error")
		mockErr = xerrors.Errorf("wrap 1: %w", mockErr)
		mockErr = fmt.Errorf("wrap 2: %w", mockErr)

		var buf bytes.Buffer
		//! clearly not concurrent safe
		SetOutput(&buf)

		Log(mockErr)

		output, err := ioutil.ReadAll(&buf)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t, "output is as expected", "error: fake error\n\n", string(output))
	})

	t.Run("plain-error", func(t *testing.T) {
		mockErr := xerrors.Errorf("base error")
		mockErr = fmt.Errorf("wrap 1: %w", mockErr)

		var buf bytes.Buffer
		//! clearly not concurrent safe
		SetOutput(&buf)

		Log(mockErr)

		output, err := ioutil.ReadAll(&buf)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t, "output is as expected", "fatal: wrap 1: base error\n\n", string(output))
	})

	t.Run("message", func(t *testing.T) {
		for _, f := range []struct {
			f     func(string, ...string)
			level string
		}{{LogInfo, "info"}, {LogSuccess, "success"}, {LogWarn, "warning"}} {
			var buf bytes.Buffer
			//! clearly not concurrent safe
			SetOutput(&buf)

			f.f("testing", Hintf("maybe do %q", "this"), BlankLine, Causef("what happened was %q", "this"))

			output, err := ioutil.ReadAll(&buf)
			assert.Success(t, "read all stderr output", err)

			assert.Equal(t, "output is as expected", f.level+": testing\n  | hint: maybe do \"this\"\n  | \n  | cause: what happened was \"this\"\n", string(output))
		}
	})

	t.Run("multi-line", func(t *testing.T) {
		var mockErr error = Error("fake header", "next line", BlankLine, Tipf("content of fake tip"))
		mockErr = xerrors.Errorf("wrap 1: %w", mockErr)
		mockErr = fmt.Errorf("wrap 1: %w", mockErr)

		var buf bytes.Buffer
		//! clearly not concurrent safe
		SetOutput(&buf)

		Log(mockErr)

		output, err := ioutil.ReadAll(&buf)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t,
			"output is as expected",
			"error: fake header\n  | next line\n  | \n  | tip: content of fake tip\n\n",
			string(output),
		)
	})
}
