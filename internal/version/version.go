package version

import (
	"strings"
)

// Version is populated at compile-time with the current coder-cli version.
var Version string = "unknown"

// VersionMatch compares the given APIVersion to the compile-time injected coder-cli version.
func VersionsMatch(apiVersion string) bool {
	withoutPatchRelease := strings.Split(Version, ".")
	if len(withoutPatchRelease) < 3 {
		return false
	}
	majorMinor := strings.Join(withoutPatchRelease[:len(withoutPatchRelease)-1], ".")
	return strings.HasPrefix(strings.TrimPrefix(apiVersion, "v"), strings.TrimPrefix(majorMinor, "v"))
}
