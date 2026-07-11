// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// GridSettings is the sketch-grid preference: the spacing between grid lines (a length
// [param.Quantity], stored in database units so it is unit-independent), whether the
// grid is shown, and how often a heavier major line is drawn. It is presented and
// edited in the document's units (see the preferences window) but the geometry uses
// SpacingModel (model/database units).
type GridSettings struct {
	spacing      param.Quantity // Length category, database units (cm)
	Visible      bool
	MajorEvery   int  // every Nth grid line is a major (heavier) line
	SnapToPoints bool // snap a sketch click to a nearby existing point
	SnapToGrid   bool // snap a sketch click to the nearest grid intersection
}

// defaultGridSpacingCm is the out-of-the-box grid spacing in database units (1 cm =
// 10 mm), a sensible default for the metric default units.
const defaultGridSpacingCm = 1.0

// NewGridSettings returns the default grid: a 10 mm spacing, visible, major every 5,
// with point and grid snapping on.
func NewGridSettings() *GridSettings {
	return &GridSettings{
		spacing:      param.Quantity{Value: defaultGridSpacingCm, Unit: param.Length},
		Visible:      true,
		MajorEvery:   5,
		SnapToPoints: true,
		SnapToGrid:   true,
	}
}

// SpacingModel returns the grid spacing in model/database units (cm) — what the grid
// geometry is laid out with.
func (g *GridSettings) SpacingModel() float64 { return g.spacing.Value }

// SetSpacingModel sets the spacing directly in model/database units (cm) — the
// unit-independent value the persisted sketch options carry (M05-F11).
func (g *GridSettings) SetSpacingModel(cm float64) error {
	if cm <= 0 {
		return fmt.Errorf("app: grid spacing %v cm is not positive", cm)
	}
	g.spacing = param.Quantity{Value: cm, Unit: param.Length}
	return nil
}

// Spacing returns the grid spacing as a length quantity.
func (g *GridSettings) Spacing() param.Quantity { return g.spacing }

// SpacingIn returns the spacing value expressed in the given units' preferred length
// unit (e.g. 10 when the spacing is 1 cm and units are mm) — the number the
// preferences window shows and edits.
func (g *GridSettings) SpacingIn(u param.UnitsOfMeasure) float64 { return u.ToPreferred(g.spacing) }

// SetSpacingIn sets the spacing from a positive value given in u's preferred length
// unit (so editing "5" in a mm document means a 5 mm grid).
func (g *GridSettings) SetSpacingIn(value float64, u param.UnitsOfMeasure) error {
	if value <= 0 {
		return errors.New("grid spacing must be positive")
	}
	g.spacing = u.FromPreferred(value, param.Length)
	return nil
}

// ChamferFlatCorners reports the default corner treatment new Chamfer tools start with:
// true blends a vertex where three chamfered edges meet into a flat triangular face (the
// Inventor default), false leaves the three chamfer planes meeting at a point.
func (s *Session) ChamferFlatCorners() bool { return s.chamferFlatCorners }

// SetChamferFlatCorners sets the default corner treatment for new chamfers.
func (s *Session) SetChamferFlatCorners(flat bool) { s.chamferFlatCorners = flat }

// TangentChainSelect reports whether new Fillet/Chamfer tools select the whole tangent chain
// through a clicked edge (Inventor's tangent propagation) rather than the single edge. Defaults
// true; Shift+click always expands regardless. See #1947.
func (s *Session) TangentChainSelect() bool { return s.tangentChainSelect }

// SetTangentChainSelect sets the default tangent-chain selection mode for new fillets/chamfers.
func (s *Session) SetTangentChainSelect(on bool) { s.tangentChainSelect = on }

// ChamferConcaveStrategy reports the default concave-edge strategy new Chamfer tools start with:
// outward fills the inside corner with material (the default), inward cuts a recessed relief.
func (s *Session) ChamferConcaveStrategy() types.ChamferConcaveStrategy {
	if !s.chamferConcaveOut {
		return types.ChamferConcaveInward
	}
	return types.ChamferConcaveOutward
}

// SetChamferConcaveStrategy sets the default concave-edge strategy for new chamfers.
func (s *Session) SetChamferConcaveStrategy(strategy types.ChamferConcaveStrategy) {
	s.chamferConcaveOut = strategy != types.ChamferConcaveInward
}

// Grid returns the session's sketch-grid settings (created on first use).
func (s *Session) Grid() *GridSettings {
	if s.grid == nil {
		s.grid = NewGridSettings()
	}
	return s.grid
}

// DocumentUnits returns the active part's display units, or metric defaults when there
// is no active part — the units the grid and dimensions are presented in.
func (s *Session) DocumentUnits() param.UnitsOfMeasure {
	if part, err := activePart(s); err == nil {
		return part.Units()
	}
	return param.DefaultUnitsOfMeasure()
}

// GridSpacingDisplay returns the grid spacing in the document's length unit and that
// unit's name (e.g. 10, "mm") — what the preferences window shows. It keeps the units
// dependency inside the app layer so the head needs only floats and strings.
func (s *Session) GridSpacingDisplay() (value float64, unit string) {
	u := s.DocumentUnits()
	return s.Grid().SpacingIn(u), u.PreferredName(param.Length)
}

// SetGridSpacingDisplay sets the grid spacing from a value given in the document's
// length unit.
func (s *Session) SetGridSpacingDisplay(value float64) error {
	return s.Grid().SetSpacingIn(value, s.DocumentUnits())
}
