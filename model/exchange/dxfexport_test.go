// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestExportImportDXFRoundTrip builds a sketch, exports it to DXF (both versions) and
// imports it back, checking the geometry survives — the round-trip contract for DXF
// export/import, mirroring the DWG test. The exported file declares centimetre units, so the
// import scale is 1 and coordinates compare directly.
func TestExportImportDXFRoundTrip(t *testing.T) {
	for _, version := range []types.DXFVersion{types.DXFR2000, types.DXFR2018} {
		src := newCentimetrePart(t)
		sk := src.Sketches().Add(xyPlane(t))
		line := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 5))
		circle := sk.Circles().AddByCenterRadius(gmath.P2(3, 4), 2.5)
		arc := sk.Arcs().AddByCenterStartEnd(gmath.P2(1, 1), gmath.P2(4, 1), gmath.P2(1, 4), true)
		spline := sk.Splines().AddByControlPoints([]gmath.Point2{{X: 0, Y: 0}, {X: 1, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 0}}, false)

		data, n, err := ExportDXF(sk, version, src.Units())
		if err != nil {
			t.Fatalf("ExportDXF(%s): %v", version, err)
		}
		if n != 4 {
			t.Errorf("%s: exported %d curves, want 4", version, n)
		}
		dst := newCentimetrePart(t)
		res, err := ImportDXF(dst, data, xyPlane(t))
		if err != nil {
			t.Fatalf("ImportDXF(%s): %v", version, err)
		}
		if res.Is3D {
			t.Fatalf("%s: round-trip imported as 3D", version)
		}
		out := dst.Sketches().Item(0)
		if out.Lines().Count() != 1 || out.Circles().Count() != 1 || out.Arcs().Count() != 1 || out.Splines().Count() != 1 {
			t.Fatalf("%s: counts L=%d C=%d A=%d S=%d", version,
				out.Lines().Count(), out.Circles().Count(), out.Arcs().Count(), out.Splines().Count())
		}
		wantPt(t, "line start", out.Lines().Item(0).StartPoint().Position(), line.StartPoint().Position())
		wantPt(t, "line end", out.Lines().Item(0).EndPoint().Position(), line.EndPoint().Position())
		wantPt(t, "circle centre", out.Circles().Item(0).CenterPoint().Position(), circle.CenterPoint().Position())
		wantScalar(t, "circle radius", float64(out.Circles().Item(0).Radius), float64(circle.Radius))
		wantScalar(t, "arc radius", float64(out.Arcs().Item(0).Radius()), float64(arc.Radius()))
		wantEndpoints(t, "arc", out.Arcs().Item(0), arc)
		rs := out.Splines().Item(0)
		if rs.PointCount() != spline.PointCount() {
			t.Fatalf("%s: spline points = %d, want %d", version, rs.PointCount(), spline.PointCount())
		}
		for i := range spline.Points {
			wantPt(t, "spline ctrl", rs.Points[i].Position(), spline.Points[i].Position())
		}
	}
}

// TestExportDXFHonorsDocumentUnit checks a sketch exported from an inch document
// declares inch $INSUNITS and scales coordinates from centimetres, and that the
// import (which scales $INSUNITS → cm) recovers the original size (#146).
func TestExportDXFHonorsDocumentUnit(t *testing.T) {
	src := newCentimetrePart(t)
	if err := src.SetLengthUnit("in"); err != nil {
		t.Fatalf("SetLengthUnit in: %v", err)
	}
	sk := src.Sketches().Add(xyPlane(t))
	sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(2.54, 0)) // 2.54 cm = 1 inch

	data, _, err := ExportDXF(sk, types.DXFR2018, src.Units())
	if err != nil {
		t.Fatalf("ExportDXF: %v", err)
	}
	// $INSUNITS = 1 (inches); the 2.54 cm edge is written as 1.0 inch.
	if !strings.Contains(string(data), "$INSUNITS") || !strings.Contains(string(data), "\n1\n") {
		t.Error("inch export must declare $INSUNITS = 1 (inches)")
	}

	dst := newCentimetrePart(t)
	if _, err := ImportDXF(dst, data, xyPlane(t)); err != nil {
		t.Fatalf("ImportDXF: %v", err)
	}
	// The inch file is scaled back to centimetres on import: 1 inch → 2.54 cm.
	end := dst.Sketches().Item(0).Lines().Item(0).EndPoint().Position()
	if math.Abs(end.X-2.54) > 1e-6 {
		t.Errorf("round-trip line end X = %g cm, want 2.54 (1 inch)", end.X)
	}
}

// TestDocumentDrawingUnit pins the $INSUNITS + cm→unit scale mapping, including the
// fallback for a unit with no $INSUNITS code.
func TestDocumentDrawingUnit(t *testing.T) {
	cases := map[string]struct {
		ins   int
		scale float64
	}{
		"mm": {4, 10}, "cm": {5, 1}, "m": {6, 0.01}, "in": {1, 1 / 2.54}, "ft": {2, 1 / 30.48},
	}
	for unit, want := range cases {
		p := newCentimetrePart(t)
		if err := p.SetLengthUnit(unit); err != nil {
			t.Fatalf("SetLengthUnit(%q): %v", unit, err)
		}
		ins, scale := documentDrawingUnit(p.Units())
		if ins != want.ins || math.Abs(scale-want.scale) > 1e-9 {
			t.Errorf("documentDrawingUnit(%q) = (%d, %g), want (%d, %g)", unit, ins, scale, want.ins, want.scale)
		}
	}
	// A unit with no $INSUNITS code falls back to centimetres (scale 1).
	p := newCentimetrePart(t)
	if err := p.SetLengthUnit("km"); err != nil {
		t.Fatalf("SetLengthUnit(km): %v", err)
	}
	if ins, scale := documentDrawingUnit(p.Units()); ins != 5 || scale != 1 {
		t.Errorf("documentDrawingUnit(km) = (%d, %g), want (5, 1) fallback", ins, scale)
	}
}

// TestExportDXFFileAndDWGFile cover the file-writing wrappers (and writeDXFFile's
// happy path): they write a non-empty file for the active sketch.
func TestExportDXFFileAndDWGFile(t *testing.T) {
	src := newCentimetrePart(t)
	sk := src.Sketches().Add(xyPlane(t))
	sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 5))

	dxfPath := filepath.Join(t.TempDir(), "s.dxf")
	if n, err := ExportDXFFile(sk, dxfPath, types.DXFR2018, src.Units()); err != nil || n != 1 {
		t.Fatalf("ExportDXFFile = (%d, %v), want 1 curve", n, err)
	}
	if fi, err := os.Stat(dxfPath); err != nil || fi.Size() == 0 {
		t.Errorf("DXF file not written: %v", err)
	}

	dwgPath := filepath.Join(t.TempDir(), "s.dwg")
	if err := ExportDWGFile(sk, dwgPath, src.Units()); err != nil {
		t.Fatalf("ExportDWGFile: %v", err)
	}
	if fi, err := os.Stat(dwgPath); err != nil || fi.Size() == 0 {
		t.Errorf("DWG file not written: %v", err)
	}
}
