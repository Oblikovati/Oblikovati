// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Loft end-CONDITION matrix (Slice 2). Each condition (Free / Angle / Direction, with impact
// and reversed) is exercised across shapes (cylindrical, rectangular, pipe, organic spline) and
// asserted both topologically valid and geometrically curved as the takeoff dictates. The
// angle/direction condition is what lets a TWO-section loft curve away from the ruled blend.

func rad(deg float64) float64 { return deg * stdmath.Pi / 180 }

// conditionedLoft builds a loft with explicit end conditions and asserts a single valid solid.
func conditionedLoft(t *testing.T, sections []LoftSection, closed bool, first, last LoftEnd) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewLoftFeatures(fs).AddConditioned(sections, closed, ops.NewBody, first, last)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("conditioned loft went sick: %+v", pf.Health())
	}
	b := fs.Result()[0]
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("conditioned loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

func twoCircles(r float64) []LoftSection {
	return []LoftSection{sec(circleOn(sketch.XYPlane(), r)), sec(circleOn(planeAtZ(4), r))}
}

// TestLoftConditionFreeIsRuled: a two-section equal-radius Free loft is a straight cylinder —
// its max radius equals the section radius (no bulge). The baseline the angle conditions beat.
func TestLoftConditionFreeIsRuled(t *testing.T) {
	b := conditionedLoft(t, twoCircles(2), false, LoftEnd{}, LoftEnd{})
	if maxX := float64(b.RangeBox().Max.X); maxX > 2.03 {
		t.Errorf("Free loft bulged: max x = %.3f, want ~2.0 (ruled cylinder)", maxX)
	}
}

// TestLoftConditionAngleBulges: a two-section equal-radius loft with a 45° takeoff at both ends
// curves OUT — the body's radius exceeds the section radius. This is the headline S2 behavior:
// curving a two-section loft, which Free cannot do.
func TestLoftConditionAngleBulges(t *testing.T) {
	end := LoftEnd{Condition: LoftAngle, Angle: rad(45)}
	b := conditionedLoft(t, twoCircles(2), false, end, end)
	if maxX := float64(b.RangeBox().Max.X); maxX < 2.15 {
		t.Errorf("Angle loft did not bulge: max x = %.3f, want > 2.15 (ruled would be 2.0)", maxX)
	}
}

// TestLoftConditionDirectionAliasesAngle: the Direction condition is the same takeoff as Angle
// (Inventor shares one id for both names), so it bulges identically.
func TestLoftConditionDirectionAliasesAngle(t *testing.T) {
	angle := conditionedLoft(t, twoCircles(2), false, LoftEnd{Condition: LoftAngle, Angle: rad(45)}, LoftEnd{Condition: LoftAngle, Angle: rad(45)})
	dir := conditionedLoft(t, twoCircles(2), false, LoftEnd{Condition: LoftDirection, Angle: rad(45)}, LoftEnd{Condition: LoftDirection, Angle: rad(45)})
	if a, d := float64(angle.RangeBox().Max.X), float64(dir.RangeBox().Max.X); stdmath.Abs(a-d) > 1e-6 {
		t.Errorf("Direction (%.4f) != Angle (%.4f); they must alias the same takeoff", d, a)
	}
}

// TestLoftConditionImpactScalesBulge: a larger impact (takeoff weight) curves the surface more,
// so the body bulges further.
func TestLoftConditionImpactScalesBulge(t *testing.T) {
	soft := conditionedLoft(t, twoCircles(2), false, LoftEnd{Condition: LoftAngle, Angle: rad(45), Impact: 1}, LoftEnd{Condition: LoftAngle, Angle: rad(45), Impact: 1})
	hard := conditionedLoft(t, twoCircles(2), false, LoftEnd{Condition: LoftAngle, Angle: rad(45), Impact: 2}, LoftEnd{Condition: LoftAngle, Angle: rad(45), Impact: 2})
	if s, h := float64(soft.RangeBox().Max.X), float64(hard.RangeBox().Max.X); h <= s {
		t.Errorf("higher impact did not bulge more: impact1 max x = %.3f, impact2 = %.3f", s, h)
	}
}

// TestLoftConditionReversedUndercut: reversing the takeoff flips it through the section plane, so
// the surface dips BELOW the start section (and above the end section) — an undercut/overhang —
// while a non-reversed takeoff stays within the section planes.
func TestLoftConditionReversedUndercut(t *testing.T) {
	end := LoftEnd{Condition: LoftAngle, Angle: rad(45)}
	rev := LoftEnd{Condition: LoftAngle, Angle: rad(45), Reversed: true}
	straight := conditionedLoft(t, twoCircles(2), false, end, end)
	under := conditionedLoft(t, twoCircles(2), false, rev, rev)
	if z := float64(straight.RangeBox().Min.Z); z < -0.01 {
		t.Errorf("non-reversed loft dipped below the start plane: min z = %.3f, want ~0", z)
	}
	if z := float64(under.RangeBox().Min.Z); z > -0.02 {
		t.Errorf("reversed loft did not undercut: min z = %.3f, want < -0.02 (below start plane)", z)
	}
}

// TestLoftConditionAngleSquare: the angle takeoff curves a rectangular (square→square) loft too —
// a valid solid that bulges past the ruled square prism.
func TestLoftConditionAngleSquare(t *testing.T) {
	secs := []LoftSection{sec(centeredSquareOn(sketch.XYPlane(), 2)), sec(centeredSquareOn(planeAtZ(4), 2))}
	end := LoftEnd{Condition: LoftAngle, Angle: rad(40)}
	b := conditionedLoft(t, secs, false, end, end)
	if maxX := float64(b.RangeBox().Max.X); maxX < 2.1 {
		t.Errorf("angled square loft did not bulge: max x = %.3f, want > 2.1 (ruled would be 2.0)", maxX)
	}
}

// TestLoftConditionAnglePipe: the takeoff applies through the tube path too — an annulus→annulus
// loft with an angle condition is still a watertight HOLLOW pipe (bore preserved), now flared.
func TestLoftConditionAnglePipe(t *testing.T) {
	sb, ib := annulusOn(sketch.XYPlane(), 2.0, 1.4)
	st, it := annulusOn(planeAtZ(4), 2.0, 1.4)
	end := LoftEnd{Condition: LoftAngle, Angle: rad(45)}
	b := conditionedLoft(t, []LoftSection{{Sketch: sb, ProfileIndex: ib}, {Sketch: st, ProfileIndex: it}}, false, end, end)
	// Hollow: volume must be well below the solid bounding cylinder (a filled body would be ~π·2²·4).
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; v > stdmath.Pi*2*2*4*0.85 {
		t.Errorf("angled pipe looks filled (bore lost): volume = %.3f", v)
	}
}

// TestLoftConditionOrganicSpline: the takeoff works on organic closed-spline sections — a valid
// curved solid (no self-intersection from the correspondence + angled blend).
func TestLoftConditionOrganicSpline(t *testing.T) {
	blob := [][2]float64{{2, 0}, {1.2, 1.6}, {-1, 1.8}, {-2, 0}, {-1, -1.6}, {1.2, -1.6}}
	secs := []LoftSection{sec(splineBlobOn(sketch.XYPlane(), blob)), sec(splineBlobOn(planeAtZ(4), blob))}
	end := LoftEnd{Condition: LoftAngle, Angle: rad(50)}
	conditionedLoft(t, secs, false, end, end)
}

// TestLoftConditionClosedIgnoresEnds: a closed loft has no end sections, so end conditions are
// ignored — a closed loft with angle conditions matches the Free closed loft exactly.
func TestLoftConditionClosedIgnoresEnds(t *testing.T) {
	mk := func() []LoftSection {
		return []LoftSection{
			sec(circleOn(sketch.XYPlane(), 2)),
			sec(circleOn(planeAtZ(3), 2)),
			sec(circleOn(planeAtZ(6), 2)),
		}
	}
	end := LoftEnd{Condition: LoftAngle, Angle: rad(45)}
	free := conditionedLoft(t, mk(), true, LoftEnd{}, LoftEnd{})
	cond := conditionedLoft(t, mk(), true, end, end)
	vf := ops.BodyGeometryProperties(free, ops.DefaultQuality()).Volume
	vc := ops.BodyGeometryProperties(cond, ops.DefaultQuality()).Volume
	if relErr(vf, vc) > 1e-9 {
		t.Errorf("closed loft honored end conditions: free vol %.6f != conditioned %.6f", vf, vc)
	}
}
