// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Feature reference re-pick (#163): the geometry half of features.edit. A feature whose
// definition is [feature.ReferenceEditable] exposes re-pickable slots (a fillet's edges, an
// extrude's profile, a mirror's plane). featureSlots renders them for FeatureDetail;
// planFeatureRepicks resolves wire refs into [feature.PickedRef] and returns an apply closure
// — resolution is read-only, so the whole batch (scalars + repicks) is validated before ANY
// mutation, matching the scalar precedent (a failed batch leaves the definition untouched).

// refKindName maps a model RefKind to its wire spelling.
func refKindName(k feature.RefKind) string {
	switch k {
	case feature.RefEdges:
		return "edges"
	case feature.RefFaces:
		return "faces"
	case feature.RefFace:
		return "face"
	case feature.RefProfile:
		return "profile"
	case feature.RefPlane:
		return "plane"
	default:
		return "unknown"
	}
}

// featureSlots renders a feature's re-pickable reference slots; nil when the definition has none.
func featureSlots(f *feature.PartFeature) []wire.FeatureSlot {
	re, ok := f.Definition().(feature.ReferenceEditable)
	if !ok {
		return nil
	}
	slots := re.EditableRefs()
	out := make([]wire.FeatureSlot, len(slots))
	for i, sl := range slots {
		out[i] = wire.FeatureSlot{Index: i, Label: sl.Label, Kind: refKindName(sl.Kind), Multi: sl.Multi, Count: sl.Count()}
	}
	return out
}

// plannedRepick is a validated repick (read-only resolution done) ready to apply.
type plannedRepick struct {
	slot  feature.EditableRefSlot
	clear bool
	ref   feature.PickedRef
}

// planFeatureRepicks validates every repick (slot in range, clearable if Clear, profile index
// and plane ref resolvable) WITHOUT mutating, and returns a closure that applies them. An error
// means nothing was applied. Edge/face topology keys are not bind-checked here — they reference
// the feature's INPUT (rollback) geometry, not the final body, and a lost reference surfaces as
// feature health after recompute, exactly as it does for features.add (parametric, not edit-time).
func planFeatureRepicks(part *compdef.PartComponentDefinition, f *feature.PartFeature, repicks []wire.FeatureRepick) (func(), error) {
	if len(repicks) == 0 {
		return func() {}, nil
	}
	re, ok := f.Definition().(feature.ReferenceEditable)
	if !ok {
		return nil, fmt.Errorf("features.edit: feature %d (%s) has no re-pickable references", uint64(f.ID()), f.Kind())
	}
	slots := re.EditableRefs()
	planned := make([]plannedRepick, 0, len(repicks))
	for _, rp := range repicks {
		p, err := planOneRepick(part, slots, rp)
		if err != nil {
			return nil, err
		}
		planned = append(planned, p)
	}
	return func() {
		for _, p := range planned {
			if p.clear {
				p.slot.Clear()
			} else {
				p.slot.Add(p.ref)
			}
		}
	}, nil
}

// planOneRepick validates a single repick against its slot.
func planOneRepick(part *compdef.PartComponentDefinition, slots []feature.EditableRefSlot, rp wire.FeatureRepick) (plannedRepick, error) {
	if rp.Slot < 0 || rp.Slot >= len(slots) {
		return plannedRepick{}, fmt.Errorf("features.edit: repick slot %d out of range (%d slots)", rp.Slot, len(slots))
	}
	sl := slots[rp.Slot]
	if rp.Clear {
		if sl.Clear == nil {
			return plannedRepick{}, fmt.Errorf("features.edit: slot %d (%s) is not clearable", rp.Slot, sl.Label)
		}
		return plannedRepick{slot: sl, clear: true}, nil
	}
	pr, err := resolvePickedRef(part, sl.Kind, rp)
	if err != nil {
		return plannedRepick{}, fmt.Errorf("features.edit: slot %d (%s): %w", rp.Slot, sl.Label, err)
	}
	return plannedRepick{slot: sl, ref: pr}, nil
}

// resolvePickedRef turns a wire repick into a [feature.PickedRef] for the slot's kind. Edge/face
// keys pass through as raw key bytes (from model.referenceKeys); a profile index and a plane ref
// are validated against the part (a missing profile / unresolvable plane is an edit-time error).
func resolvePickedRef(part *compdef.PartComponentDefinition, kind feature.RefKind, rp wire.FeatureRepick) (feature.PickedRef, error) {
	switch kind {
	case feature.RefEdges, feature.RefFaces, feature.RefFace:
		return feature.PickedRef{Key: []byte(rp.Ref)}, nil
	case feature.RefProfile:
		return resolveProfilePick(part, rp)
	case feature.RefPlane:
		return resolvePlanePick(part, rp.Ref)
	default:
		return feature.PickedRef{}, fmt.Errorf("unsupported slot kind")
	}
}

// resolveProfilePick validates the (sketchIndex, profileIndex) against the part's sketches.
func resolveProfilePick(part *compdef.PartComponentDefinition, rp wire.FeatureRepick) (feature.PickedRef, error) {
	sketches := part.Sketches()
	if rp.SketchIndex < 0 || rp.SketchIndex >= sketches.Count() {
		return feature.PickedRef{}, fmt.Errorf("sketch index %d out of range (%d sketches)", rp.SketchIndex, sketches.Count())
	}
	sk := sketches.Item(rp.SketchIndex)
	if n := sk.Profiles().Count(); rp.ProfileIndex < 0 || rp.ProfileIndex >= n {
		return feature.PickedRef{}, fmt.Errorf("profile index %d out of range (sketch has %d profiles)", rp.ProfileIndex, n)
	}
	return feature.PickedRef{Sketch: sk, Profile: rp.ProfileIndex}, nil
}

// resolvePlanePick resolves a plane ref — a planar-face key, a work plane ("plane/N"), or an
// origin plane ("origin/plane/xy") — to a plane pick. A face ref also carries its key so the
// mirror tracks the face through edits.
func resolvePlanePick(part *compdef.PartComponentDefinition, ref string) (feature.PickedRef, error) {
	wr := toWorkRef(ref)
	pl, err := part.WorkGeometry().ResolvePlaneRef(wr)
	if err != nil {
		return feature.PickedRef{}, err
	}
	pr := feature.PickedRef{Origin: pl.Origin(), Normal: pl.Normal().AsVector()}
	if key, isFace := feature.FaceRefKey(wr); isFace {
		pr.PlaneKey = key
	}
	return pr, nil
}
