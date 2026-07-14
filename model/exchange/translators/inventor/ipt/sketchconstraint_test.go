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

// TestDecodePointOnLine verifies point-on-line decode on a real part. A point-on-line is a 0x3e
// coincidence node pinning a curve vertex to a line's INTERIOR (a coincidence at an endpoint is a
// plain corner). TorquimeterShaft (real_shaft_splitcluster) has two: the step-transition vertices
// lie on the interior of the shaft's vertical edge. The stepped/L-profile shafts, whose vertices
// only meet at corners, decode none — proving endpoints aren't mistaken for interior points.
func TestDecodePointOnLine(t *testing.T) {
	count := func(file string) []GeoConstraint {
		var pol []GeoConstraint
		for _, g := range DecodeGeometricConstraints(segFor(t, file)) {
			if g.Kind == GeoPointOnLine {
				pol = append(pol, g)
			}
		}
		return pol
	}
	pol := count("real_shaft_splitcluster.ipt")
	if len(pol) != 2 {
		t.Fatalf("splitcluster shaft: got %d point-on-line, want 2", len(pol))
	}
	for _, g := range pol {
		if !onSegmentInterior(g.Pt, g.L1[0], g.L1[1]) {
			t.Errorf("point %v not strictly interior to line %v", g.Pt, g.L1)
		}
	}
	for _, f := range []string{"real_shaft_stepped.ipt", "18_lprofile.ipt"} {
		if n := count(f); len(n) != 0 {
			t.Errorf("%s: got %d point-on-line, want 0 (corners are not interior points)", f, len(n))
		}
	}
}

