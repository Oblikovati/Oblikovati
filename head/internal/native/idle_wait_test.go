//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"
	"time"
)

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
	for i := 0; i < 3; i++ {
		w.WaitEvents(0.01)
	}

	// Sum several idle waits: under xvfb no spontaneous events arrive, so blocking waits total
	// ~N*d, while a non-blocking poll would total ~0. The lower bound (2*d) sits well above
	// timing noise yet far under the real N*d, tolerating an occasional stray early return.
	const d = 0.1
	const n = 5
	start := time.Now()
	for i := 0; i < n; i++ {
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
	for i := 0; i < 3; i++ {
		w.WaitEvents(0.01) // drain startup events so the blocking wait below starts on an empty queue
	}

	const postAfter = 150 * time.Millisecond
	go func() {
		time.Sleep(postAfter)
		PostEmptyEvent()
	}()
	// Backstop so a broken wake never hangs the suite: a far-later post forces a return, and the
	// upper-bound assertion below then fails cleanly instead of the test timing out.
	go func() {
		time.Sleep(3 * time.Second)
		PostEmptyEvent()
	}()

	start := time.Now()
	w.WaitEventsBlocking()
	elapsed := time.Since(start)

	if elapsed < postAfter/2 {
		t.Errorf("WaitEventsBlocking returned after %v, want >= %v — it did not block (busy return? #1493)",
			elapsed, postAfter/2)
	}
	if elapsed > 2*time.Second {
		t.Errorf("WaitEventsBlocking returned after %v, want < 2s — PostEmptyEvent did not wake it (#1493)", elapsed)
	}
}
