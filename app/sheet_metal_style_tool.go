// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sheetmetal"
)

// SheetMetalStyleTool edits the active part's sheet-metal rule (M13-F01 UI): gauge thickness,
// default bend radius, K-factor, corner-relief shape and notch size, and the minimum remnant
// gap. It picks no geometry — it is a settings panel — so it seeds its buffers from the rule on
// Start and re-authors the rule on Commit, then recomputes so every dependent wall/bend rebuilds
// at the new gauge (the same full rebuild [router.sheetMetalSetStyle] performs).
//
// Lengths are held in the document's preferred unit (the value the dialog field shows); Commit
// converts them back to model units and re-authors the Thickness/BendRadius parameters so the
// rule stays parameter-backed.
type SheetMetalStyleTool struct {
	dialogTool
	thickness   float64 // preferred (display) units
	bendRadius  float64 // preferred (display) units
	reliefWidth float64 // preferred (display) units
	reliefDepth float64 // preferred (display) units
	gap         float64 // preferred (display) units
	kFactor     float64 // unitless (0..1)
	reliefShape types.ReliefShape
}

// NewSheetMetalStyleTool returns a style-editor tool.
func NewSheetMetalStyleTool() *SheetMetalStyleTool { return &SheetMetalStyleTool{} }

func (t *SheetMetalStyleTool) Name() string { return "Sheet Metal Style" }

// Start seeds the edit buffers from the active part's rule, converting its model-unit lengths
// to the document's preferred unit. A no-op when the active part is not sheet metal.
func (t *SheetMetalStyleTool) Start(s *Session) {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return
	}
	rule := part.SheetMetal()
	if rule == nil {
		return
	}
	u := s.DocumentUnits()
	toPref := func(v float64) float64 { return u.ToPreferred(param.Q(v, param.Length)) }
	t.thickness = toPref(rule.Thickness())
	t.bendRadius = toPref(rule.BendRadius())
	t.reliefWidth = toPref(rule.ReliefWidth())
	t.reliefDepth = toPref(rule.ReliefDepth())
	t.gap = toPref(rule.Gap())
	t.kFactor = rule.Unfold().KFactor
	t.reliefShape = rule.Relief().Shape
}

// CanCommit requires positive gauge/radius and a K-factor in the open interval (0,1) — the
// physically meaningful range (the neutral fibre lies inside the sheet).
func (t *SheetMetalStyleTool) CanCommit() bool {
	return t.thickness > 0 && t.bendRadius > 0 && t.kFactor > 0 && t.kFactor < 1
}

// Commit re-authors the rule from the edit buffers and recomputes the part.
func (t *SheetMetalStyleTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	rule := part.SheetMetal()
	if rule == nil {
		return errors.New("sheet-metal style: the active part has no rule")
	}
	if err := t.apply(part, rule); err != nil {
		return err
	}
	// A rule edit changes inputs every wall/bend reads live, but those reads are not tracked
	// feature dependencies — invalidate the whole program so the sheet rebuilds at the new gauge.
	part.Features().MarkAllDirty()
	part.Recompute()
	s.recordEdit(part, "Sheet Metal Style")
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// apply writes the buffers onto the rule: the two parameter-backed lengths are re-authored as
// unit expressions (keeping the rule parameter-backed); relief, gap, and K-factor land directly.
func (t *SheetMetalStyleTool) apply(part *compdef.PartComponentDefinition, rule *sheetmetal.Rule) error {
	u := part.Units()
	expr := func(displayValue float64) string {
		return u.Format(u.FromPreferred(displayValue, param.Length))
	}
	if err := part.SetSheetMetalLengthParam(compdef.ThicknessParamName(), expr(t.thickness)); err != nil {
		return err
	}
	if err := part.SetSheetMetalLengthParam(compdef.BendRadiusParamName(), expr(t.bendRadius)); err != nil {
		return err
	}
	toModel := func(v float64) float64 { return u.FromPreferred(v, param.Length).Value }
	relief := rule.Relief()
	relief.Shape = t.reliefShape
	relief.Width = sheetmetal.Constant(toModel(t.reliefWidth))
	relief.Depth = sheetmetal.Constant(toModel(t.reliefDepth))
	rule.SetRelief(relief)
	rule.SetGap(sheetmetal.Constant(toModel(t.gap)))
	rule.SetUnfold(sheetmetal.KFactorMethod(t.kFactor))
	return nil
}

// Buffer accessors for the property dialog. Lengths are in the document's preferred unit.

func (t *SheetMetalStyleTool) Thickness() float64     { return t.thickness }
func (t *SheetMetalStyleTool) SetThickness(v float64) { t.thickness = v }

func (t *SheetMetalStyleTool) BendRadius() float64     { return t.bendRadius }
func (t *SheetMetalStyleTool) SetBendRadius(v float64) { t.bendRadius = v }

func (t *SheetMetalStyleTool) KFactor() float64     { return t.kFactor }
func (t *SheetMetalStyleTool) SetKFactor(v float64) { t.kFactor = v }

func (t *SheetMetalStyleTool) ReliefWidth() float64     { return t.reliefWidth }
func (t *SheetMetalStyleTool) SetReliefWidth(v float64) { t.reliefWidth = v }

func (t *SheetMetalStyleTool) ReliefDepth() float64     { return t.reliefDepth }
func (t *SheetMetalStyleTool) SetReliefDepth(v float64) { t.reliefDepth = v }

func (t *SheetMetalStyleTool) Gap() float64     { return t.gap }
func (t *SheetMetalStyleTool) SetGap(v float64) { t.gap = v }

// ReliefShapeIndex / SetReliefShapeIndex expose the relief shape as a 0-based combo index
// (the [types.ReliefShape] enum order: round, square, tear).
func (t *SheetMetalStyleTool) ReliefShapeIndex() int     { return int(t.reliefShape) }
func (t *SheetMetalStyleTool) SetReliefShapeIndex(i int) { t.reliefShape = types.ReliefShape(i) }
