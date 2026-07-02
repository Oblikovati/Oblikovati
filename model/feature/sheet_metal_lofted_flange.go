// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Lofted Flange feature (M13-F02). A lofted flange is a constant-thickness wall
// that transitions between TWO open profiles on two sketch planes — the sheet-metal way to
// build a transition piece (e.g. a square-to-round duct). Each profile is thickened in its
// plane into a band; the two bands loft into one solid. Both profiles must have the same
// vertex count so their bands correspond point-for-point. Thickness is read live from the
// rule. The result is a new body (or joined to the running part).

// SheetMetalLoftedFlangeDefinition is the lofted-flange recipe: the two open profile sketches
// and the boolean operation (new body / join).
type SheetMetalLoftedFlangeDefinition struct {
	ProfileA  *sketch.Sketch
	ProfileB  *sketch.Sketch
	Operation ops.PartFeatureOperation
}

// SheetMetalLoftedFlangeFeature lofts the transition wall each recompute.
type SheetMetalLoftedFlangeFeature struct {
	def      *SheetMetalLoftedFlangeDefinition
	featName string
}

// Definition returns the lofted-flange recipe.
func (f *SheetMetalLoftedFlangeFeature) Definition() *SheetMetalLoftedFlangeDefinition {
	return f.def
}

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalLoftedFlangeFeature) Kind() string { return "sheet-metal-lofted-flange" }

// Recompute thickens each profile into a band and lofts the two into a transition wall.
func (f *SheetMetalLoftedFlangeFeature) Recompute(in Input) (Output, error) {
	t, err := sheetThickness(in.Params)
	if err != nil {
		return Output{}, err
	}
	bandA, err := loftBand(f.def.ProfileA, t)
	if err != nil {
		return Output{}, err
	}
	bandB, err := loftBand(f.def.ProfileB, t)
	if err != nil {
		return Output{}, err
	}
	if len(bandA) != len(bandB) {
		return Output{}, fmt.Errorf("sheet-metal lofted flange: profiles have %d and %d vertices; both must match to loft", len(bandA), len(bandB))
	}
	wall, err := sweptSolid([][]math.Point3{bandA, bandB}, false, featOr(f.featName, "loftedFlange"))
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal lofted flange: %w", err)
	}
	bodies, err := combine(in, wall, f.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// loftBand thickens an open profile into its closed 3D band on the profile's sketch plane.
func loftBand(sk *sketch.Sketch, thickness float64) ([]math.Point3, error) {
	profile, err := openProfilePoints(sk)
	if err != nil {
		return nil, err
	}
	band2D := profileBand2D(profile, thickness)
	plane := sk.Plane()
	band := make([]math.Point3, len(band2D))
	for i, p := range band2D {
		band[i] = plane.ToModel(p)
	}
	return band, nil
}

// SheetMetalLoftedFlangeFeatures adds lofted-flange features into the engine.
type SheetMetalLoftedFlangeFeatures struct{ engine *PartFeatures }

// NewSheetMetalLoftedFlangeFeatures binds the collection to a feature engine.
func NewSheetMetalLoftedFlangeFeatures(engine *PartFeatures) *SheetMetalLoftedFlangeFeatures {
	return &SheetMetalLoftedFlangeFeatures{engine}
}

// Add appends a lofted-flange feature, naming it LoftedFlange1, … .
func (c *SheetMetalLoftedFlangeFeatures) Add(def *SheetMetalLoftedFlangeDefinition) *PartFeature {
	f := &SheetMetalLoftedFlangeFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("LoftedFlange"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalLoftedFlangeFeature)(nil)
