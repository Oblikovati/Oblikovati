// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// hemCorneredSheet folds a flange on one edge and a hem on the adjacent edge, so the hem's bend
// meets the flange's at a corner, and returns the result under the given corner relief (#2072).
func hemCorneredSheet(t *testing.T, corner CornerReliefSpec) *topo.Body {
	t.Helper()
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} }) // isolate the CORNER cut
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return corner })
	fs.SetBendTransition(func() types.BendTransition { return types.NoBendTransition })

	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	hem := NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: edgeY.ReferenceKey(), Type: SingleHem, Length: constClosure(0.5), Gap: constClosure(0.1),
	})
	fs.Recompute()
	if !hem.Health().OK() {
		t.Fatalf("hem meeting the flange went sick: %s", hem.Health().Reason)
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("hem+flange corner is not a valid solid: %+v", r)
	}
	return body
}

// TestHemCornerReliefRemovesTheSharedCorner is the #2072 item-5 regression: a HEM meeting a flange at
// a corner cuts the styled corner relief, just as a flange-to-flange corner does — before this the
// hem placed its bend but relieved nothing, so the junction it formed was left full of material.
func TestHemCornerReliefRemovesTheSharedCorner(t *testing.T) {
	t.Parallel()
	relieved := query.BodyGeometryProperties(
		hemCorneredSheet(t, CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4}), ops.DefaultQuality()).Volume
	// The same part with the corner left to tear removes nothing there, so it has MORE material.
	torn := query.BodyGeometryProperties(
		hemCorneredSheet(t, CornerReliefSpec{Shape: types.CornerTear}), ops.DefaultQuality()).Volume

	if relieved >= torn-1e-6 {
		t.Errorf("square corner relief volume %g is not less than the un-relieved %g — the hem did "+
			"not cut its corner (#2072)", relieved, torn)
	}
}

// TestHemCornerReliefRefusesShapingTransition: a shaping bend transition (arc/straight/intersection)
// is not built yet, and a hem meeting a flange is a junction — so the hem refuses it there rather
// than silently ignoring it, exercising the corner-relief error path in the hem's recompute.
func TestHemCornerReliefRefusesShapingTransition(t *testing.T) {
	t.Parallel()
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4} })
	fs.SetBendTransition(func() types.BendTransition { return types.StraightLineBendTransition })

	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	hem := NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: edgeY.ReferenceKey(), Type: SingleHem, Length: constClosure(0.5), Gap: constClosure(0.1),
	})
	fs.Recompute()
	if hem.Health().OK() {
		t.Error("a hem forming a junction under a not-yet-built shaping transition should be sick")
	}
}

// TestHemNeedsABody: a hem with no prior solid to fold goes sick rather than panicking — foldHem
// surfaces the missing body.
func TestHemNeedsABody(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	hem := NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: []byte("x"), Type: SingleHem, Length: constClosure(0.5), Gap: constClosure(0.1),
	})
	fs.Recompute()
	if hem.Health().OK() {
		t.Error("a hem with no prior body should be sick")
	}
}

// TestHemRejectsUnresolvableEdge: a hem whose edge key resolves to nothing goes sick — foldHem
// surfaces the resolve failure instead of indexing an empty edge slice.
func TestHemRejectsUnresolvableEdge(t *testing.T) {
	t.Parallel()
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	hem := NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: []byte("no-such-edge"), Type: SingleHem, Length: constClosure(0.5), Gap: constClosure(0.1),
	})
	fs.Recompute()
	if hem.Health().OK() {
		t.Error("a hem on an unresolvable edge should be sick")
	}
}
