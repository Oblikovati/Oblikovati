// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Coil feature (M48 #2239 split of sketched_features.go). Sweeps a sketch profile along a helix: the
// definition, the feature wrapper and Recompute, the coil tool builder and the CoilFeatures adder
// collection. The shared sketch-profile binding helpers live in sketched_features.go.

// CoilDefinition is the recipe for a coil (helical sweep).
type CoilDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Pitch        func() float64
	Revolutions  func() float64
	// Height is the total axial rise — any TWO of pitch/revolutions/height
	// specify the coil (the reference's pitch+height and revolution+height
	// modes, M08 PBI-096 #316); all three is overdetermined.
	Height    func() float64
	Taper     float64
	Operation ops.PartFeatureOperation
	// Variable-pitch rail + end conditions (M06-F09, #624; coil_variable.go).
	PitchRows []CoilPitchRow
	StartEnd  CoilEndCondition
	EndEnd    CoilEndCondition
	// Handedness is the winding sense; the zero value is right-handed (#1883).
	Handedness CoilHandedness
	// Spiral sweeps a FLAT spiral instead of a helix: Pitch becomes the radial step per turn and
	// there is no axial rise (Inventor's kSpiralCoilExtent, #1883). See coil_placement.go.
	Spiral bool
}

// CoilFeature sweeps a profile along a helix.
type CoilFeature struct {
	def      *CoilDefinition
	featName string
	tool     *topo.Body // last helical solid, exposed so a pattern can replicate it
}

func (c *CoilFeature) Definition() *CoilDefinition { return c.def }
func (c *CoilFeature) Kind() string                { return "coil" }

// Operation and ToolBody let a pattern/mirror replicate this feature (see [ToolFeature]).
func (c *CoilFeature) Operation() ops.PartFeatureOperation { return c.def.Operation }
func (c *CoilFeature) ToolBody() *topo.Body                { return c.tool }

// Recompute resolves the profile and sweeps it along a helix about the axis (pitch per
// revolution × revolutions) into a faceted solid, then applies the operation.
func (c *CoilFeature) Recompute(in Input) (Output, error) {
	prof, err := resolveSingleProfile(c.def.Sketch, c.def.ProfileIndex, "coil")
	if err != nil {
		return Output{}, err
	}
	if c.def.Axis == nil {
		return Output{}, errors.New("coil: no axis")
	}
	advance, totalTurns, err := coilRail(c.def)
	if err != nil {
		return Output{}, err
	}
	place, err := coilPlacerFor(c.def, advance)
	if err != nil {
		return Output{}, err
	}
	sections := coilSections(prof, c.def.Sketch.Plane(), c.def.Axis, place, totalTurns, c.def.Handedness)
	c.tool, err = c.coilTool(sections)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, c.tool, c.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// coilTool builds the coiled body from its helical cross-sections: for the Surface operation
// (kSurfaceOperation, #1858) an OPEN coiled sheet (the profile boundary swept along the helix, no
// end caps) via sweptShell; otherwise the coiled solid. combine() adds a surface tool as a surface
// body (no boolean).
func (c *CoilFeature) coilTool(sections [][]math.Point3) (*topo.Body, error) {
	feat := featOr(c.featName, "coil")
	build := sweptSolid
	if c.def.Operation == ops.Surface {
		build = sweptShell
	}
	body, err := build(sections, false, feat)
	if err != nil {
		return nil, err
	}
	return body, coilClearsItsOwnTurns(body, sections, c.def.Axis)
}

// CoilFeatures adds coils into the engine.
type CoilFeatures struct{ engine *PartFeatures }

// NewCoilFeatures binds the collection to an engine.
func NewCoilFeatures(engine *PartFeatures) *CoilFeatures { return &CoilFeatures{engine} }

// AddDefinition adds a coil from a fully-populated definition (height mode,
// taper, variable pitch — #316/#624).
func (c *CoilFeatures) AddDefinition(def *CoilDefinition) *PartFeature {
	cf := &CoilFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Coil"))
	cf.featName = pf.name
	return pf
}

// Add adds a coil of the profile about axis with the given pitch (per revolution),
// number of revolutions, taper, and boolean operation.
func (c *CoilFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, pitch, revolutions func() float64, taper float64, op ops.PartFeatureOperation) *PartFeature {
	def := &CoilDefinition{
		Sketch: skt, ProfileIndex: profileIndex, Axis: axis,
		Pitch: pitch, Revolutions: revolutions, Taper: taper, Operation: op,
	}
	cf := &CoilFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Coil"))
	cf.featName = pf.name
	return pf
}
