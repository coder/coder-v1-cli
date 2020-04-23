package wush

type promise struct {
	wait chan struct{}
}

func newPromise() promise {
	return promise{wait: make(chan struct{})}
}

// Wait may be called any number of times by value recievers.
func (o promise) Wait() {
	<-o.wait
}

// Release may only be called once from the value setter.
func (o promise) Release() {
	close(o.wait)
}
