// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestSplineHandleReshapesCurve: activating a handle and pointing it away
// from the natural tangent must reshape the sampled curve; deactivating it
// must restore the natural fit (M06-F11, #626).
func TestSplineHandleReshapesCurve(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	sp := s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false)
	natural := append([]gmath.Point2(nil), sampleSplineEntity(sp)...)

	h, err := s.SplineHandles().Activate(sp, 1)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if w := h.Weight(); math.Abs(w-1) > 1e-9 {
		t.Errorf("fresh handle weight = %v, want 1", w)
	}
	if err := h.SetTangent(gmath.V2(0, 1), 1); err != nil { // force a vertical tangent at the apex
		t.Fatalf("SetTangent: %v", err)
	}
	reshaped := sampleSplineEntity(sp)
	maxGap := 0.0
	for i := range natural {
		if d := float64(natural[i].DistanceTo(reshaped[i])); d > maxGap {
			maxGap = d
		}
	}
	if maxGap < 1e-3 {
		t.Errorf("handle did not reshape the curve (max gap %g)", maxGap)
	}
	for _, want := range []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)} {
		found := false
		for _, p := range reshaped {
			if float64(p.DistanceTo(want)) < 1e-9 {
				found = true
			}
		}
		if !found {
			t.Errorf("reshaped curve no longer passes through fit point %v", want)
		}
	}

	if !s.SplineHandles().Deactivate(sp, 1) {
		t.Fatal("Deactivate must report the handle existed")
	}
	restored := sampleSplineEntity(sp)
	for i := range natural {
		if float64(natural[i].DistanceTo(restored[i])) > 1e-12 {
			t.Fatal("deactivation did not restore the natural curve")
		}
	}
}

// TestSplineHandleRules: control splines refuse handles; out-of-range fit
// indices error; re-activation returns the same handle.
func TestSplineHandleRules(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ctrl := s.Splines().AddByControlPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false)
	if _, err := s.SplineHandles().Activate(ctrl, 0); err == nil {
		t.Error("a control-point spline must refuse handles")
	}
	sp := s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false)
	if _, err := s.SplineHandles().Activate(sp, 7); err == nil {
		t.Error("an out-of-range fit index must be rejected")
	}
	h1, err := s.SplineHandles().Activate(sp, 0)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	h2, _ := s.SplineHandles().Activate(sp, 0)
	if h1 != h2 {
		t.Error("re-activation must return the existing handle")
	}
}

// TestSplineExtrasRoundTrip: fit method and active handles survive the .obk
// recipe round-trip.
func TestSplineExtrasRoundTrip(t *testing.T) {
	col := NewSketches()
	s := col.Add(XYPlane())
	sp := s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false)
	sp.FitMethod = types.SplineFitChord
	h, err := s.SplineHandles().Activate(sp, 1)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if err := h.SetTangent(gmath.V2(0, 1), 2); err != nil {
		t.Fatalf("SetTangent: %v", err)
	}
	endBefore := h.End.Position()

	restored := roundTrip(t, col)
	rsp := restored.Splines().Item(0)
	if rsp.FitMethod != types.SplineFitChord {
		t.Errorf("restored fit method = %v, want chord", rsp.FitMethod)
	}
	handles := rsp.Handles()
	if len(handles) != 1 || handles[0].FitIndex != 1 {
		t.Fatalf("restored handles = %+v, want the fit-1 handle", handles)
	}
	if got := handles[0].End.Position(); float64(got.DistanceTo(endBefore)) > 1e-9 {
		t.Errorf("restored handle end = %v, want %v", got, endBefore)
	}
}

// TestTextBoxAnchorConstraintLifecycle: the anchor record is auto-created
// with its text box, refuses direct deletion, and dies with the text.
func TestTextBoxAnchorConstraintLifecycle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	txt := s.TextBoxes().Add(gmath.P2(1, 1), "label", 0.5, 0, TextLeft)
	anchor := findTextAnchor(s, txt)
	if anchor == nil {
		t.Fatal("adding a text box must auto-create its anchor constraint")
	}
	if anchor.Deletable() {
		t.Error("the anchor record must not be deletable")
	}
	if err := s.GeometricConstraints().DeleteAllowed(anchor); err == nil {
		t.Error("DeleteAllowed must refuse the system-owned anchor")
	}
	s.deleteEntity(txt)
	if findTextAnchor(s, txt) != nil {
		t.Error("the anchor record must die with its text box")
	}
}

