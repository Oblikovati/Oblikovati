// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The curved-survivor rim carry regression (curved-host-collapse-rootcause.md). transformLoop's ENDS
// branch used to hard-code the OUTGOING survivor edge's curve to nil (a straight chord), collapsing a
// curved wall's rim arc to a chord across the face. These white-box tests prove the two required
// behaviours on named fixtures: a CURVED survivor rim is carried and trimmed to its retained sub-arc
// (the wall keeps its arc), and a STRAIGHT (B1/B9-style) survivor stays nil, byte-identical.

// rimCircleFixture is a named fake of a partial cylinder's TOP rim circle (radius 50 in the z=100 plane,
// axis +z) — the survivor edge whose arc the ENDS branch used to drop. sectorSweep is the sector's angular
// span (90° for a B1-style minor rim, 270° for a B5-style major rim); the returned parent Arc3d covers the
// whole sector, and (tIn, tOut) are the two corner tangent points that replaced its endpoints — placed OFF
// the circle at radius √(50²+r²) exactly as the fillet's cap-contact points sit (the root-cause 50.99 receipt).
type rimCircleFixture struct {
	sectorSweep float64
	parent      geom.Arc3d
	tIn, tOut   math.Point3
}

func newRimCircleFixture(t *testing.T, sectorDeg, filletR float64) rimCircleFixture {
	t.Helper()
	sweep := sectorDeg * stdmath.Pi / 180
	center := math.P3(0, 0, 100)
	parent, err := geom.NewArc3d(center, math.V3(0, 0, 1), math.V3(1, 0, 0), 50, 0, sweep)
	if err != nil {
		t.Fatalf("build parent rim arc (sector %.0f°): %v", sectorDeg, err)
	}
	// The tangent points sit a small angular inset from the sector ends, pushed radially OUT to the
	// cap-contact radius √(50²+r²) so they are genuinely off the rim circle (the defect's premise).
	inset := 0.13 // ~7.4° inset, comfortably inside any sector so the retained span stays a proper arc
	off := stdmath.Sqrt(50*50 + filletR*filletR)
	return rimCircleFixture{
		sectorSweep: sweep,
		parent:      parent,
		tIn:         offCirclePoint(center, off, inset),       // near the start end
		tOut:        offCirclePoint(center, off, sweep-inset), // near the far end
	}
}

// offCirclePoint is a point at the given angle on the z=100 plane, at radius `rad` (> 50 ⇒ off the rim circle).
func offCirclePoint(center math.Point3, rad, ang float64) math.Point3 {
	return math.P3(float64(center.X)+rad*stdmath.Cos(ang), float64(center.Y)+rad*stdmath.Sin(ang), float64(center.Z))
}

// TestAddCornerRoundCarriesCurvedSurvivor proves addCornerRound carries a CURVED (Arc3d) outgoing survivor
// on the tOut segment and reports true, while a STRAIGHT (nil) survivor stays nil and reports false — the
// byte-identity guarantee for the whole planar corpus + the 24 fingerprint pins.
func TestAddCornerRoundCarriesCurvedSurvivor(t *testing.T) {
	t.Parallel()
	fix := newRimCircleFixture(t, 270, 10)
	c := corner{ta: fix.tIn, tb: fix.tOut, mid: fix.tIn.Midpoint(fix.tOut)} // no chords ⇒ the constant-fillet arc branch

	var curved filletLoop
	if !addCornerRound(&curved, c, fix.tIn, fix.tOut, fix.parent) {
		t.Fatal("addCornerRound with a curved survivor must report true (it carries the parent rim arc)")
	}
	if _, ok := curved.curves[len(curved.curves)-1].(geom.Arc3d); !ok {
		t.Fatalf("curved survivor: tOut segment curve = %T, want geom.Arc3d (the carried rim arc)", curved.curves[len(curved.curves)-1])
	}

	var straight filletLoop
	if addCornerRound(&straight, c, fix.tIn, fix.tOut, nil) {
		t.Fatal("addCornerRound with a straight survivor must report false (nothing to trim)")
	}
	if last := straight.curves[len(straight.curves)-1]; last != nil {
		t.Fatalf("straight survivor: tOut segment curve = %v, want nil (byte-identical to the pre-fix planar path)", last)
	}
}

