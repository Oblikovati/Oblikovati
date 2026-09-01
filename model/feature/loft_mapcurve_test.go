// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft MAP-CURVE matrix (Slice 6, MapPointCurves): an explicit point correspondence that overrides
// the automatic minimum-twist alignment. With two identical squares the automatic alignment yields a
// straight (untwisted) prism; a map curve pairing a corner with the NEXT corner forces a 90° twist —
// the mid cross-section rotates 45°, so its bounding box grows.

func twoSquares() []LoftSection {
	return []LoftSection{sec(centeredSquareOn(sketch.XYPlane(), 2)), sec(centeredSquareOn(planeAtZ(4), 2))}
}

func mappedSquares(t *testing.T, mapCurves []func() []math.Point3) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewLoftFeatures(fs).AddGuided(twoSquares(), false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{MapCurves: mapCurves})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("map-curve loft went sick: %+v", pf.Health())
	}
	b := fs.Result()[0]
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("map-curve loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

// TestLoftMapCurveDefaultIsUntwisted: identical squares with NO map curve auto-align to a straight
// prism — max x stays at the section half-width.
func TestLoftMapCurveDefaultIsUntwisted(t *testing.T) {
	t.Parallel()
	if maxX := float64(mappedSquares(t, nil).RangeBox().Max.X); maxX > 2.05 {
		t.Errorf("auto-aligned identical squares twisted: max x = %.3f, want ≈2.0", maxX)
	}
}

// TestLoftMapCurveForcesTwist: a map curve pairing the start square's (2,2) corner with the end
// square's NEXT corner (-2,2) forces a 90° twist in the correspondence. A linearly-blended 90°
// twist pinches the mid section to the inscribed diamond (half the area), so the twisted loft holds
// distinctly less volume than the auto-aligned (untwisted) prism — proving the override took effect.
func TestLoftMapCurveForcesTwist(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	auto := query.BodyGeometryProperties(mappedSquares(t, nil), ops.DefaultQuality()).Volume
	mc := func() []math.Point3 { return []math.Point3{math.P3(2, 2, 0), math.P3(-2, 2, 4)} }
	twisted := query.BodyGeometryProperties(mappedSquares(t, []func() []math.Point3{mc}), ops.DefaultQuality()).Volume
	if twisted >= auto*0.9 {
		t.Errorf("map curve did not change the correspondence: twisted vol %.3f vs auto %.3f (want twisted clearly less, the mid pinches)", twisted, auto)
	}
}

// TestLoftMapCurveRoundTrip: a map curve survives a recipe save/restore.
func TestLoftMapCurveRoundTrip(t *testing.T) {
	t.Parallel()
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(4), 2)
	idx := sketchList{sks: []*sketch.Sketch{bottom, top}}
	fs := NewPartFeatures(nil)
	mc := func() []math.Point3 { return []math.Point3{math.P3(2, 2, 0), math.P3(-2, 2, 4)} }
	NewLoftFeatures(fs).AddGuided(
		[]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}},
		false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{MapCurves: []func() []math.Point3{mc}},
	)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*LoftFeature).Definition()
	if len(got.MapCurves) != 1 {
		t.Fatalf("map-curve round-trip: got %d curves, want 1", len(got.MapCurves))
	}
	if pts := got.MapCurves[0](); len(pts) != 2 || float64(pts[0].X) != 2 || float64(pts[1].X) != -2 {
		t.Errorf("map-curve round-trip lost the anchors: %v", pts)
	}
}
