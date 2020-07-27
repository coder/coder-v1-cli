// +build windows

package main

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"go.coder.com/flog"
)

// windows does have a unix.SIGWINCH equivalent, so we poll the terminal size
// and send resize events when needed
func resizeEvents(ctx context.Context, termfd int) chan resizeEvent {
	events := make(chan resizeEvent)
	t := time.Tick(time.Millisecond * 100)

	go func() {
		var lastEvent *resizeEvent

		for {
			select {
			case <-ctx.Done():
				break
			case <-t:
				width, height, err := terminal.GetSize(termfd)
				if err != nil {
					flog.Error("get term size: %v", err)
					break
				}
				event := resizeEvent{
					height: uint16(height),
					width:  uint16(width),
				}
				if !event.equal(lastEvent) {
					events <- event
				}
				lastEvent = &event
			}
		}
	}()

	return events
}
