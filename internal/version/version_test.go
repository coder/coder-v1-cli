package version

import (
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
)

func TestVersion(t *testing.T) {
	Version = "1.12.1"
	match := VersionsMatch("1.12.2")
	assert.True(t, "versions match", match)

	Version = "v1.14.1"
	match = VersionsMatch("1.15.2")
	assert.True(t, "versions do not match", !match)

	Version = "v1.15.4"
	match = VersionsMatch("1.15.2")
	assert.True(t, "versions do match", match)

	Version = "1.15.4"
	match = VersionsMatch("v1.15.2")
	assert.True(t, "versions do match", match)

	Version = "1.15.4"
	match = VersionsMatch("v2.15.2")
	assert.True(t, "versions do not match", !match)
}