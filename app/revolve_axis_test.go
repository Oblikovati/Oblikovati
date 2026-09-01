// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"slices"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// #2018: the revolve axis was the one input with no way to point at it — an origin-axis combo and
// nothing else. Worse, the combo LIED: a pre-selected centerline outranks it in addRevolve, so the
// panel read "Y Axis" while the feature spun about the centerline. These tests assert what the
// panel now reads (AxisName) alongside the geometry it produces, so the two cannot drift apart
// again.

// The 2×2 square at x∈[2,4] revolved a full turn about a vertical axis is a washer whose radii
// are its distance from that axis. About Y (the tool's default) that is 24π; about the line x=1 it
// is 16π — two values no mix-up can produce by accident, which is what makes the axis assertions
// below bite.
const (
	washerAboutY      = stdmath.Pi * (4*4 - 2*2) * 2
	washerAboutXIsOne = stdmath.Pi * (3*3 - 1*1) * 2
)

// revolvedBodyVolume commits the tool and returns the volume of the single body it produced.
func revolvedBodyVolume(t *testing.T, s *Session) float64 {
	t.Helper()
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after revolve, want 1", def.SurfaceBodies().Count())
	}
	return ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
}

// TestAxisChipNamesThePreselectedCenterline is the #2018 regression: the panel must name the axis
// the feature will actually use. A profile whose sketch carries a centerline pre-selects it, and
// before this the panel went on showing the origin-axis combo's "Y Axis".
func TestAxisChipNamesThePreselectedCenterline(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	cl := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1) // pick the profile → pre-select the centerline

	if !rv.AxisPicked() {
		t.Fatal("a pre-selected centerline must read as a picked axis, not as the combo's default")
	}
	if got := rv.AxisName(); got != "Centerline" {
		t.Errorf("axis chip reads %q, want \"Centerline\" — the axis the revolve will really use", got)
	}
}

// TestAxisChipNamesTheOriginAxisWhenNothingIsPicked: with no centerline anywhere the chip falls
// back to the quick-pick's axis, and the quick-pick stays live.
func TestAxisChipNamesTheOriginAxisWhenNothingIsPicked(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1)

	if rv.AxisPicked() {
		t.Fatal("nothing was picked, so the quick-pick must still drive the axis")
	}
	if got := rv.AxisName(); got != "Y Axis" {
		t.Errorf("axis chip reads %q, want \"Y Axis\"", got)
	}
	rv.SetAxis(feature.OriginZAxis)
	if got := rv.AxisName(); got != "Z Axis" {
		t.Errorf("after choosing Z the chip reads %q, want \"Z Axis\"", got)
	}
}

// TestClearAxisReturnsToTheOriginAxis: the chip's × drops the pick, and both the caption and the
// geometry go back to the quick-pick's axis.
func TestClearAxisReturnsToTheOriginAxis(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	horiz := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	horiz.SetCenterline(true) // an X-axis centerline: revolving about it is NOT the washer
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1)
	if !rv.AxisPicked() {
		t.Fatal("the lone centerline should pre-select")
	}
	rv.ClearAxis()

	if rv.AxisPicked() || rv.AxisName() != "Y Axis" {
		t.Fatalf("after clearing, chip reads %q (picked=%v), want \"Y Axis\" unpicked", rv.AxisName(), rv.AxisPicked())
	}
	if got := revolvedBodyVolume(t, s); relErrApp(got, washerAboutY) > 0.01 {
		t.Errorf("volume %g, want ≈%g — clearing must revolve about Y, not the cleared centerline", got, washerAboutY)
	}
}

// TestRevolveAcceptsAWorkAxisPick: a work axis clicked in the viewport or the browser becomes the
// axis. Before #2018 the tool's filter admitted only sketch entities, so a work axis could not be
// picked at all and the origin axes were reachable only through the combo.
func TestRevolveAcceptsAWorkAxisPick(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	// A vertical work axis OFFSET to x=1, so the resulting washer (16π) can only come from this
	// axis — the tool's Y default would give 24π.
	up, err := math.UnitVector3FromVector(math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("unit vector: %v", err)
	}
	axis := part.WorkAxes().AddByLine(math.P3(1, 0, 0), up)
	axis.SetName("Spin Axis")

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1)
	if !acceptsKind(rv.AcceptedKinds(), SelectWorkAxis) {
		t.Fatal("the axis step must accept work axes")
	}
	s.SelectBrowserNode(BrowserNode{Select: WorkAxisHandle{Axis: axis}})

	if got := rv.AxisName(); got != "Spin Axis" {
		t.Errorf("axis chip reads %q, want the picked work axis's name", got)
	}
	if got := revolvedBodyVolume(t, s); relErrApp(got, washerAboutXIsOne) > 0.01 {
		t.Errorf("volume %g, want ≈%g about the picked work axis", got, washerAboutXIsOne)
	}
}

// acceptsKind reports whether a tool's accepted-kind list admits kind.
func acceptsKind(kinds []SelectionKind, kind SelectionKind) bool {
	return slices.Contains(kinds, kind)
}

// TestRevolveAcceptsAPlainSketchLineAsAxis: the model revolves about any sketch line, but the tool
// used to discard everything but a centerline, so a construction line drawn as the axis did
// nothing at all when clicked.
func TestRevolveAcceptsAPlainSketchLineAsAxis(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	// A plain line at x=1 (NOT a centerline): revolving about it gives 16π, which the tool's Y
	// default cannot produce.
	plain := profile.Sketch.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 2))
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1)
	rv.Pick(s, SketchEntityHandle{Entity: plain})

	if got := rv.AxisName(); got != "Sketch Line" {
		t.Errorf("axis chip reads %q, want \"Sketch Line\"", got)
	}
	if got := revolvedBodyVolume(t, s); relErrApp(got, washerAboutXIsOne) > 0.01 {
		t.Errorf("volume %g, want ≈%g about the picked sketch line", got, washerAboutXIsOne)
	}
}

// TestEditingARevolveKeepsItsUserWorkAxis: re-opening a revolve seeded the tool by matching the
// definition's axis against the three ORIGIN refs only, so a user work axis matched nothing and
// the tool fell back to its Y default — silently rewriting the axis on the next OK (#2018).
func TestEditingARevolveKeepsItsUserWorkAxis(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sideways, err := math.UnitVector3FromVector(math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("unit vector: %v", err)
	}
	axis := part.WorkAxes().AddByLine(math.P3(0, 0, 0), sideways)
	axis.SetName("Tilt Axis")
	pf := feature.NewRevolveFeatures(part.Features()).Add(profile.Sketch, 0, axis, func() float64 { return 0 }, ops.NewBody)

	edited := editRevolveTool(pf, pf.Definition().(*feature.RevolveFeature))

	if got := edited.AxisName(); got != "Tilt Axis" {
		t.Fatalf("re-opened revolve names axis %q, want \"Tilt Axis\"", got)
	}
	if err := edited.writeEditAxis(s, pf.Definition().(*feature.RevolveFeature).Definition()); err != nil {
		t.Fatalf("writeEditAxis: %v", err)
	}
	if got := pf.Definition().(*feature.RevolveFeature).Definition().Axis; got != axis {
		t.Errorf("committing the edit wrote axis %v, want the original user work axis", got)
	}
}
