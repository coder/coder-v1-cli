package clog

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/internal/x/xsync"
)

func TestErrGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		egroup := LoggedErrGroup()

		var buf bytes.Buffer
		SetOutput(xsync.Writer(&buf))

		egroup.Go(func() error { return nil })
		egroup.Go(func() error { return nil })
		egroup.Go(func() error { return nil })

		err := egroup.Wait()
		assert.Success(t, "error group wait", err)
		assert.Equal(t, "empty log buffer", "", buf.String())
	})
	t.Run("failure_count", func(t *testing.T) {
		egroup := LoggedErrGroup()

		var buf bytes.Buffer
		SetOutput(xsync.Writer(&buf))

		egroup.Go(func() error { return errors.New("whoops") })
		egroup.Go(func() error { return Error("rich error", "second line") })

		err := egroup.Wait()
		assert.ErrorContains(t, "error group wait", err, "2 failures emitted")
		assert.True(t, "log buf contains", strings.Contains(buf.String(), "fatal: whoops\n\n"))
		assert.True(t, "log buf contains", strings.Contains(buf.String(), "error: rich error\n  | second line\n\n"))
	})
}
