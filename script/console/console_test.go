// SPDX-License-Identifier: GPL-2.0-only

package console_test

import (
	"context"
	"testing"
	"time"

	"oblikovati.org/script"
	"oblikovati.org/script/console"
	"oblikovati.org/script/runner"
)

// scriptedEngine is a fake script.Engine whose behaviour is the injected fn, so a
// console test can drive print()/result/blocking/cancellation deterministically without
// the real Lua VM (no cgo, fast).
type scriptedEngine struct {
	fn func(ctx context.Context, g script.Globals) script.Result
}

func (e *scriptedEngine) Run(ctx context.Context, _ string, g script.Globals, _ script.Limits) script.Result {
	return e.fn(ctx, g)
}

// nopCaller is a no-op client.Caller; the console tests never reach the host, so the
// engine fn just ignores g.Call.
type nopCaller struct{}

func (nopCaller) Call(string, []byte) ([]byte, error) { return []byte("{}"), nil }

// newTestController builds a Controller over a scripted engine with unbounded limits.
func newTestController(fn func(ctx context.Context, g script.Globals) script.Result) *console.Controller {
	r := runner.New(&scriptedEngine{fn: fn}, nopCaller{}, nil)
	return console.NewController(r, script.Limits{})
}

// waitUntil polls cond until true or a short deadline (the run is async). FIRST: fast
// and self-validating.
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestControllerStreamsOutputAndRecordsResult(t *testing.T) {
	c := newTestController(func(_ context.Context, g script.Globals) script.Result {
		g.Print("hello")
		g.Print("world")
		return script.Result{Stdout: "hello\nworld\n"}
	})
	if err := c.Run("print('hi')"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	waitUntil(t, func() bool { return !c.Console().Running() })

	snap := c.Console().Snapshot()
	if !snap.HasLast || snap.Last.Err != nil {
		t.Fatalf("expected a successful last result, got %+v", snap)
	}
	if len(snap.Output) != 2 || snap.Output[0] != "hello" || snap.Output[1] != "world" {
		t.Fatalf("output = %v, want [hello world]", snap.Output)
	}
}

// TestControllerRejectsConcurrentRun: a second Run while one is in flight is refused
// with ErrBusy (so the "Run" button stays disabled mid-run).
func TestControllerRejectsConcurrentRun(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	c := newTestController(func(_ context.Context, _ script.Globals) script.Result {
		close(started)
		<-release
		return script.Result{}
	})
	if err := c.Run("blocking"); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	<-started
	if err := c.Run("second"); err != runner.ErrBusy {
		t.Fatalf("second Run error = %v, want ErrBusy", err)
	}
	close(release)
	waitUntil(t, func() bool { return !c.Console().Running() })
}

// TestControllerStopCancelsRun: Stop cancels the run's context, unwinding a script that
// waits on ctx, and the cancellation surfaces as the recorded result error.
func TestControllerStopCancelsRun(t *testing.T) {
	c := newTestController(func(ctx context.Context, _ script.Globals) script.Result {
		<-ctx.Done()
		return script.Result{Err: ctx.Err()}
	})
	if err := c.Run("loop"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	waitUntil(t, func() bool { return c.Console().Running() })
	c.Stop()
	waitUntil(t, func() bool { return !c.Console().Running() })

	if snap := c.Console().Snapshot(); snap.Last.Err == nil {
		t.Fatalf("stopped run should record a cancellation error, got %+v", snap.Last)
	}
}

func TestConsoleClearEmptiesOutput(t *testing.T) {
	co := console.New()
	co.AppendOutput("a")
	co.AppendOutput("b")
	co.Clear()
	if snap := co.Snapshot(); len(snap.Output) != 0 {
		t.Fatalf("after Clear, output = %v, want empty", snap.Output)
	}
}

// TestConsoleSnapshotIsCopy: mutating a returned snapshot must not corrupt the console.
func TestConsoleSnapshotIsCopy(t *testing.T) {
	co := console.New()
	co.AppendOutput("keep")
	snap := co.Snapshot()
	snap.Output[0] = "tampered"
	if got := co.Snapshot().Output[0]; got != "keep" {
		t.Fatalf("snapshot is not a copy: console output became %q", got)
	}
}

// TestConsoleCapsOutput: a flood of lines is bounded — the buffer drops the oldest and
// keeps the most recent (the runaway-print guard).
func TestConsoleCapsOutput(t *testing.T) {
	co := console.New()
	const flood = 6000
	for i := 0; i < flood; i++ {
		co.AppendOutput("line")
	}
	co.AppendOutput("last")
	snap := co.Snapshot()
	if len(snap.Output) >= flood {
		t.Fatalf("output not capped: len = %d", len(snap.Output))
	}
	if snap.Output[len(snap.Output)-1] != "last" {
		t.Fatalf("most recent line dropped: tail = %q", snap.Output[len(snap.Output)-1])
	}
}
