// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// nearPoint reports whether two points coincide within a geometric tolerance.
// (near, for scalars, lives in paramat_test.go.)
func nearPoint(p, q math.Point3) bool { return p.DistanceTo(q) < 1e-9 }

// --- similarityScale -------------------------------------------------------

func TestSimilarityScaleUniform(t *testing.T) {
	cases := []struct {
		name string
		m    math.Matrix4
		want float64
	}{
		{"identity", math.Identity4(), 1},
		{"uniform-scale", math.Scale4(2, 2, 2), 2},
		{"translation-only", math.Translation4(math.V3(5, -3, 2)), 1},
		{"rotation-only", math.Rotation4(stdmath.Pi/3, unitZ(t), math.P3(0, 0, 0)), 1},
		// A reflection is a valid similarity: the scale is the positive axis length,
		// the orientation flip is the caller's concern, not folded into the factor.
		{"reflection", math.Scale4(-1, -1, -1), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := similarityScale(c.m)
			if err != nil {
				t.Fatalf("similarityScale: %v", err)
			}
			if !near(got, c.want) {
				t.Errorf("scale = %g, want %g", got, c.want)
			}
		})
	}
}

func TestSimilarityScaleRejectsNonUniform(t *testing.T) {
	if _, err := similarityScale(math.Scale4(2, 3, 2)); err == nil {
		t.Error("non-uniform scale should be rejected")
	}
}

func TestSimilarityScaleRejectsDegenerate(t *testing.T) {
	if _, err := similarityScale(math.Scale4(0, 1, 1)); err == nil {
		t.Error("zero-length axis should be rejected")
	}
}

// --- TransformCurve --------------------------------------------------------

func TestTransformLineSegment(t *testing.T) {
	seg := NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	out, err := TransformCurve(seg, math.Translation4(math.V3(1, 2, 3)))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(LineSegment)
	if !nearPoint(got.StartPoint, math.P3(1, 2, 3)) || !nearPoint(got.EndPoint, math.P3(2, 2, 3)) {
		t.Errorf("segment endpoints = %v→%v, want (1,2,3)→(2,2,3)", got.StartPoint, got.EndPoint)
	}
}

func TestTransformLine(t *testing.T) {
	ln, err := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformCurve(ln, math.Rotation4(stdmath.Pi/2, unitZ(t), math.P3(0, 0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	// +X rotated 90° about +Z is +Y.
	if d := out.(Line).Dir.AsVector(); !near(d.X, 0) || !near(d.Y, 1) {
		t.Errorf("rotated line dir = %v, want ≈ (0,1,0)", d)
	}
}

func TestTransformCircleScalesRadius(t *testing.T) {
	c, err := NewCircle(math.P3(1, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformCurve(c, math.Scale4(3, 3, 3))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(Circle)
	if !near(got.Radius, 6) {
		t.Errorf("radius = %g, want 6", got.Radius)
	}
	if !nearPoint(got.Center, math.P3(3, 0, 0)) {
		t.Errorf("center = %v, want (3,0,0)", got.Center)
	}
}

func TestTransformArcPreservesAngles(t *testing.T) {
	a, err := NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0.25, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformCurve(a, math.Scale4(2, 2, 2))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(Arc3d)
	if !near(got.Radius, 4) || !near(got.StartAngle, 0.25) || !near(got.SweepAngle, 1.5) {
		t.Errorf("arc = r%g start%g sweep%g, want r4 start0.25 sweep1.5", got.Radius, got.StartAngle, got.SweepAngle)
	}
}

func TestTransformCurveRejectsNonUniform(t *testing.T) {
	c, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	if _, err := TransformCurve(c, math.Scale4(2, 3, 2)); err == nil {
		t.Error("non-uniform scale should be rejected before transforming a curve")
	}
}

func TestTransformCurveUnsupportedType(t *testing.T) {
	p, err := NewPolyline([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TransformCurve(p, math.Identity4()); err == nil {
		t.Error("an unsupported curve type should return NotYetImplemented")
	}
}

// --- TransformSurface ------------------------------------------------------

func TestTransformPlane(t *testing.T) {
	pl, err := NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformSurface(pl, math.Translation4(math.V3(0, 0, 4)))
	if err != nil {
		t.Fatal(err)
	}
	if !nearPoint(out.(Plane).Origin, math.P3(0, 0, 5)) {
		t.Errorf("plane origin = %v, want (0,0,5)", out.(Plane).Origin)
	}
}

func TestTransformCylinderScalesRadius(t *testing.T) {
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformSurface(cyl, math.Scale4(2, 2, 2))
	if err != nil {
		t.Fatal(err)
	}
	if !near(out.(Cylinder).Radius, 4) {
		t.Errorf("cylinder radius = %g, want 4", out.(Cylinder).Radius)
	}
}

func TestTransformConePreservesHalfAngle(t *testing.T) {
	cone, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformSurface(cone, math.Translation4(math.V3(1, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(Cone)
	if !near(got.HalfAngle, stdmath.Pi/6) || !nearPoint(got.Apex, math.P3(1, 1, 1)) {
		t.Errorf("cone = apex%v half%g, want apex(1,1,1) half%g", got.Apex, got.HalfAngle, stdmath.Pi/6)
	}
}

func TestTransformSphereScalesRadius(t *testing.T) {
	s, err := NewSphere(math.P3(1, 1, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformSurface(s, math.Scale4(2, 2, 2))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(Sphere)
	if !near(got.Radius, 4) || !nearPoint(got.Center, math.P3(2, 2, 2)) {
		t.Errorf("sphere = center%v r%g, want center(2,2,2) r4", got.Center, got.Radius)
	}
}

func TestTransformTorusScalesBothRadii(t *testing.T) {
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	out, err := TransformSurface(tor, math.Scale4(3, 3, 3))
	if err != nil {
		t.Fatal(err)
	}
	got := out.(Torus)
	if !near(got.MajorRadius, 12) || !near(got.MinorRadius, 3) {
		t.Errorf("torus = major%g minor%g, want major12 minor3", got.MajorRadius, got.MinorRadius)
	}
}

func TestTransformSurfaceRejectsNonUniform(t *testing.T) {
	s, _ := NewSphere(math.P3(0, 0, 0), 1)
	if _, err := TransformSurface(s, math.Scale4(2, 3, 2)); err == nil {
		t.Error("non-uniform scale should be rejected before transforming a surface")
	}
}

func TestTransformSurfaceUnsupportedType(t *testing.T) {
	if _, err := TransformSurface(sampleBSplineSurface(t), math.Identity4()); err == nil {
		t.Error("an unsupported surface type should return NotYetImplemented")
	}
}

// unitZ is the +Z rotation axis used across the transform cases.
func unitZ(t *testing.T) math.UnitVector3 {
	t.Helper()
	u, err := math.UnitVector3FromVector(math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	return u
}