// TestTrimCarriedRimArcMajor proves a >π retained span (a B5-style 270° sector rim) is trimmed to a MAJOR
// sub-arc (|sweep| > π) whose endpoints land back ON the rim circle (radius 50) — never snapping to the
// minor complement, which is the collapse the projection + subArcMajor reuse prevents.
func TestTrimCarriedRimArcMajor(t *testing.T) {
	t.Parallel()
	fix := newRimCircleFixture(t, 270, 10)
	fl := loopWithCarriedRim(fix)
	trimCarriedRimArcs(&fl, []int{0})

	arc, ok := fl.curves[0].(geom.Arc3d)
	if !ok {
		t.Fatalf("major rim: trimmed curve = %T, want geom.Arc3d", fl.curves[0])
	}
	if stdmath.Abs(arc.SweepAngle) <= stdmath.Pi {
		t.Fatalf("major rim: trimmed sweep %.1f° collapsed to the minor complement, want > 180°", arc.SweepAngle*180/stdmath.Pi)
	}
	assertArcEndpointsOnCircle(t, "major", arc)
}

// TestTrimCarriedRimArcLargeMinor proves a LARGE minor retained span (>π/2, <π — an E1/E2-style sphere
// meridian rim) carries a MINOR sub-arc (|sweep| < π) re-fit through its shorter-arc midpoint: the in-sector
// side, and NOT dropped by the quadrant gate (which a major-only gate would wrongly do, un-greening E1/E2).
func TestTrimCarriedRimArcLargeMinor(t *testing.T) {
	t.Parallel()
	fix := newRimCircleFixture(t, 170, 10) // retained ~155° — comfortably above the π/2 gate, below π
	fl := loopWithCarriedRim(fix)
	trimCarriedRimArcs(&fl, []int{0})

	arc, ok := fl.curves[0].(geom.Arc3d)
	if !ok {
		t.Fatalf("large-minor rim: trimmed curve = %T, want geom.Arc3d (a >π/2 rim must be carried)", fl.curves[0])
	}
	if stdmath.Abs(arc.SweepAngle) >= stdmath.Pi {
		t.Fatalf("large-minor rim: trimmed sweep %.1f° over-spanned, want < 180° (the in-sector side)", arc.SweepAngle*180/stdmath.Pi)
	}
	assertArcEndpointsOnCircle(t, "large-minor", arc)
}

// TestTrimCarriedRimSmallMinorStaysChord proves a SMALL minor retained span (≤π/2 — a B1/B9-style 90° sector
// rim, retained ~75°) is RESTORED to a straight chord (nil), byte-identical to the pre-fix planar path, so
// the whole planar corpus + the fingerprint pins are untouched (the reviewer's Critical: B1/B9 must not drift).
func TestTrimCarriedRimSmallMinorStaysChord(t *testing.T) {
	t.Parallel()
	fix := newRimCircleFixture(t, 90, 10) // retained ~75° — below the π/2 quadrant gate
	fl := loopWithCarriedRim(fix)
	trimCarriedRimArcs(&fl, []int{0})

	if fl.curves[0] != nil {
		t.Fatalf("small-minor rim: curve = %v, want nil (a ≤π/2 rim keeps its faithful base chord, byte-identical)", fl.curves[0])
	}
}

// loopWithCarriedRim builds the two-point filletLoop addCornerRound would leave for a curved survivor: the
// tOut point (index 0) carrying the FULL parent rim arc, and the far corner tangent point (index 1). Both
// points are the OFF-circle tangent points, so the trim must project them before measuring the span.
func loopWithCarriedRim(fix rimCircleFixture) filletLoop {
	var fl filletLoop
	fl.add(fix.tOut, fix.parent) // segment 0: tOut → tIn, carrying the parent arc to be trimmed
	fl.add(fix.tIn, nil)
	return fl
}

// assertArcEndpointsOnCircle checks the trimmed sub-arc's own endpoints lie on the parent rim circle
// (radius 50) — proving the off-circle tangent points were projected before the sub-arc was built.
func assertArcEndpointsOnCircle(t *testing.T, label string, arc geom.Arc3d) {
	t.Helper()
	for _, p := range []math.Point3{arc.PointAt(0), arc.PointAt(1)} {
		if d := stdmath.Abs(float64(p.DistanceTo(arc.Center)) - arc.Radius); d > 1e-6*arc.Radius {
			t.Fatalf("%s rim: sub-arc endpoint %v is %.4g off the radius-%.0f circle, want on it (projection failed)", label, p, d, arc.Radius)
		}
	}
}

