// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
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

// TestTrimCarriedRimArcMinor proves a <π retained span (a B1-style 90° sector rim) trims to a MINOR sub-arc
// (|sweep| < π) — the correct in-sector side, matching B1/B9 staying green.
func TestTrimCarriedRimArcMinor(t *testing.T) {
	fix := newRimCircleFixture(t, 90, 10)
	fl := loopWithCarriedRim(fix)
	trimCarriedRimArcs(&fl, []int{0})

	arc, ok := fl.curves[0].(geom.Arc3d)
	if !ok {
		t.Fatalf("minor rim: trimmed curve = %T, want geom.Arc3d", fl.curves[0])
	}
	if stdmath.Abs(arc.SweepAngle) >= stdmath.Pi {
		t.Fatalf("minor rim: trimmed sweep %.1f° over-spanned, want < 180° (the in-sector side)", arc.SweepAngle*180/stdmath.Pi)
	}
	assertArcEndpointsOnCircle(t, "minor", arc)
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
	fix := newRimCircleFixture(t, 270, 10)
	got := projectOntoArcCircle(fix.parent, fix.tOut)
	if d := stdmath.Abs(float64(got.DistanceTo(fix.parent.Center)) - fix.parent.Radius); d > 1e-9 {
		t.Fatalf("projected point is %.4g off the radius-50 circle, want 0", d)
	}
	if float64(got.Z) != 100 {
		t.Fatalf("projected z = %v, want 100 (stays in the rim plane)", got.Z)
	}
}
