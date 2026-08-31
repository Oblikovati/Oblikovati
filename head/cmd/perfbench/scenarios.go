//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"time"

	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/perf/benchprof"
)

// activeAssemblyOrNil returns the active document's assembly content, erroring when there
// is none or it is not an assembly (the app-package helper is unexported).
func activeAssemblyOrNil(s *app.Session) (*compdef.AssemblyComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, fmt.Errorf("no active document")
	}
	asm, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return nil, fmt.Errorf("active document content is %T, not an assembly", d.Content())
	}
	return asm, nil
}

// ScenarioResult is the timing + memory for a one-shot scenario (cold load, UI stress,
// propagation). FirstFrameMs is only set for cold load.
type ScenarioResult struct {
	DurationMs   float64 `json:"durationMs"`
	FirstFrameMs float64 `json:"firstFrameMs,omitempty"`
	HeapMB       float64 `json:"heapMB"`
	AllocMB      float64 `json:"allocMB"`
	NumGC        uint32  `json:"numGC"`
	PauseMs      uint64  `json:"pauseMs"`
	PeakRSSMB    float64 `json:"peakRSSMB"`
}

// OrbitResult is the per-frame distribution of the 360° orbit, the headline render
// number.
type OrbitResult struct {
	Frames    int     `json:"frames"`
	MedianMs  float64 `json:"medianMs"`
	P95Ms     float64 `json:"p95Ms"`
	MaxMs     float64 `json:"maxMs"`
	MedianFPS float64 `json:"medianFPS"`
	HeapMB    float64 `json:"heapMB"`
	AllocMB   float64 `json:"allocMB"`
	NumGC     uint32  `json:"numGC"`
	PauseMs   uint64  `json:"pauseMs"`
}

// orbitScenario sweeps the camera a full turn around the framed assembly, rendering one
// frame per step through the real DrawChrome path and recording each frame's wall time —
// the ViewCube-orbit stress that exercises command-buffer recording and per-instance frustum
// culling (M34-F1: as the view turns, the off-screen part of the car is dropped before upload).
func orbitScenario(win *native.Window, s *app.Session, frames int, hoverpick bool) OrbitResult {
	if frames < 2 {
		frames = 2
	}
	prof, _ := benchprof.Start("orbit")
	yaw := 2 * stdmath.Pi / float64(frames)
	times := make([]float64, 0, frames)
	for i := 0; i < frames; i++ {
		s.SetCamera(s.Camera().Orbit(yaw, 0))
		t0 := time.Now()
		drawFrame(win, s)
		if hoverpick {
			frameHoverPick(s) // the live head's per-frame viewport-centre hover-pick (RayCastFaces over the scene)
		}
		times = append(times, msSince(t0))
	}
	sum, _ := prof.Stop()
	return orbitStats(times, sum)
}

// frameHoverPick reproduces the live head's per-frame hover-pick: a ray through the viewport centre
// against the scene. It is what turns an orbit slow when picking re-tessellates curved faces every
// frame; since M48/C3 the pick resolves against the analytic surfaces (no face tessellation), and
// measured here it guards that per-frame pick cost at scene scale. The camera pixel size is set by
// DrawChrome's updateViewportCamera on the frame just drawn.
func frameHoverPick(s *app.Session) {
	cam := s.Camera()
	if cam.Width <= 0 || cam.Height <= 0 {
		return
	}
	s.PickAt(float64(cam.Width)/2, float64(cam.Height)/2, app.NewSelectionFilter())
}

// uiStressScenario measures the per-frame model-browser tree the UI rebuilds every frame
// (app.BuildBrowser) — the O(N) Go allocation + timeline sort that runs regardless of how
// many rows are on screen (M34-F3, the no-clipper cost). It times iters builds.
func uiStressScenario(s *app.Session, iters int) ScenarioResult {
	if iters < 1 {
		iters = 1
	}
	prof, _ := benchprof.Start("uistress")
	t0 := time.Now()
	for i := 0; i < iters; i++ {
		_ = app.BuildBrowser(s)
	}
	res, _ := finishScenario(prof, t0)
	res.DurationMs /= float64(iters) // per-build cost
	res.AllocMB /= float64(iters)
	return res
}

