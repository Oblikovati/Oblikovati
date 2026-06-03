// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// MoveFeature rigidly relocates a running body by a transform (translate / rotate /
// reflect / uniform scale). Unlike a pattern it creates no copy: the body is moved
// in place, so its reference keys are preserved (the identity lineage map) and picks
// against it survive the move (Inventor's MoveFeature / direct-edit move-body).

// MoveDefinition is the recipe: which running body (by index) and the transform.
type MoveDefinition struct {
	BodyIndex int
	Transform math.Matrix4
}

// MoveFeature applies the definition's transform to the target body each recompute.
type MoveFeature struct{ def *MoveDefinition }

// Definition returns the move recipe.
func (m *MoveFeature) Definition() *MoveDefinition { return m.def }

// Kind implements [Feature].
func (m *MoveFeature) Kind() string { return "move" }

// Recompute transforms the target body in place, leaving the others untouched.
func (m *MoveFeature) Recompute(in Input) (Output, error) {
	if !validIndex(m.def.BodyIndex, in.Bodies) {
		return Output{}, fmt.Errorf("move: invalid body index %d (have %d)", m.def.BodyIndex, len(in.Bodies))
	}
	moved, err := ops.TransformBody(in.Bodies[m.def.BodyIndex], m.def.Transform, keepLineage)
	if err != nil {
		return Output{}, err
	}
	out := append([]*topo.Body(nil), in.Bodies...)
	out[m.def.BodyIndex] = moved
	return Output{Bodies: out}, nil
}

// keepLineage is the identity lineage map: an in-place move keeps reference keys so
// downstream features that picked the moved body's faces still resolve.
func keepLineage(l topo.Lineage) topo.Lineage { return l }

// AddMove relocates the running body at bodyIndex by transform.
func (c *ModifyFeatures) AddMove(bodyIndex int, transform math.Matrix4) *PartFeature {
	return c.engine.Add(&MoveFeature{def: &MoveDefinition{BodyIndex: bodyIndex, Transform: transform}})
}
