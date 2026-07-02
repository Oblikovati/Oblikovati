// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Rip feature (M13-F03). A rip cuts a narrow slit of a given gap width along a
// sketch line, through the full thickness — opening a seam so a closed or folded sheet can be
// developed flat (a rip across a corner or roll lets the part unfold). The geometry is a thin
// rectangular cutter (the line offset ±gap/2) subtracted through-all, so a rip is the boolean
// complement of a sliver prism — the same machinery the Cut feature uses, narrowed to a line.

// ripOvershoot extends the cutter a hair past the line's endpoints so a rip that reaches a
// boundary cuts cleanly through it rather than leaving a coincident-face sliver.
const ripOvershoot = 1e-4

// SheetMetalRipDefinition is the rip recipe: the sketch line to rip along and the gap width the
// slit opens (parameter-backed).
type SheetMetalRipDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	Gap       func() float64
}

// SheetMetalRipFeature slits the running sheet along the rip line each recompute.
type SheetMetalRipFeature struct {
	def      *SheetMetalRipDefinition
	featName string
}

// Definition returns the rip recipe.
func (f *SheetMetalRipFeature) Definition() *SheetMetalRipDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalRipFeature) Kind() string { return "sheet-metal-rip" }

// Recompute resolves the rip line, builds the slit cutter, and subtracts it through the sheet.
func (f *SheetMetalRipFeature) Recompute(in Input) (Output, error) {
	a, b, err := sketchLineEnds(f.def.Sketch, f.def.LineIndex, "sheet-metal rip")
	if err != nil {
		return Output{}, err
	}
	gap := evalFloat(f.def.Gap)
	if gap <= 0 {
		return Output{}, fmt.Errorf("sheet-metal rip: gap must be positive, got %g", gap)
	}
	plane := f.def.Sketch.Plane()
	sp, err := throughAllSpan(Extent{Type: ThroughAllExtent}, in.Bodies, plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildPrism(ripPolygon(a, b, gap), plane, sp, 0, f.featName)
	bodies, err := combine(in, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// ripPolygon is the slit cutter cross-section: the rectangle bounded by the line offset ±gap/2
// perpendicular, extended by ripOvershoot past each end so a boundary-reaching rip cuts through.
func ripPolygon(a, b math.Point2, gap float64) []math.Point2 {
	dir := a.VectorTo(b)
	if l := dir.Length(); l > 0 {
		dir = dir.Scale(1 / l)
	}
	perp := math.V2(-dir.Y, dir.X).Scale(gap / 2)
	a = a.TranslateBy(dir.Scale(-ripOvershoot))
	b = b.TranslateBy(dir.Scale(ripOvershoot))
	return []math.Point2{
		a.TranslateBy(perp.Negate()), b.TranslateBy(perp.Negate()),
		b.TranslateBy(perp), a.TranslateBy(perp),
	}
}

// sketchLineEnds returns the 2D endpoints of a sketch line, erroring (named by what) when the
// line index is out of range.
func sketchLineEnds(sk *sketch.Sketch, lineIndex int, what string) (a, b math.Point2, err error) {
	lines := sk.Lines()
	if lineIndex < 0 || lineIndex >= lines.Count() {
		return a, b, fmt.Errorf("%s: line index %d out of range (%d lines)", what, lineIndex, lines.Count())
	}
	l := lines.Item(lineIndex)
	return l.StartPoint().Position(), l.EndPoint().Position(), nil
}

// SheetMetalRipFeatures adds rip features into the engine.
type SheetMetalRipFeatures struct{ engine *PartFeatures }

// NewSheetMetalRipFeatures binds the collection to a feature engine.
func NewSheetMetalRipFeatures(engine *PartFeatures) *SheetMetalRipFeatures {
	return &SheetMetalRipFeatures{engine}
}

// Add appends a rip feature, naming it Rip1, Rip2, … .
func (c *SheetMetalRipFeatures) Add(def *SheetMetalRipDefinition) *PartFeature {
	f := &SheetMetalRipFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Rip"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalRipFeature)(nil)
