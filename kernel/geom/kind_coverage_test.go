// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

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
	CurveTorusQuadric:  TorusQuadricArc{},
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

// TestSurfaceKindOfNamesEveryProbe: SurfaceKindOf is the bare-Surface form of Kind(), so it must
// agree with the method on every kind — a consumer that groups surfaces by it (kernel/brep's merge
// bucket, #3523) is only sound while the two answers are the same one.
func TestSurfaceKindOfNamesEveryProbe(t *testing.T) {
	t.Parallel()
	for k, probe := range surfaceKindProbes {
		got, ok := SurfaceKindOf(probe)
		if !ok || got != k {
			t.Errorf("SurfaceKindOf(%T) = %v, %v; want %v, true", probe, got, ok, k)
		}
	}
}

// TestSurfaceKindOfDeclinesASurfaceThatNamesNoKind: the false arm is the one a caller must handle,
// so it has to be reachable. saddleSurface (plate_fill_test.go) is a Surface with no Kind method.
func TestSurfaceKindOfDeclinesASurfaceThatNamesNoKind(t *testing.T) {
	t.Parallel()
	if _, ok := SurfaceKindOf(saddleSurface{}); ok {
		t.Error("SurfaceKindOf claims a kind for a surface with no Kind method")
	}
}

// populatedSurfaceKindProbes is the same ten types with their fields FILLED. surfaceKindProbes above
// uses zero values, which proves the mapping only where every field is zero.
func populatedSurfaceKindProbes(t *testing.T) map[SurfaceKind]KindedSurface {
	t.Helper()
	z, x, down := mustUnit(t, 0, 0, 1), mustUnit(t, 1, 0, 0), mustUnit(t, 0, 0, -1)
	cyl := Cylinder{Origin: math.P3(1, 2, 3), AxisDir: z, Ref: x, Radius: 2.5}
	return map[SurfaceKind]KindedSurface{
		SurfacePlane:    Plane{Origin: math.P3(1, 2, 3), UAxis: x, VAxis: mustUnit(t, 0, 1, 0)},
		SurfaceCylinder: cyl,
		SurfaceSphere:   Sphere{Center: math.P3(-1, 4, 2), Radius: 3.25},
		SurfaceCone:     Cone{Apex: math.P3(0, 1, 5), AxisDir: down, Ref: x, HalfAngle: 0.4},
		SurfaceTorus: Torus{Center: math.P3(2, 2, 2), AxisDir: z, Ref: x,
			MajorRadius: 5, MinorRadius: 1.5},
		SurfaceBSpline: BSplineSurface{UDegree: 1, VDegree: 1,
			Ctrl:   [][]math.Point3{{math.P3(0, 0, 0), math.P3(1, 0, 0)}, {math.P3(0, 1, 0), math.P3(1, 1, 1)}},
			UKnots: []float64{0, 0, 1, 1}, VKnots: []float64{0, 0, 1, 1}},
		SurfaceEllipticalCylinder: EllipticalCylinder{Origin: math.P3(1, 1, 0), AxisDir: z, Ref: x,
			MajorRadius: 4, MinorRadius: 2},
		SurfaceEllipticalCone: EllipticalCone{Apex: math.P3(0, 0, 9), AxisDir: down, Ref: x,
			MajorAngle: 0.5, MinorAngle: 0.3},
		SurfaceOffset:           OffsetSurface{Base: cyl, Distance: 0.75},
		SurfaceThreadedCylinder: ThreadedCylinder{Cylinder: cyl, Pitch: 0.2, Depth: 0.1},
	}
}

// TestKindIgnoresTheReceiversFields is the property kernel/brep's merge bucket rests on (#3523): the
// bucket reads Kind() as a proxy for the CONCRETE TYPE, and that proxy is faithful only while Kind()
// is a per-type constant. A Kind() that branched on a field would put two values of ONE type in two
// buckets and the merge would stop offering them — which the zero-value coverage row above cannot
// see, because it never varies a field.
func TestKindIgnoresTheReceiversFields(t *testing.T) {
	t.Parallel()
	filled := populatedSurfaceKindProbes(t)
	if len(filled) != int(surfaceKindCount) {
		t.Fatalf("populatedSurfaceKindProbes has %d entries, want %d — a new kind needs a populated probe",
			len(filled), int(surfaceKindCount))
	}
	for k, s := range filled {
		if got := s.Kind(); got != k {
			t.Errorf("a populated %T reports Kind %v, but the zero value reports %v: Kind() reads its "+
				"receiver's fields, so it is not a proxy for the concrete type", s, got, k)
		}
	}
}
