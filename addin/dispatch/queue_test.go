// SPDX-License-Identifier: GPL-2.0-only

package dispatch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor polls cond until true or the deadline, failing the test on timeout. Used
// to observe queue state without sleeping a fixed (flaky) duration.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition not met within deadline: %s", msg)
}

func TestSubmitDrainRoundTrip(t *testing.T) {
	d := New(4)
	type res struct {
		out []byte
		err error
	}
	got := make(chan res, 1)
	go func() {
		out, err := d.Submit(context.Background(), func() ([]byte, error) {
			return []byte("hello"), nil
		})
		got <- res{out, err}
	}()
	waitFor(t, func() bool { return d.Pending() == 1 }, "job queued")
	if ran := d.Drain(0); ran != 1 {
		t.Fatalf("Drain ran %d jobs, want 1", ran)
	}
	r := <-got
	if r.err != nil || string(r.out) != "hello" {
		t.Fatalf("Submit returned (%q, %v), want (\"hello\", nil)", r.out, r.err)
	}
}

func TestSubmitBlocksUntilDrain(t *testing.T) {
	d := New(4)
	returned := make(chan struct{})
	go func() {
		_, _ = d.Submit(context.Background(), func() ([]byte, error) { return nil, nil })
		close(returned)
	}()
	waitFor(t, func() bool { return d.Pending() == 1 }, "job queued")
	select {
	case <-returned:
		t.Fatal("Submit returned before Drain ran the job")
	case <-time.After(20 * time.Millisecond):
	}
	d.Drain(0)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Submit did not return after Drain")
	}
}

func TestConcurrentSubmitsEachGetOwnResult(t *testing.T) {
	d := New(64)
	const n = 50
	var wg sync.WaitGroup
	mismatch := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, err := d.Submit(context.Background(), func() ([]byte, error) {
				return []byte{byte(i)}, nil
			})
			if err != nil || len(out) != 1 || out[0] != byte(i) {
				mismatch <- "wrong result for job"
			}
		}(i)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for {
		d.Drain(0)
		select {
		case <-done:
			close(mismatch)
			if m, ok := <-mismatch; ok {
				t.Fatal(m)
			}
			return
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestSubmitContextCancel(t *testing.T) {
	d := New(4)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := d.Submit(ctx, func() ([]byte, error) { return nil, nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit err = %v, want context.Canceled", err)
	}
}

func TestDrainMaxLimit(t *testing.T) {
	d := New(8)
	for i := 0; i < 3; i++ {
		go func() { _, _ = d.Submit(context.Background(), func() ([]byte, error) { return nil, nil }) }()
	}
	waitFor(t, func() bool { return d.Pending() == 3 }, "three jobs queued")
	if ran := d.Drain(2); ran != 2 {
		t.Fatalf("Drain(2) ran %d, want 2", ran)
	}
	waitFor(t, func() bool { return d.Pending() == 1 }, "one job remaining")
	if ran := d.Drain(0); ran != 1 {
		t.Fatalf("Drain(0) ran %d, want 1", ran)
	}
}

func TestSubmitAfterClose(t *testing.T) {
	d := New(4)
	d.Close()
	d.Close() // idempotent
	if _, err := d.Submit(context.Background(), func() ([]byte, error) { return nil, nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit after Close err = %v, want ErrClosed", err)
	}
}

// TestWakeupFiresOnEnqueue pins the #1493 wake path: the head's render-on-demand loop sleeps
// when idle, so a job an add-in submits from another goroutine must rouse the consumer or it
// would not run until the next OS input event. SetWakeup's callback must fire when (and only
// when) a job is enqueued — before it drains — so the head can post a window wake.
func TestWakeupFiresOnEnqueue(t *testing.T) {
	d := New(4)
	var woke int32
	d.SetWakeup(func() { atomic.AddInt32(&woke, 1) })

	if got := atomic.LoadInt32(&woke); got != 0 {
		t.Fatalf("wakeup fired %d times before any Submit, want 0", got)
	}
	go func() { _, _ = d.Submit(context.Background(), func() ([]byte, error) { return nil, nil }) }()
	waitFor(t, func() bool { return atomic.LoadInt32(&woke) == 1 }, "wakeup fired once on enqueue")
	d.Drain(0)
	if got := atomic.LoadInt32(&woke); got != 1 {
		t.Fatalf("wakeup fired %d times, want exactly 1 (one enqueue)", got)
	}
}

// TestWakeupOptional confirms a dispatcher with no wakeup registered still works (the head is
// the only caller that sets one; tests and the bridge do not).
func TestWakeupOptional(t *testing.T) {
	d := New(4)
	got := make(chan []byte, 1)
	go func() {
		out, _ := d.Submit(context.Background(), func() ([]byte, error) { return []byte("ok"), nil })
		got <- out
	}()
	waitFor(t, func() bool { return d.Pending() == 1 }, "job queued without a wakeup")
	d.Drain(0)
	if string(<-got) != "ok" {
		t.Fatal("Submit without a wakeup did not run the job")
	}
}

func TestJobErrorPropagates(t *testing.T) {
	d := New(4)
	want := errors.New("boom")
	got := make(chan error, 1)
	go func() {
		_, err := d.Submit(context.Background(), func() ([]byte, error) { return nil, want })
		got <- err
	}()
	waitFor(t, func() bool { return d.Pending() == 1 }, "job queued")
	d.Drain(0)
	if err := <-got; !errors.Is(err, want) {
		t.Fatalf("Submit err = %v, want %v", err, want)
	}
}
