// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// RestDefinition is a rest pad (#486, M20-F10 plastic features): a flat landing bounded by a closed
// sketch profile, raised above the face by Depth (joined) or recessed into it (cut), along the sketch-
// plane normal, with an optional draft. It is the profile-based sibling of the snap-fit; geometrically
// it extrudes the profile and combines it with the running body, like emboss, but is its own feature
// so it serializes as a "rest" and can grow rest-specific options (the surrounding land) later.
type RestDefinition struct {
	Sketch         *sketch.Sketch
	ProfileIndices []int
	Depth          func() float64
	Recessed       bool    // cut a recess into the part instead of raising a pad
	Taper          float64 // draft angle (radians)
}

// RestFeature adds a raised or recessed rest pad.
type RestFeature struct {
	def      *RestDefinition
	tool     *topo.Body
	featName string
}

// Definition returns the feature's definition.
func (f *RestFeature) Definition() *RestDefinition { return f.def }

// Kind names the feature type.
func (f *RestFeature) Kind() string { return "rest" }

// Recompute extrudes the profile to the rest depth and joins (pad) or cuts (recess) it into the body.
func (f *RestFeature) Recompute(in Input) (Output, error) {
	profiles, err := resolveClosedProfiles(f.def.Sketch, f.def.ProfileIndices, "rest")
	if err != nil {
		return Output{}, err
	}
	d := callOrZero(f.def.Depth)
	if d <= 0 {
		return Output{}, fmt.Errorf("rest: depth %g must be > 0", d)
	}
	sp, op := orderedSpan(0, d), ops.Join
	if f.def.Recessed {
		sp, op = orderedSpan(0, -d), ops.Cut // recess: cut into the part, below the sketch plane
	}
	f.tool = buildProfilePrisms(profiles, f.def.Sketch.Plane(), sp, f.def.Taper, featOr(f.featName, "rest"))
	bodies, err := combine(in.Bodies, f.tool, op)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// AddRest adds a rest pad over the given closed profiles: raised by depth (recessed cuts instead),
// with an optional draft taper.
func (c *PlasticFeatures) AddRest(skt *sketch.Sketch, profileIndices []int, depth func() float64, recessed bool, taper float64) *PartFeature {
	return c.engine.Add(&RestFeature{def: &RestDefinition{
		Sketch: skt, ProfileIndices: profileIndices, Depth: depth, Recessed: recessed, Taper: taper,
	}})
}
