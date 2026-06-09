// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// ThickenTool is the interactive Thicken command: with a surface (sheet) body active, set
// the wall thickness in the property window and OK to turn it into a solid slab. It takes no
// face picks — it thickens the running surface body.
type ThickenTool struct {
	thickness float64
	added     *feature.PartFeature
}

// NewThickenTool returns a thicken tool with a default 1-unit thickness.
func NewThickenTool() *ThickenTool { return &ThickenTool{thickness: 1} }

// Name implements [Tool].
func (t *ThickenTool) Name() string { return "Thicken" }

// Start clears the selection filter (thicken needs no picks).
func (t *ThickenTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// Pick is a no-op — thicken acts on the running surface body, not a selection.
func (t *ThickenTool) Pick(*Session, Selectable) {}

// SetThickness/Thickness set the wall thickness (database units).
func (t *ThickenTool) SetThickness(d float64) { t.thickness = d }
func (t *ThickenTool) Thickness() float64     { return t.thickness }

// CanCommit reports whether the thickness is positive.
func (t *ThickenTool) CanCommit() bool { return t.thickness > 0 }

// Commit thickens the active part's running surface body into a solid and recomputes; a sick
// feature (no surface body, or not thickenable) keeps the tool open by returning an error.
func (t *ThickenTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddThicken(t.thickness)
	part.Recompute()
	s.recordEdit(part, "Thicken")
	if !t.added.Health().OK() {
		return errors.New("thicken: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ThickenTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user.
func (t *ThickenTool) Prompt(*Session) string { return "Set the thickness, then click OK" }

// Cancel restores the default selection filter.
func (t *ThickenTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
