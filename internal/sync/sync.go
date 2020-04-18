package sync

import (
	"context"

	"go.coder.com/flog"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder/internal/client"
)

type Sync struct {
	*client.Client
	client.Environment
}

func (s Sync) Run() error {
	conn, err := s.Wush(s.Environment, "ls", "/proc")
	if err != nil {
		flog.Fatal("establish wush: %v", err)
	}
	defer conn.Close(websocket.StatusAbnormalClosure, "idk")

	flog.Info("waiting for messages")

	ctx := context.Background()
	for {
		typ, msg, err := conn.Read(ctx)
		if err != nil {
			return xerrors.Errorf("read: %w", err)
		}
		flog.Info("%s %s", typ, msg)
	}
}
