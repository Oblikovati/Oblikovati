// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// allPoints flattens traced polylines into a single point slice, failing if empty.
func allPoints(t *testing.T, loops [][]math.Point3) []math.Point3 {
	t.Helper()
	var pts []math.Point3
	for _, l := range loops {
		pts = append(pts, l...)
	}
	if len(pts) == 0 {
		t.Fatal("trace produced no points")
	}
	return pts
}

func TestSpherePlaneIntersectionIsEquator(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	for _, p := range allPoints(t, IntersectSurfaceSurface(sp, pl, SurfaceGrid{})) {
		if stdmath.Abs(float64(p.Z)) > 1e-6 {
			t.Errorf("intersection point %v not on z=0 plane", p)
		}
		if r := float64(p.AsVector().Length()); stdmath.Abs(r-5) > 1e-6 {
			t.Errorf("intersection point radius %v, want 5 (on sphere)", r)
		}
	}
}

func TestCylinderPlaneIntersectionIsCircle(t *testing.T) {
	cy, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	loops := IntersectSurfaceSurface(cy, pl, SurfaceGrid{VMin: -5, VMax: 5})
	for _, p := range allPoints(t, loops) {
		if stdmath.Abs(float64(p.Z)) > 1e-6 {
			t.Errorf("point %v not on z=0", p)
		}
		if r := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(r-3) > 1e-6 {
			t.Errorf("point radial %v, want 3 (on cylinder)", r)
		}
	}
}

func TestSurfaceIntersectionNone(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 100), 5) // far above the plane
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if loops := IntersectSurfaceSurface(sp, pl, SurfaceGrid{}); loops != nil {
		t.Errorf("disjoint surfaces should not intersect, got %v", loops)
	}
}

func TestSurfaceIntersectionUnboundedBaseNeedsWindow(t *testing.T) {
	// A plane base has an unbounded domain; with no explicit window the grid is degenerate
	// and yields nothing (the caller must window an unbounded base).
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	if loops := IntersectSurfaceSurface(pl, sp, SurfaceGrid{}); loops != nil {
		t.Errorf("unbounded base without a window should yield nil, got %v", loops)
	}
}

// TestSurfaceIntersectionMetamorphic checks the traced circle is stable under grid
// refinement: a denser grid keeps every point on both surfaces.
func TestSurfaceIntersectionMetamorphic(t *testing.T) {
	sp, _ := NewSphere(math.P3(1, 0, 0), 4)
	pl, _ := NewPlane(math.P3(0, 0, 2), math.V3(0, 0, 1)) // z=2 cuts the sphere
	for _, steps := range []int{48, 192} {
		loops := IntersectSurfaceSurface(sp, pl, SurfaceGrid{USteps: steps, VSteps: steps})
		for _, p := range allPoints(t, loops) {
			if stdmath.Abs(float64(p.Z)-2) > 1e-6 {
				t.Errorf("steps=%d: point %v not on z=2", steps, p)
			}
			if r := float64(p.VectorTo(math.P3(1, 0, 0)).Length()); stdmath.Abs(r-4) > 1e-6 {
				t.Errorf("steps=%d: point %v not on sphere", steps, p)
			}
		}
	}
}

func TestSphereSilhouetteIsGreatCircle(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	// Viewing along +Z, the silhouette is the z=0 equator (normal ⟂ Z there).
	for _, p := range allPoints(t, Silhouette(sp, math.V3(0, 0, 1), SurfaceGrid{})) {
		if stdmath.Abs(float64(p.Z)) > 1e-6 {
			t.Errorf("silhouette point %v not on the equator", p)
		}
		if r := float64(p.AsVector().Length()); stdmath.Abs(r-5) > 1e-6 {
			t.Errorf("silhouette point radius %v, want 5", r)
		}
	}
}

func TestCylinderSilhouetteIsTwoLines(t *testing.T) {
	cy, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	// Viewing along +X (⟂ the axis), the silhouette is the two contour lines at x=0, y=±3.
	pts := allPoints(t, Silhouette(cy, math.V3(1, 0, 0), SurfaceGrid{VMin: -4, VMax: 4}))
	for _, p := range pts {
		if stdmath.Abs(float64(p.X)) > 1e-6 || stdmath.Abs(stdmath.Abs(float64(p.Y))-3) > 1e-6 {
			t.Errorf("cylinder silhouette point %v, want x=0 / |y|=3", p)
		}
	}
}

// TestAppendCellSegmentsSaddle covers the four-crossing (saddle) cell branch with a
// checkerboard field whose corners alternate sign, so all four edges cross zero.
func TestAppendCellSegmentsSaddle(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	saddle := func(u, v float64) float64 { return stdmath.Cos(stdmath.Pi * (u + v)) }
	segs := appendCellSegments(nil, pl, saddle, 0, 0, 1, 1)
	if len(segs) != 2 {
		t.Fatalf("saddle cell = %d segments, want 2", len(segs))
	}
}

// TestBisectEdgeExactZero covers the exact-zero-at-midpoint early return of bisectEdge.
func TestBisectEdgeExactZero(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	// f(u) = u − 0.5 is exactly zero at the edge midpoint (u = 0.5).
	f := func(u, _ float64) float64 { return u - 0.5 }
	got := bisectEdge(pl, f, 0, 0, f(0, 0), 1, 0)
	want := pl.PointAt(0.5, 0)
	if got.DistanceTo(want) > 1e-9 {
		t.Errorf("bisectEdge exact-zero = %v, want %v", got, want)
	}
}

// TestTraceHelpersDirect covers the small numeric helpers directly.
func TestTraceHelpersDirect(t *testing.T) {
	if finiteOr(stdmath.Inf(1), 7) != 7 || finiteOr(3, 7) != 3 {
		t.Error("finiteOr should fall back only for infinities")
	}
	if lerp(0, 10, 0.25) != 2.5 {
		t.Error("lerp wrong")
	}
	if straddlesZero(1, 2) || !straddlesZero(-1, 2) || !straddlesZero(2, -1) {
		t.Error("straddlesZero wrong")
	}
}
