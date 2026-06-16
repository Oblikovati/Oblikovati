// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Bend feature (M13-F02). A bend folds a flat sheet along a sketch line through a
// given angle over the rule's bend radius — the in-environment counterpart of the part bend
// (#651), differing in that the radius defaults to the active sheet-metal rule so a radius
// edit repropagates, and the bend records its place in the sheet-metal feature history for
// the flat pattern (F04). The geometry is the shared straight→arc→straight fold (bendSolid).

// SheetMetalBendDefinition is the bend recipe: the sketch bend line (sketch + line index),
// the bend angle (parameter-backed; nil ⇒ 90°), an optional radius override (nil ⇒ the rule's
// bend radius), and a flip that folds to the opposite side of the sketch plane.
type SheetMetalBendDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	Angle     func() float64
	Radius    func() float64
	Flip      bool
}

// SheetMetalBendFeature folds the running sheet along the bend line each recompute.
type SheetMetalBendFeature struct {
	def      *SheetMetalBendDefinition
	featName string
}

// Definition returns the bend recipe.
func (f *SheetMetalBendFeature) Definition() *SheetMetalBendDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalBendFeature) Kind() string { return "sheet-metal-bend" }

// Recompute resolves the bend line and folds the sheet over the rule radius through the angle.
func (f *SheetMetalBendFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal bend")
	if err != nil {
		return Output{}, err
	}
	point, dir, up, err := sketchBendLine(f.def.Sketch, f.def.LineIndex, f.def.Flip, "sheet-metal bend")
	if err != nil {
		return Output{}, err
	}
	radius := f.resolveRadius(in.Params)
	angle := f.resolveAngle()
	if radius <= 0 || angle <= 0 {
		return Output{}, fmt.Errorf("sheet-metal bend: radius/angle must be positive (r=%g a=%g)", radius, angle)
	}
	bent, err := bendSolid(body, point, dir, up, radius, angle, featOr(f.featName, "bend"))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, bent)}, nil
}

// resolveRadius returns the bend radius: the override closure, else the rule's BendRadius
// parameter, else 0 (which Recompute rejects).
func (f *SheetMetalBendFeature) resolveRadius(ps *param.Parameters) float64 {
	if f.def.Radius != nil {
		return f.def.Radius()
	}
	if ps != nil {
		if p, ok := ps.ByName(flangeBendParamName); ok {
			return p.ModelValue()
		}
	}
	return 0
}

// resolveAngle returns the bend angle, defaulting to a 90° fold.
func (f *SheetMetalBendFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return stdmath.Pi / 2
	}
	return f.def.Angle()
}

// BendSpecs reports the single bend this fold introduces, for the flat pattern. A nil
// radius override defers to the rule's default (signalled by a non-positive radius).
func (f *SheetMetalBendFeature) BendSpecs(_ float64) []BendSpec {
	radius := 0.0
	if f.def.Radius != nil {
		radius = f.def.Radius()
	}
	return []BendSpec{{Angle: f.resolveAngle(), Radius: radius}}
}

// SheetMetalBendFeatures adds bend features into the engine.
type SheetMetalBendFeatures struct{ engine *PartFeatures }

// NewSheetMetalBendFeatures binds the collection to a feature engine.
func NewSheetMetalBendFeatures(engine *PartFeatures) *SheetMetalBendFeatures {
	return &SheetMetalBendFeatures{engine}
}

// Add appends a bend feature, naming it Bend1, Bend2, … .
func (c *SheetMetalBendFeatures) Add(def *SheetMetalBendDefinition) *PartFeature {
	f := &SheetMetalBendFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Bend"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalBendFeature)(nil)
