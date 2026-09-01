// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// A contour flange's corners are BENDS (#1961). They were swept sharp, which is geometry no press
// brake can make. These tests check the corner is a real arc of the asked-for radius — a rounded
// corner that is merely "about right" is a bend radius that is wrong.

// TestCornerArcIsACircleOfTheRadius: every sampled point has to sit at the radius from the arc's
// centre. A quadratic Bezier through the same three points passes a glance and fails this.
func TestCornerArcIsACircleOfTheRadius(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct{ a, b, corner math.Point2 }{
		"right angle": {math.P2(0, 0), math.P2(2, 0), math.P2(2, 2)},
		"sharp turn":  {math.P2(0, 0), math.P2(2, 0), math.P2(1, 1.7)},
		"gentle turn": {math.P2(0, 0), math.P2(2, 0), math.P2(4, 0.5)},
		"clockwise":   {math.P2(0, 0), math.P2(2, 0), math.P2(2, -2)},
	} {
		t.Run(name, func(t *testing.T) {
			const radius = 0.4
			arc, err := cornerArc(c.a, c.b, c.corner, radius)
			if err != nil {
				t.Fatalf("cornerArc: %v", err)
			}
			if len(arc) < 3 {
				t.Fatalf("arc has %d points, want a sampled curve", len(arc))
			}
			// The centre is equidistant from the two ends and at the radius from both; fit it from
			// the first point and the known tangency, then check every sample against it.
			centre := arcCentreFrom(t, arc, radius)
			for i, p := range arc {
				if d := float64(centre.DistanceTo(p)); stdmath.Abs(d-radius) > 1e-6 {
					t.Errorf("point %d sits %.6f from the centre, want the %g radius", i, d, radius)
				}
			}
		})
	}
}

// arcCentreFrom recovers the arc's centre: the point equidistant from the first, middle and last
// samples. Three points on a circle determine it, so this is a check independent of how the arc
// was built.
func arcCentreFrom(t *testing.T, arc []math.Point2, radius float64) math.Point2 {
	t.Helper()
	a, b, c := arc[0], arc[len(arc)/2], arc[len(arc)-1]
	d := 2 * (float64(a.X)*(float64(b.Y)-float64(c.Y)) + float64(b.X)*(float64(c.Y)-float64(a.Y)) +
		float64(c.X)*(float64(a.Y)-float64(b.Y)))
	if stdmath.Abs(d) < 1e-12 {
		t.Fatal("the three samples are collinear — that is not an arc")
	}
	sq := func(p math.Point2) float64 { return float64(p.X)*float64(p.X) + float64(p.Y)*float64(p.Y) }
	ux := (sq(a)*(float64(b.Y)-float64(c.Y)) + sq(b)*(float64(c.Y)-float64(a.Y)) + sq(c)*(float64(a.Y)-float64(b.Y))) / d
	uy := (sq(a)*(float64(c.X)-float64(b.X)) + sq(b)*(float64(a.X)-float64(c.X)) + sq(c)*(float64(b.X)-float64(a.X))) / d
	return math.P2(ux, uy)
}

// TestCornerArcIsTangentToBothSegments: the arc has to leave the incoming segment and rejoin the
// outgoing one, or the wall kinks where the bend was supposed to smooth it.
func TestCornerArcIsTangentToBothSegments(t *testing.T) {
	t.Parallel()
	a, b, c := math.P2(0, 0), math.P2(2, 0), math.P2(2, 2)
	const radius = 0.5
	arc, err := cornerArc(a, b, c, radius)
	if err != nil {
		t.Fatalf("cornerArc: %v", err)
	}
	// tan(45°)=1, so the arc leaves at (1.5, 0) and rejoins at (2, 0.5).
	if got := arc[0]; stdmath.Abs(float64(got.X)-1.5) > 1e-9 || stdmath.Abs(float64(got.Y)) > 1e-9 {
		t.Errorf("arc starts at %v, want (1.5, 0) on the incoming segment", got)
	}
	if got := arc[len(arc)-1]; stdmath.Abs(float64(got.X)-2) > 1e-9 || stdmath.Abs(float64(got.Y)-0.5) > 1e-9 {
		t.Errorf("arc ends at %v, want (2, 0.5) on the outgoing segment", got)
	}
}

// TestRadiusTooBigForTheProfileIsRefused: the arc eats radius·tan(half the turn) of straight either
// side, so a radius the segments cannot give is refused rather than allowed to overrun the corner
// and fold the profile back on itself.
func TestRadiusTooBigForTheProfileIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := cornerArc(math.P2(0, 0), math.P2(0.3, 0), math.P2(0.3, 2), 1.0); err == nil {
		t.Error("a radius larger than the segment should be refused")
	}
	if _, err := roundProfileCorners([]math.Point2{{X: 0}, {X: 0.3}, {X: 0.3, Y: 2}}, 1.0); err == nil {
		t.Error("rounding a profile that cannot take the radius should be refused")
	}
}

