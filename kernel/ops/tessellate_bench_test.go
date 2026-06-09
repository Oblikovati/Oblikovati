// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
)

// Performance accountability for tessellation — the import→render hot path. TessellateBody runs in the
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
	return stepBodies(tb, filepath.Join("..", "exchange", "step", "testdata", "occ", name+".step"))
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
// path. Run: go test -bench=TessellateBody ./kernel/ops
func BenchmarkTessellateBody(b *testing.B) {
	for _, name := range perfFixtures {
		bodies := occBodies(b, name)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, body := range bodies {
					TessellateBody(body, DefaultQuality())
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
				constrainedDelaunay(pts, loops)
			}
		})
	}
}

// circleWithInteriorGrid builds a CDT input: an n-point circle boundary (n hard constraints) plus an
// interior point grid — a stand-in for the dense trimmed-wall inputs the mesher feeds the CDT.
func circleWithInteriorGrid(n int) (pts [][2]float64, loops [][]int) {
	for i := 0; i < n; i++ {
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
	const budget = 2 * time.Second
	bodies := map[string][]*topo.Body{}
	for _, name := range perfFixtures {
		bodies[name] = occBodies(t, name)
	}
	start := time.Now()
	for _, bs := range bodies {
		for _, body := range bs {
			TessellateBody(body, DefaultQuality())
		}
	}
	if d := time.Since(start); d > budget {
		t.Errorf("tessellating the OCC fixtures took %v; budget %v — a mesher perf regression?", d, budget)
	}
}

// TestHeavyModelBudget guards the heavy REAL model named by OBK_PERF_STEP (e.g. an imported EDF duct:
// trimmed cones + freeform NURBS) — the case the committed fixtures don't reach, and exactly where the
// O(n²) CDT regression bit (50 ms → 3.5 s). Skipped when OBK_PERF_STEP is unset, so set it locally:
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
		TessellateBody(body, DefaultQuality())
	}
	if d := time.Since(start); d > budget {
		t.Errorf("tessellating %s took %v; budget %v — a mesher perf regression?", path, d, budget)
	}
}
