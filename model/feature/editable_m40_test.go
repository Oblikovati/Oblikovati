// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// M40 S4 (#1639): loft, the fillet variants, grill and boss must all be editable after creation.
// These guard the EditableParams round-trip and the EditableRefs slot mechanics for each, mirroring
// editable_m10_test.go / editable_refs_test.go.

func TestS4FeaturesExposeEditableParams(t *testing.T) {
	assertParamsRoundTrip(t, "loft", &LoftFeature{def: &LoftDefinition{}}, 4)
	assertParamsRoundTrip(t, "grill", &GrillFeature{def: &GrillDefinition{}}, 1)
	assertParamsRoundTrip(t, "faceFillet", &FaceFilletFeature{def: &FaceFilletDefinition{Radius: constFloat(1)}}, 1)
}

// TestLoftEditableRefsSkipNonSketchSections checks that only sketch-profile sections become
// re-pickable slots (point/face sections have no profile to re-select) and that re-picking a
// section rebinds its (sketch, index) pair, with Snapshot restoring the prior binding.
func TestLoftEditableRefsSkipNonSketchSections(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	apex := math.P3(0, 0, 5)
	loft := &LoftFeature{def: &LoftDefinition{Sections: []LoftSection{
		{Sketch: sk, ProfileIndex: 0},
		{Point: &apex}, // a point section — not a sketch profile, so no slot
	}}}
	slots := loft.EditableRefs()
	if len(slots) != 1 {
		t.Fatalf("loft exposed %d ref slots, want 1 (only the sketch section)", len(slots))
	}
	if slots[0].Label != "Section 1" || slots[0].Kind != RefProfile || slots[0].Count() != 1 {
		t.Fatalf("loft slot = %+v, want one RefProfile 'Section 1' count 1", slots[0])
	}
	undo := slots[0].Snapshot()
	sk2 := sketch.NewSketches().Add(sketch.XYPlane())
	slots[0].Add(PickedRef{Sketch: sk2, Profile: 3})
	if loft.def.Sections[0].Sketch != sk2 || loft.def.Sections[0].ProfileIndex != 3 {
		t.Fatalf("after Add, section 0 = (%v,%d), want (sk2,3)", loft.def.Sections[0].Sketch, loft.def.Sections[0].ProfileIndex)
	}
	undo()
	if loft.def.Sections[0].Sketch != sk || loft.def.Sections[0].ProfileIndex != 0 {
		t.Fatalf("after restore, section 0 = (%v,%d), want (sk,0)", loft.def.Sections[0].Sketch, loft.def.Sections[0].ProfileIndex)
	}
}

// TestFaceFilletEditableRefs checks the two face-set slots re-pick independently.
func TestFaceFilletEditableRefs(t *testing.T) {
	ff := &FaceFilletFeature{def: &FaceFilletDefinition{
		FaceKeysA: [][]byte{[]byte("a1")}, FaceKeysB: [][]byte{[]byte("b1")}, Radius: constFloat(1)}}
	slots := ff.EditableRefs()
	if len(slots) != 2 || slots[0].Label != "Face set 1" || slots[1].Label != "Face set 2" {
		t.Fatalf("face-fillet slots = %v, want [Face set 1, Face set 2]", paramSlotLabels(slots))
	}
	slots[0].Add(PickedRef{Key: []byte("a2")})
	if len(ff.def.FaceKeysA) != 2 || len(ff.def.FaceKeysB) != 1 {
		t.Fatalf("after Add to set 1, A=%d B=%d, want 2 and 1", len(ff.def.FaceKeysA), len(ff.def.FaceKeysB))
	}
}

// TestFullRoundFilletEditableRefs checks the three face slots (no scalar param — the radius is
// derived from face spacing, so it is ReferenceEditable only).
func TestFullRoundFilletEditableRefs(t *testing.T) {
	fr := &FullRoundFilletFeature{def: &FullRoundFilletDefinition{
		Side1Keys: [][]byte{[]byte("s1")}, CenterKeys: [][]byte{[]byte("c")}, Side2Keys: [][]byte{[]byte("s2")}}}
	slots := fr.EditableRefs()
	want := []string{"Side face 1", "Center face", "Side face 2"}
	if got := paramSlotLabels(slots); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("full-round slots = %v, want %v", got, want)
	}
	if _, ok := any(fr).(Editable); ok {
		t.Error("full-round fillet should not be Editable (no scalar param)")
	}
}

// TestBossEditableRefs checks the boss gained its single placement-face slot (parity with hole).
func TestBossEditableRefs(t *testing.T) {
	boss := &BossFeature{def: &BossDefinition{PlacementFaceKey: []byte("face-1"), Diameter: constFloat(2), Height: constFloat(4)}}
	slot := boss.EditableRefs()[0]
	if slot.Label != "Placement face" || slot.Kind != RefFace || slot.Count() != 1 {
		t.Fatalf("boss slot = %+v, want one RefFace 'Placement face' count 1", slot)
	}
	slot.Add(PickedRef{Key: []byte("face-2")})
	if string(boss.def.PlacementFaceKey) != "face-2" {
		t.Fatalf("after Add, boss face = %q, want face-2", boss.def.PlacementFaceKey)
	}
}

// TestGrillEditableRefs checks grill exposes its boundary profile slot.
func TestGrillEditableRefs(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	grill := &GrillFeature{def: &GrillDefinition{Sketch: sk, Boundaries: []int{0}}}
	slot := grill.EditableRefs()[0]
	if slot.Kind != RefProfile || slot.Count() != 1 {
		t.Fatalf("grill slot = %+v, want one RefProfile count 1", slot)
	}
}

// TestLoftSectionSlotCountReflectsClearedSketch covers the slot's Count closure reporting 0 once its
// section's sketch is cleared out from under it (a bound → unbound profile), and 1 while bound.
func TestLoftSectionSlotCountReflectsClearedSketch(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sec := LoftSection{Sketch: sk, ProfileIndex: 0}
	slot := loftSectionSlot(1, &sec)
	if slot.Count() != 1 {
		t.Fatalf("Count with a bound sketch = %d, want 1", slot.Count())
	}
	sec.Sketch = nil
	if slot.Count() != 0 {
		t.Errorf("Count after the sketch is cleared = %d, want 0", slot.Count())
	}
}

func paramSlotLabels(slots []EditableRefSlot) []string {
	out := make([]string, len(slots))
	for i, s := range slots {
		out[i] = s.Label
	}
	return out
}