// TestProjectOntoArcCircle proves an off-circle point (the fillet's cap-contact tangent point at radius
// √(50²+10²)=50.99, the root-cause receipt) is dropped exactly onto the radius-50 rim circle.
func TestProjectOntoArcCircle(t *testing.T) {
	t.Parallel()
	fix := newRimCircleFixture(t, 270, 10)
	got := projectOntoArcCircle(fix.parent, fix.tOut)
	if d := stdmath.Abs(float64(got.DistanceTo(fix.parent.Center)) - fix.parent.Radius); d > 1e-9 {
		t.Fatalf("projected point is %.4g off the radius-50 circle, want 0", d)
	}
	if float64(got.Z) != 100 {
		t.Fatalf("projected z = %v, want 100 (stays in the rim plane)", got.Z)
	}
}

// The subs-branch survivor-arc carry regression (i3-recon-rootcause.md). transformLoop's `subs` branch
// (a fillet A/B-corner vertex pulled back to its tangent point) used to hard-code the LEAVING survivor
// edge's curve to nil — chording a curved host rim (I3's r=300 annular-sector outer arc) to a straight
// line, slicing −63% off the host and folding the neighbour cone. These white-box tests prove the two
// required behaviours on named fixtures: a CURVED survivor rim leaving the tangent point is carried, a
// STRAIGHT survivor stays nil (byte-identical), and the carry fires ONLY when chording erases a MATERIAL
// circular segment (I3's large host rim), never a minor one (N5's small boss rim, kept byte-identical).

// leavingEdgeUse builds a one-edge face carrying `curve` and returns the edge use LEAVING its start vertex
// — the `u` transformLoop's subs branch passes to addSubstVertex. A plane surface and single-edge loop are
// enough: addSubstVertex only reads survivorCurve(u) = u.Edge().Geometry().
func leavingEdgeUse(t *testing.T, curve geom.Curve3, start, end math.Point3) *topo.EdgeUse {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage())
	v0 := bld.AddVertex(start, topo.NewLineage())
	v1 := bld.AddVertex(end, topo.NewLineage())
	e := bld.AddEdge(curve, v0, v1, topo.NewLineage())
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("build fixture plane: %v", err)
	}
	f := bld.AddFace(plane, topo.NewLineage(), topo.OuterLoop(topo.Fwd(e)))
	return f.Loops()[0].EdgeUses()[0]
}

// TestAddSubstVertexCarriesCurvedSurvivor proves addSubstVertex carries a CURVED (Arc3d) leaving survivor
// on the tangent-point segment and reports its index, while a STRAIGHT (LineSegment) leaving edge stays
// nil and reports −1 — the byte-identity guarantee for every straight-survivor subs case (the whole planar
// corpus + the fingerprint pins).
func TestAddSubstVertexCarriesCurvedSurvivor(t *testing.T) {
	t.Parallel()
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 300, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("build survivor arc: %v", err)
	}
	tan := math.P3(-190, 300, 0) // the fillet corner pulled back to its tangent point

	var curved filletLoop
	uArc := leavingEdgeUse(t, arc, arc.PointAt(0), arc.PointAt(1))
	if idx := addSubstVertex(&curved, tan, uArc); idx != 0 {
		t.Fatalf("curved survivor: addSubstVertex returned idx %d, want 0 (it carries the parent rim arc)", idx)
	}
	if _, ok := curved.curves[0].(geom.Arc3d); !ok {
		t.Fatalf("curved survivor: carried curve = %T, want geom.Arc3d", curved.curves[0])
	}

	var straight filletLoop
	seg := geom.NewLineSegment(math.P3(-200, 200, 0), math.P3(-200, 300, 0))
	uLine := leavingEdgeUse(t, seg, seg.PointAt(0), seg.PointAt(1))
	if idx := addSubstVertex(&straight, tan, uLine); idx != -1 {
		t.Fatalf("straight survivor: addSubstVertex returned idx %d, want -1 (nothing to carry)", idx)
	}
	if straight.curves[0] != nil {
		t.Fatalf("straight survivor: curve = %v, want nil (byte-identical to the pre-fix planar path)", straight.curves[0])
	}
}

