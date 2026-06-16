// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Punch Tool feature (M13-F03). A punch stamps every closed profile of a sketch
// through the sheet in one shot — the die-pattern counterpart of the single-profile Cut, for a
// row of louvers, a grid of vents, or a perforation array placed from one sketch. By default it
// punches through all the material; a depth limits it to a coined/embossed cutout. The geometry
// reuses the shared profile-prism builder + through-all span, so a punch is the boolean
// complement of the sketch's thickened profiles.

// SheetMetalPunchDefinition is the punch recipe: the sketch whose closed profiles are stamped,
// and an optional depth (nil ⇒ through all).
type SheetMetalPunchDefinition struct {
	Sketch    *sketch.Sketch
	Direction ExtentDirection
	Depth     func() float64 // nil ⇒ punch through all the material
}

// SheetMetalPunchFeature stamps the sketch's profiles through the running sheet each recompute.
type SheetMetalPunchFeature struct {
	def      *SheetMetalPunchDefinition
	featName string
}

// Definition returns the punch recipe.
func (f *SheetMetalPunchFeature) Definition() *SheetMetalPunchDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalPunchFeature) Kind() string { return "sheet-metal-punch" }

// Recompute resolves every closed profile of the sketch, builds the punch tool, and subtracts
// it from the sheet.
func (f *SheetMetalPunchFeature) Recompute(in Input) (Output, error) {
	n := f.def.Sketch.Profiles().Count()
	if n == 0 {
		return Output{}, fmt.Errorf("sheet-metal punch: the sketch has no closed profile to punch")
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	profiles, err := resolveClosedProfiles(f.def.Sketch, indices, "sheet-metal punch")
	if err != nil {
		return Output{}, err
	}
	plane := f.def.Sketch.Plane()
	sp, err := f.punchSpan(in.Bodies, plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildProfilePrisms(profiles, plane, sp, 0, f.featName)
	bodies, err := combine(in.Bodies, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// punchSpan returns the punch's span: a fixed depth when Depth is set, else through all the
// running material on the chosen side of the sketch plane.
func (f *SheetMetalPunchFeature) punchSpan(bodies []*topo.Body, plane sketch.Plane) (span, error) {
	if f.def.Depth != nil {
		return distanceSpan(Extent{Type: DistanceExtent, Direction: f.def.Direction, Distance: f.def.Depth})
	}
	return throughAllSpan(Extent{Type: ThroughAllExtent, Direction: f.def.Direction}, bodies, plane)
}

// SheetMetalPunchFeatures adds punch features into the engine.
type SheetMetalPunchFeatures struct{ engine *PartFeatures }

// NewSheetMetalPunchFeatures binds the collection to a feature engine.
func NewSheetMetalPunchFeatures(engine *PartFeatures) *SheetMetalPunchFeatures {
	return &SheetMetalPunchFeatures{engine}
}

// Add appends a punch feature, naming it Punch1, Punch2, … .
func (c *SheetMetalPunchFeatures) Add(def *SheetMetalPunchDefinition) *PartFeature {
	f := &SheetMetalPunchFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Punch"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalPunchFeature)(nil)
