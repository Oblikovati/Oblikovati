// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"reflect"
	"testing"

	"oblikovati.org/math"
)

// sketch2DEntities is the universe of concrete 2D entity types — one typed-nil sample each,
// keyed for the coverage tables below. Every 2D entity kind with a Kind() in entity_kind.go
// appears exactly once. A new 2D entity type MUST be appended here, which then forces it into
// each PARTIAL coverage map (the exhaustiveness test fails until it is classified) — so a new
// kind can never silently fall out of an optional capability (audit I10, #1633).
var sketch2DEntities = []Entity{
	(*Point)(nil), (*Line)(nil), (*Circle)(nil), (*Arc)(nil), (*Ellipse)(nil),
	(*EllipticalArc)(nil), (*Spline)(nil), (*SplineHandle)(nil), (*FixedSpline)(nil),
	(*OffsetSpline)(nil), (*EquationCurve)(nil), (*BlockInstance)(nil), (*SketchImage)(nil),
	(*TextBox)(nil), (*FillRegion)(nil), (*ProjectedPoint)(nil),
}

// pointDefinedCoverage records which entity types define draggable points (a drag pins them so
// the curve translates rigidly). The geometric curves do; annotations, derived and projected
// geometry legitimately do not. Spline is true since #1633 fixed its silent hole.
var pointDefinedCoverage = map[reflect.Type]bool{
	kindType((*Point)(nil)): true, kindType((*Line)(nil)): true, kindType((*Circle)(nil)): true,
	kindType((*Arc)(nil)): true, kindType((*Ellipse)(nil)): true, kindType((*EllipticalArc)(nil)): true,
	kindType((*Spline)(nil)):       true,
	kindType((*SplineHandle)(nil)): false, kindType((*FixedSpline)(nil)): false,
	kindType((*OffsetSpline)(nil)): false, kindType((*EquationCurve)(nil)): false,
	kindType((*BlockInstance)(nil)): false, kindType((*SketchImage)(nil)): false,
	kindType((*TextBox)(nil)): false, kindType((*FillRegion)(nil)): false,
	kindType((*ProjectedPoint)(nil)): false,
}

// smoothCurveCoverage records the sealed SmoothCurve set: only a Line, Arc or Spline can report
// a tangent/curvature frame at an endpoint for a G2 smooth join (constraints_smooth.go).
var smoothCurveCoverage = map[reflect.Type]bool{
	kindType((*Line)(nil)): true, kindType((*Arc)(nil)): true, kindType((*Spline)(nil)): true,
	kindType((*Point)(nil)): false, kindType((*Circle)(nil)): false, kindType((*Ellipse)(nil)): false,
	kindType((*EllipticalArc)(nil)): false, kindType((*SplineHandle)(nil)): false,
	kindType((*FixedSpline)(nil)): false, kindType((*OffsetSpline)(nil)): false,
	kindType((*EquationCurve)(nil)): false, kindType((*BlockInstance)(nil)): false,
	kindType((*SketchImage)(nil)): false, kindType((*TextBox)(nil)): false,
	kindType((*FillRegion)(nil)): false, kindType((*ProjectedPoint)(nil)): false,
}

// circularCurveCoverage records the sealed CircularCurve set: only a Circle or Arc is defined by
// a center and radius (entities.go) for the concentric/tangent/equal-radius constraints.
var circularCurveCoverage = map[reflect.Type]bool{
	kindType((*Circle)(nil)): true, kindType((*Arc)(nil)): true,
	kindType((*Point)(nil)): false, kindType((*Line)(nil)): false, kindType((*Ellipse)(nil)): false,
	kindType((*EllipticalArc)(nil)): false, kindType((*Spline)(nil)): false,
	kindType((*SplineHandle)(nil)): false, kindType((*FixedSpline)(nil)): false,
	kindType((*OffsetSpline)(nil)): false, kindType((*EquationCurve)(nil)): false,
	kindType((*BlockInstance)(nil)): false, kindType((*SketchImage)(nil)): false,
	kindType((*TextBox)(nil)): false, kindType((*FillRegion)(nil)): false,
	kindType((*ProjectedPoint)(nil)): false,
}

// kindType keys a coverage map by an entity's concrete type (reflect.Type is safe on a typed-nil
// pointer, unlike calling its methods).
func kindType(e Entity) reflect.Type { return reflect.TypeOf(e) }

// TestPointDefinedCoverage asserts each entity's actual pointDefined membership matches the
// table — a type-assert result, not a method call, so typed-nil samples are safe.
func TestPointDefinedCoverage(t *testing.T) {
	assertCoverage[pointDefined](t, "pointDefined", pointDefinedCoverage)
}

func TestSmoothCurveCoverage(t *testing.T) {
	assertCoverage[SmoothCurve](t, "SmoothCurve", smoothCurveCoverage)
}

func TestCircularCurveCoverage(t *testing.T) {
	assertCoverage[CircularCurve](t, "CircularCurve", circularCurveCoverage)
}

// assertCoverage checks that (a) every 2D entity is classified in the table (no unclassified
// type, so a new kind forces a decision), (b) the table has no stale entries, and (c) each
// type's real interface satisfaction matches its recorded value — an implementation the table
// denies, or a table entry with no implementation, fails CI (#1633).
func assertCoverage[C any](t *testing.T, name string, table map[reflect.Type]bool) {
	t.Helper()
	if len(table) != len(sketch2DEntities) {
		t.Fatalf("%s coverage has %d entries, universe has %d — a stale or missing classification",
			name, len(table), len(sketch2DEntities))
	}
	for _, e := range sketch2DEntities {
		want, listed := table[kindType(e)]
		if !listed {
			t.Errorf("%s: %T is unclassified — mark it true/false in the coverage table", name, e)
			continue
		}
		if _, got := any(e).(C); got != want {
			t.Errorf("%s: %T implements=%v, table says %v", name, e, got, want)
		}
	}
}

// TestSplineDragPinsItsPoints is the regression for the #1633 spline hole: dragging a spline
// now pins all its interpolation points, so DefiningPoints returns them (before the fix a
// spline fell through to nil and a drag pinned nothing).
func TestSplineDragPinsItsPoints(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	sp := sk.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(2, 1)}, false)
	got := DefiningPoints(sp)
	if len(got) != 3 {
		t.Fatalf("spline DefiningPoints returned %d points, want 3 (the drag pins nothing bug, #1633)", len(got))
	}
}
