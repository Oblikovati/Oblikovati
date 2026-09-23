// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"fmt"
	stdmath "math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Performance accountability for tessellation — the import→render hot path. boolean.TessellateBody runs in the
// viewport every time geometry changes (cached per geometry version), so a pathological mesher (e.g.
// routing faces through the O(n²) constrained-Delaunay, which once made ONE cone face take 2.4s and
// froze import) must be caught, not silently shipped. The benchmarks measure; the budget TESTS fail
// the suite on a catastrophic regression.

// perfFixtures are the committed OCC solids that exercise the analytic / sphere-cap-CDT / hole paths.
var perfFixtures = []string{
	"filleted_box", "sphere", "cone_frustum", "cone_sharp",
	"drilled_box", "torus", "partial_sphere", "chamfered_box",
}

// occBodies imports a committed OCC fixture's bodies once.
func occBodies(tb testing.TB, name string) []*topo.Body {
	return stepBodies(tb, filepath.Join("..", "..", "exchange", "step", "testdata", "occ", name+".step"))
}

// stepBodies imports a STEP file's bodies, skipping the caller if the file is unavailable.
func stepBodies(tb testing.TB, path string) []*topo.Body {
	tb.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Skipf("%s unavailable: %v", path, err)
		return nil
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		tb.Fatalf("import %s: %v", path, err)
	}
	return bodies
}

// BenchmarkTessellateBody measures full-body tessellation per committed fixture — the import-render hot
// path. Run: go test -bench=boolean.TessellateBody ./kernel/ops
func BenchmarkTessellateBody(b *testing.B) {
	for _, name := range perfFixtures {
		bodies := occBodies(b, name)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, body := range bodies {
					tessellate.TessellateBody(body, ops.DefaultQuality())
				}
			}
		})
	}
}

// BenchmarkChartFaceCovering measures the chart-driven mesher on ONE face at each covering size a chart
// region can have. A torus face wraps in BOTH parameters, so the covering replicates its points over
// 3 × 3 = 9 period shifts; a cylinder wall wraps in one, over 3. Every interior grid node's membership
// runs chartRegion.covers, which walks |shifts| × |contours| point-in-polygon tests, and nothing
// measured what the wrapping costs (#3527).
//
// The two faces are picked out of the corpus bodies the chart rows already build, so this measures the
// same geometry those rows gate rather than a fixture of its own.
//
// Baseline, amd64, -benchtime 200x: the torus band 7.26 ms/op, 11.18 MB/op, 69 655 allocs/op; the
// cylinder wall 5.00 ms/op, 5.31 MB/op, 26 676 allocs/op. Doubly periodic costs about 1.45× singly
// periodic here on 2.1× the allocation, which is the shape of a 9-shift covering against a 3-shift one
// — it is a baseline to compare against, not a budget: nothing fails on it.
func BenchmarkChartFaceCovering(b *testing.B) {
	for _, c := range []struct {
		name string
		face func(b *testing.B) *topo.Face
	}{
		{"torus band, 9 shifts", func(b *testing.B) *topo.Face {
			return faceOnSurfaceKind[geom.Torus](b, ringMinus(b, mustCylinder(b, math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8)))
		}},
		{"cylinder wall, 3 shifts", func(b *testing.B) *topo.Face {
			return faceOnSurfaceKind[geom.Cylinder](b, rodBall(b, ops.Join))
		}},
	} {
		f := c.face(b)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tessellate.TessellateFace(f, ops.DefaultQuality())
			}
		})
	}
}

// faceOnSurfaceKind returns the body's first face on a surface of kind S, failing the benchmark when the
// body has none — so a fixture that stops producing the face measures nothing rather than something else.
func faceOnSurfaceKind[S geom.Surface](b *testing.B, body *topo.Body) *topo.Face {
	b.Helper()
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(S); ok {
			return f
		}
	}
	b.Fatalf("the fixture body has no face on a %T surface", *new(S))
	return nil
}

// BenchmarkWholeBodyWeld measures the weld the tessellation post-condition runs over a body's whole
// vertex set (WeldedFreeEdgeCount → tornAcrossMeshes → weldAcrossMeshes), with allocations, because that
// weld concatenates every mesh's positions into ONE transient slice before gridding them: O(V) memory on
// top of the O(V) index it has to build anyway. It ran on every tessellated body and nothing measured it
// (#3527). A change that removes the transient — a two-pass extent then a per-mesh walk — shows here as
// bytes per operation, which is the only way to tell it apart from noise.
//
// Baseline, amd64, -benchtime 200x: torus 981 µs/op and 1.69 MB/op, filleted_box 336 µs/op and
// 666 kB/op, drilled_box 133 µs/op and 213 kB/op. The transient copy is one of the two O(V) allocations
// inside those figures; how much of them it is has not been attributed, and this row is where a change
// that removes it would show.
func BenchmarkWholeBodyWeld(b *testing.B) {
	for _, name := range []string{"torus", "filleted_box", "drilled_box"} {
		var meshes []*tessellate.Mesh
		for _, body := range occBodies(b, name) {
			m, _ := tessellate.TessellateBody(body, ops.DefaultQuality())
			meshes = append(meshes, m)
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, m := range meshes {
					tessellate.WeldedFreeEdgeCount(m)
				}
			}
		})
	}
}

