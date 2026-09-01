// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// bezierCurve3 builds a clamped cubic Bézier from four control points.
func bezierCurve3(t *testing.T, p0, p1, p2, p3 math.Point3) BSplineCurve {
	t.Helper()
	c, err := NewBSplineCurveUniformWeights(3, []math.Point3{p0, p1, p2, p3}, []float64{0, 0, 0, 0, 1, 1, 1, 1})
	if err != nil {
		t.Fatalf("bezier: %v", err)
	}
	return c
}

func TestCoonsFillInterpolatesBoundaries(t *testing.T) {
	t.Parallel()
	// A square opening with two curved (z-lifted) opposite edges.
	c0 := bezierCurve3(t, math.P3(0, 0, 0), math.P3(0.33, 0, 0.5), math.P3(0.66, 0, 0.5), math.P3(1, 0, 0)) // v=0, arched up
	c1 := bezierCurve3(t, math.P3(0, 1, 0), math.P3(0.33, 1, -0.3), math.P3(0.66, 1, -0.3), math.P3(1, 1, 0))
	d0 := bezierCurve3(t, math.P3(0, 0, 0), math.P3(0, 0.33, 0), math.P3(0, 0.66, 0), math.P3(0, 1, 0)) // u=0
	d1 := bezierCurve3(t, math.P3(1, 0, 0), math.P3(1, 0.33, 0), math.P3(1, 0.66, 0), math.P3(1, 1, 0)) // u=1

	fill, err := CoonsFill(c0, c1, d0, d1)
	if err != nil {
		t.Fatalf("CoonsFill: %v", err)
	}
	// The four boundaries are interpolated exactly.
	for i := 0; i <= 10; i++ {
		s := float64(i) / 10
		if !fill.PointAt(s, 0).IsEqualTo(c0.PointAt(s), 1e-9) {
			t.Fatalf("v=0 boundary not interpolated at u=%g: %v vs %v", s, fill.PointAt(s, 0), c0.PointAt(s))
		}
		if !fill.PointAt(s, 1).IsEqualTo(c1.PointAt(s), 1e-9) {
			t.Fatalf("v=1 boundary not interpolated at u=%g", s)
		}
		if !fill.PointAt(0, s).IsEqualTo(d0.PointAt(s), 1e-9) {
			t.Fatalf("u=0 boundary not interpolated at v=%g", s)
		}
		if !fill.PointAt(1, s).IsEqualTo(d1.PointAt(s), 1e-9) {
			t.Fatalf("u=1 boundary not interpolated at v=%g", s)
		}
	}
}

func TestCoonsFillRejectsIncompatible(t *testing.T) {
	t.Parallel()
	c0 := bezierCurve3(t, math.P3(0, 0, 0), math.P3(0.33, 0, 0), math.P3(0.66, 0, 0), math.P3(1, 0, 0))
	c1 := bezierCurve3(t, math.P3(0, 1, 0), math.P3(0.33, 1, 0), math.P3(0.66, 1, 0), math.P3(1, 1, 0))
	d0 := bezierCurve3(t, math.P3(0, 0, 0), math.P3(0, 0.33, 0), math.P3(0, 0.66, 0), math.P3(0, 1, 0))
	// d1 with a degree-2 (different) curve → incompatible v-pair.
	bad, _ := NewBSplineCurveUniformWeights(2, []math.Point3{math.P3(1, 0, 0), math.P3(1, 0.5, 0), math.P3(1, 1, 0)}, []float64{0, 0, 0, 1, 1, 1})
	if _, err := CoonsFill(c0, c1, d0, bad); err == nil {
		t.Error("incompatible v-pair should error")
	}
	// Mismatched corner: shift d0's start away from c0's start.
	d0bad := bezierCurve3(t, math.P3(0, 0, 5), math.P3(0, 0.33, 0), math.P3(0, 0.66, 0), math.P3(0, 1, 0))
	d1 := bezierCurve3(t, math.P3(1, 0, 0), math.P3(1, 0.33, 0), math.P3(1, 0.66, 0), math.P3(1, 1, 0))
	if _, err := CoonsFill(c0, c1, d0bad, d1); err == nil {
		t.Error("inconsistent corner should error")
	}
}

// degree3Curve builds a clamped cubic from 5 control points (clamped uniform knots).
func degree3Curve(t *testing.T, pts ...math.Point3) BSplineCurve {
	t.Helper()
	c, err := NewBSplineCurveUniformWeights(3, pts, clampedUniformKnots(4, 3))
	if err != nil {
		t.Fatalf("curve: %v", err)
	}
	return c
}

