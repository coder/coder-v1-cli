package sync

import (
	"crypto/sha256"
	"encoding/hex"

	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/entclient"
)

type Syncer struct {
	Environment entclient.Environment
	RemoteDir   string
	LocalDir    string
	Client      entclient.Client
}

func (s *Syncer) syncExists(name string) bool {
	c, err := mutagenCmd("sync", "list", name)
	if err != nil {
		return false
	}
	_, err = c.CombinedOutput()
	if err != nil {
		// We probably couldn't find the sync.
		return false
	}
	return true
}

// Create creates a sync session.
func (s *Syncer) Create() error {
	var (
		alpha = s.LocalDir
		beta  = "coder." + s.Environment.Name + ":" + s.RemoteDir
	)
	// Checksum the name so it doesn't have to be massive.
	csm := sha256.Sum256([]byte(alpha + beta))
	name := "cdr-" + hex.EncodeToString(csm[:])[:8]
	if s.syncExists(name) {
		flog.Info("sync already exists, entering monitor")
		return becomeMutagen( "sync", "monitor", name)
	}
	flog.Info("trying to create sync %s", name)
	return becomeMutagen(
		"sync", "create",
		"-l", "com.coder=true",
		// Give the sync a stable name to prevent bloat.
		"-n", name,
		alpha, beta,
	)
}

// Monitor monitors the most recent sync session.
func (s *Syncer) Monitor() error {
	flog.Info("you may exit the command without interrupting your sync")
	flog.Info("call 'coder mutagen sync monitor' to watch the sync again")
	return becomeMutagen(
		"monitor",
	)
}
