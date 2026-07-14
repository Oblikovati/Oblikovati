// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"os"
	"testing"
)

// segFor opens a testdata part and returns its PmDCSegment.
func segFor(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile("../testdata/" + file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	d, err := Open(data)
	if err != nil {
		t.Fatalf("open %s: %v", file, err)
	}
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		t.Fatalf("%s: no PmDCSegment", file)
	}
	return seg
}

// TestDecodeGeometricConstraintsSingleVariable verifies each single-variable corpus part decodes to
// exactly the one geometric constraint it was authored with (plus none for the coincidence-only
// base) — the ground truth that pins the 2027 t44 discriminator and the geometry-based H/V and
// parallel/perpendicular classification.
func TestDecodeGeometricConstraintsSingleVariable(t *testing.T) {
	cases := []struct {
		file                    string
		kind                    GeoKind
		horiz, vert, para, perp int
	}{
		{"k_base.ipt", 0, 0, 0, 0, 0}, // only coincidences → no geometric constraints
		{"k_horiz.ipt", GeoHorizontal, 1, 0, 0, 0},
		{"k_vert.ipt", GeoVertical, 0, 1, 0, 0},
		{"k_para.ipt", GeoParallel, 0, 0, 1, 0},
		{"k_perp.ipt", GeoPerpendicular, 0, 0, 0, 1},
	}
	for _, tc := range cases {
		gcs := DecodeGeometricConstraints(segFor(t, tc.file))
		var h, v, pa, pe int
		for _, g := range gcs {
			switch g.Kind {
			case GeoHorizontal:
				h++
			case GeoVertical:
				v++
			case GeoParallel:
				pa++
			case GeoPerpendicular:
				pe++
			}
		}
		if h != tc.horiz || v != tc.vert || pa != tc.para || pe != tc.perp {
			t.Errorf("%s: got H=%d V=%d ∥=%d ⊥=%d, want H=%d V=%d ∥=%d ⊥=%d",
				tc.file, h, v, pa, pe, tc.horiz, tc.vert, tc.para, tc.perp)
		}
	}
}

// TestDecodeCollinearAndEqualLength verifies the batch-2 line constraints. Collinear shares the
// line-relate discriminator (0x00400000) with parallel/perpendicular and IS parallel, so it must
// be classified by geometry (both lines on one infinite line) BEFORE parallel — otherwise a
// collinear pair would mis-decode as parallel. Equal-length uses the coincidence discriminator
// (0x0000003e) but references two lines (not the usual line↔endpoint), and is accepted only when
// the lengths actually match. The base decodes to neither.
func TestDecodeCollinearAndEqualLength(t *testing.T) {
	count := func(file string) (collinear, equal, para int) {
		for _, g := range DecodeGeometricConstraints(segFor(t, file)) {
			switch g.Kind {
			case GeoCollinear:
				collinear++
			case GeoEqualLength:
				equal++
			case GeoParallel:
				para++
			}
		}
		return
	}
	if c, e, p := count("k2_base.ipt"); c != 0 || e != 0 {
		t.Errorf("k2_base: got collinear=%d equal=%d, want 0/0 (para=%d)", c, e, p)
	}
	// Collinear must NOT leak into the parallel bucket.
	if c, e, p := count("k2_collinear.ipt"); c != 1 || e != 0 || p != 0 {
		t.Errorf("k2_collinear: got collinear=%d equal=%d parallel=%d, want 1/0/0", c, e, p)
	}
	if c, e, _ := count("k2_equall.ipt"); c != 0 || e != 1 {
		t.Errorf("k2_equall: got collinear=%d equal=%d, want 0/1", c, e)
	}
	// The collinear constraint binds the two collinear lines (both on y=0).
	for _, g := range DecodeGeometricConstraints(segFor(t, "k2_collinear.ipt")) {
		if g.Kind == GeoCollinear && !linesCollinear(g.L1, g.L2) {
			t.Errorf("collinear bound non-collinear lines %v %v", g.L1, g.L2)
		}
	}
}

// TestDecodeMidpoint verifies the midpoint constraint (disc 0x00000000, line + a point at its
// midpoint). It shares the discriminator with radius/diameter dimensions, but those reference the
// 0x10 sentinel (not a line) so they don't resolve here. The pinned point's coordinate is computed
// from the resolved line — k2_midpoint pins a point to L1 (0,0)-(3,0), whose midpoint is (1.5,0).
// The base decodes to no midpoint (guards against a stray disc-0 node firing).
func TestDecodeMidpoint(t *testing.T) {
	countMid := func(file string) (mids []Point2D) {
		for _, g := range DecodeGeometricConstraints(segFor(t, file)) {
			if g.Kind == GeoMidpoint {
				mids = append(mids, g.Pt)
			}
		}
		return
	}
	if m := countMid("k2_base.ipt"); len(m) != 0 {
		t.Errorf("k2_base: got %d midpoints, want 0", len(m))
	}
	m := countMid("k2_midpoint.ipt")
	if len(m) != 1 {
		t.Fatalf("k2_midpoint: got %d midpoints, want 1", len(m))
	}
	if absf(m[0].X-1.5) > 1e-6 || absf(m[0].Y) > 1e-6 {
		t.Errorf("midpoint = %v, want (1.5,0) (midpoint of L1 (0,0)-(3,0))", m[0])
	}
}

