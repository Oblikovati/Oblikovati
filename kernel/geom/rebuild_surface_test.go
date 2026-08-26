// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// wavyMultiSpanSurface is a gently undulating, low-curvature height field
// z = 0.15·sin(½πx)·sin(½πy) carried on a fine interpolating net (many spans) — the kind of
// over-defined surface Rebuild cleans up to far fewer control points.
func wavyMultiSpanSurface(t *testing.T) BSplineSurface {
	t.Helper()
	const n = 12
	ctrl := make([][]math.Point3, n+1)
	w := make([][]float64, n+1)
	for i := 0; i <= n; i++ {
		ctrl[i] = make([]math.Point3, n+1)
		w[i] = make([]float64, n+1)
		x := float64(i) / n
		for j := 0; j <= n; j++ {
			y := float64(j) / n
			ctrl[i][j] = math.P3(x, y, 0.15*stdmath.Sin(0.5*stdmath.Pi*x)*stdmath.Sin(0.5*stdmath.Pi*y))
			w[i][j] = 1
		}
	}
	uk := clampedUniformKnots(n, 3)
	s, err := NewBSplineSurface(3, 3, ctrl, w, uk, clampedUniformKnots(n, 3))
	if err != nil {
		t.Fatalf("wavy surface: %v", err)
	}
	return s
}

// clampedUniformKnots returns a clamped degree-p knot vector with evenly spaced interiors for
// nctrl = n+1 control points.
func clampedUniformKnots(n, p int) []float64 {
	nctrl := n + 1
	knots := make([]float64, nctrl+p+1)
	interior := nctrl - p - 1
	for i := 0; i < p+1; i++ {
		knots[i] = 0
	}
	for j := 1; j <= interior; j++ {
		knots[p+j] = float64(j) / float64(interior+1)
	}
	for i := nctrl; i < nctrl+p+1; i++ {
		knots[i] = 1
	}
	return knots
}

// bruteForceSurfaceDeviation is an INDEPENDENT same-parameter oracle over a finer grid.
func bruteForceSurfaceDeviation(t *testing.T, src, rebuilt BSplineSurface) float64 {
	t.Helper()
	maxDev := 0.0
	for i := 0; i <= 60; i++ {
		f := float64(i) / 60
		for j := 0; j <= 60; j++ {
			g := float64(j) / 60
			d := float64(src.PointAt(f, g).DistanceTo(rebuilt.PointAt(f, g)))
			if d > maxDev {
				maxDev = d
			}
		}
	}
	return maxDev
}

func TestRebuildSurfaceMultiSpanToSingleSpanExact(t *testing.T) {
	// A genuinely single-span bicubic patch is artificially carried on many spans by inserting
	// knots (F01) — same geometry, 7x7 control points. Rebuild must collapse it back to a clean
	// 4x4 single span near-exactly: the canonical "wavy multi-span -> single span" acceptance.
	single := bicubicPatch(t)
	multi := single
	for _, u := range []float64{0.25, 0.5, 0.75} {
		var err error
		if multi, err = multi.InsertKnotU(u, 1); err != nil {
			t.Fatalf("InsertKnotU: %v", err)
		}
	}
	for _, v := range []float64{0.25, 0.5, 0.75} {
		var err error
		if multi, err = multi.InsertKnotV(v, 1); err != nil {
			t.Fatalf("InsertKnotV: %v", err)
		}
	}
	if len(multi.Ctrl) != 7 || len(multi.Ctrl[0]) != 7 {
		t.Fatalf("multi-span net = %dx%d, want 7x7", len(multi.Ctrl), len(multi.Ctrl[0]))
	}
	rebuilt, dev, err := RebuildSurface(multi, 3, 3, 4, 4, 0)
	if err != nil {
		t.Fatalf("RebuildSurface: %v", err)
	}
	if len(rebuilt.Ctrl) != 4 || len(rebuilt.Ctrl[0]) != 4 {
		t.Errorf("rebuilt net = %dx%d, want a 4x4 single span", len(rebuilt.Ctrl), len(rebuilt.Ctrl[0]))
	}
	if dev > 1e-6 {
		t.Errorf("collapsing a multi-span bicubic to a single span should be near-exact, dev=%g", dev)
	}
}

func TestRebuildSurfaceReducesCVsWithinTolerance(t *testing.T) {
	src := wavyMultiSpanSurface(t) // 13x13 = 169 control points
	rebuilt, dev, err := RebuildSurface(src, 3, 3, 10, 10, 0)
	if err != nil {
		t.Fatalf("RebuildSurface: %v", err)
	}
	if len(rebuilt.Ctrl) != 10 || len(rebuilt.Ctrl[0]) != 10 {
		t.Errorf("net = %dx%d, want 10x10", len(rebuilt.Ctrl), len(rebuilt.Ctrl[0]))
	}
	if len(rebuilt.Ctrl)*len(rebuilt.Ctrl[0]) >= len(src.Ctrl)*len(src.Ctrl[0]) {
		t.Errorf("rebuild should use fewer CVs than source %dx%d", len(src.Ctrl), len(src.Ctrl[0]))
	}
	if dev > 3e-3 {
		t.Errorf("deviation %g too large for the undulation rebuilt to 10x10 cubic", dev)
	}
}

func TestRebuildSurfaceDeviationMatchesOracle(t *testing.T) {
	src := wavyMultiSpanSurface(t)
	rebuilt, dev, err := RebuildSurface(src, 3, 3, 6, 6, 0)
	if err != nil {
		t.Fatalf("RebuildSurface: %v", err)
	}
	oracle := bruteForceSurfaceDeviation(t, src, rebuilt)
	if stdmath.Abs(dev-oracle) > 1e-3 {
		t.Errorf("reported deviation %g disagrees with oracle %g", dev, oracle)
	}
}

// bicubicPatch is a single-span (4x4) bicubic Bézier surface with a saddle-ish bump.
func bicubicPatch(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := range 4 {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := range 4 {
			ctrl[i][j] = math.P3(float64(i), float64(j), float64((i-1)*(j-1))*0.3)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	s, err := NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("bicubic patch: %v", err)
	}
	return s
}

func TestRebuildSurfaceIdempotentOnClean(t *testing.T) {
	// A single-span bicubic Bézier patch rebuilt to the same 4x4 target is near-exact.
	src := bicubicPatch(t)
	_, dev, err := RebuildSurface(src, 3, 3, 4, 4, 0)
	if err != nil {
		t.Fatalf("RebuildSurface: %v", err)
	}
	if dev > 1e-7 {
		t.Errorf("rebuild of an already-clean bicubic patch should be idempotent, dev=%g", dev)
	}
}

func TestRebuildSurfaceValidates(t *testing.T) {
	src := wavyMultiSpanSurface(t)
	if _, _, err := RebuildSurface(src, 3, 3, 2, 5, 0); err == nil {
		t.Error("nu < uDeg+1 should error")
	}
}
