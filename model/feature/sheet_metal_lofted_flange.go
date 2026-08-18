// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// codeLoftedFlangeUnmodeled marks a lofted flange whose converge / end-bend-radius solid is not
// modelled yet (#2086).
const codeLoftedFlangeUnmodeled diag.Code = "sheet-metal.lofted-flange-unmodeled"

// Sheet-metal Lofted Flange feature (M13-F02). A lofted flange is a constant-thickness wall
// that transitions between TWO open profiles on two sketch planes — the sheet-metal way to
// build a transition piece (e.g. a square-to-round duct). Each profile is thickened in its
// plane into a band; the two bands loft into one solid. Both profiles must have the same
// vertex count so their bands correspond point-for-point. Thickness is read live from the
// rule. The result is a new body (or joined to the running part).
//
// The transition is calculated two ways (Inventor's LoftedFlangeOutputTypeEnum, #1966). Die-formed
// is the SMOOTH wall: each pair of corresponding band points is joined by a cubic Hermite that
// leaves both profiles perpendicular to their planes (the natural forming direction), sampled
// finely. A press-brake output facets that smooth wall into flat plates joined by bends so it can
// be folded on a brake, the facet count set by the tolerance — chord deviation, facet angle, or
// facet width. See sheet_metal_lofted_flange_facet.go for the curve and faceting math.

// LoftedFlangeOutputType aliases the API enum (ADR-0018); the die-formed and press-brake values
// keep the existing call sites terse.
type LoftedFlangeOutputType = types.LoftedFlangeOutputType

const (
	DieFormedLoftedFlange                = types.DieFormedLoftedFlange
	PressBrakeChordToleranceLoftedFlange = types.PressBrakeChordToleranceLoftedFlange
	PressBrakeFacetAngleLoftedFlange     = types.PressBrakeFacetAngleLoftedFlange
	PressBrakeFacetDistanceLoftedFlange  = types.PressBrakeFacetDistanceLoftedFlange
)

// SheetMetalLoftedFlangeDefinition is the lofted-flange recipe: the two open profile sketches, the
// boolean operation, and — Inventor's output settings (#1966) — how the transition is calculated
// (Output), the facet tolerance for the press-brake modes (FacetTolerance), whether the corners
// converge to a point, and an optional end-bend radius.
type SheetMetalLoftedFlangeDefinition struct {
	ProfileA  *sketch.Sketch
	ProfileB  *sketch.Sketch
	Operation ops.PartFeatureOperation
	Output    LoftedFlangeOutputType
	// FacetTolerance bounds a press-brake output's facets; its meaning depends on Output (a length
	// for chord/distance, an angle in radians for facet-angle). Zero ⇒ a sensible default.
	FacetTolerance float64
	// Converge merges the transition's corners to a point instead of carrying them through.
	Converge bool
	// Radius rounds the end bends; 0 ⇒ the rule's bend radius. Recorded for parity (#2086).
	Radius func() float64
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
	bandA, bandB, err := f.loftBands(in.Params)
	if err != nil {
		return Output{}, err
	}
	// The output type (die-formed / press-brake facets) IS modelled; converge and the end-bend
	// radius are recorded but their solid is a follow-up (#2086), so report rather than pass off a
	// sharp-cornered wall as converged.
	if f.def.Converge || evalFloat(f.def.Radius) > 0 {
		in.Diag.Recordf(codeLoftedFlangeUnmodeled, diag.Warning,
			"lofted flange: converge=%v and end-bend radius are recorded but not modelled yet (#2086)", f.def.Converge)
	}
	sections := loftedFlangeSections(bandA, bandB, f.def.ProfileA.Plane().Normal(),
		f.def.ProfileB.Plane().Normal(), f.def.Output, f.def.FacetTolerance)
	wall, err := sweptSolid(sections, false, featOr(f.featName, "loftedFlange"))
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal lofted flange: %w", err)
	}
	bodies, err := combine(in, wall, f.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// loftBands thickens both profiles into their 3D bands, erroring when a profile cannot be read or
// the two bands do not correspond point-for-point.
func (f *SheetMetalLoftedFlangeFeature) loftBands(ps *param.Parameters) (bandA, bandB []math.Point3, err error) {
	t, err := sheetThickness(ps)
	if err != nil {
		return nil, nil, err
	}
	if bandA, err = loftBand(f.def.ProfileA, t); err != nil {
		return nil, nil, err
	}
	if bandB, err = loftBand(f.def.ProfileB, t); err != nil {
		return nil, nil, err
	}
	if len(bandA) != len(bandB) {
		return nil, nil, fmt.Errorf("sheet-metal lofted flange: profiles have %d and %d vertices; both must match to loft", len(bandA), len(bandB))
	}
	return bandA, bandB, nil
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
