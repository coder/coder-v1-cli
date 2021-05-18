package clog

import (
	"fmt"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

// ErrGroup wraps the /x/sync/errgroup.(Group) and adds clog logging and rich error propagation.
//
// Take for example, a case in which we are concurrently stopping a slice of workspaces.
// In this case, we want to log errors as they happen, not pass them through the callstack as errors.
// When the operations complete, we want to log how many, if any, failed. The caller is still expected
// to handle success and info logging.
type ErrGroup interface {
	Go(f func() error)
	Wait() error
}

type group struct {
	egroup   errgroup.Group
	failures int32
}

// LoggedErrGroup gives an error group with error logging and error propagation handled automatically.
func LoggedErrGroup() ErrGroup {
	return &group{
		egroup:   errgroup.Group{},
		failures: 0,
	}
}

func (g *group) Go(f func() error) {
	g.egroup.Go(func() error {
		if err := f(); err != nil {
			atomic.AddInt32(&g.failures, 1)
			Log(err)

			// this error does not matter because we discard it in Wait.
			return xerrors.New("")
		}
		return nil
	})
}

func (g *group) Wait() error {
	_ = g.egroup.Wait() // ignore this error because we are already tracking failures manually
	if g.failures == 0 {
		return nil
	}
	failureWord := "failure"
	if g.failures > 1 {
		failureWord += "s"
	}
	return Fatal(fmt.Sprintf("%d %s emitted", g.failures, failureWord))
}
