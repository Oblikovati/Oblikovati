// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// ThickenTool is the interactive Thicken command: with a surface (sheet) body active, set
// the wall thickness in the property window and OK to turn it into a solid slab. It takes no
// face picks — it thickens the running surface body.
// The #1876 options — which side to offset toward, whether to join/cut/intersect the result
// into the running solid, and whether to leave it as a surface — were carried by the definition
// and set by the wire handler, but the tool set none of them, so the ribbon could only ever
// build the symmetric, joined, solid thicken (#2050).
type ThickenTool struct {
	dialogTool
	thickness float64
	approxIdx int // index into ApproximationOptions (#331; 0 = exact)
	dirIdx    int // index into ThickenDirectionOptions (0 = symmetric)
	opIdx     int // index into ThickenOperationOptions (0 = join)
	asSurface bool
	added     *feature.PartFeature
}

// NewThickenTool returns a thicken tool with a default 1-unit thickness, offsetting
// symmetrically and joining the result — the pre-#1876 behaviour, kept as the default.
func NewThickenTool() *ThickenTool { return &ThickenTool{thickness: 1} }

// thickenDirections maps the direction combo index to the kernel enum.
var thickenDirections = []ops.ThickenDirection{ops.ThickenSymmetric, ops.ThickenPositive, ops.ThickenNegative}

// ThickenDirectionOptions labels the direction combo, in index order.
func ThickenDirectionOptions() []string {
	return []string{"Symmetric", "Positive (+normal)", "Negative (−normal)"}
}

// thickenOperations maps the operation combo index to the boolean the result takes with the
// running solid.
var thickenOperations = []ops.PartFeatureOperation{ops.Join, ops.Cut, ops.Intersect, ops.NewBody}

// ThickenOperationOptions labels the operation combo, in index order.
func ThickenOperationOptions() []string { return []string{"Join", "Cut", "Intersect", "New body"} }

// DirectionIndex / SetDirectionIndex select which side the thicken offsets toward.
func (t *ThickenTool) DirectionIndex() int { return t.dirIdx }
func (t *ThickenTool) SetDirectionIndex(i int) {
	t.dirIdx = clampRange(i, len(thickenDirections))
}

// OperationIndex / SetOperationIndex select the boolean the result takes with the running solid.
func (t *ThickenTool) OperationIndex() int { return t.opIdx }
func (t *ThickenTool) SetOperationIndex(i int) {
	t.opIdx = clampRange(i, len(thickenOperations))
}

// AsSurface / SetAsSurface choose whether the result stays an offset SURFACE rather than
// becoming a solid slab.
func (t *ThickenTool) AsSurface() bool      { return t.asSurface }
func (t *ThickenTool) SetAsSurface(on bool) { t.asSurface = on }

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
	def := pf.Definition().(*feature.ThickenFeature)
	def.SetApproximation(approximationAt(t.approxIdx))
	// nil faceKeys thickens the whole body; walls/chain/blend keep their documented defaults,
	// which the recipe restore also uses.
	def.SetThickenOptions(thickenDirections[t.dirIdx], thickenOperations[t.opIdx], t.asSurface, nil, true, false, false)
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
