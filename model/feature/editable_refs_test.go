// SPDX-License-Identifier: GPL-2.0-only

package feature

import "testing"

// TestFeatureEditableRefs covers the EditableRefs slot exposure across the re-pickable feature
// kinds (the #163 feature-edit re-pick surface) and the slot-builder helpers behind them: each
// feature lists at least one labelled slot, and exercising the slot's accessors (Count/Keys/
// Snapshot) covers the keyRef/profileRef/planeRef closures.
func TestFeatureEditableRefs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		feat ReferenceEditable
	}{
		{"fillet", &FilletFeature{def: &FilletDefinition{}}},
		{"chamfer", &ChamferFeature{def: &ChamferDefinition{}}},
		{"shell", &ShellFeature{def: &ShellDefinition{}}},
		{"draft", &FaceDraftFeature{def: &FaceDraftDefinition{}}},
		{"hole", &HoleFeature{def: &HoleDefinition{}}},
		{"extrude", &ExtrudeFeature{def: &ExtrudeDefinition{}}},
		{"revolve", &RevolveFeature{def: &RevolveDefinition{}}},
		{"coil", &CoilFeature{def: &CoilDefinition{}}},
		{"rib", &RibFeature{def: &RibDefinition{}}},
		{"emboss", &EmbossFeature{def: &EmbossDefinition{}}},
		{"mirror", &MirrorFeature{def: &MirrorDefinition{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			slots := c.feat.EditableRefs()
			if len(slots) == 0 {
				t.Fatalf("%s exposed no editable ref slots", c.name)
			}
			for i, sl := range slots {
				if sl.Label == "" {
					t.Errorf("%s slot %d has an empty label", c.name, i)
				}
				if sl.Count != nil {
					_ = sl.Count() // exercises the slot's count closure
				}
				if sl.Keys != nil {
					_ = sl.Keys() // exercises the key-snapshot closure
				}
				if sl.Snapshot != nil {
					restore := sl.Snapshot()
					if restore != nil {
						restore() // exercises the undo-snapshot closure
					}
				}
			}
		})
	}
}
