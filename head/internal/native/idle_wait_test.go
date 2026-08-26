//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"
	"time"
)

// drainUntilQuiet consumes the window-creation events (show/focus/refresh) the server queues after
// CreateWindow, returning once the queue has genuinely gone quiet: it waits in short slices until one
// slice runs its FULL timeout without an event cutting it short.
//
// A fixed number of drain calls cannot establish that. WaitEvents returns as soon as an event
// arrives, so a bounded drain budget only proves that no event arrived DURING it — the server is free
// to deliver a creation event afterwards, and the wait under test then returns on that stale event
// instead of on the one the test posted. That is not a hypothetical: TestPostEmptyEventWakesBlockingWait
// began failing on CI with head/internal/native byte-identical to its last green run (a runner-image
// change moved the startup timing), reporting returns of 119µs and 15ms against a 75ms floor.
//
// The slice/2 threshold separates the two outcomes with wide margin: a timeout wait returns at ~slice,
// an event-interrupted one returns in microseconds to a few ms.
func drainUntilQuiet(t *testing.T, w *Window) {
	t.Helper()
	const slice = 20 * time.Millisecond
	for range 50 { // bounded so a pathologically chatty server cannot hang the test
		start := time.Now()
		w.WaitEvents(slice.Seconds())
		if time.Since(start) >= slice/2 {
			return // the wait ran its timeout rather than being cut short — nothing left queued
		}
	}
	t.Log("event queue never went quiet after 50 drain slices; the timing assertions below may be flaky")
}

// TestWaitEventsBlocksWhenIdle pins the #1493 fix: the interactive loop renders on demand by
// sleeping in WaitEvents between frames instead of spinning at vsync. On a software Vulkan
// rasterizer (a VM's llvmpipe) every frame is drawn on the CPU, so an idle window that
// re-rendered each vsync pegged the cores. The behavioural guarantee is that WaitEvents
// actually BLOCKS for ~the timeout when no input arrives (a regression to a non-blocking poll
// would return instantly and reintroduce the spin), while still respecting the timeout so the
// loop never hangs.
func TestWaitEventsBlocksWhenIdle(t *testing.T) {
	w, err := CreateWindow(320, 240, "Oblikovati (#1493 idle-wait test)")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer w.Destroy()

	// Drain the queue of window-creation events (show/focus/refresh) so the timed calls below
	// sleep on a genuinely empty queue rather than returning early on a startup event.
	w.BeginFrame()
	w.EndFrame(0.1, 0.1, 0.12)
	drainUntilQuiet(t, w)

	// Sum several idle waits: under xvfb no spontaneous events arrive, so blocking waits total
	// ~N*d, while a non-blocking poll would total ~0. The lower bound (2*d) sits well above
	// timing noise yet far under the real N*d, tolerating an occasional stray early return.
	const d = 0.1
	const n = 5
	start := time.Now()
	for range n {
		w.WaitEvents(d)
	}
	elapsed := time.Since(start).Seconds()

	if elapsed < 2*d {
		t.Errorf("%d idle WaitEvents(%.2fs) took %.3fs total; want >= %.2fs — the loop is not "+
			"blocking when idle (reverted to a busy poll? #1493)", n, d, elapsed, 2*d)
	}
	if elapsed > n*d+1.0 {
		t.Errorf("%d idle WaitEvents(%.2fs) took %.3fs total; want < %.2fs — WaitEvents is not "+
			"respecting its timeout (the loop could hang)", n, d, elapsed, n*d+1.0)
	}
}

// TestPostEmptyEventWakesBlockingWait pins the #1493 wake path: when fully idle the loop blocks
// indefinitely in WaitEventsBlocking, so a background producer (an add-in submitting work, a
// finished update check) MUST be able to rouse it from another goroutine via PostEmptyEvent —
// otherwise its change would not render until the user moved the mouse. The wait must block
// (not return instantly) yet return promptly once the event is posted.
func TestPostEmptyEventWakesBlockingWait(t *testing.T) {
	w, err := CreateWindow(320, 240, "Oblikovati (#1493 wake test)")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer w.Destroy()

	w.BeginFrame()
	w.EndFrame(0.1, 0.1, 0.12)

	// RETRY, because "the queue is empty" is not something a test can establish once and rely on.
	// A server-delivered startup event landing after the drain cuts the blocking wait short, and
	// that early return says nothing about the wake path — it is the queue's noise, not a busy
	// return. So each attempt re-drains, posts, and measures; an attempt that was cut short is
	// discarded and retried rather than reported as a failure. Only a wait that genuinely blocked
	// is judged. (Without this the test failed on CI at 119µs and 15ms against the 75ms floor, with
	// head/internal/native byte-identical to its last green run — a runner-image change alone.)
	const postAfter = 150 * time.Millisecond
	var elapsed time.Duration
	for attempt := range 5 {
		drainUntilQuiet(t, w)
		posted := make(chan struct{})
		go func() {
			time.Sleep(postAfter)
			PostEmptyEvent()
			close(posted)
		}()
		// Backstop so a broken wake never hangs the suite: a far-later post forces a return, and the
		// upper-bound assertion below then fails cleanly instead of the test timing out.
		go func() {
			time.Sleep(3 * time.Second)
			PostEmptyEvent()
		}()

		start := time.Now()
		w.WaitEventsBlocking()
		elapsed = time.Since(start)
		<-posted // let this attempt's post land so the next attempt's drain can absorb it

		if elapsed >= postAfter/2 {
			break // this wait actually blocked — it is the one worth judging
		}
		t.Logf("attempt %d returned in %v (a queued event, not the posted one); retrying", attempt, elapsed)
	}

	if elapsed < postAfter/2 {
		t.Errorf("WaitEventsBlocking returned after %v on every attempt, want >= %v — it did not block "+
			"(busy return? #1493)", elapsed, postAfter/2)
	}
	if elapsed > 2*time.Second {
		t.Errorf("WaitEventsBlocking returned after %v, want < 2s — PostEmptyEvent did not wake it (#1493)", elapsed)
	}
}
