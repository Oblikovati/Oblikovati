// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Cut feature (M13-F03). A cut removes a closed sketch profile from the sheet,
// normal to the sketch plane. By default it cuts through all the material (the common case —
// holes, slots, openings); a distance limits it. This is the normal cut; the across-bend
// variant (unfold the spanned region, cut flat, refold) builds on the flat pattern (F04) and
// is a follow-up — AcrossBend is reserved and rejected until then.
//
// The geometry reuses the shared profile-prism builder + the through-all span computation, so
// a cut is the boolean complement of the base Face's thickening.

// SheetMetalCutDefinition is the cut recipe: the profile to remove, the side to cut toward,
// an optional depth (nil ⇒ through all), and the (reserved) across-bend flag.
type SheetMetalCutDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Direction    ExtentDirection
	Distance     func() float64 // nil ⇒ cut through all the material
	AcrossBend   bool           // reserved for F04; rejected until the flat pattern lands
}

// SheetMetalCutFeature removes the profile from the running sheet each recompute.
type SheetMetalCutFeature struct {
	def      *SheetMetalCutDefinition
	featName string
}

// Definition returns the cut recipe.
func (f *SheetMetalCutFeature) Definition() *SheetMetalCutDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalCutFeature) Kind() string { return "sheet-metal-cut" }

// Recompute resolves the profile, builds the cutter prism, and subtracts it from the sheet.
func (f *SheetMetalCutFeature) Recompute(in Input) (Output, error) {
	if f.def.AcrossBend {
		return Output{}, fmt.Errorf("sheet-metal cut: across-bend cuts need the flat pattern (M13-F04); not yet supported")
	}
	profiles, err := resolveClosedProfiles(f.def.Sketch, []int{f.def.ProfileIndex}, "sheet-metal cut")
	if err != nil {
		return Output{}, err
	}
	plane := f.def.Sketch.Plane()
	sp, err := f.cutSpan(in.Bodies, plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildProfilePrisms(profiles, plane, sp, 0, f.featName, in.Diag)
	bodies, err := combine(in, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// cutSpan returns the cutter's span: a fixed depth when Distance is set, else through all the
// running material on the chosen side of the sketch plane.
func (f *SheetMetalCutFeature) cutSpan(bodies []*topo.Body, plane sketch.Plane) (span, error) {
	if f.def.Distance != nil {
		ext := Extent{Type: DistanceExtent, Direction: f.def.Direction, Distance: f.def.Distance}
		return distanceSpan(ext)
	}
	return throughAllSpan(Extent{Type: ThroughAllExtent, Direction: f.def.Direction}, bodies, plane)
}

// SheetMetalCutFeatures adds cut features into the engine.
type SheetMetalCutFeatures struct{ engine *PartFeatures }

// NewSheetMetalCutFeatures binds the collection to a feature engine.
func NewSheetMetalCutFeatures(engine *PartFeatures) *SheetMetalCutFeatures {
	return &SheetMetalCutFeatures{engine}
}

// Add appends a cut feature, naming it Cut1, Cut2, … .
func (c *SheetMetalCutFeatures) Add(def *SheetMetalCutDefinition) *PartFeature {
	f := &SheetMetalCutFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Cut"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalCutFeature)(nil)
