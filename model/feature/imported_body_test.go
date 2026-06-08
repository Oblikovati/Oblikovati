// SPDX-License-Identifier: GPL-2.0-only

package feature

import "testing"

// TestImportedBodiesGetUniqueNames guards that each body of a multi-solid import gets a
// distinct, readable name (not the bare kind "importedBody" repeated) — repeated identical
// names collide as Dear ImGui ids in the model browser and break it.
func TestImportedBodiesGetUniqueNames(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	imp := NewImportedBodies(fs)
	a := imp.AddAt(nil, "edf.step", "step", 0)
	b := imp.AddAt(nil, "edf.step", "step", 1)
	c := imp.AddAt(nil, "edf.step", "step", 2)

	if a.Name() != "Imported Body1" {
		t.Errorf("first imported body = %q, want Imported Body1", a.Name())
	}
	seen := map[string]bool{}
	for _, pf := range []*PartFeature{a, b, c} {
		if pf.Name() == "importedBody" {
			t.Errorf("imported body kept the bare kind name %q", pf.Name())
		}
		if seen[pf.Name()] {
			t.Errorf("duplicate imported-body name %q", pf.Name())
		}
		seen[pf.Name()] = true
	}
}