func findTextAnchor(s *Sketch, txt *TextBox) *TextBoxAnchorConstraint {
	for _, c := range s.GeometricConstraints().All() {
		if a, ok := c.(*TextBoxAnchorConstraint); ok && a.Text == txt {
			return a
		}
	}
	return nil
}

// TestCustomConstraintRules: a custom tag needs its owner, enumerates, can be
// deleted, and round-trips with its operands.
func TestCustomConstraintRules(t *testing.T) {
	col := NewSketches()
	s := col.Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	if _, err := s.GeometricConstraints().AddCustom("", "tag", []Entity{l}); err == nil {
		t.Error("a custom constraint without a clientId must be rejected")
	}
	c, err := s.GeometricConstraints().AddCustom("com.x.cam", "toolpath-edge", []Entity{l})
	if err != nil {
		t.Fatalf("AddCustom: %v", err)
	}
	if err := s.GeometricConstraints().DeleteAllowed(c); err != nil {
		t.Errorf("a custom constraint must be deletable by its owner: %v", err)
	}

	if _, err := s.GeometricConstraints().AddCustom("com.x.cam", "toolpath-edge", []Entity{l}); err != nil {
		t.Fatalf("re-AddCustom: %v", err)
	}
	restored := roundTrip(t, col)
	found := false
	for _, rc := range restored.GeometricConstraints().All() {
		if cc, ok := rc.(*CustomConstraint); ok {
			found = true
			if cc.ClientID != "com.x.cam" || cc.Name != "toolpath-edge" || len(cc.Entities) != 1 {
				t.Errorf("restored custom = %+v, want owner/name/1 operand", cc)
			}
		}
	}
	if !found {
		t.Error("the custom constraint did not survive the round-trip")
	}
}

// TestMoveableStatusClassification: free geometry drags, fixed geometry does
// not, fully-dimensioned geometry needs a dimension change, and derived
// curves own no drag DOFs.
func TestMoveableStatusClassification(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	free := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	if got := s.MoveableStatus(free); got != types.MoveableFree {
		t.Errorf("unconstrained line = %v, want freeToMove", got)
	}

	pinned := s.Points().Add(gmath.P2(3, 3))
	s.GeometricConstraints().AddFix(pinned)
	if got := s.MoveableStatus(pinned); got != types.MoveableFixed {
		t.Errorf("fixed point = %v, want fixed", got)
	}

	fixedSpl := s.FixedSplines().Add([]gmath.Point2{gmath.P2(0, 2), gmath.P2(1, 3), gmath.P2(2, 2)})
	if got := s.MoveableStatus(fixedSpl); got != types.MoveableFixed {
		t.Errorf("fixed spline = %v, want fixed (no drag DOFs)", got)
	}
}

// TestMoveableStatusByDimensionChange: a point pinned only by driving
// dimensions against a fixed anchor reports moveable-by-dimension-change.
func TestMoveableStatusByDimensionChange(t *testing.T) {
	col := NewSketches()
	s := col.Add(XYPlane())
	anchor := s.Points().Add(gmath.P2(0, 0))
	s.GeometricConstraints().AddFix(anchor)
	p := s.Points().Add(gmath.P2(2, 0))
	// Pin p's two DOFs numerically: coincident-in-y via vertical-style
	// horizontal alignment plus a driving distance.
	s.GeometricConstraints().AddHorizontal(anchor, p)
	if _, err := s.DimensionConstraints().AddDistance(anchor, p, "2 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if got := s.MoveableStatus(p); got != types.MoveableByDimensionChange {
		t.Errorf("dimension-pinned point = %v, want byDimensionChange", got)
	}
}
