// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
)

// The drawing style system (M14-F01 PBI-138, #385): a drawing follows a drafting standard
// (ISO or ANSI) whose preset fixes the dimension/text/line appearance. Switching the
// standard re-points the active preset, so every annotation re-renders to it. V1 ships the
// two built-in presets; user-authored styles are a follow-up.

// compile-time: the style types satisfy the api/contract surface (ADR-0018).
var (
	_ contract.DrawingStylesManager  = (*StylesManager)(nil)
	_ contract.DrawingStandardStyle  = (*StandardStyle)(nil)
	_ contract.DrawingDimensionStyle = (*DimensionStyle)(nil)
	_ contract.DrawingTextStyle      = (*TextStyle)(nil)
	_ contract.DrawingLineStyle      = (*LineStyle)(nil)
)

// DimensionStyle is the appearance of a dimension under a drafting standard.
type DimensionStyle struct {
	name          string
	textHeightMM  float64
	arrowSizeMM   float64
	decimalPlaces int
	unit          types.DimensionUnit
	lineWeightMM  float64
}

func (d *DimensionStyle) Name() string              { return d.name }
func (d *DimensionStyle) TextHeightMM() float64     { return d.textHeightMM }
func (d *DimensionStyle) ArrowSizeMM() float64      { return d.arrowSizeMM }
func (d *DimensionStyle) DecimalPlaces() int        { return d.decimalPlaces }
func (d *DimensionStyle) Unit() types.DimensionUnit { return d.unit }
func (d *DimensionStyle) LineWeightMM() float64     { return d.lineWeightMM }

// TextStyle is the appearance of annotation text under a drafting standard.
type TextStyle struct {
	name     string
	fontName string
	heightMM float64
}

func (t *TextStyle) Name() string      { return t.name }
func (t *TextStyle) FontName() string  { return t.fontName }
func (t *TextStyle) HeightMM() float64 { return t.heightMM }

// LineStyle is the appearance of a drawing line under a drafting standard.
type LineStyle struct {
	name     string
	weightMM float64
}

func (l *LineStyle) Name() string      { return l.name }
func (l *LineStyle) WeightMM() float64 { return l.weightMM }

// StandardStyle is one drafting standard's complete style preset.
type StandardStyle struct {
	standard  types.DraftingStandard
	dimension *DimensionStyle
	text      *TextStyle
	line      *LineStyle
}

func (s *StandardStyle) Standard() types.DraftingStandard               { return s.standard }
func (s *StandardStyle) DimensionStyle() contract.DrawingDimensionStyle { return s.dimension }
func (s *StandardStyle) TextStyle() contract.DrawingTextStyle           { return s.text }
func (s *StandardStyle) LineStyle() contract.DrawingLineStyle           { return s.line }

// StylesManager is a drawing's style system: the active drafting standard and the built-in
// preset for each standard.
type StylesManager struct {
	active  types.DraftingStandard
	presets map[types.DraftingStandard]*StandardStyle
}

// newStylesManager builds the manager with the ISO and ANSI presets, active on ISO (the
// metric default a new drawing opens in).
func newStylesManager() *StylesManager {
	return &StylesManager{
		active: types.DraftingISO,
		presets: map[types.DraftingStandard]*StandardStyle{
			types.DraftingISO:  isoStyle(),
			types.DraftingANSI: ansiStyle(),
		},
	}
}

// ActiveStandard returns the drawing's current drafting standard.
func (m *StylesManager) ActiveStandard() types.DraftingStandard { return m.active }

// ActiveStyle returns the active standard's style preset.
func (m *StylesManager) ActiveStyle() contract.DrawingStandardStyle { return m.presets[m.active] }

// SetActiveStandard switches the active standard (ignoring an unknown one), re-pointing the
// active preset so every annotation re-renders to it.
func (m *StylesManager) SetActiveStandard(std types.DraftingStandard) {
	if _, ok := m.presets[std]; ok {
		m.active = std
	}
}

// Standards returns the available drafting standards in display order.
func (m *StylesManager) Standards() []types.DraftingStandard {
	return []types.DraftingStandard{types.DraftingISO, types.DraftingANSI}
}

// isoStyle is the ISO (metric) preset: 3.5 mm text/arrows, 2 decimals, millimetres.
func isoStyle() *StandardStyle {
	return &StandardStyle{
		standard:  types.DraftingISO,
		dimension: &DimensionStyle{name: "Default (ISO)", textHeightMM: 3.5, arrowSizeMM: 3.5, decimalPlaces: 2, unit: types.DimensionMillimeter, lineWeightMM: 0.25},
		text:      &TextStyle{name: "Note Text (ISO)", fontName: "Arial", heightMM: 3.5},
		line:      &LineStyle{name: "Visible (ISO)", weightMM: 0.5},
	}
}

// ansiStyle is the ANSI (imperial) preset: 0.12 in (≈3 mm) text/arrows, 3 decimals, inches.
func ansiStyle() *StandardStyle {
	return &StandardStyle{
		standard:  types.DraftingANSI,
		dimension: &DimensionStyle{name: "Default (ANSI)", textHeightMM: 3.0, arrowSizeMM: 3.0, decimalPlaces: 3, unit: types.DimensionInch, lineWeightMM: 0.3},
		text:      &TextStyle{name: "Note Text (ANSI)", fontName: "Arial", heightMM: 3.0},
		line:      &LineStyle{name: "Visible (ANSI)", weightMM: 0.6},
	}
}
