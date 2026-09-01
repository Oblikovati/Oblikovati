// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// flatGrid is a planar n×n bicubic-capable B-spline at z=0 over [0,1]² with evenly spaced CVs.
func flatGrid(t *testing.T, n int) BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(float64(i)/float64(n-1), float64(j)/float64(n-1), 0)
			w[i][j] = 1
		}
	}
	k := clampedUniformKnots(n-1, 3)
	s, err := NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("flat grid: %v", err)
	}
	return s
}

func TestDisplaceControlPointsMovesLimitSurfaceAndKeepsStructure(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	before := s.PointAt(0.5, 0.5)
	moved, err := s.DisplaceControlPoints([]ControlPointDelta{{U: 2, V: 2, Delta: math.V3(0, 0, 1)}})
	if err != nil {
		t.Fatalf("DisplaceControlPoints: %v", err)
	}
	// Degree and knot vectors are untouched (the Class-A invariant).
	if moved.UDegree != s.UDegree || moved.VDegree != s.VDegree {
		t.Errorf("degree changed: %dx%d, want %dx%d", moved.UDegree, moved.VDegree, s.UDegree, s.VDegree)
	}
	if len(moved.UKnots) != len(s.UKnots) || len(moved.VKnots) != len(s.VKnots) {
		t.Errorf("knot vector length changed")
	}
	for i := range s.UKnots {
		if moved.UKnots[i] != s.UKnots[i] {
			t.Fatalf("U knot %d changed: %g vs %g", i, moved.UKnots[i], s.UKnots[i])
		}
	}
	// The limit surface rose toward the lifted centre CV.
	after := moved.PointAt(0.5, 0.5)
	if after.Z <= before.Z+0.1 {
		t.Errorf("limit surface barely moved: z %g -> %g", before.Z, after.Z)
	}
	// A corner far from the moved CV is essentially unaffected.
	if c := moved.PointAt(0, 0); c.Z > 1e-9 {
		t.Errorf("corner moved by %g, expected ~0", c.Z)
	}
}

func TestDisplaceControlPointsRejectsOutOfRange(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	if _, err := s.DisplaceControlPoints([]ControlPointDelta{{U: 5, V: 0, Delta: math.V3(0, 0, 1)}}); err == nil {
		t.Error("out-of-range U index should error")
	}
	if _, err := s.DisplaceControlPoints([]ControlPointDelta{{U: 0, V: -1, Delta: math.V3(0, 0, 1)}}); err == nil {
		t.Error("negative V index should error")
	}
}

func TestFalloffDeltasSingleDriver(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	// radius 0 → only the driver moves.
	d := s.FalloffDeltas([][2]int{{2, 2}}, math.V3(0, 0, 1), 0, FalloffSmooth)
	if len(d) != 1 || d[0].U != 2 || d[0].V != 2 {
		t.Fatalf("radius-0 falloff should move only the driver, got %+v", d)
	}
	if !d[0].Delta.IsEqualTo(math.V3(0, 0, 1), 1e-12) {
		t.Errorf("driver delta = %v, want full move", d[0].Delta)
	}
}

func TestFalloffDeltasRegionDecays(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	// CV spacing is 0.25; a radius of 0.6 reaches the immediate and diagonal neighbours.
	deltas := s.FalloffDeltas([][2]int{{2, 2}}, math.V3(0, 0, 1), 0.6, FalloffSmooth)
	byIndex := map[[2]int]float64{}
	for _, d := range deltas {
		byIndex[[2]int{d.U, d.V}] = float64(d.Delta.Z)
	}
	if byIndex[[2]int{2, 2}] != 1 {
		t.Errorf("driver weight = %g, want 1", byIndex[[2]int{2, 2}])
	}
	nb := byIndex[[2]int{2, 3}] // 0.25 away
	if nb <= 0 || nb >= 1 {
		t.Errorf("neighbour weight = %g, want strictly between 0 and 1", nb)
	}
	if _, far := byIndex[[2]int{0, 0}]; far {
		t.Error("a CV outside the radius should not move")
	}
}

func TestFalloffDeltasLinearVsSmooth(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	lin := weightAt(s.FalloffDeltas([][2]int{{2, 2}}, math.V3(0, 0, 1), 0.6, FalloffLinear), 2, 3)
	smooth := weightAt(s.FalloffDeltas([][2]int{{2, 2}}, math.V3(0, 0, 1), 0.6, FalloffSmooth), 2, 3)
	// At t = 0.25/0.6 ≈ 0.42 (inside the half-radius), smoothstep stays nearer 1 than the linear
	// ramp, so the smooth weight exceeds the linear one; both strictly inside (0,1).
	if !(lin > 0 && lin < 1 && smooth > 0 && smooth < 1) {
		t.Fatalf("weights out of range: linear=%g smooth=%g", lin, smooth)
	}
	if smooth <= lin {
		t.Errorf("smooth weight %g should exceed linear %g at this distance", smooth, lin)
	}
}

// weightAt returns the Z displacement weight of control (u,v) in a delta list (0 if absent).
func weightAt(deltas []ControlPointDelta, u, v int) float64 {
	for _, d := range deltas {
		if d.U == u && d.V == v {
			return float64(d.Delta.Z)
		}
	}
	return 0
}

func TestFalloffDeltasRowDriver(t *testing.T) {
	t.Parallel()
	s := flatGrid(t, 5)
	// Drive an entire V-row (constant U index 2): rigid row, radius 0.
	drivers := [][2]int{{2, 0}, {2, 1}, {2, 2}, {2, 3}, {2, 4}}
	deltas := s.FalloffDeltas(drivers, math.V3(0, 0, 2), 0, FalloffConstant)
	if len(deltas) != 5 {
		t.Fatalf("rigid row should move exactly its 5 CVs, got %d", len(deltas))
	}
	for _, d := range deltas {
		if d.U != 2 || !d.Delta.IsEqualTo(math.V3(0, 0, 2), 1e-12) {
			t.Errorf("row CV (%d,%d) delta = %v, want full move on row 2", d.U, d.V, d.Delta)
		}
	}
}
