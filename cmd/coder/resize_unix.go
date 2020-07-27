// +build !windows

package main

import (
	"context"
	"os"
	"os/signal"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"

	"go.coder.com/flog"
)

func resizeEvents(ctx context.Context, termfd int) chan resizeEvent {
	sigs := make(chan os.Signal, 16)
	signal.Notify(sigs, unix.SIGWINCH)

	events := make(chan resizeEvent)

	go func() {
		for ctx.Err() == nil {
			width, height, err := terminal.GetSize(termfd)
			if err != nil {
				flog.Error("get term size: %v", err)
				break
			}
			events <- resizeEvent{
				height: uint16(height),
				width:  uint16(width),
			}

			<-sigs
		}
	}()

	return events
}
