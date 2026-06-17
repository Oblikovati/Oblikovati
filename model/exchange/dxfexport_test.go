// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestExportImportDXFRoundTrip builds a sketch, exports it to DXF (both versions) and
// imports it back, checking the geometry survives — the round-trip contract for DXF
// export/import, mirroring the DWG test. The exported file declares centimetre units, so the
// import scale is 1 and coordinates compare directly.
func TestExportImportDXFRoundTrip(t *testing.T) {
	for _, version := range []types.DXFVersion{types.DXFR2000, types.DXFR2018} {
		src := compdef.NewPartComponentDefinition()
		sk := src.Sketches().Add(xyPlane(t))
		line := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 5))
		circle := sk.Circles().AddByCenterRadius(gmath.P2(3, 4), 2.5)
		arc := sk.Arcs().AddByCenterStartEnd(gmath.P2(1, 1), gmath.P2(4, 1), gmath.P2(1, 4), true)
		spline := sk.Splines().AddByControlPoints([]gmath.Point2{{X: 0, Y: 0}, {X: 1, Y: 2}, {X: 3, Y: 2}, {X: 4, Y: 0}}, false)

		data, n, err := ExportDXF(sk, version)
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
