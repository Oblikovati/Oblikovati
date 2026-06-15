// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// Simplify is a model-reduction feature (M20-F13): it removes a selected set of faces (healing
// the openings, like Delete Face) and/or fills internal voids, producing a lighter body for
// downstream use (display, drawings, analysis). It bundles the two reductions into one history
// step distinct from the single Delete Face edit.

// SimplifyDefinition is the reduction recipe: faces to remove and whether to fill voids.
type SimplifyDefinition struct {
	RemoveFaceKeys [][]byte
	FillVoids      bool
}

// SimplifyFeature reduces the running body.
type SimplifyFeature struct{ def *SimplifyDefinition }

// Definition returns the simplify recipe.
func (s *SimplifyFeature) Definition() *SimplifyDefinition { return s.def }

// Kind implements [Feature].
func (s *SimplifyFeature) Kind() string { return "simplify" }

// Recompute removes the selected faces (healing the openings) then fills internal voids.
func (s *SimplifyFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if len(s.def.RemoveFaceKeys) == 0 && !s.def.FillVoids {
		return Output{}, fmt.Errorf("simplify: nothing to do (no faces to remove and fillVoids off)")
	}
	result := body
	if len(s.def.RemoveFaceKeys) > 0 {
		if result, err = ops.DeleteFaces(result, s.def.RemoveFaceKeys); err != nil {
			return Output{}, fmt.Errorf("simplify: %w", err)
		}
	}
	if s.def.FillVoids {
		result = ops.FillInternalVoids(result, ops.DefaultQuality())
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// AddSimplify reduces the running body by removing faceKeys (healing) and optionally filling
// internal voids.
func (c *ModifyFeatures) AddSimplify(faceKeys [][]byte, fillVoids bool) *PartFeature {
	return c.engine.Add(&SimplifyFeature{def: &SimplifyDefinition{RemoveFaceKeys: faceKeys, FillVoids: fillVoids}})
}
