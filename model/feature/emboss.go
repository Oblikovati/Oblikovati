// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// EmbossDefinition is the recipe for an emboss: a closed sketch profile raised from (or engraved
// into) the part by a depth along the sketch-plane normal. Engrave cuts; raise joins.
type EmbossDefinition struct {
	Sketch         *sketch.Sketch
	ProfileIndices []int
	Depth          func() float64
	Engrave        bool    // cut into the face instead of raising from it
	Taper          float64 // draft angle (radians)
}

// EmbossFeature raises or engraves a closed profile on the part (Inventor's Emboss): the profile
// is extruded a shallow depth along the sketch-plane normal and joined (raise) or cut (engrave).
// (Wrapping the emboss onto a curved face is a refinement; the planar emboss works today.)
type EmbossFeature struct {
	def      *EmbossDefinition
	featName string
	tool     *topo.Body // last raised/engraved prism, exposed so a pattern can replicate it
}

// Definition returns the emboss recipe.
func (f *EmbossFeature) Definition() *EmbossDefinition { return f.def }

// Kind implements [Feature].
func (f *EmbossFeature) Kind() string { return "emboss" }

// Operation and ToolBody let a pattern/mirror replicate this feature with the right boolean —
// an engrave cuts, a raise joins (see [ToolFeature]).
func (f *EmbossFeature) Operation() ops.PartFeatureOperation {
	if f.def.Engrave {
		return ops.Cut
	}
	return ops.Join
}
func (f *EmbossFeature) ToolBody() *topo.Body { return f.tool }

// Recompute extrudes the profile(s) by the depth and joins (raise) or cuts (engrave) the part.
func (f *EmbossFeature) Recompute(in Input) (Output, error) {
	profiles, err := resolveClosedProfiles(f.def.Sketch, f.def.ProfileIndices, "emboss")
	if err != nil {
		return Output{}, err
	}
	d := callOrZero(f.def.Depth)
	if d <= 0 {
		return Output{}, fmt.Errorf("emboss: depth %g must be > 0", d)
	}
	sp, op := orderedSpan(0, d), ops.Join
	if f.def.Engrave {
		sp, op = orderedSpan(0, -d), ops.Cut // cut into the part, below the sketch plane
	}
	f.tool = buildProfilePrisms(profiles, f.def.Sketch.Plane(), sp, f.def.Taper, featOr(f.featName, "emboss"))
	bodies, err := combine(in.Bodies, f.tool, op)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// EmbossFeatures adds emboss features into the engine.
type EmbossFeatures struct{ engine *PartFeatures }

// NewEmbossFeatures binds the collection to an engine.
func NewEmbossFeatures(engine *PartFeatures) *EmbossFeatures { return &EmbossFeatures{engine} }

// Add adds an emboss of the sketch's closed profile(s); engrave cuts instead of raising.
func (c *EmbossFeatures) Add(skt *sketch.Sketch, profileIndices []int, depth func() float64, engrave bool, taper float64) *PartFeature {
	def := &EmbossDefinition{Sketch: skt, ProfileIndices: profileIndices, Depth: depth, Engrave: engrave, Taper: taper}
	ef := &EmbossFeature{def: def}
	pf := c.engine.Add(ef)
	pf.SetName(c.engine.UniqueName("Emboss"))
	ef.featName = pf.name
	return pf
}

// resolveClosedProfiles returns the closed sketch profiles at the given indices — the shared
// profile resolution for solid-from-region features (extrude, emboss). `what` names the caller
// in errors.
func resolveClosedProfiles(sk *sketch.Sketch, indices []int, what string) ([]*sketch.Profile, error) {
	all := sk.Profiles()
	if len(indices) == 0 {
		return nil, fmt.Errorf("%s: no profile selected", what)
	}
	out := make([]*sketch.Profile, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= all.Count() {
			return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", what, idx, all.Count())
		}
		p := all.Item(idx)
		if !p.IsClosed() {
			return nil, fmt.Errorf("%s: profile is open (cannot form a solid)", what)
		}
		out = append(out, p)
	}
	return out, nil
}

// buildProfilePrisms extrudes each closed profile to a prism over span sp (with taper), merging
// several into one tool body — the shared prism builder for extrude/emboss.
func buildProfilePrisms(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string) *topo.Body {
	prisms := make([]*topo.Body, len(profiles))
	for i, p := range profiles {
		name := feat
		if len(profiles) > 1 {
			name = fmt.Sprintf("%s/p%d", feat, i)
		}
		prisms[i] = buildPrism(p.OuterLoop().Polygon(), plane, sp, taper, name)
	}
	if len(prisms) == 1 {
		return prisms[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(feat, "merged", 0)), true, prisms...)
}
