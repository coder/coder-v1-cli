package tlscertificates_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/coder-cli/wsnet/tlscertificates"
)

func TestLoadDirectory(t *testing.T) {
	t.Parallel()

	t.Run("ValidDirectory", func(t *testing.T) {
		// Load the testdata certs
		certs, err := tlscertificates.LoadCertsFromDirectory("testdata")
		require.NoError(t, err)
		// ca-certificates.crt is 6 certs
		// Comodo is 1
		// VeriSign is 1
		require.Len(t, certs, 6+1+1)
	})

	t.Run("NonExistantDir", func(t *testing.T) {
		_, err := tlscertificates.LoadCertsFromDirectory("not-exists")
		require.Error(t, err)
		require.Regexp(t, "no such file or directory", err.Error())
	})
}