// TestDecodeTangent verifies the line↔circle tangent constraint. Its node uses a non-standard
// layout — the line ref is at +32 (a two-ref constraint holds the 0x10 sentinel there) and the
// circle ref is the +44 discriminator word (high bit set), naming the circle by its entity id
// (centre ref + 1). k2_tangent makes the line y=8.5 tangent to the circle centred (0,10) r=1.5
// (their gap is exactly 1.5). Accepted only when the line is genuinely tangent, so the base — with
// the same line and circle but no constraint — decodes to none.
func TestDecodeTangent(t *testing.T) {
	if ts := DecodeTangentConstraints(segFor(t, "k2_base.ipt")); len(ts) != 0 {
		t.Errorf("k2_base: got %d tangents, want 0", len(ts))
	}
	ts := DecodeTangentConstraints(segFor(t, "k2_tangent.ipt"))
	if len(ts) != 1 {
		t.Fatalf("k2_tangent: got %d tangents, want 1", len(ts))
	}
	tc := ts[0]
	if absf(tc.Center.X) > 1e-6 || absf(tc.Center.Y-10) > 1e-6 || absf(tc.Radius-1.5) > 1e-6 {
		t.Errorf("tangent circle = centre %v r %.3g, want (0,10) r 1.5", tc.Center, tc.Radius)
	}
	if !lineTangentToCircle(tc.Line, tc.Center, tc.Radius) {
		t.Errorf("decoded tangent line %v is not tangent to the circle", tc.Line)
	}
}

// TestHorizontalConstraintBindsRightLine checks the decoded horizontal constraint names the
// horizontal line (0,0)-(3,0), proving the line resolves to its endpoint coordinates.
func TestHorizontalConstraintBindsRightLine(t *testing.T) {
	gcs := DecodeGeometricConstraints(segFor(t, "k_horiz.ipt"))
	if len(gcs) != 1 || gcs[0].Kind != GeoHorizontal {
		t.Fatalf("want one horizontal constraint, got %+v", gcs)
	}
	l := gcs[0].L1
	if !isHorizontal(l) {
		t.Errorf("horizontal constraint bound to non-horizontal line %v", l)
	}
	// the authored horizontal line spans x 0..3 at y=0
	minX, maxX := minOf(l[0].X, l[1].X), maxOf(l[0].X, l[1].X)
	if absf(l[0].Y) > 1e-6 || minX > 1e-6 || absf(maxX-3) > 1e-6 {
		t.Errorf("horizontal line = %v, want (0,0)-(3,0)", l)
	}
}

// TestDecodeDistanceDimensions verifies the point-to-point distance dimensions decode with the
// right endpoints and value: a horizontal 7 cm dimension (k_disth7) and a vertical 4 cm one
// (k_distv4), and none on the coincidence-only base.
func TestDecodeDistanceDimensions(t *testing.T) {
	cases := []struct {
		file  string
		value float64
	}{
		{"k_disth7.ipt", 7},
		{"k_distv4.ipt", 4},
	}
	for _, tc := range cases {
		dims := DecodeDistanceDimensions(segFor(t, tc.file))
		found := false
		for _, dm := range dims {
			if absf(dm.Value-tc.value) < 1e-6 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no distance dimension of value %.1f (got %+v)", tc.file, tc.value, dims)
		}
	}
	if dims := DecodeDistanceDimensions(segFor(t, "k_base.ipt")); len(dims) != 0 {
		t.Errorf("k_base has %d distance dimensions, want 0 (coincidence-only)", len(dims))
	}
}

// TestDecodeRevolveRadii checks the stepped shaft's radius dimensions decode to the x-positions of
// its dimensioned vertical edges (0.5 and 1.5 cm from the centreline) — the value-matched-to-edge
// gate that keeps a coordinate leak from being taken as a radius.
func TestDecodeRevolveRadii(t *testing.T) {
	radii := DecodeRevolveRadii(segFor(t, "real_shaft_stepped.ipt"))
	want := map[int64]bool{5000: false, 15000: false}
	for _, r := range radii {
		want[int64(r*1e4)] = true
	}
	for x, seen := range want {
		if !seen {
			t.Errorf("no revolve radius at x=%.1f cm (got %v)", float64(x)/1e4, radii)
		}
	}
}

// TestDecodeAxialLengths checks the stepped shaft's axial step-length dimensions decode to the
// vertical gaps between its horizontal edges that match a driving parameter (0.2 and 1.65 cm), and
// that a non-revolve part decodes none (the gate that stopped the L-profile over-constraining).
func TestDecodeAxialLengths(t *testing.T) {
	got := map[int64]bool{}
	for _, ax := range DecodeAxialLengths(segFor(t, "real_shaft_stepped.ipt")) {
		got[int64(ax.Value*1e4)] = true
	}
	for _, want := range []int64{2000, 16500} {
		if !got[want] {
			t.Errorf("no axial length of %.2f cm (got %v)", float64(want)/1e4, got)
		}
	}
	if ax := DecodeAxialLengths(segFor(t, "18_lprofile.ipt")); len(ax) != 0 {
		t.Errorf("L-profile (an extrude) decoded %d axial lengths, want 0 (revolve-only)", len(ax))
	}
}

func minOf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxOf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