// subsRimFixture is a named fake of a large planar HOST's boundary rim arc (the annular sector's outer arc)
// leaving a fillet corner: `parent` is the whole sector rim (radius R, sweep sectorDeg°), `from` the moved
// tangent point (pushed OFF the circle to √(R²+r²) by the fillet pull-back, near the start), and `to` the
// far rim endpoint (ON the circle). `scale` is the model bounding-diagonal the material gate divides by.
type subsRimFixture struct {
	parent   geom.Arc3d
	from, to math.Point3
	scale    float64
}

func newSubsRimFixture(t *testing.T, radius, sectorDeg, filletR, scale float64) subsRimFixture {
	t.Helper()
	center := math.P3(0, 0, 0)
	parent, err := geom.NewArc3d(center, math.V3(0, 0, 1), math.V3(1, 0, 0), radius, 0, sectorDeg*stdmath.Pi/180)
	if err != nil {
		t.Fatalf("build parent rim arc (R=%.0f sector=%.0f°): %v", radius, sectorDeg, err)
	}
	off := stdmath.Sqrt(radius*radius + filletR*filletR)
	return subsRimFixture{
		parent: parent,
		from:   offCirclePoint(center, off, 0.05), // moved tangent point, off-circle, small inset from the start
		to:     parent.PointAt(1),                 // the far rim endpoint, unmoved (on the circle)
		scale:  scale,
	}
}

// loopWithCarriedSubRim builds the two-point subs loop addSubstVertex leaves: the tangent point (index 0)
// carrying the FULL parent rim arc, then the far rim endpoint (index 1).
func loopWithCarriedSubRim(fix subsRimFixture) filletLoop {
	var fl filletLoop
	fl.add(fix.from, fix.parent) // segment 0: from → to, the parent arc to be trimmed (or reverted to nil)
	fl.add(fix.to, nil)
	return fl
}

// TestTrimCarriedSubArcCarriesLargeHostRim proves a large host rim (I3: R=300, ~90° sector, model scale
// 427 — chording it erases ~24000, ~13% of scale²) is carried as a trimmed sub-arc whose endpoints land
// back ON the parent circle (never the crude un-trimmed full parent that blows the body up).
func TestTrimCarriedSubArcCarriesLargeHostRim(t *testing.T) {
	t.Parallel()
	fix := newSubsRimFixture(t, 300, 90, 10, 427)
	fl := loopWithCarriedSubRim(fix)
	trimCarriedSubArcs(&fl, []int{0}, fix.scale)

	arc, ok := fl.curves[0].(geom.Arc3d)
	if !ok {
		t.Fatalf("large host rim: trimmed curve = %T, want geom.Arc3d (a material rim must be carried)", fl.curves[0])
	}
	assertArcEndpointsOnCircle(t, "large-host-rim", arc)
}

// TestTrimCarriedSubArcMinorStaysChord proves a small boss rim (N5: R=20, ~76° sector, model scale 206 —
// chording it erases only ~64, ~0.17% of scale²) is RESTORED to a straight chord (nil), byte-identical to
// the pre-carry planar path, so an already-green minor-face body (N5/B1/B9) is not perturbed.
func TestTrimCarriedSubArcMinorStaysChord(t *testing.T) {
	t.Parallel()
	fix := newSubsRimFixture(t, 20, 76, 10, 206)
	fl := loopWithCarriedSubRim(fix)
	trimCarriedSubArcs(&fl, []int{0}, fix.scale)

	if fl.curves[0] != nil {
		t.Fatalf("minor boss rim: curve = %v, want nil (chording erases a minor segment — keep the faithful chord)", fl.curves[0])
	}
}

// TestTrimCarriedSubArcDisabledByZeroScale proves modelScale=0 — the sentinel the specialized obstacle/
// runout/canal rebuild callers pass — restores the chord (nil), keeping those paths byte-identical to the
// pre-carry planar retrim regardless of the arc's size.
func TestTrimCarriedSubArcDisabledByZeroScale(t *testing.T) {
	t.Parallel()
	fix := newSubsRimFixture(t, 300, 90, 10, 0)
	fl := loopWithCarriedSubRim(fix)
	trimCarriedSubArcs(&fl, []int{0}, fix.scale)

	if fl.curves[0] != nil {
		t.Fatalf("scale-0 (carry disabled): curve = %v, want nil (specialized rebuild paths stay byte-identical)", fl.curves[0])
	}
}
