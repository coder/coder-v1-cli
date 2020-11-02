package clog

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"golang.org/x/xerrors"
)

func TestError(t *testing.T) {
	t.Run("oneline", func(t *testing.T) {
		var mockErr error = Error("fake error")
		mockErr = xerrors.Errorf("wrap 1: %w", mockErr)
		mockErr = fmt.Errorf("wrap 2: %w", mockErr)

		reader, writer, err := os.Pipe()
		assert.Success(t, "create pipe", err)

		//! clearly not thread safe
		os.Stderr = writer

		Log(mockErr)
		writer.Close()

		output, err := ioutil.ReadAll(reader)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t, "output is as expected", "error: fake error\n\n", string(output))
	})

	t.Run("plain-error", func(t *testing.T) {
		mockErr := xerrors.Errorf("base error")
		mockErr = fmt.Errorf("wrap 1: %w", mockErr)

		reader, writer, err := os.Pipe()
		assert.Success(t, "create pipe", err)

		//! clearly not thread safe
		os.Stderr = writer

		Log(mockErr)
		writer.Close()

		output, err := ioutil.ReadAll(reader)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t, "output is as expected", "fatal: wrap 1: base error\n\n", string(output))
	})

	t.Run("multi-line", func(t *testing.T) {
		var mockErr error = Error("fake header", "next line", BlankLine, Tipf("content of fake tip"))
		mockErr = xerrors.Errorf("wrap 1: %w", mockErr)
		mockErr = fmt.Errorf("wrap 1: %w", mockErr)

		reader, writer, err := os.Pipe()
		assert.Success(t, "create pipe", err)

		//! clearly not thread safe
		os.Stderr = writer

		Log(mockErr)
		writer.Close()

		output, err := ioutil.ReadAll(reader)
		assert.Success(t, "read all stderr output", err)

		assert.Equal(t,
			"output is as expected",
			"error: fake header\n  | next line\n  | \n  | tip: content of fake tip\n\n",
			string(output),
		)
	})
}
