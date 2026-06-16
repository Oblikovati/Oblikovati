// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Contour Roll feature (M13-F02). A contour roll revolves an open profile (the
// wall cross-section) around an axis line into a rolled shell — a cylinder, cone, or partial
// roll — the sheet-metal way to make tubes and rolled sections. The profile is thickened in
// its plane into a band; the band revolves about the axis through the sweep angle (default a
// full 360°). Thickness is read live from the rule. The result is a new body or joined to the
// running part.

// rollSegments caps the facet count of a full revolution (matching the part revolve's).
const rollSegments = 48

// SheetMetalContourRollDefinition is the contour-roll recipe: the open profile sketch, the
// index of the axis line in that sketch (a centerline), the sweep angle (parameter-backed;
// nil ⇒ 360°), and the boolean operation.
type SheetMetalContourRollDefinition struct {
	Profile   *sketch.Sketch
	AxisLine  int
	Angle     func() float64
	Operation ops.PartFeatureOperation
}

// SheetMetalContourRollFeature revolves the profile band each recompute.
type SheetMetalContourRollFeature struct {
	def      *SheetMetalContourRollDefinition
	featName string
}

// Definition returns the contour-roll recipe.
func (f *SheetMetalContourRollFeature) Definition() *SheetMetalContourRollDefinition {
	return f.def
}

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalContourRollFeature) Kind() string { return "sheet-metal-contour-roll" }

// Recompute thickens the profile into a band and revolves it about the axis.
func (f *SheetMetalContourRollFeature) Recompute(in Input) (Output, error) {
	t, err := sheetThickness(in.Params)
	if err != nil {
		return Output{}, err
	}
	axis, err := f.resolveAxis()
	if err != nil {
		return Output{}, err
	}
	band, err := loftBand(f.def.Profile, t) // the thickened profile, on its plane, in 3D
	if err != nil {
		return Output{}, err
	}
	angle := f.resolveAngle()
	if angle <= 0 {
		return Output{}, fmt.Errorf("sheet-metal contour roll: angle must be positive, got %g", angle)
	}
	sections, full := rollSections(band, axis, angle)
	rolled, err := sweptSolid(sections, full, featOr(f.featName, "contourRoll"))
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal contour roll: %w", err)
	}
	bodies, err := combine(in.Bodies, rolled, f.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// resolveAxis resolves the roll axis from the profile sketch's axis line.
func (f *SheetMetalContourRollFeature) resolveAxis() (*WorkAxis, error) {
	sk := f.def.Profile
	if sk == nil {
		return nil, fmt.Errorf("sheet-metal contour roll: no profile sketch")
	}
	lines := sk.Lines()
	if f.def.AxisLine < 0 || f.def.AxisLine >= lines.Count() {
		return nil, fmt.Errorf("sheet-metal contour roll: axis line index %d out of range (%d lines)", f.def.AxisLine, lines.Count())
	}
	return centerlineAxis(lines.Item(f.def.AxisLine), sk)
}

// resolveAngle returns the sweep angle, defaulting to a full revolution.
func (f *SheetMetalContourRollFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return 2 * stdmath.Pi
	}
	return f.def.Angle()
}

// rollSections revolves the band base about the axis through angle, returning the rotated
// cross-sections and whether the revolution is full (a closed loft).
func rollSections(base []math.Point3, axis *WorkAxis, angle float64) ([][]math.Point3, bool) {
	full := angle >= 2*stdmath.Pi-1e-9
	k, step := rollSegments, 2*stdmath.Pi/float64(rollSegments)
	if !full {
		segs := stdmath.Max(3, stdmath.Round(rollSegments*angle/(2*stdmath.Pi)))
		k, step = int(segs)+1, angle/segs
	}
	sections := make([][]math.Point3, k)
	for s := 0; s < k; s++ {
		m := math.Rotation4(step*float64(s), axis.Direction(), axis.Origin())
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			sec[i] = m.TransformPoint(p)
		}
		sections[s] = sec
	}
	return sections, full
}

// SheetMetalContourRollFeatures adds contour-roll features into the engine.
type SheetMetalContourRollFeatures struct{ engine *PartFeatures }

// NewSheetMetalContourRollFeatures binds the collection to a feature engine.
func NewSheetMetalContourRollFeatures(engine *PartFeatures) *SheetMetalContourRollFeatures {
	return &SheetMetalContourRollFeatures{engine}
}

// Add appends a contour-roll feature, naming it ContourRoll1, … .
func (c *SheetMetalContourRollFeatures) Add(def *SheetMetalContourRollDefinition) *PartFeature {
	f := &SheetMetalContourRollFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("ContourRoll"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalContourRollFeature)(nil)
