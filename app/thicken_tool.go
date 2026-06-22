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
	dialogTool
	thickness float64
	approxIdx int // index into ApproximationOptions (#331; 0 = exact)
	added     *feature.PartFeature
}

// NewThickenTool returns a thicken tool with a default 1-unit thickness.
func NewThickenTool() *ThickenTool { return &ThickenTool{thickness: 1} }

// Name implements [Tool].
func (t *ThickenTool) Name() string { return "Thicken" }

// Start is a no-op; thicken needs no picks.
func (t *ThickenTool) Start(*Session) {}

// AcceptedKinds declares no restriction (thicken acts on the whole body and gathers no picks).
func (t *ThickenTool) AcceptedKinds() []SelectionKind { return nil }

// Pick is a no-op — thicken acts on the running surface body, not a selection.

// SetThickness/Thickness set the wall thickness (database units).
func (t *ThickenTool) SetThickness(d float64) { t.thickness = d }
func (t *ThickenTool) Thickness() float64     { return t.thickness }

// ApproximationIndex / SetApproximationIndex select the #331 approximation request
// (index into ApproximationOptions).
func (t *ThickenTool) ApproximationIndex() int { return t.approxIdx }
func (t *ThickenTool) SetApproximationIndex(i int) {
	t.approxIdx = clampRange(i, len(featureApproximations))
}

// CanCommit reports whether the thickness is positive.
func (t *ThickenTool) CanCommit() bool { return t.thickness > 0 }

// Commit thickens the active part's running surface body into a solid and recomputes; a sick
// feature (no surface body, or not thickenable) keeps the tool open by returning an error.
func (t *ThickenTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addThicken(part.Features())
	part.Recompute()
	s.recordEdit(part, "Thicken")
	if !t.added.Health().OK() {
		return errors.New("thicken: " + t.added.Health().Reason)
	}
	return nil
}

// addThicken builds the thicken feature into engine fs — shared by Commit and the preview.
func (t *ThickenTool) addThicken(fs *feature.PartFeatures) *feature.PartFeature {
	pf := feature.NewModifyFeatures(fs).AddThicken(t.thickness)
	pf.Definition().(*feature.ThickenFeature).SetApproximation(approximationAt(t.approxIdx))
	return pf
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ThickenTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached thicken feature the viewport previews before commit.
func (t *ThickenTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addThicken(fs), nil
	})
}

// Prompt guides the user.
func (t *ThickenTool) Prompt(*Session) string { return "Set the thickness, then click OK" }

// Cancel is a no-op; the engine restores the ambient filter.
func (t *ThickenTool) Cancel(*Session) {}
