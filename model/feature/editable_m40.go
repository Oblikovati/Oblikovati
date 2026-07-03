// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/model/param"
)

// M40 audit S4 (#1639): close the edit-after-create gap for loft, the fillet variants, grill and
// boss. Each mirrors the base-fillet pattern in editable.go — EditableParams for scalars, an
// EditableRefs slot per geometric input — so double-clicking any of them opens the same generic
// edit dialog and re-picking its references rebinds the feature rather than forcing delete-and-
// recreate (which destroys every downstream reference through topological-naming loss).
var (
	_ ReferenceEditable = (*LoftFeature)(nil)
	_ Editable          = (*LoftFeature)(nil)
	_ ReferenceEditable = (*FaceFilletFeature)(nil)
	_ Editable          = (*FaceFilletFeature)(nil)
	_ ReferenceEditable = (*FullRoundFilletFeature)(nil)
	_ ReferenceEditable = (*GrillFeature)(nil)
	_ Editable          = (*GrillFeature)(nil)
	_ ReferenceEditable = (*BossFeature)(nil)
)

// EditableParams exposes the loft's start/end takeoff conditions — the angle each end section
// leaves its plane at and the impact (weight) of that takeoff. A loft driven by LiveEnds (a
// parameter feeding the end conditions afresh each recompute) overrides these static fields, so
// editing them there is a no-op until the parameter is cleared.
func (l *LoftFeature) EditableParams() []EditableParam {
	return []EditableParam{
		floatParam("Start angle", param.Angle, &l.def.First.Angle),
		floatParam("Start weight", param.Unitless, &l.def.First.Impact),
		floatParam("End angle", param.Angle, &l.def.Last.Angle),
		floatParam("End weight", param.Unitless, &l.def.Last.Impact),
	}
}

// EditableRefs exposes each sketch-profile loft section for re-selection (Section 1, Section 2 …).
// Point and face sections are not sketch profiles, so they are skipped; rails/centerline are live
// point-providers with no pickable key, so they are not re-picked here (a follow-up slice).
func (l *LoftFeature) EditableRefs() []EditableRefSlot {
	slots := make([]EditableRefSlot, 0, len(l.def.Sections))
	for i := range l.def.Sections {
		if l.def.Sections[i].Sketch == nil {
			continue
		}
		slots = append(slots, loftSectionSlot(i+1, &l.def.Sections[i]))
	}
	return slots
}

// loftSectionSlot builds a single re-pickable profile slot over one section's (sketch, index) pair,
// labelled by its 1-based position so the dialog lists "Section 1", "Section 2" …
func loftSectionSlot(n int, sec *LoftSection) EditableRefSlot {
	return EditableRefSlot{
		Label: fmt.Sprintf("Section %d", n), Kind: RefProfile, Multi: false,
		Count: func() int {
			if sec.Sketch == nil {
				return 0
			}
			return 1
		},
		Add: func(r PickedRef) { sec.Sketch = r.Sketch; sec.ProfileIndex = r.Profile },
		Snapshot: func() func() {
			sk, i := sec.Sketch, sec.ProfileIndex
			return func() { sec.Sketch = sk; sec.ProfileIndex = i }
		},
	}
}

// EditableRefs exposes the two face sets a face fillet rounds between.
func (f *FaceFilletFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{
		keyRefSlotMulti("Face set 1", RefFaces, &f.def.FaceKeysA),
		keyRefSlotMulti("Face set 2", RefFaces, &f.def.FaceKeysB),
	}
}

// EditableParams exposes the face fillet radius.
func (f *FaceFilletFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Radius", param.Length, &f.def.Radius)}
}

// EditableRefs exposes the full-round fillet's two side faces and the center face it replaces with a
// round. There is no editable radius — it is derived from the side-face spacing.
func (f *FullRoundFilletFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{
		keyRefSlotMulti("Side face 1", RefFaces, &f.def.Side1Keys),
		keyRefSlotMulti("Center face", RefFaces, &f.def.CenterKeys),
		keyRefSlotMulti("Side face 2", RefFaces, &f.def.Side2Keys),
	}
}

// EditableRefs exposes the grill's boundary vent profile for re-selection.
func (g *GrillFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndices(&g.def.Sketch, &g.def.Boundaries)}
}

// EditableParams exposes the grill's draft angle.
func (g *GrillFeature) EditableParams() []EditableParam {
	return []EditableParam{floatParam("Draft", param.Angle, &g.def.Draft)}
}

// EditableRefs exposes the boss stud's placement face — the parity HoleFeature already has (its
// diameter/height params live in editable_m10.go).
func (b *BossFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotSingle("Placement face", RefFace, &b.def.PlacementFaceKey)}
}
