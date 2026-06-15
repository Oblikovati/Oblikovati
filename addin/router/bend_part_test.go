// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// TestBendPartViaWireFoldsBody drives the full wire path for an M20-F17 bend — a bend line
// across the extruded block, then features.add(bendPart) — and checks it recomputes healthy
// into a single body (the fold-up geometry is asserted by the model-level test) (#651).
func TestBendPartViaWireFoldsBody(t *testing.T) {
	r, s, _ := extrudedPartViaAPI(t) // block x∈[0,4] y∈[0,3] z∈[0,5] cm, Extrusion1
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"line","points":[[2,0],[2,3]]}`, &struct{}{})
	call(t, r, s, "features.add",
		`{"kind":"bendPart","args":{"sketchIndex":1,"lineIndex":0,"bendType":"radiusAndAngle","radius":"5 mm","angle":"90 deg"}}`,
		&struct{}{})

	tree := modelTreeOf(t, r, s)
	if tree.Bodies != 1 {
		t.Fatalf("after bend body count = %d, want 1 (bend replaces, not adds)", tree.Bodies)
	}
	bend := tree.Features[len(tree.Features)-1]
	if bend.Kind != "bend-part" || bend.Health != "" {
		t.Fatalf("bend feature = kind %q health %q, want bend-part / healthy", bend.Kind, bend.Health)
	}
}

// TestBendPartScalarsEditable exposes the two driving scalars through features.edit.
func TestBendPartScalarsEditable(t *testing.T) {
	r, s, _ := extrudedPartViaAPI(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"line","points":[[2,0],[2,3]]}`, &struct{}{})
	call(t, r, s, "features.add",
		`{"kind":"bendPart","args":{"sketchIndex":1,"lineIndex":0,"radius":"5 mm","angle":"90 deg"}}`,
		&struct{}{})
	bendID := modelTreeOf(t, r, s).Features[1].ID

	var detail wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, bendID), &detail)
	if len(detail.Feature.Scalars) != 2 || detail.Feature.Scalars[0].Label != "Radius" {
		t.Fatalf("bend scalars = %+v, want [Radius, Angle]", detail.Feature.Scalars)
	}
}

// TestBendPartUnknownTypeFails rejects an unknown bend type with a precise error.
func TestBendPartUnknownTypeFails(t *testing.T) {
	r, s, _ := extrudedPartViaAPI(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"line","points":[[2,0],[2,3]]}`, &struct{}{})
	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"bendPart","args":{"sketchIndex":1,"lineIndex":0,"bendType":"twist"}}`)); err == nil {
		t.Error("expected an error for an unknown bendType")
	}
}
