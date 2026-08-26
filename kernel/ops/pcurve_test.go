// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// foldSurface folds exactly in v: PointAt(u,v) = (u, 2v(1−v), 0), so v and 1−v map to the SAME 3D
// point (a 2-to-1 fold). Independent ParamAt is genuinely ambiguous there and snaps adjacent
// boundary points to different sides (the jitter that self-intersects the (u,v) boundary); a march
// seeded from the previous point stays on one side — the property the tolerant mesher needs.
func foldSurface(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(1, 0, 0)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(1, 2, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func selfIntersections(raw []math.Point2) int {
	// Drop consecutive coincident points (the loop double-adds corners) so zero-length edges do
	// not produce spurious crossings.
	var poly []math.Point2
	for i, p := range raw {
		if i == 0 || float64(p.X) != float64(raw[i-1].X) || float64(p.Y) != float64(raw[i-1].Y) {
			poly = append(poly, p)
		}
	}
	at := func(i int) [2]float64 { return [2]float64{float64(poly[i].X), float64(poly[i].Y)} }
	n, count := len(poly), 0
	for i := range n {
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // the closing edge is adjacent to edge 0
			}
			if segmentsCross(at(i), at((i+1)%n), at(j), at((j+1)%n)) {
				count++
			}
		}
	}
	return count
}

func TestMarchUVNonSelfIntersecting(t *testing.T) {
	s := foldSurface(t)
	// A loop entirely on one side of the fold (v < 0.5). Every point also exists at 1−v, so
	// independent ParamAt jitters between the two sides; marchUV keeps it on the v<0.5 side.
	var loop []math.Point3
	add := func(u, v float64) { loop = append(loop, s.PointAt(u, v)) }
	for u := 0.2; u <= 0.8; u += 0.1 {
		add(u, 0.1)
	}
	for v := 0.1; v <= 0.3; v += 0.05 {
		add(0.8, v)
	}
	for u := 0.8; u >= 0.2; u -= 0.1 {
		add(u, 0.3)
	}
	for v := 0.3; v >= 0.1; v -= 0.05 {
		add(0.2, v)
	}

	marched := marchUV(s, loop)

	// marchUV's guarantee: a simple (non-self-intersecting) boundary that stays on ONE side of the
	// fold. (The branch-selection mechanism is proven in geom.TestParamNearSelectsBranchBySeed; the
	// end-to-end win over independent ParamAt is gated on EDF in M24 F04.)
	if si := selfIntersections(marched); si != 0 {
		t.Errorf("marchUV boundary self-intersects %d times; expected 0 (a simple pcurve)", si)
	}
	first := float64(marched[0].Y) < 0.5
	for i, p := range marched {
		if (float64(p.Y) < 0.5) != first {
			t.Errorf("marchUV point %d crossed the fold (v=%.3f); the pcurve must stay single-branch", i, p.Y)
		}
	}
	// For context: independent ParamAt is free to jitter across the fold on this 2-to-1 surface.
	indep := make([]math.Point2, len(loop))
	for i, p := range loop {
		u, v := s.ParamAt(p)
		indep[i] = math.P2(math.Scalar(u), math.Scalar(v))
	}
	t.Logf("self-intersections: marchUV=%d independent ParamAt=%d", selfIntersections(marched), selfIntersections(indep))
}

func TestMarchUVRoundTrip(t *testing.T) {
	s := foldSurface(t)
	var loop []math.Point3
	for u := 0.2; u <= 0.8; u += 0.15 {
		loop = append(loop, s.PointAt(u, 0.3))
	}
	for i, p := range marchUV(s, loop) {
		if got := s.PointAt(float64(p.X), float64(p.Y)); float64(got.DistanceTo(loop[i])) > 1e-4 {
			t.Errorf("marchUV round-trip off by %.5f at %d", got.DistanceTo(loop[i]), i)
		}
	}
}