func TestFillSurfaceAllG0InterpolatesBoundaries(t *testing.T) {
	t.Parallel()
	c0 := degree3Curve(t, math.P3(0, 0, 8), math.P3(0.25, 0, 6), math.P3(0.5, 0, 4), math.P3(0.75, 0, 2), math.P3(1, 0, 0))
	c1 := degree3Curve(t, math.P3(0, 1, 8), math.P3(0.25, 1, 6), math.P3(0.5, 1, 4), math.P3(0.75, 1, 2), math.P3(1, 1, 0))
	d0 := degree3Curve(t, math.P3(0, 0, 8), math.P3(0, 0.25, 8), math.P3(0, 0.5, 8), math.P3(0, 0.75, 8), math.P3(0, 1, 8))
	d1 := degree3Curve(t, math.P3(1, 0, 0), math.P3(1, 0.25, 0), math.P3(1, 0.5, 0), math.P3(1, 0.75, 0), math.P3(1, 1, 0))
	fill, err := FillSurface(c0, c1, d0, d1, [4]FillSide{})
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	for i := 0; i <= 8; i++ {
		s := float64(i) / 8
		if !fill.PointAt(s, 0).IsEqualTo(c0.PointAt(s), 1e-9) || !fill.PointAt(0, s).IsEqualTo(d0.PointAt(s), 1e-9) {
			t.Fatalf("all-G0 fill should interpolate the boundaries at %g", s)
		}
	}
}

func TestFillSurfaceG2MatchesNeighbour(t *testing.T) {
	t.Parallel()
	// Neighbour patch to the west (x∈[-1,0]); the fill's u=0 boundary is its u-max iso-curve.
	nb := uPatch(t, -1, func(i, j int) float64 { return 0.5 * float64(i*i) })
	d0, err := NewBSplineCurveUniformWeights(3, nb.Ctrl[len(nb.Ctrl)-1], clampedUniformKnots(4, 3))
	if err != nil {
		t.Fatalf("d0 iso: %v", err)
	}
	z0 := float64(nb.Ctrl[len(nb.Ctrl)-1][0].Z) // the neighbour's u-max corner height (constant along v here)
	c0 := degree3Curve(t, math.P3(0, 0, z0), math.P3(0.25, 0, z0*0.75), math.P3(0.5, 0, z0*0.5), math.P3(0.75, 0, z0*0.25), math.P3(1, 0, 0))
	c1 := degree3Curve(t, math.P3(0, 1, z0), math.P3(0.25, 1, z0*0.75), math.P3(0.5, 1, z0*0.5), math.P3(0.75, 1, z0*0.25), math.P3(1, 1, 0))
	d1 := degree3Curve(t, math.P3(1, 0, 0), math.P3(1, 0.25, 0), math.P3(1, 0.5, 0), math.P3(1, 0.75, 0), math.P3(1, 1, 0))

	fill, err := FillSurface(c0, c1, d0, d1, [4]FillSide{2: {Adjacent: nb, AdjEdge: UMaxEdge, Order: 2}})
	if err != nil {
		t.Fatalf("FillSurface G2: %v", err)
	}
	// The fill's u=0 cross-derivatives equal the neighbour's u=1 ones (C2 across the seam ⇒ G2).
	for _, v := range []float64{0, 0.5, 1} {
		fd := fill.SurfaceDersAt(0, v, 2, 0)
		nd := nb.SurfaceDersAt(1, v, 2, 0)
		for k := 0; k <= 2; k++ {
			if !fd[k][0].IsEqualTo(nd[k][0], 1e-7) {
				t.Fatalf("G2 fill seam mismatch at v=%g order %d: %v vs %v", v, k, fd[k][0], nd[k][0])
			}
		}
	}
}

func TestGrevilleAbscissaeClampedEnds(t *testing.T) {
	t.Parallel()
	g := grevilleAbscissae([]float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}, 3)
	if len(g) != 5 || g[0] != 0 || g[len(g)-1] != 1 {
		t.Fatalf("greville = %v, want 5 values spanning [0,1]", g)
	}
	for i := 1; i < len(g); i++ {
		if g[i] <= g[i-1] {
			t.Errorf("greville abscissae should be increasing: %v", g)
		}
	}
}