// TestCollinearCornerIsNotRounded: a straight-through vertex is not a bend and must not be given
// an arc, which would put a kink in a straight run.
func TestCollinearCornerIsNotRounded(t *testing.T) {
	t.Parallel()
	arc, err := cornerArc(math.P2(0, 0), math.P2(1, 0), math.P2(2, 0), 0.3)
	if err != nil || len(arc) != 0 {
		t.Errorf("a collinear corner produced %d points (err %v), want none", len(arc), err)
	}
}

// TestNoRadiusLeavesTheProfileAlone: with no bend radius the profile keeps its drawn corners, so a
// part built before this still comes out as it did.
func TestNoRadiusLeavesTheProfileAlone(t *testing.T) {
	t.Parallel()
	pts := []math.Point2{{X: 0}, {X: 1}, {X: 1, Y: 1}}
	got, err := roundProfileCorners(pts, 0)
	if err != nil || len(got) != len(pts) {
		t.Errorf("a zero radius changed the profile to %v (err %v)", got, err)
	}
}

// TestContourFlangeBandIsTheFullGauge: a mitred corner has to stand the GAUGE away from both faces,
// not the gauge times the cosine of the half-turn. The L-profile gives an exact figure to check
// against: two 1 cm runs of 2 mm material is 0.4 cm² of section, plus t² = 0.04 for the outer
// mitre at the right-angle corner, over a 4 cm edge — 1.76 cm³ of wall on a 3.2 cm³ sheet.
//
// Before the mitre was fixed this came out 16% light, because the corner was pinched to 71% of the
// gauge and nothing measured it.
func TestContourFlangeBandIsTheFullGauge(t *testing.T) {
	t.Parallel()
	const sheet, wall = 4 * 4 * 0.2, (2*0.2 + 0.2*0.2) * 4
	if got := contourFlangeVolume(t, 0); stdmath.Abs(got-(sheet+wall)) > 1e-6 {
		t.Errorf("sharp contour flange = %.6f cm³, want %.6f (%.2f sheet + %.2f wall)",
			got, sheet+wall, sheet, wall)
	}
}

// TestContourFlangeRoundsItsCorner: a bend cuts the corner short, so the wall loses the material
// the sharp mitre had — more of it as the radius grows. A vanishing bend must approach the sharp
// wall, which is what says the rounding is a corner treatment and not a change to the whole sweep.
func TestContourFlangeRoundsItsCorner(t *testing.T) {
	t.Parallel()
	sharp := contourFlangeVolume(t, 0)
	small := contourFlangeVolume(t, 0.05)
	large := contourFlangeVolume(t, 0.3)
	if !(large < small && small < sharp) {
		t.Errorf("volumes sharp %.6f, r=0.05 %.6f, r=0.3 %.6f; a bigger bend should cut more corner",
			sharp, small, large)
	}
	if sharp-small > 0.1 {
		t.Errorf("a 0.5 mm bend cut %.6f cm³ — it should barely differ from the sharp corner", sharp-small)
	}
}

// TestContourFlangeStartsANewBody: the operation decides whether the wall joins the sheet or
// stands alone, which is the difference between one body and two.
func TestContourFlangeStartsANewBody(t *testing.T) {
	t.Parallel()
	if got := contourFlangeBodyCount(t, ops.Join); got != 1 {
		t.Errorf("a joined contour flange left %d bodies, want 1", got)
	}
	if got := contourFlangeBodyCount(t, ops.NewBody); got != 2 {
		t.Errorf("a new-body contour flange left %d bodies, want 2", got)
	}
}

// contourFlangeVolume sweeps the L-profile with the given corner radius and measures the result.
func contourFlangeVolume(t *testing.T, radius float64) float64 {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	def := &SheetMetalContourFlangeDefinition{EdgeKey: edge.ReferenceKey(), Profile: lProfile()}
	if radius > 0 {
		def.Radius = constClosure(radius)
	} else {
		def.Radius = constClosure(0) // explicit: keep the drawn corners
	}
	pf := NewSheetMetalContourFlangeFeatures(fs).Add(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("contour flange sick: %+v", pf.Health())
	}
	return smSolidVolume(fs.Result()[0])
}

// contourFlangeBodyCount sweeps the L-profile under the given operation and reports how many
// bodies the part ends with.
func contourFlangeBodyCount(t *testing.T, op ops.PartFeatureOperation) int {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalContourFlangeFeatures(fs).Add(&SheetMetalContourFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Profile: lProfile(), Operation: op, Radius: constClosure(0),
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("contour flange sick: %+v", pf.Health())
	}
	return len(fs.Result())
}
