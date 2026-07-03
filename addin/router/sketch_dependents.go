// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"strings"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Sketch consumed/owned-by state + dependents (#154). A sketch is "consumed" when a feature
// uses it as profile/path/centerline input — the same relationship the model browser nests by
// ([feature.PartFeature.ConsumedSketches]). The owning feature is the first to consume it;
// every consumer is a dependent that a delete or edit affects.

// sketchConsumers returns the features that consume sk, in history order.
func sketchConsumers(part *compdef.PartComponentDefinition, sk *sketch.Sketch) []*feature.PartFeature {
	var out []*feature.PartFeature
	feats := part.Features()
	for i := 0; i < feats.Count(); i++ {
		f := feats.Item(i)
		for _, c := range f.ConsumedSketches() {
			if c == sk {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

// sketchDependentRefs renders a sketch's consuming features as wire dependents.
func sketchDependentRefs(part *compdef.PartComponentDefinition, sk *sketch.Sketch) []wire.SketchDependent {
	cons := sketchConsumers(part, sk)
	out := make([]wire.SketchDependent, len(cons))
	for i, f := range cons {
		out[i] = wire.SketchDependent{ID: uint64(f.ID()), Name: f.Name(), Kind: f.Kind()}
	}
	return out
}

// sketchDependents lists the features that consume the addressed sketch (#154).
func sketchDependents(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.SketchDependentsResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchDependentsResult{}, err
	}
	return wire.SketchDependentsResult{Dependents: sketchDependentRefs(part, sk)}, nil
}

// dependentNames joins the dependent feature names for a delete-guard error message.
func dependentNames(deps []wire.SketchDependent) string {
	names := make([]string, len(deps))
	for i, d := range deps {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

// rejectIfConsumed returns an error naming the dependents when the sketch is still used by a
// feature — Inventor's "Delete is only valid for sketches not used by a feature" (#154).
func rejectIfConsumed(part *compdef.PartComponentDefinition, sk *sketch.Sketch) error {
	deps := sketchDependentRefs(part, sk)
	if len(deps) == 0 {
		return nil
	}
	return fmt.Errorf("sketch.delete: sketch %q is used by %d feature(s): %s — delete the features first (or re-pick their profiles)",
		sk.Name(), len(deps), dependentNames(deps))
}