// TestOnSegmentInterior locks the interior test used to separate point-on-line from corner
// coincidences: a point strictly between the endpoints is interior; an endpoint or an off-line
// point is not.
func TestOnSegmentInterior(t *testing.T) {
	a, b := Point2D{0, 0}, Point2D{4, 0}
	cases := []struct {
		p    Point2D
		want bool
	}{
		{Point2D{2, 0}, true},     // midpoint — interior
		{Point2D{0, 0}, false},    // endpoint a
		{Point2D{4, 0}, false},    // endpoint b
		{Point2D{2, 0.01}, false}, // off the line
		{Point2D{5, 0}, false},    // collinear but past b
	}
	for _, c := range cases {
		if got := onSegmentInterior(c.p, a, b); got != c.want {
			t.Errorf("onSegmentInterior(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

// TestDecodeCircleRelations verifies concentric and equal-radius decode. Both are a coincidence
// node (0x3e) whose two refs resolve to circles (named by entity id = centre ref + 1), the same
// shape equal-length uses for lines. They share the discriminator and are told apart by geometry:
// k2_concentric constrains two same-centre circles (0,10) r1.5 and r0.8; k2_equalr constrains two
// same-radius circles r1.5 at (0,10) and (6,10). The base, with the circles but no constraint,
// decodes to none.
func TestDecodeCircleRelations(t *testing.T) {
	kinds := func(file string) (conc, eq int) {
		for _, cr := range DecodeCircleRelations(segFor(t, file)) {
			if cr.Kind == GeoConcentric {
				conc++
			} else if cr.Kind == GeoEqualRadius {
				eq++
			}
		}
		return
	}
	if c, e := kinds("k2_base.ipt"); c != 0 || e != 0 {
		t.Errorf("k2_base: got concentric=%d equal-radius=%d, want 0/0", c, e)
	}
	if c, e := kinds("k2_concentric.ipt"); c != 1 || e != 0 {
		t.Errorf("k2_concentric: got concentric=%d equal-radius=%d, want 1/0", c, e)
	}
	if c, e := kinds("k2_equalr.ipt"); c != 0 || e != 1 {
		t.Errorf("k2_equalr: got concentric=%d equal-radius=%d, want 0/1", c, e)
	}
	// The concentric pair shares a centre; the equal-radius pair shares a radius.
	for _, cr := range DecodeCircleRelations(segFor(t, "k2_concentric.ipt")) {
		if !samePoint2D(cr.C1, cr.C2) {
			t.Errorf("concentric circles have different centres: %v %v", cr.C1, cr.C2)
		}
	}
	for _, cr := range DecodeCircleRelations(segFor(t, "k2_equalr.ipt")) {
		if absf(cr.R1-cr.R2) > 1e-6 {
			t.Errorf("equal-radius circles have different radii: %.3g %.3g", cr.R1, cr.R2)
		}
	}
}

// TestDecodeGround verifies ground decode. A ground node (t44 = groundDisc) freezes one entity;
// k2_ground grounds the horizontal line L1 = (0,0)-(3,0). The base, with the geometry but no
// ground, decodes none.
func TestDecodeGround(t *testing.T) {
	if gs := DecodeGroundConstraints(segFor(t, "k2_base.ipt")); len(gs) != 0 {
		t.Errorf("k2_base: got %d ground constraints, want 0", len(gs))
	}
	gs := DecodeGroundConstraints(segFor(t, "k2_ground.ipt"))
	if len(gs) != 1 {
		t.Fatalf("k2_ground: got %d ground constraints, want 1", len(gs))
	}
	g := gs[0]
	if g.Kind != GroundLine {
		t.Fatalf("ground kind = %v, want GroundLine", g.Kind)
	}
	// L1 spans x 0..3 at y=0.
	minX, maxX := minOf(g.Line[0].X, g.Line[1].X), maxOf(g.Line[0].X, g.Line[1].X)
	if absf(g.Line[0].Y) > 1e-6 || absf(g.Line[1].Y) > 1e-6 || minX > 1e-6 || absf(maxX-3) > 1e-6 {
		t.Errorf("grounded line = %v, want (0,0)-(3,0)", g.Line)
	}
}

// TestDecodeSymmetry verifies symmetry decode end-to-end on a part whose symmetric points are
// CURVE VERTICES (so the resolver recovers them). k2_symv has three vertical lines — axis at x=5,
// left at x=2, right at x=8 — with both endpoint pairs constrained symmetric about the axis; it
// decodes to two symmetry constraints, each mirror-valid. The base, with the geometry but no
// symmetry, decodes none. The two point refs and the axis-line ref at t44 all resolve.
func TestDecodeSymmetry(t *testing.T) {
	if ss := DecodeSymmetryConstraints(segFor(t, "k2_symv_base.ipt")); len(ss) != 0 {
		t.Errorf("k2_symv_base: got %d symmetry constraints, want 0", len(ss))
	}
	ss := DecodeSymmetryConstraints(segFor(t, "k2_symv.ipt"))
	if len(ss) != 2 {
		t.Fatalf("k2_symv: got %d symmetry constraints, want 2", len(ss))
	}
	for _, s := range ss {
		if !pointsSymmetric(s.P1, s.P2, s.Axis) {
			t.Errorf("decoded symmetry not mirror-valid: p1=%v p2=%v axis=%v", s.P1, s.P2, s.Axis)
		}
		// The axis is the vertical line at x=5.
		if !isVertical(s.Axis) || absf(s.Axis[0].X-5) > 1e-6 {
			t.Errorf("symmetry axis = %v, want vertical at x=5", s.Axis)
		}
	}
	// The symmetry nodes must NOT leak into the distance-dimension decoder (same high-bit t44
	// shape, but t44 references the axis line, not a text point).
	if dims := DecodeDistanceDimensions(segFor(t, "k2_symv.ipt")); len(dims) != 0 {
		t.Errorf("k2_symv: got %d distance dimensions, want 0 (symmetry must not leak)", len(dims))
	}
}

// TestPointsSymmetric locks the mirror predicate: two points reflect across a vertical axis when
// equidistant on opposite sides at the same height; a same-side or offset point does not.
func TestPointsSymmetric(t *testing.T) {
	axis := [2]Point2D{{5, 0}, {5, 6}} // vertical at x=5
	cases := []struct {
		p1, p2 Point2D
		want   bool
	}{
		{Point2D{2, 1}, Point2D{8, 1}, true},  // mirror about x=5
		{Point2D{2, 4}, Point2D{8, 4}, true},  // mirror at a different height
		{Point2D{2, 1}, Point2D{8, 2}, false}, // different height
		{Point2D{2, 1}, Point2D{7, 1}, false}, // not equidistant
		{Point2D{2, 1}, Point2D{2, 1}, false}, // coincident (on same side)
	}
	for _, c := range cases {
		if got := pointsSymmetric(c.p1, c.p2, axis); got != c.want {
			t.Errorf("pointsSymmetric(%v,%v) = %v, want %v", c.p1, c.p2, got, c.want)
		}
	}
}

// TestDecodeRadiusDimensions verifies radius/diameter decode for both a circle and an arc. The
// node has t44 == 0 (shared with midpoint) but its first ref is the 0x10 sentinel (not a line) and
// its second ref resolves to a circle (centre ref + 1) or arc (highest point ref + 1) entity id.
// Radius and diameter are byte-identical, so k_rad/k_dia and k_arcrad/k_arcdia each decode to one
// dimension of the curve's own radius. k_rad/k_dia dimension circle C1 = (0,10) r1.5; the arc pair
// dimension an arc of radius 2 centred (10,0). The coincidence-only bases decode none.
func TestDecodeRadiusDimensions(t *testing.T) {
	circleCase := func(file string) []RadiusDim { return DecodeRadiusDimensions(segFor(t, file)) }
	if r := circleCase("k_base.ipt"); len(r) != 0 {
		t.Errorf("k_base: got %d radius dims, want 0", len(r))
	}
	for _, file := range []string{"k_rad.ipt", "k_dia.ipt"} {
		rs := circleCase(file)
		if len(rs) != 1 {
			t.Fatalf("%s: got %d radius dims, want 1", file, len(rs))
		}
		r := rs[0]
		if r.Arc {
			t.Errorf("%s: dimensioned an arc, want a circle", file)
		}
		if absf(r.Center.X) > 1e-6 || absf(r.Center.Y-10) > 1e-6 || absf(r.Radius-1.5) > 1e-6 {
			t.Errorf("%s: circle radius dim = centre %v r %.3g, want (0,10) r 1.5", file, r.Center, r.Radius)
		}
	}
	if r := DecodeRadiusDimensions(segFor(t, "k_arcbase.ipt")); len(r) != 0 {
		t.Errorf("k_arcbase: got %d radius dims, want 0", len(r))
	}
	for _, file := range []string{"k_arcrad.ipt", "k_arcdia.ipt"} {
		rs := DecodeRadiusDimensions(segFor(t, file))
		if len(rs) != 1 {
			t.Fatalf("%s: got %d radius dims, want 1", file, len(rs))
		}
		r := rs[0]
		if !r.Arc {
			t.Errorf("%s: dimensioned a circle, want an arc", file)
		}
		if absf(r.Radius-2) > 1e-6 {
			t.Errorf("%s: arc radius dim = r %.3g, want 2", file, r.Radius)
		}
		if absf(r.Center.X-10) > 1e-6 || absf(r.Center.Y) > 1e-6 {
			t.Errorf("%s: arc centre = %v, want (10,0)", file, r.Center)
		}
	}
}

// TestDecodeAngleDimensions verifies two-line angle decode. An angle-dimension node has its first
// ref = the 0x10 sentinel and its second (+40) and t44 (+44) refs both resolving to lines. k_angle2
// dimensions L1 = (0,0)-(4,0) (horizontal) against L2 = (0,1)-(3,5) (direction (3,4)); the unsigned
// angle between them is atan2(4,3) = 53.13°. The value is read from the geometry (the label's inline
// coords are a decoy), so the base — with both lines but no dimension — decodes none.
func TestDecodeAngleDimensions(t *testing.T) {
	if a := DecodeAngleDimensions(segFor(t, "k_anglebase.ipt")); len(a) != 0 {
		t.Errorf("k_anglebase: got %d angle dims, want 0", len(a))
	}
	as := DecodeAngleDimensions(segFor(t, "k_angle2.ipt"))
	if len(as) != 1 {
		t.Fatalf("k_angle2: got %d angle dims, want 1", len(as))
	}
	if absf(as[0].Degrees-53.13010235) > 1e-4 {
		t.Errorf("angle = %.6f deg, want 53.13010 (atan2(4,3))", as[0].Degrees)
	}
	// The two dimensioned lines must be the horizontal L1 and the (3,4)-direction L2.
	if !isHorizontal(as[0].L1) && !isHorizontal(as[0].L2) {
		t.Errorf("neither dimensioned line is L1 (horizontal): %v %v", as[0].L1, as[0].L2)
	}
}

// TestLineAngleDegrees locks the angle measure (unsigned, [0,180], atan2(|cross|,dot)): perpendicular
// lines read 90°, a (3,4) direction against horizontal reads 53.13°, and the measure is orientation-
// independent (reversing a segment gives the same unsigned angle).
func TestLineAngleDegrees(t *testing.T) {
	horiz := [2]Point2D{{0, 0}, {4, 0}}
	vert := [2]Point2D{{0, 0}, {0, 3}}
	diag := [2]Point2D{{0, 0}, {3, 4}}
	if got := lineAngleDegrees(horiz, vert); absf(got-90) > 1e-6 {
		t.Errorf("perpendicular angle = %.6f, want 90", got)
	}
	if got := lineAngleDegrees(horiz, diag); absf(got-53.13010235) > 1e-4 {
		t.Errorf("(3,4) vs horizontal = %.6f, want 53.13010", got)
	}
	rev := [2]Point2D{{3, 4}, {0, 0}} // diag reversed
	if got := lineAngleDegrees(horiz, rev); absf(got-(180-53.13010235)) > 1e-4 {
		t.Errorf("reversed diag = %.6f, want 126.87 (supplementary — direction flips)", got)
	}
}

// TestDecodeOffsetDimensions verifies both offset (distance-from-line) forms. The node shares the
// angle head (r1 = 0x10 sentinel, r2 = a line) but its t44 reference is a POINT (point-to-line) or a
// PARALLEL line (line-to-line). k_offpt7 offsets line L2 (x=7) from point (0,0) → 7 cm; k_offln6
// offsets line L1 (y=0) from the parallel line L3 (y=6) → 6 cm, reduced to L3's endpoint (0,6). The
// base (same lines, no dimension) decodes none, and the parallel line-to-line offset must NOT leak
// into the angle decoder.
func TestDecodeOffsetDimensions(t *testing.T) {
	if o := DecodeOffsetDimensions(segFor(t, "k_base.ipt")); len(o) != 0 {
		t.Errorf("k_base: got %d offset dims, want 0", len(o))
	}
	pt := DecodeOffsetDimensions(segFor(t, "k_offpt7.ipt"))
	if len(pt) != 1 {
		t.Fatalf("k_offpt7: got %d offset dims, want 1", len(pt))
	}
	if absf(pt[0].Value-7) > 1e-6 {
		t.Errorf("point-to-line offset = %.4f, want 7", pt[0].Value)
	}
	if !isVertical(pt[0].Line) || absf(pt[0].Line[0].X-7) > 1e-6 {
		t.Errorf("offset reference line = %v, want vertical at x=7", pt[0].Line)
	}
	ln := DecodeOffsetDimensions(segFor(t, "k_offln6.ipt"))
	if len(ln) != 1 {
		t.Fatalf("k_offln6: got %d offset dims, want 1", len(ln))
	}
	if absf(ln[0].Value-6) > 1e-6 {
		t.Errorf("line-to-line offset = %.4f, want 6", ln[0].Value)
	}
	// The parallel line-to-line offset must not be mistaken for an angle dimension.
	if a := DecodeAngleDimensions(segFor(t, "k_offln6.ipt")); len(a) != 0 {
		t.Errorf("k_offln6: got %d angle dims, want 0 (a parallel offset is not an angle)", len(a))
	}
}

// TestPointLineDistance locks the perpendicular point-to-line measure used by offset decode.
func TestPointLineDistance(t *testing.T) {
	line := [2]Point2D{{7, 0}, {7, 4}} // vertical at x=7
	if got := pointLineDistance(Point2D{0, 0}, line); absf(got-7) > 1e-9 {
		t.Errorf("distance (0,0)→x=7 = %.6f, want 7", got)
	}
	horiz := [2]Point2D{{0, 0}, {3, 0}} // y=0
	if got := pointLineDistance(Point2D{1, 6}, horiz); absf(got-6) > 1e-9 {
		t.Errorf("distance (1,6)→y=0 = %.6f, want 6", got)
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
