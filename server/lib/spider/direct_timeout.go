package spider

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/jollaman999/utils/logger"
)

// driverCallTimeout bounds one exported driver call (connect plus handler
// call), matching the 60s the cb-spider REST client gave each request.
var driverCallTimeout = 60 * time.Second

// logDriverPanic records a driver panic with the stack of the goroutine it
// happened in.
var logDriverPanic = func(p any, stack []byte) {
	logger.Println(logger.ERROR, false, fmt.Sprintf("spider: driver call panicked: %v\n%s", p, stack))
}

// withTimeout runs fn and gives up after d. cb-spider drivers take no
// context, so a call that times out keeps running in its goroutine until the
// driver returns on its own, as cb-spider kept working after the REST client
// gave up.
//
// A panic in fn is logged with its stack and returned as an error, as
// cb-spider's REST server turned it into a 500 through Echo's Recover
// middleware (api-runtime/rest-runtime/CBSpiderRuntime.go, ApiServer). After
// a timeout nobody is waiting and the panic is only logged.
func withTimeout[T any](d time.Duration, fn func() (T, error)) (T, error) {
	type result struct {
		v        T
		err      error
		panicked bool
		p        any
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				logDriverPanic(p, debug.Stack())
				done <- result{panicked: true, p: p}
			}
		}()
		v, err := fn()
		done <- result{v: v, err: err}
	}()

	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case r := <-done:
		if r.panicked {
			var zero T
			return zero, fmt.Errorf("driver call panicked: %v", r.p)
		}
		return r.v, r.err
	case <-timer.C:
		var zero T
		return zero, fmt.Errorf("driver call timed out after %s", d)
	}
}