// BenchmarkConstrainedDelaunay measures the CDT at increasing boundary (constraint) + total sizes — to
// catch any super-linear regression in its scaling, the failure mode behind the import freeze.
func BenchmarkConstrainedDelaunay(b *testing.B) {
	for _, n := range []int{32, 64, 128, 256} {
		pts, loops := circleWithInteriorGrid(n)
		b.Run(fmt.Sprintf("boundary%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tessellate.ConstrainedDelaunay(pts, loops)
			}
		})
	}
}

// circleWithInteriorGrid builds a CDT input: an n-point circle boundary (n hard constraints) plus an
// interior point grid — a stand-in for the dense trimmed-wall inputs the mesher feeds the CDT.
func circleWithInteriorGrid(n int) (pts [][2]float64, loops [][]int) {
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		pts = append(pts, [2]float64{50 * stdmath.Cos(a), 50 * stdmath.Sin(a)})
	}
	loop := make([]int, n)
	for i := range loop {
		loop[i] = i
	}
	for y := -40; y <= 40; y += 8 {
		for x := -40; x <= 40; x += 8 {
			pts = append(pts, [2]float64{float64(x), float64(y)})
		}
	}
	return pts, [][]int{loop}
}

// TestTessellationBudget is the accountability guard: tessellating every committed OCC fixture must stay
// WELL under a generous wall-clock budget (~40× the tens-of-ms baseline). A pathological mesher
// regression that affects these shapes fails the suite instead of degrading import silently. Generous
// so it never flakes on a slow CI box — it catches catastrophic (10×+) regressions; the benchmarks
// track micro ones. The heavy real model is guarded separately (see TestHeavyModelBudget).
func TestTessellationBudget(t *testing.T) {
	// 2.15s, not 2s: the race-detector CI box runs close to the old 2s ceiling and flaked
	// often (~2.1s actuals), so add 150ms of headroom. This still catches catastrophic
	// (10×+) regressions; the benchmarks track micro ones.
	const budget = 2150 * time.Millisecond
	bodies := map[string][]*topo.Body{}
	for _, name := range perfFixtures {
		bodies[name] = occBodies(t, name)
	}
	start := time.Now()
	for _, bs := range bodies {
		for _, body := range bs {
			tessellate.TessellateBody(body, ops.DefaultQuality())
		}
	}
	if d := time.Since(start); d > budget {
		t.Errorf("tessellating the OCC fixtures took %v; budget %v — a mesher perf regression?", d, budget)
	}
}

// TestHeavyModelBudget guards the heavy REAL model named by OBK_PERF_STEP (e.g. an imported EDF duct:
// trimmed cones + freeform NURBS) — the case the committed fixtures don't reach, and exactly where the
// O(n²) CDT regression bit (50 ms → 3.5 s).
//
// It is therefore the ONLY row that gates the NURBS path's cost, and it is opt-in, so anything that
// makes a freeform face more expensive is invisible to the gate. The achieved-chord report of #3517
// (face_chord_achieved.go) is one such thing: it costs one surface point inversion per mesh edge, which
// on an analytic face is a few Gauss-Newton steps and on a NURBS face is the expensive one — measured
// per face, occtparity J3 face 3 went 13 ms → 808 ms and K2 face 4 26 ms → 2.46 s, 50–95×. Here, on the
// EDF duct, the same change reads 0.34 s → 0.41 s against this 700 ms budget. If a future change moves
// that per-face figure again, THIS is the row that sees it, and the committed fixtures will not.
//
// Skipped when OBK_PERF_STEP is unset, so set it locally:
//
//	OBK_PERF_STEP=/path/EDF.STEP go test -run TestHeavyModelBudget ./kernel/ops
func TestHeavyModelBudget(t *testing.T) {
	path := os.Getenv("OBK_PERF_STEP")
	if path == "" {
		t.Skip("set OBK_PERF_STEP=/path/model.step to run the heavy-model tessellation budget")
	}
	bodies := stepBodies(t, path)
	const budget = 700 * time.Millisecond // ~14× the ~50 ms baseline; the metricWall freeze was 3.5 s
	start := time.Now()
	for _, body := range bodies {
		tessellate.TessellateBody(body, ops.DefaultQuality())
	}
	if d := time.Since(start); d > budget {
		t.Errorf("tessellating %s took %v; budget %v — a mesher perf regression?", path, d, budget)
	}
}
