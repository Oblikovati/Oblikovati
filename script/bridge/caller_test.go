// SPDX-License-Identifier: GPL-2.0-only

package bridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"oblikovati.org/addin/dispatch"
	"oblikovati.org/app"
)

// recordingRoute is a named fake RouteFunc that records the method it saw and returns a
// scripted reply — exercising the caller wiring without the real router.
type recordingRoute struct {
	reply  []byte
	err    error
	method string
}

func (r *recordingRoute) route(_ *app.Session, method string, _ []byte) ([]byte, error) {
	r.method = method
	return r.reply, r.err
}

func TestDirectCallerInvokesRoute(t *testing.T) {
	rec := &recordingRoute{reply: []byte(`{"ok":true}`)}
	c := NewDirectCaller(rec.route, app.NewSession())
	out, err := c.Call("documents.list", []byte(`{}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if rec.method != "documents.list" {
		t.Errorf("method = %q, want documents.list", rec.method)
	}
	if string(out) != `{"ok":true}` {
		t.Errorf("reply not forwarded: %s", out)
	}
}

func TestDirectCallerWithoutRouteErrors(t *testing.T) {
	c := NewDirectCaller(nil, app.NewSession())
	if _, err := c.Call("x", nil); err == nil {
		t.Fatal("a caller with no route must error, not panic")
	}
}

func TestDirectCallerSurfacesRouteError(t *testing.T) {
	rec := &recordingRoute{err: errors.New("nope")}
	c := NewDirectCaller(rec.route, app.NewSession())
	if _, err := c.Call("x", nil); err == nil {
		t.Fatal("route error must surface")
	}
}

func TestDispatchedCallerRunsOnDrainGoroutine(t *testing.T) {
	rec := &recordingRoute{reply: []byte(`{"drained":true}`)}
	d := dispatch.New(4)
	c := NewDispatchedCaller(rec.route, app.NewSession(), d, context.Background())

	// The call blocks until something drains the dispatcher (the "session goroutine").
	resCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		out, err := c.Call("model.tree", []byte(`{}`))
		resCh <- out
		errCh <- err
	}()
	waitForPending(t, d)
	d.Drain(0)

	if err := <-errCh; err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out := <-resCh; string(out) != `{"drained":true}` {
		t.Errorf("reply not forwarded: %s", out)
	}
	if rec.method != "model.tree" {
		t.Errorf("method = %q, want model.tree", rec.method)
	}
}

func TestDispatchedCallerCancelledContextErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := dispatch.New(1)
	c := NewDispatchedCaller((&recordingRoute{}).route, app.NewSession(), d, ctx)
	if _, err := c.Call("x", nil); err == nil {
		t.Fatal("a cancelled context must fail the submit")
	}
}

// waitForPending blocks until the dispatcher has a queued job (the worker reached
// Submit), so the test drains deterministically without sleeping a fixed interval.
func waitForPending(t *testing.T, d *dispatch.Dispatcher) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for d.Pending() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no job was submitted to the dispatcher")
		}
	}
}
