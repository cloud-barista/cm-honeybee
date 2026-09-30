package spider

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWithTimeoutReturnsValue(t *testing.T) {
	want := errors.New("driver error")
	v, err := withTimeout(time.Second, func() (int, error) { return 7, want })
	if v != 7 || err != want {
		t.Fatalf("got (%d, %v), want (7, %v)", v, err, want)
	}
}

func TestWithTimeoutBlockedCall(t *testing.T) {
	release := make(chan struct{})
	finished := make(chan struct{})
	defer func() {
		close(release)
		<-finished
	}()

	d := 20 * time.Millisecond
	v, err := withTimeout(d, func() ([]VMInfo, error) {
		defer close(finished)
		<-release
		return []VMInfo{{}}, nil
	})
	if v != nil {
		t.Fatalf("value = %v, want nil", v)
	}
	if err == nil || !strings.Contains(err.Error(), "timed out after "+d.String()) {
		t.Fatalf("err = %v, want a timeout after %s", err, d)
	}
}

func TestWithTimeoutPanicBeforeDeadline(t *testing.T) {
	old := logDriverPanic
	var logged []byte
	logDriverPanic = func(p any, stack []byte) { logged = stack }
	defer func() { logDriverPanic = old }()

	v, err := withTimeout(time.Second, func() (int, error) { panic("driver boom") })
	if v != 0 {
		t.Fatalf("value = %d, want 0", v)
	}
	if err == nil || !strings.Contains(err.Error(), "driver call panicked: driver boom") {
		t.Fatalf("err = %v, want the panic value", err)
	}
	if !strings.Contains(string(logged), "TestWithTimeoutPanicBeforeDeadline") {
		t.Fatalf("logged stack does not show the driver goroutine:\n%s", logged)
	}
}

func TestWithTimeoutPanicAfterDeadline(t *testing.T) {
	old := logDriverPanic
	logged := make(chan any, 1)
	logDriverPanic = func(p any, stack []byte) { logged <- p }
	defer func() { logDriverPanic = old }()

	release := make(chan struct{})
	_, err := withTimeout(20*time.Millisecond, func() (int, error) {
		<-release
		panic("late boom")
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	close(release)
	select {
	case p := <-logged:
		if p != "late boom" {
			t.Fatalf("logged %v, want late boom", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("late panic was not logged")
	}
}
