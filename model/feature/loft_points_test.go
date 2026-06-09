// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft POINT-section matrix (Slice 2b): a section can be a single point (apex) so the loft tapers
// to a tip. Sharp/Free gives a straight cone (or pyramid); TangentToPlane domes the apex (tangent
// to its plane), with impact scaling the dome and reversed dishing it. Apex sections are valid
// only at the ends.

// apexPoint returns a point (apex) section on plane at (0,0) lifted to height z.
func apexPoint(plane sketch.Plane, z float64) LoftSection {
	p := math.P3(0, 0, math.Scalar(z))
	return LoftSection{Sketch: sketch.NewSketches().Add(plane), Point: &p}
}

// TestLoftPointSharpCone: a circle base lofted to a Sharp apex is a straight cone — V = πr²h/3.
func TestLoftPointSharpCone(t *testing.T) {
	secs := []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(4), 4)}
	b := conditionedLoft(t, secs, false, LoftEnd{}, LoftEnd{Condition: LoftSharpPoint})
	want := stdmath.Pi * 4 / 3 * 4 // πr²h/3, r=2 h=4
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v, want) > 0.03 {
		t.Errorf("cone volume = %g, want ≈%g", v, want)
	}
}

// TestLoftPointTangentDomes: a TangentToPlane apex domes OUTWARD, so the solid holds more volume
// than the straight cone between the same base and tip.
func TestLoftPointTangentDomes(t *testing.T) {
	base := func() []LoftSection {
		return []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(4), 4)}
	}
	cone := ops.BodyGeometryProperties(conditionedLoft(t, base(), false, LoftEnd{}, LoftEnd{Condition: LoftSharpPoint}), ops.DefaultQuality()).Volume
	dome := ops.BodyGeometryProperties(conditionedLoft(t, base(), false, LoftEnd{}, LoftEnd{Condition: LoftTangentToPlane}), ops.DefaultQuality()).Volume
	if dome <= cone*1.05 {
		t.Errorf("tangent-to-plane apex did not dome out: dome vol %.3f, cone vol %.3f", dome, cone)
	}
}

// TestLoftPointImpactScalesDome: a larger impact bulges the dome more.
func TestLoftPointImpactScalesDome(t *testing.T) {
	mk := func() []LoftSection {
		return []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(4), 4)}
	}
	soft := ops.BodyGeometryProperties(conditionedLoft(t, mk(), false, LoftEnd{}, LoftEnd{Condition: LoftTangentToPlane, Impact: 1}), ops.DefaultQuality()).Volume
	hard := ops.BodyGeometryProperties(conditionedLoft(t, mk(), false, LoftEnd{}, LoftEnd{Condition: LoftTangentToPlane, Impact: 2}), ops.DefaultQuality()).Volume
	if hard <= soft {
		t.Errorf("higher impact did not dome more: impact1 vol %.3f, impact2 vol %.3f", soft, hard)
	}
}

// TestLoftPointReversedDishes: reversing the tangent-to-plane apex pulls it inward, dishing the
// tip concave — less volume than the straight cone.
func TestLoftPointReversedDishes(t *testing.T) {
	mk := func() []LoftSection {
		return []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(4), 4)}
	}
	cone := ops.BodyGeometryProperties(conditionedLoft(t, mk(), false, LoftEnd{}, LoftEnd{Condition: LoftSharpPoint}), ops.DefaultQuality()).Volume
	dish := ops.BodyGeometryProperties(conditionedLoft(t, mk(), false, LoftEnd{}, LoftEnd{Condition: LoftTangentToPlane, Reversed: true}), ops.DefaultQuality()).Volume
	if dish >= cone {
		t.Errorf("reversed tangent apex did not dish concave: dished vol %.3f, cone vol %.3f", dish, cone)
	}
}

// TestLoftPointSquarePyramid: a square base lofted to a Sharp apex is a pyramid — V = base·h/3.
func TestLoftPointSquarePyramid(t *testing.T) {
	secs := []LoftSection{sec(centeredSquareOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(6), 6)}
	b := conditionedLoft(t, secs, false, LoftEnd{}, LoftEnd{Condition: LoftSharpPoint})
	want := 16.0 * 6 / 3 // base area (4×4) × h / 3
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v, want) > 0.03 {
		t.Errorf("pyramid volume = %g, want ≈%g", v, want)
	}
}

// TestLoftPointAtStart: an apex as the FIRST section (an inverted cone, tip at the bottom) is a
// valid solid — point sections work at either end.
func TestLoftPointAtStart(t *testing.T) {
	secs := []LoftSection{apexPoint(sketch.XYPlane(), 0), sec(circleOn(planeAtZ(4), 2))}
	b := conditionedLoft(t, secs, false, LoftEnd{Condition: LoftSharpPoint}, LoftEnd{})
	want := stdmath.Pi * 4 / 3 * 4
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v, want) > 0.03 {
		t.Errorf("inverted cone volume = %g, want ≈%g", v, want)
	}
}

// TestLoftPointValidation: a point in the middle, or an all-point loft, is rejected.
func TestLoftPointValidation(t *testing.T) {
	mid := []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), apexPoint(planeAtZ(2), 2), sec(circleOn(planeAtZ(4), 2))}
	if err := loftError(t, mid); err == nil {
		t.Error("a point section in the middle should be rejected")
	}
	allPts := []LoftSection{apexPoint(sketch.XYPlane(), 0), apexPoint(planeAtZ(4), 4)}
	if err := loftError(t, allPts); err == nil {
		t.Error("an all-point loft should be rejected")
	}
}

// loftError returns the health-reason error of building a loft (nil if it stays healthy).
func loftError(t *testing.T, sections []LoftSection) error {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	pf := NewLoftFeatures(fs).Add(sections, false, ops.NewBody)
	fs.Recompute()
	if pf.Health().OK() {
		return nil
	}
	return errLoft(pf.Health().Reason)
}

type errLoft string

func (e errLoft) Error() string { return string(e) }
