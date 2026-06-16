// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// SheetMetalFlangeTool is the interactive Flange command (M13-F02): activate it, click a
// straight edge of the sheet, set the height (and optionally the bend angle), and OK to fold a
// wall up over a bend at the rule's gauge. The thickness and default bend radius come from the
// active rule, so the property panel only needs the height and angle.
type SheetMetalFlangeTool struct {
	edge   *EdgeHandle
	height float64 // flange height, model units
	angle  float64 // bend angle, radians (90° default)
	added  *feature.PartFeature
}

// NewSheetMetalFlangeTool returns a flange tool defaulting to a 10 mm, 90° flange.
func NewSheetMetalFlangeTool() *SheetMetalFlangeTool {
	return &SheetMetalFlangeTool{height: 1.0, angle: halfPiAngle}
}

// halfPiAngle is a 90° bend in radians — the default flange fold.
const halfPiAngle = 1.5707963267948966

// Name implements [Tool].
func (t *SheetMetalFlangeTool) Name() string { return "Sheet Metal Flange" }

// Start filters selection to edges so a click picks the edge to flange from.
func (t *SheetMetalFlangeTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
}

// Pick captures the clicked edge (a re-pick replaces it).
func (t *SheetMetalFlangeTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		t.edge = &e
	}
}

// SetHeight/Height and SetAngle/Angle set the flange dimensions (height in model units, angle
// in radians) the property panel edits.
func (t *SheetMetalFlangeTool) SetHeight(h float64) { t.height = h }
func (t *SheetMetalFlangeTool) Height() float64     { return t.height }
func (t *SheetMetalFlangeTool) SetAngle(a float64)  { t.angle = a }
func (t *SheetMetalFlangeTool) Angle() float64      { return t.angle }

// CanCommit reports whether an edge is picked and the height and angle are positive.
func (t *SheetMetalFlangeTool) CanCommit() bool {
	return t.edge != nil && t.height > 0 && t.angle > 0
}

// Commit folds the wall on the picked edge at the rule gauge and recomputes; a sick result
// (e.g. an edge that is not a straight sheet boundary) keeps the tool open via an error.
func (t *SheetMetalFlangeTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal flange: pick an edge and set a positive height/angle")
	}
	height, angle := t.height, t.angle
	t.added = feature.NewSheetMetalFlangeFeatures(part.Features()).Add(&feature.SheetMetalFlangeDefinition{
		EdgeKey: t.edge.Edge.ReferenceKey(),
		Height:  func() float64 { return height },
		Angle:   func() float64 { return angle },
	})
	part.Recompute()
	s.recordEdit(part, "Sheet Metal Flange")
	if !t.added.Health().OK() {
		return errors.New("sheet-metal flange: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel abandons the tool.
func (t *SheetMetalFlangeTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SheetMetalFlangeTool) AddedFeature() *feature.PartFeature { return t.added }
