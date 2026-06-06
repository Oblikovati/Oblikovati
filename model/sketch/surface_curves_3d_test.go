// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati/kernel/geom"
	gmath "oblikovati/math"
)

func plane(t *testing.T, origin, normal gmath.Vector3) geom.Surface {
	t.Helper()
	p, err := geom.NewPlane(gmath.P3(origin.X, origin.Y, origin.Z), normal)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}

func sphere(t *testing.T, r float64) geom.Surface {
	t.Helper()
	s, err := geom.NewSphere(gmath.P3(0, 0, 0), r)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	return s
}

// TestIntersectionCurve3D checks a sphere∩plane intersection produces points on both.
func TestIntersectionCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	c := s.AddIntersectionCurve3D(sphere(t, 5), plane(t, gmath.V3(0, 0, 0), gmath.V3(0, 0, 1)), geom.SurfaceGrid{})
	loops := c.Evaluate()
	if len(loops) == 0 {
		t.Fatal("intersection produced no polylines")
	}
	for _, l := range loops {
		for _, p := range l {
			if math.Abs(float64(p.Z)) > 1e-6 || math.Abs(float64(p.AsVector().Length())-5) > 1e-6 {
				t.Errorf("intersection point %v not on sphere∩plane", p)
			}
		}
	}
	if !isDerivedCurve3D(c) {
		t.Error("intersection curve should be a derived curve")
	}
}

// TestSilhouetteCurve3D checks a sphere silhouette lies on the equator.
func TestSilhouetteCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	c := s.AddSilhouetteCurve3D(sphere(t, 5), gmath.V3(0, 0, 1), geom.SurfaceGrid{})
	loops := c.Evaluate()
	if len(loops) == 0 {
		t.Fatal("silhouette produced no polylines")
	}
	for _, l := range loops {
		for _, p := range l {
			if math.Abs(float64(p.Z)) > 1e-6 {
				t.Errorf("silhouette point %v not on the equator", p)
			}
		}
	}
}

// TestProjectToSurfaceCurve3D checks projecting a line onto a plane lands it on the plane.
func TestProjectToSurfaceCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	src := geom.NewLineSegment(gmath.P3(0, 0, 5), gmath.P3(4, 0, 5)) // 5 above z=0
	c := s.AddProjectToSurfaceCurve3D(src, plane(t, gmath.V3(0, 0, 0), gmath.V3(0, 0, 1)))
	pts := c.Evaluate()
	if len(pts) == 0 {
		t.Fatal("projection produced no points")
	}
	for _, p := range pts {
		if math.Abs(float64(p.Z)) > 1e-9 {
			t.Errorf("projected point %v not on z=0", p)
		}
	}
	if pts[0].DistanceTo(gmath.P3(0, 0, 0)) > 1e-9 {
		t.Errorf("projection start = %v, want (0,0,0)", pts[0])
	}
}

// TestOnFaceCurve3D checks a parameter-space curve maps onto the surface.
func TestOnFaceCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	pl := plane(t, gmath.V3(0, 0, 0), gmath.V3(0, 0, 1))
	uv := []gmath.Point2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 2}}
	c := s.AddOnFaceCurve3D(pl, uv)
	pts := c.Evaluate()
	if len(pts) != 3 {
		t.Fatalf("on-face curve = %d points, want 3", len(pts))
	}
	// On the z=0 plane, surface points stay at z=0.
	for _, p := range pts {
		if math.Abs(float64(p.Z)) > 1e-9 {
			t.Errorf("on-face point %v not on z=0", p)
		}
	}
}

// TestOffsetCurve3 checks offsetting a straight segment yields a parallel segment at the
// requested distance.
func TestOffsetCurve3(t *testing.T) {
	s := NewSketches3D().Add()
	src := geom.NewLineSegment(gmath.P3(0, 0, 0), gmath.P3(10, 0, 0)) // along +X
	// Offset in the XY plane (normal +Z): direction = Z × X = +Y, distance 2.
	c := s.AddOffsetCurve3(src, 2, gmath.V3(0, 0, 1))
	pts := c.Evaluate()
	for _, p := range pts {
		if math.Abs(float64(p.Y)-2) > 1e-9 || math.Abs(float64(p.Z)) > 1e-9 {
			t.Errorf("offset point %v, want y=2 z=0", p)
		}
	}
}

// TestProjectToSurfaceCurve3DExplicitSamples checks the explicit sample-count path.
func TestProjectToSurfaceCurve3DExplicitSamples(t *testing.T) {
	s := NewSketches3D().Add()
	src := geom.NewLineSegment(gmath.P3(0, 0, 5), gmath.P3(4, 0, 5))
	c := s.AddProjectToSurfaceCurve3D(src, plane(t, gmath.V3(0, 0, 0), gmath.V3(0, 0, 1)))
	c.Samples = 4
	if got := c.Evaluate(); len(got) != 5 {
		t.Errorf("Samples=4 ⇒ %d points, want 5", len(got))
	}
}

// TestOffsetCurve3DegenerateTangent checks the offset leaves a point unmoved when its
// tangent is parallel to the offset normal (no defined offset direction).
func TestOffsetCurve3DegenerateTangent(t *testing.T) {
	src := geom.NewLineSegment(gmath.P3(0, 0, 0), gmath.P3(0, 0, 5)) // along +Z
	p := offsetPoint3(src, 0.5, 3, gmath.V3(0, 0, 1))                // normal also +Z
	if p.DistanceTo(gmath.P3(0, 0, 2.5)) > 1e-9 {
		t.Errorf("degenerate offset moved the point to %v, want it on the curve", p)
	}
}

// TestSurfaceCurvesSkippedBySerialize checks derived curves are excluded from the .obk
// projection (they rebind from references on recompute) while other geometry round-trips.
func TestSurfaceCurvesSkippedBySerialize(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	s.AddIntersectionCurve3D(sphere(t, 5), plane(t, gmath.V3(0, 0, 0), gmath.V3(0, 0, 1)), geom.SurfaceGrid{})

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(data[0].Entities) != 1 {
		t.Errorf("serialized %d entities, want 1 (the line; derived curve skipped)", len(data[0].Entities))
	}
}
