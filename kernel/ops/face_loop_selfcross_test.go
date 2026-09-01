// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSelfCrossingFaceLoopsMeasuresThePinchedOffArea is the falsifiable guard for the detector.
//
// It builds complex/D8's defect in MINIATURE and from closed form: a cylindrical band whose far
// boundary overshoots the band's own u=0 ruling by d and comes back along the top rim, exactly as
// D8's far-end trim overshoots its corner round's ruling by 0.2527 rad. The lobe that overshoot
// pinches off is a triangle of area h·d²/(2(W+d)) in the developed metric — no oracle needed — and
// that is the quantity the shipped face's shoelace is wrong by.
//
// Falsify by relaxing segmentsCross to a non-strict test, or by dropping the crossing point from
// pinchedOffArea: the area then stops matching the closed form and this goes RED.
func TestSelfCrossingFaceLoopsMeasuresThePinchedOffArea(t *testing.T) {
	t.Parallel()
	const r, w, l, h, d = 24.0, 30.0, 100.0, 8.0, 6.0
	body := lobedBandBody(t, r, w, l, h, d)
	bad := SelfCrossingFaceLoops(body, PropertyQuality())
	if len(bad) != 1 {
		t.Fatalf("a band whose boundary crosses its own ruling must report exactly one loop, got %d", len(bad))
	}
	want := h * d * d / (2 * (w + d))
	if rel := stdmath.Abs(bad[0].Area-want) / want; rel > 1e-9 { // tol:numeric (relative area fraction)
		t.Errorf("pinched-off area %.10g, closed form %.10g (rel %.4g)", bad[0].Area, want, rel)
	}
}

// TestSelfCrossingFaceLoopsPassesASimpleTrim pins the other side: the SAME band with no overshoot is
// a simple polygon and must not be reported. Without this the detector could pass the test above by
// reporting every face.
func TestSelfCrossingFaceLoopsPassesASimpleTrim(t *testing.T) {
	t.Parallel()
	const r, w, l, h = 24.0, 30.0, 100.0, 8.0
	body := lobedBandBody(t, r, w, l, h, 0)
	if bad := SelfCrossingFaceLoops(body, PropertyQuality()); len(bad) != 0 {
		t.Errorf("a simple band trim must not be reported, got %d: %+v", len(bad), bad)
	}
}

// lobedBandBody builds a radius-r cylindrical band, in metric (u,v) corners (0,0) → (w,0) → (w,l−h)
// → (−d,l) → (0,l) → back down the u=0 ruling. With d > 0 the slanted far boundary crosses that
// closing ruling and pinches off the triangle the guard measures; with d = 0 the same trim is simple.
func lobedBandBody(t *testing.T, r, w, l, h, d float64) *topo.Body {
	t.Helper()
	cyl, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	corners := [][2]float64{{0, 0}, {w / r, 0}, {w / r, l - h}, {-d / r, l}, {0, l}}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("sx", "body", 0)))
	lin := topo.NewLineage(topo.Tok("sx", "x", 0))
	verts := make([]*topo.Vertex, len(corners))
	for i, c := range corners {
		verts[i] = bld.AddVertex(cyl.PointAt(c[0], c[1]), lin)
	}
	uses := make([]topo.Use, len(corners))
	for i, c := range corners {
		j := (i + 1) % len(corners)
		uses[i] = topo.Fwd(bld.AddEdge(lobedBandEdge(t, cyl, c, corners[j]), verts[i], verts[j], lin))
	}
	bld.AddFace(cyl, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// lobedBandEdge is the curve between two (u,v) corners: the exact iso-v rim arc when they share v,
// else the straight chord between them (an axial ruling shares u; the slanted far boundary is the
// piece whose two endpoints are all the trim gives, which is exactly D8's discretized situation).
func lobedBandEdge(t *testing.T, cyl geom.Cylinder, a, b [2]float64) geom.Curve3 {
	t.Helper()
	if a[1] != b[1] {
		return geom.NewLineSegment(cyl.PointAt(a[0], a[1]), cyl.PointAt(b[0], b[1]))
	}
	arc, err := geom.NewArc3d(math.P3(0, 0, math.Scalar(a[1])), math.V3(0, 0, 1), math.V3(1, 0, 0), cyl.Radius, a[0], b[0]-a[0])
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	return arc
}

// TestSelfCrossingFaceLoopsMeasuresTheCrossingPairsOwnFidelity is the ADDED measurement's own guard,
// gated on a CLOSED FORM. On the lobed band the crossing pair is the u = 0 closing ruling (chart length
// = chord exactly) and the slant that overshoots it, which spans Δu = (w+d)/r about the cylinder while
// rising h: chart hypot(w+d, h), 3D chord hypot(2r·sin(Δu/2), h). The worse of the two is the slant's,
// 1.0948765… — well inside the half-turn cut, so the crossing IS measured on the surface and its Area
// may be quoted as one.
//
// Falsify by measuring the ratio against the loop's chart diagonal, or against a segment index other
// than the crossing pair's: the value stops matching this closed form.
func TestSelfCrossingFaceLoopsMeasuresTheCrossingPairsOwnFidelity(t *testing.T) {
	t.Parallel()
	const r, w, l, h, d = 24.0, 30.0, 100.0, 8.0, 6.0
	bad := SelfCrossingFaceLoops(lobedBandBody(t, r, w, l, h, d), PropertyQuality())
	if len(bad) != 1 {
		t.Fatalf("a band whose boundary crosses its own ruling must report exactly one loop, got %d", len(bad))
	}
	du := (w + d) / r
	want := stdmath.Hypot(w+d, h) / stdmath.Hypot(2*r*stdmath.Sin(du/2), h)
	if rel := stdmath.Abs(bad[0].ChartChordRatio-want) / want; rel > 1e-9 { // tol:numeric (ratio)
		t.Errorf("crossing pair chart/chord %.12g, closed form %.12g (rel %.4g)", bad[0].ChartChordRatio, want, rel)
	}
	if !bad[0].ChartFaithful() {
		t.Errorf("a %.6g ratio is inside the half-turn cut %.6g and must read faithful",
			bad[0].ChartChordRatio, selfCrossChartFaithfulRatio)
	}
}

// TestChartFaithfulCutIsTheHalfTurn pins selfCrossChartFaithfulRatio on its DERIVATION rather than on
// the corpus numbers it happens to separate: the ratio θ/(2 sin(θ/2)) of a segment spanning angle θ
// about a periodic direction equals the constant exactly at θ = π, the span past which the 3D chord
// starts shrinking while the chart keeps growing. Falsify by re-tuning the constant to fit the corpus
// and this goes RED, which is the point — it is a closed form, not a calibration.
func TestChartFaithfulCutIsTheHalfTurn(t *testing.T) {
	t.Parallel()
	atHalfTurn := stdmath.Pi / (2 * stdmath.Sin(stdmath.Pi/2))
	if stdmath.Abs(selfCrossChartFaithfulRatio-atHalfTurn) > 1e-15 { // tol:numeric (ratio)
		t.Errorf("selfCrossChartFaithfulRatio is %.17g, the half-turn ratio θ/(2 sin(θ/2)) at θ=π is %.17g",
			selfCrossChartFaithfulRatio, atHalfTurn)
	}
	for _, theta := range []float64{4.42, 6.20} { // complex/F2's two measured spans
		if got := theta / (2 * stdmath.Sin(theta/2)); got <= selfCrossChartFaithfulRatio {
			t.Errorf("a %.2f rad segment measures ratio %.4g, which must exceed the half-turn cut %.4g",
				theta, got, selfCrossChartFaithfulRatio)
		}
	}
}
