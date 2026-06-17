// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// TestExportImportRoundTrip builds a sketch with lines, circles, arcs and splines, exports
// it to DWG, imports the bytes back, and checks the resulting sketch carries the same
// geometry. It is the round-trip contract for DWG export/import: ExportDWG and ImportDWG
// are inverses for the supported 2D entity types.
//
// The exported R2000 file is unitless (no header section yet), so it is imported into a
// part whose preferred length unit is the database unit (cm); the unit scale is then 1 and
// coordinates compare directly.
func TestExportImportRoundTrip(t *testing.T) {
	src := newCentimetrePart(t)
	sk := src.Sketches().Add(xyPlane(t))
	line := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 5))
	circle := sk.Circles().AddByCenterRadius(gmath.P2(3, 4), 2.5)
	// CCW arc: centre (1,1), from (4,1) to (1,4).
	arc := sk.Arcs().AddByCenterStartEnd(gmath.P2(1, 1), gmath.P2(4, 1), gmath.P2(1, 4), true)
	spline := sk.Splines().AddByControlPoints([]gmath.Point2{{X: 0, Y: 0}, {X: 1, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 0}}, false)

	data, err := ExportDWG(sk, src.Units())
	if err != nil {
		t.Fatalf("ExportDWG: %v", err)
	}

	dst := newCentimetrePart(t)
	res, err := ImportDWG(dst, data, xyPlane(t))
	if err != nil {
		t.Fatalf("ImportDWG: %v", err)
	}
	if res.Is3D {
		t.Fatal("round-trip imported as 3D")
	}
	out := dst.Sketches().Item(0)

	// Lines.
	if got := out.Lines().Count(); got != 1 {
		t.Fatalf("lines = %d, want 1", got)
	}
	rl := out.Lines().Item(0)
	wantPt(t, "line start", rl.StartPoint().Position(), line.StartPoint().Position())
	wantPt(t, "line end", rl.EndPoint().Position(), line.EndPoint().Position())

	// Circles.
	if got := out.Circles().Count(); got != 1 {
		t.Fatalf("circles = %d, want 1", got)
	}
	rc := out.Circles().Item(0)
	wantPt(t, "circle centre", rc.CenterPoint().Position(), circle.CenterPoint().Position())
	wantScalar(t, "circle radius", float64(rc.Radius), float64(circle.Radius))

	// Arcs.
	if got := out.Arcs().Count(); got != 1 {
		t.Fatalf("arcs = %d, want 1", got)
	}
	ra := out.Arcs().Item(0)
	wantPt(t, "arc centre", ra.Center.Position(), arc.Center.Position())
	wantScalar(t, "arc radius", float64(ra.Radius()), float64(arc.Radius()))
	wantEndpoints(t, "arc", ra, arc)

	// Splines.
	if got := out.Splines().Count(); got != 1 {
		t.Fatalf("splines = %d, want 1", got)
	}
	rs := out.Splines().Item(0)
	if rs.PointCount() != spline.PointCount() {
		t.Fatalf("spline points = %d, want %d", rs.PointCount(), spline.PointCount())
	}
	for i := range spline.Points {
		wantPt(t, "spline ctrl", rs.Points[i].Position(), spline.Points[i].Position())
	}
}

// newCentimetrePart returns a part whose preferred length unit is the database unit (cm),
// so a unitless import scales by 1.
func newCentimetrePart(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	p := compdef.NewPartComponentDefinition()
	if err := p.Units().SetPreferred(param.Length, "cm"); err != nil {
		t.Fatalf("SetPreferred cm: %v", err)
	}
	return p
}

func wantPt(t *testing.T, what string, got, want gmath.Point2) {
	t.Helper()
	if math.Abs(got.X-want.X) > 1e-7 || math.Abs(got.Y-want.Y) > 1e-7 {
		t.Errorf("%s = (%g,%g), want (%g,%g)", what, got.X, got.Y, want.X, want.Y)
	}
}

func wantScalar(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-7 {
		t.Errorf("%s = %g, want %g", what, got, want)
	}
}

// wantEndpoints checks the two arcs share the same endpoint pair, ignoring which is
// labelled start vs end (DWG normalises arcs to CCW, which can swap the labels).
func wantEndpoints(t *testing.T, what string, got, want *sketch.Arc) {
	t.Helper()
	gs, ge := got.Start.Position(), got.End.Position()
	ws, we := want.Start.Position(), want.End.Position()
	same := func(a, b gmath.Point2) bool { return math.Abs(a.X-b.X) < 1e-7 && math.Abs(a.Y-b.Y) < 1e-7 }
	matched := (same(gs, ws) && same(ge, we)) || (same(gs, we) && same(ge, ws))
	if !matched {
		t.Errorf("%s endpoints = (%v,%v), want the pair (%v,%v)", what, gs, ge, ws, we)
	}
}

// TestExportDWGHonorsDocumentUnit checks a DWG exported from an inch document
// re-imports at the same physical size (1 inch ↔ 2.54 cm).
func TestExportDWGHonorsDocumentUnit(t *testing.T) {
	src := newCentimetrePart(t)
	if err := src.SetLengthUnit("in"); err != nil {
		t.Fatalf("SetLengthUnit in: %v", err)
	}
	sk := src.Sketches().Add(xyPlane(t))
	sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(2.54, 0))
	data, err := ExportDWG(sk, src.Units())
	if err != nil {
		t.Fatalf("ExportDWG: %v", err)
	}
	dst := newCentimetrePart(t)
	if _, err := ImportDWG(dst, data, xyPlane(t)); err != nil {
		t.Fatalf("ImportDWG: %v", err)
	}
	end := dst.Sketches().Item(0).Lines().Item(0).EndPoint().Position()
	if math.Abs(end.X-2.54) > 1e-6 {
		t.Errorf("DWG round-trip line end X = %g cm, want 2.54", end.X)
	}
}
