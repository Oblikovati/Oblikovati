// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/model/sketch"
)

// Sheet-metal Cosmetic Bend feature (M13-F03). A cosmetic bend marks a fold line on the sheet
// for manufacturing WITHOUT deforming the model — the part stays flat (or as built), but the
// bend joins the bend table and bend-allowance preview ([compdef.Bends]) so the developed length
// and the fold annotation are reported. It is the geometry-neutral counterpart of the Bend
// feature: same sketch-line + angle + radius inputs, but Recompute leaves the body unchanged.
// Use it to call out a bend a downstream process applies, or to annotate a part modelled flat.

// SheetMetalCosmeticBendDefinition is the cosmetic-bend recipe: the sketch bend line (sketch +
// line index), the bend angle (parameter-backed; nil ⇒ 90°), and an optional inside-radius
// override (nil ⇒ the rule's bend radius).
type SheetMetalCosmeticBendDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	Angle     func() float64
	Radius    func() float64
}

// SheetMetalCosmeticBendFeature records the cosmetic bend each recompute, passing the body
// through unchanged.
type SheetMetalCosmeticBendFeature struct {
	def      *SheetMetalCosmeticBendDefinition
	featName string
}

// Definition returns the cosmetic-bend recipe.
func (f *SheetMetalCosmeticBendFeature) Definition() *SheetMetalCosmeticBendDefinition {
	return f.def
}

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalCosmeticBendFeature) Kind() string { return "sheet-metal-cosmetic-bend" }

// Recompute validates the bend line resolves (so a dangling line reports the feature sick) and
// leaves the geometry untouched — a cosmetic bend annotates, it does not fold.
func (f *SheetMetalCosmeticBendFeature) Recompute(in Input) (Output, error) {
	if _, _, _, err := sketchBendLine(f.def.Sketch, f.def.LineIndex, false, "sheet-metal cosmetic bend"); err != nil {
		return Output{}, err
	}
	return Output{Bodies: in.Bodies}, nil
}

// BendSpecs reports the cosmetic bend for the bend table / allowance preview. A nil radius
// override defers to the rule's default (signalled by a non-positive radius).
func (f *SheetMetalCosmeticBendFeature) BendSpecs(_ float64) []BendSpec {
	radius := 0.0
	if f.def.Radius != nil {
		radius = f.def.Radius()
	}
	return []BendSpec{{Angle: f.resolveAngle(), Radius: radius}}
}

// resolveAngle returns the bend angle, defaulting to a 90° fold.
func (f *SheetMetalCosmeticBendFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return stdmath.Pi / 2
	}
	return f.def.Angle()
}

// SheetMetalCosmeticBendFeatures adds cosmetic-bend features into the engine.
type SheetMetalCosmeticBendFeatures struct{ engine *PartFeatures }

// NewSheetMetalCosmeticBendFeatures binds the collection to a feature engine.
func NewSheetMetalCosmeticBendFeatures(engine *PartFeatures) *SheetMetalCosmeticBendFeatures {
	return &SheetMetalCosmeticBendFeatures{engine}
}

// Add appends a cosmetic-bend feature, naming it CosmeticBend1, CosmeticBend2, … .
func (c *SheetMetalCosmeticBendFeatures) Add(def *SheetMetalCosmeticBendDefinition) *PartFeature {
	f := &SheetMetalCosmeticBendFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("CosmeticBend"))
	f.featName = pf.Name()
	return pf
}

var (
	_ Feature     = (*SheetMetalCosmeticBendFeature)(nil)
	_ BendLineage = (*SheetMetalCosmeticBendFeature)(nil)
)
