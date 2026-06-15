// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Grill (M20-F10 #863) is a plastic ventilation feature: a boundary region of a thin wall is
// cut open and bridged by a structure of ribs, spars and islands that stay solid (the reference
// GrillFeature). The structure is drawn as the inner loops of the boundary profile(s): a sketch
// rectangle (the vent) with rib/spar/island shapes inside it resolves to one profile whose
// outer loop is the boundary and whose holes are the kept structure. Cutting that profile
// through the wall removes the open area and leaves the structure bridging the vent.
//
// Geometrically every kept element (rib/spar/island) is a hole of the cut profile; they share
// one cut depth. A draft angle tapers the vent walls.

// GrillDefinition is the grill recipe: the boundary vent profile(s) (whose inner loops are the
// bridging structure), in one sketch, and a draft angle on the cut walls.
type GrillDefinition struct {
	Sketch     *sketch.Sketch
	Boundaries []int
	Draft      float64
}

// GrillFeature cuts the vent(s), leaving each boundary profile's inner-loop structure bridging.
type GrillFeature struct {
	def      *GrillDefinition
	featName string
}

// Definition returns the grill recipe.
func (g *GrillFeature) Definition() *GrillDefinition { return g.def }

// Kind implements [Feature].
func (g *GrillFeature) Kind() string { return "grill" }

// Recompute cuts the boundary profiles (honoring their structure holes) through the running
// wall, leaving the ribs/spars/islands bridging the vent.
func (g *GrillFeature) Recompute(in Input) (Output, error) {
	if _, err := lastBody(in, "grill"); err != nil {
		return Output{}, err
	}
	if len(g.def.Boundaries) == 0 {
		return Output{}, fmt.Errorf("grill: no boundary profile selected")
	}
	boundaries, err := resolveClosedProfiles(g.def.Sketch, g.def.Boundaries, "grill boundary")
	if err != nil {
		return Output{}, err
	}
	plane := g.def.Sketch.Plane()
	sp := throughSpan(in.Bodies, plane)
	cutTool := buildProfilePrisms(boundaries, plane, sp, g.def.Draft, featOr(g.featName, "grill"))
	bodies, err := combine(in.Bodies, cutTool, ops.Cut)
	if err != nil {
		return Output{}, fmt.Errorf("grill: %w", err)
	}
	return Output{Bodies: bodies}, nil
}

// throughSpan spans the whole running material along the sketch-plane normal (plus a margin on
// each side) so the vent cut passes cleanly through the wall whichever side the sketch sits on.
func throughSpan(bodies []*topo.Body, plane sketch.Plane) span {
	lo, hi := normalExtent(bodies, plane)
	return span{near: lo - throughAllMargin, far: hi + throughAllMargin}
}

// GrillFeatures adds grill features into the engine.
type GrillFeatures struct{ engine *PartFeatures }

// NewGrillFeatures binds the collection to a feature engine.
func NewGrillFeatures(engine *PartFeatures) *GrillFeatures { return &GrillFeatures{engine: engine} }

// Add creates a grill from its definition.
func (c *GrillFeatures) Add(def *GrillDefinition) *PartFeature {
	g := &GrillFeature{def: def}
	pf := c.engine.Add(g)
	pf.SetName(c.engine.UniqueName("Grill"))
	g.featName = pf.name
	return pf
}
