// SPDX-License-Identifier: GPL-2.0-only

package geom

import "testing"

// The kind enums are only useful if every value maps to exactly one type and vice versa.
// These coverage tests enumerate the enum and assert a probe instance of each kind reports
// itself — so adding a SurfaceKind/CurveKind value without a Kind() method (or with the
// wrong one), or a new type without an enum value, fails here rather than silently widening
// a consumer's default (#1403, audit I6). Kind() ignores the receiver's fields, so a
// zero-value struct is a sufficient probe.

var surfaceKindProbes = map[SurfaceKind]KindedSurface{
	SurfacePlane:              Plane{},
	SurfaceCylinder:           Cylinder{},
	SurfaceSphere:             Sphere{},
	SurfaceCone:               Cone{},
	SurfaceTorus:              Torus{},
	SurfaceBSpline:            BSplineSurface{},
	SurfaceEllipticalCylinder: EllipticalCylinder{},
	SurfaceEllipticalCone:     EllipticalCone{},
	SurfaceOffset:             OffsetSurface{},
	SurfaceThreadedCylinder:   ThreadedCylinder{},
}

func TestSurfaceKindCoverage(t *testing.T) {
	t.Parallel()
	if len(surfaceKindProbes) != int(surfaceKindCount) {
		t.Fatalf("surfaceKindProbes has %d entries, want %d (one per SurfaceKind) — a new kind needs a probe",
			len(surfaceKindProbes), int(surfaceKindCount))
	}
	for k := range surfaceKindCount {
		probe, ok := surfaceKindProbes[k]
		if !ok {
			t.Errorf("SurfaceKind %v has no probe — add its type to surfaceKindProbes", k)
			continue
		}
		if got := probe.Kind(); got != k {
			t.Errorf("probe for %v reports Kind %v — its Kind() method is wrong", k, got)
		}
	}
}

var curveKindProbes = map[CurveKind]KindedCurve{
	CurveLine:          Line{},
	CurveLineSegment:   LineSegment{},
	CurvePolyline:      Polyline{},
	CurveCircle:        Circle{},
	CurveArc:           Arc3d{},
	CurveEllipse:       EllipseFull{},
	CurveEllipticalArc: EllipticalArc{},
	CurveHyperbolicArc: HyperbolicArc{},
	CurveParabola:      Parabola{},
	CurveBSpline:       BSplineCurve{},
	CurveHelix:         Helix3d{},
	CurveVariableHelix: VariableHelix3d{},
	CurveSpiric:        SpiricArc{},
	CurveTorusCyl:      TorusCylinderArc{},
	CurveRuledQuadric:  RuledQuadricArc{},
}

func TestCurveKindCoverage(t *testing.T) {
	t.Parallel()
	if len(curveKindProbes) != int(curveKindCount) {
		t.Fatalf("curveKindProbes has %d entries, want %d (one per CurveKind) — a new kind needs a probe",
			len(curveKindProbes), int(curveKindCount))
	}
	for k := range curveKindCount {
		probe, ok := curveKindProbes[k]
		if !ok {
			t.Errorf("CurveKind %v has no probe — add its type to curveKindProbes", k)
			continue
		}
		if got := probe.Kind(); got != k {
			t.Errorf("probe for %v reports Kind %v — its Kind() method is wrong", k, got)
		}
	}
}

// TestSurfaceKindNamesComplete asserts every SurfaceKind has a distinct non-placeholder name
// (used verbatim in error messages), so a new kind cannot ship with a "SurfaceKind(?)" label.
func TestSurfaceKindNamesComplete(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for k := range surfaceKindCount {
		name := k.String()
		if name == "SurfaceKind(?)" || seen[name] {
			t.Errorf("SurfaceKind %d has a missing or duplicate name %q", k, name)
		}
		seen[name] = true
	}
}

// TestCurveKindNamesComplete is the curve analogue.
func TestCurveKindNamesComplete(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for k := range curveKindCount {
		name := k.String()
		if name == "CurveKind(?)" || seen[name] {
			t.Errorf("CurveKind %d has a missing or duplicate name %q", k, name)
		}
		seen[name] = true
	}
}
