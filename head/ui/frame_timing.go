//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"
	"time"
)

// Frame-phase timing, enabled by OBK_FRAME_TIMING (a diagnostic for orbit-perf investigation). It
// reports, throttled to ~once a second, the wall time of the single-view phases: pick (the hover
// PickAt in updateViewportCamera), draw-list assembly, the instanced mesh build, and the native
// GPU submit. Disabled (zero overhead) when the env var is unset.
var frameTimingOn = os.Getenv("OBK_FRAME_TIMING") != ""

var frameStats struct {
	n              int
	buildNs, gpuNs int64 // set by renderViewportImage for the current frame
	browserNs      int64 // set by recordBrowser for the current frame (model-tree walk cost)
}

// recordBrowser stores the wall time the model browser took to build+emit this frame
// (the no-clipper O(N) tree walk, M34-F3). start comes from frameClock, so this is a
// no-op when timing is off.
func recordBrowser(start time.Time) {
	if frameTimingOn {
		frameStats.browserNs = time.Since(start).Nanoseconds()
	}
}

// frameClock returns now when timing is on, else the zero time (so the arithmetic is cheap and the
// disabled path allocates nothing).
func frameClock() time.Time {
	if frameTimingOn {
		return time.Now()
	}
	return time.Time{}
}

// frameTiming logs the per-phase breakdown once every ~60 frames. t0..t2,t3 bracket pick, draw-list,
// and (pick→list, list→render-call). The build/GPU split inside the render call is recorded by
// renderViewportImage into frameStats.
func frameTiming(t0, t1, t2, t3 time.Time) {
	if !frameTimingOn {
		return
	}
	frameStats.n++
	if frameStats.n%60 != 0 {
		return
	}
	pick := t1.Sub(t0)
	list := t2.Sub(t1)
	render := t3.Sub(t2)
	fmt.Fprintf(os.Stderr, "[frame %d] pick=%v drawlist=%v render=%v (build=%v gpu=%v) browser=%v\n",
		frameStats.n, pick.Round(time.Microsecond), list.Round(time.Microsecond), render.Round(time.Microsecond),
		time.Duration(frameStats.buildNs).Round(time.Microsecond), time.Duration(frameStats.gpuNs).Round(time.Microsecond),
		time.Duration(frameStats.browserNs).Round(time.Microsecond))
}