// propagationScenario moves a top-level occurrence (the "engine block") and times the
// re-flatten that recomputes every descendant's global matrix — the parametric-edit
// settle the spec calls out.
func propagationScenario(s *app.Session) ScenarioResult {
	asm, err := activeAssemblyOrNil(s)
	if err != nil || asm.Occurrences().Count() == 0 {
		return ScenarioResult{}
	}
	occ := asm.Occurrences().Item(0)
	occ.SetTransform(occ.Transform().Mul(math.Translation4(math.V3(25, 0, 0))))
	prof, _ := benchprof.Start("propagation")
	t0 := time.Now()
	_ = s.VisibleInstances() // re-flatten = recompute descendant global matrices
	res, _ := finishScenario(prof, t0)
	return res
}

// drawFrame renders one full chrome frame (viewport + browser + ribbon), the same path
// the live app runs.
func drawFrame(win *native.Window, s *app.Session) {
	win.BeginFrame()
	ui.DrawChrome(win, s)
	win.EndFrame(ui.WindowClearColor())
}

// finishScenario stops the profiler and returns the elapsed wall time plus the memory
// summary as a ScenarioResult.
func finishScenario(prof *benchprof.Run, t0 time.Time) (ScenarioResult, error) {
	sum, err := prof.Stop()
	return ScenarioResult{
		DurationMs: msSince(t0),
		HeapMB:     mb(sum.HeapAllocBytes),
		AllocMB:    mb(sum.TotalAllocBytes),
		NumGC:      sum.NumGC,
		PauseMs:    sum.PauseTotalNs / 1e6,
		PeakRSSMB:  mb(sum.PeakRSSBytes),
	}, err
}

// orbitStats reduces per-frame times to the reported distribution.
func orbitStats(times []float64, sum benchprof.MemSummary) OrbitResult {
	median := math.Median(append([]float64(nil), times...))
	r := OrbitResult{
		Frames:   len(times),
		MedianMs: median,
		P95Ms:    math.Percentile(append([]float64(nil), times...), 0.95),
		MaxMs:    math.Percentile(append([]float64(nil), times...), 1),
		HeapMB:   mb(sum.HeapAllocBytes),
		AllocMB:  mb(sum.TotalAllocBytes) / float64(len(times)), // per-frame churn
		NumGC:    sum.NumGC,
		PauseMs:  sum.PauseTotalNs / 1e6,
	}
	if median > 0 {
		r.MedianFPS = 1000 / median
	}
	return r
}

// printSummary writes the human-readable report to stdout.
func printSummary(res Result) {
	fmt.Printf("perfbench %s (%s): %d placements, %d unique meshes\n",
		res.Profile, res.Source, res.LeafPlacements, res.UniqueMeshes)
	fmt.Printf("  cold load:   %.0f ms load, %.1f ms first frame, peakRSS %.0f MB\n",
		res.ColdLoad.DurationMs, res.ColdLoad.FirstFrameMs, res.ColdLoad.PeakRSSMB)
	fmt.Printf("  orbit:       median %.1f ms (%.0f fps), p95 %.1f ms, max %.1f ms over %d frames, %.0f MB/orbit\n",
		res.Orbit.MedianMs, res.Orbit.MedianFPS, res.Orbit.P95Ms, res.Orbit.MaxMs, res.Orbit.Frames, res.Orbit.AllocMB)
	fmt.Printf("  ui stress:   %.3f ms/build, %.2f MB/build\n", res.UIStress.DurationMs, res.UIStress.AllocMB)
	fmt.Printf("  propagation: %.1f ms re-flatten, %.0f MB\n", res.Propagation.DurationMs, res.Propagation.AllocMB)
}

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000 }
func mb(b uint64) float64         { return float64(b) / (1 << 20) }

// totalTransforms sums the per-group instance counts — the leaf placement total.
func totalTransforms(groups []app.InstanceGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Transforms)
	}
	return n
}
