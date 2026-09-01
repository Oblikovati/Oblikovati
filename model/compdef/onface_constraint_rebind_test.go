// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestOnFace3DConstraintRebindsOnReload: an onFace constraint on a feature-backed solid round-trips
// and re-binds its face source after a save/reload, so the point-on-face pin stays active and
// associative — the constraint analogue of projected-geometry rebind (#1839 AC2).
func TestOnFace3DConstraintRebindsOnReload(t *testing.T) {
	t.Parallel()
	def := extrudeBlock(t) // feature-backed block; its face keys survive recompute + reload
	faceKey := string(def.SurfaceBodies().All()[0].Faces()[0].ReferenceKey())

	sk := def.Sketches3D().Add()
	p := sk.AddPoint3D(math.P3(10, 0, 0))
	sk.GeometricConstraints3D().Add(sketch.NewOnFace3D(p, compdef.NewFaceRefSource(def, faceKey), faceKey))

	recipe, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	got := compdef.NewPartComponentDefinition()
	if err := got.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	var restored *sketch.OnFace3D
	for _, c := range got.Sketches3D().Item(0).GeometricConstraints3D().All() {
		if oc, ok := c.(*sketch.OnFace3D); ok {
			restored = oc
		}
	}
	if restored == nil {
		t.Fatal("onFace constraint lost on reload")
	}
	if restored.SurfaceRef() != faceKey {
		t.Errorf("restored SurfaceRef = %q, want %q", restored.SurfaceRef(), faceKey)
	}
	// Bound + the face resolves ⇒ the constraint is active again (contributes its point's DOFs).
	if len(restored.Variables()) != 3 {
		t.Errorf("restored onFace has %d variables, want 3 (rebound + active)", len(restored.Variables()))
	}
}
