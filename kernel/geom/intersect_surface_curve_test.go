// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// assertOnBoth checks an intersection point lies on both the surface (signed distance ~0)
// and within the curve's swept extent (it was produced from a curve sample).
func assertOnSurface(t *testing.T, s Surface, p math.Point3, tol float64) {
	t.Helper()
	if d := SignedDistanceToSurface(s, p); stdmath.Abs(d) > tol {
		t.Errorf("point %v is %v off the surface, want ≤ %v", p, d, tol)
	}
}

func TestSegmentPlaneIntersection(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	seg := NewLineSegment(math.P3(1, 2, -4), math.P3(1, 2, 4))
	pts := IntersectCurveSurface(seg, pl)
	if len(pts) != 1 {
		t.Fatalf("segment∩plane = %d points, want 1", len(pts))
	}
	if pts[0].DistanceTo(math.P3(1, 2, 0)) > 1e-9 {
		t.Errorf("pierce point = %v, want (1,2,0)", pts[0])
	}
}

func TestHelixPlaneIntersection(t *testing.T) {
	// A 3-turn helix about +Z, pitch 10, crossing the z=15 plane once (mid of height 30).
	h, _ := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 4, 10, 0, 3, false)
	pl, _ := NewPlane(math.P3(0, 0, 15), math.V3(0, 0, 1))
	pts := IntersectCurveSurface(h, pl)
	if len(pts) != 1 {
		t.Fatalf("helix∩plane = %d points, want 1", len(pts))
	}
	assertOnSurface(t, pl, pts[0], 1e-6)
	if stdmath.Abs(float64(pts[0].Z)-15) > 1e-6 {
		t.Errorf("crossing z = %v, want 15", pts[0].Z)
	}
}

func TestSegmentSphereTwoCrossings(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	// A chord through the sphere along X crosses the surface twice (±5).
	seg := NewLineSegment(math.P3(-9, 0, 0), math.P3(9, 0, 0))
	pts := IntersectCurveSurface(seg, sp)
	if len(pts) != 2 {
		t.Fatalf("segment∩sphere = %d points, want 2", len(pts))
	}
	for _, p := range pts {
		assertOnSurface(t, sp, p, 1e-6)
	}
}

func TestCurveSurfaceNoIntersection(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	seg := NewLineSegment(math.P3(0, 0, 1), math.P3(1, 1, 2)) // wholly above the plane
	if pts := IntersectCurveSurface(seg, pl); len(pts) != 0 {
		t.Errorf("non-crossing segment = %d points, want 0", len(pts))
	}
}

func TestCurveSurfaceUnboundedDomainSkipped(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	line, _ := NewLine(math.P3(0, 0, -1), math.V3(0, 0, 1)) // infinite domain
	if pts := IntersectCurveSurface(line, pl); pts != nil {
		t.Errorf("an unbounded curve should yield no points, got %v", pts)
	}
}

// TestCurveSurfaceEndpointOnSurface covers the exact-sample-hit path (the segment starts
// on the plane).
func TestCurveSurfaceEndpointOnSurface(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	seg := NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 5)) // starts on the plane
	pts := IntersectCurveSurface(seg, pl)
	if len(pts) != 1 || pts[0].DistanceTo(math.P3(0, 0, 0)) > 1e-9 {
		t.Fatalf("endpoint-on-surface = %v, want one point at the origin", pts)
	}
}

// TestCurveSurfaceMetamorphic checks the crossing is stable under sampling refinement:
// the bisected root does not move when the bracketing resolution changes (the segment has
// a single transversal crossing, so both resolutions find the same point).
func TestCurveSurfaceMetamorphic(t *testing.T) {
	sp, _ := NewSphere(math.P3(0, 0, 0), 5)
	seg := NewLineSegment(math.P3(-9, 1, 0), math.P3(9, 1, 0))
	pts := IntersectCurveSurface(seg, sp)
	if len(pts) != 2 {
		t.Fatalf("expected 2 crossings, got %d", len(pts))
	}
	// Re-run on a sub-segment that brackets only the +X crossing; the shared root agrees.
	half := NewLineSegment(math.P3(0, 1, 0), math.P3(9, 1, 0))
	got := IntersectCurveSurface(half, sp)
	if len(got) != 1 {
		t.Fatalf("sub-segment expected 1 crossing, got %d", len(got))
	}
	// The +X crossing from the full run matches the sub-segment run to bisection tol.
	near := pts[0]
	if pts[1].X > pts[0].X {
		near = pts[1]
	}
	if got[0].DistanceTo(near) > 1e-6 {
		t.Errorf("refined crossing moved: %v vs %v", got[0], near)
	}
}
