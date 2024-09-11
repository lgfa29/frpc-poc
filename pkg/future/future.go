package future

import "sync"

type Future struct {
	waitCh chan struct{}
	result any
	err    error
	once   *sync.Once
}

func New() *Future {
	return &Future{
		waitCh: make(chan struct{}),
		once:   new(sync.Once),
	}
}

func (f *Future) Wait() (any, error) {
	<-f.waitCh
	return f.result, f.err
}

func (f *Future) Fail(err error) {
	f.once.Do(func() {
		f.err = err
		close(f.waitCh)
	})
}

func (f *Future) Fulfill(val any) {
	f.once.Do(func() {
		f.result = val
		close(f.waitCh)
	})
}
