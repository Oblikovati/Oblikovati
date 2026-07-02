// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Sheet-metal modify tools (M13-F02/F03): the Modify-panel features that fold or cut the
// running sheet. Each is an interactive Tool that picks geometry and commits.

// SheetMetalBendTool folds the sheet along a picked sketch line.
type SheetMetalBendTool struct {
	line  *SketchEntityHandle
	added *feature.PartFeature
}

// NewSheetMetalBendTool returns a bend tool awaiting a sketch line.
func NewSheetMetalBendTool() *SheetMetalBendTool { return &SheetMetalBendTool{} }

func (t *SheetMetalBendTool) Name() string   { return "Sheet Metal Bend" }
func (t *SheetMetalBendTool) Start(*Session) {}

// AcceptedKinds declares the bend picks a sketch entity (the bend line).
func (t *SheetMetalBendTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectSketchEntity}
}
func (t *SheetMetalBendTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.line = &h
	}
}
func (t *SheetMetalBendTool) CanCommit() bool                    { return t.line != nil }
func (t *SheetMetalBendTool) Cancel(s *Session)                  {}
func (t *SheetMetalBendTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalBendTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if t.line == nil {
		return errors.New("sheet-metal bend: pick a sketch line crossing the sheet")
	}
	added, err := t.addBend(part, part.Features())
	if err != nil {
		return err
	}
	t.added = added
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Bend")
}

// addBend builds the sheet-metal bend feature into engine fs — shared by Commit and preview.
func (t *SheetMetalBendTool) addBend(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) (*feature.PartFeature, error) {
	sk, idx, ok := lineHandleInPart(part, t.line.Entity)
	if !ok {
		return nil, errors.New("sheet-metal bend: the pick is not a sketch line")
	}
	return feature.NewSheetMetalBendFeatures(fs).Add(&feature.SheetMetalBendDefinition{Sketch: sk, LineIndex: idx}), nil
}

// DraftFeature returns the unattached sheet-metal bend feature the viewport previews (#1626).
func (t *SheetMetalBendTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addBend(part, fs)
	})
}

// SheetMetalFoldTool folds the sheet along a picked sketch line, hinged at its centerline.
type SheetMetalFoldTool struct {
	line  *SketchEntityHandle
	added *feature.PartFeature
}

// NewSheetMetalFoldTool returns a fold tool awaiting a sketch line.
func NewSheetMetalFoldTool() *SheetMetalFoldTool { return &SheetMetalFoldTool{} }

func (t *SheetMetalFoldTool) Name() string   { return "Sheet Metal Fold" }
func (t *SheetMetalFoldTool) Start(*Session) {}

// AcceptedKinds declares the fold picks a sketch entity (the fold line).
func (t *SheetMetalFoldTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectSketchEntity}
}
func (t *SheetMetalFoldTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.line = &h
	}
}
func (t *SheetMetalFoldTool) CanCommit() bool                    { return t.line != nil }
func (t *SheetMetalFoldTool) Cancel(s *Session)                  {}
func (t *SheetMetalFoldTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalFoldTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if t.line == nil {
		return errors.New("sheet-metal fold: pick a sketch line crossing the sheet")
	}
	added, err := t.addFold(part, part.Features())
	if err != nil {
		return err
	}
	t.added = added
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Fold")
}

// addFold builds the sheet-metal fold feature into engine fs — shared by Commit and preview.
func (t *SheetMetalFoldTool) addFold(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) (*feature.PartFeature, error) {
	sk, idx, ok := lineHandleInPart(part, t.line.Entity)
	if !ok {
		return nil, errors.New("sheet-metal fold: the pick is not a sketch line")
	}
	return feature.NewSheetMetalFoldFeatures(fs).Add(&feature.SheetMetalFoldDefinition{
		Sketch: sk, LineIndex: idx, Location: feature.CenterlineOfBend,
	}), nil
}

// DraftFeature returns the unattached sheet-metal fold feature the viewport previews (#1626).
func (t *SheetMetalFoldTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFold(part, fs)
	})
}

// SheetMetalCornerTool applies a corner treatment (chamfer) at the picked corner edges.
type SheetMetalCornerTool struct {
	edges []EdgeHandle
	size  float64
	added *feature.PartFeature
}

// NewSheetMetalCornerTool returns a corner tool defaulting to a 3 mm chamfer.
func NewSheetMetalCornerTool() *SheetMetalCornerTool { return &SheetMetalCornerTool{size: 0.3} }

func (t *SheetMetalCornerTool) Name() string   { return "Sheet Metal Corner" }
func (t *SheetMetalCornerTool) Start(*Session) {}

// AcceptedKinds declares the corner picks edges (the corner edges to treat).
func (t *SheetMetalCornerTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edges for the unified highlight.
func (t *SheetMetalCornerTool) Picks() []Selectable { return edgeSelectables(t.edges) }
func (t *SheetMetalCornerTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		t.edges = append(t.edges, e)
	}
}
func (t *SheetMetalCornerTool) SetSize(v float64)                  { t.size = v }
func (t *SheetMetalCornerTool) Size() float64                      { return t.size }
func (t *SheetMetalCornerTool) CanCommit() bool                    { return len(t.edges) > 0 && t.size > 0 }
func (t *SheetMetalCornerTool) Cancel(s *Session)                  {}
func (t *SheetMetalCornerTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalCornerTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal corner: pick a corner edge and set a positive size")
	}
	t.added = t.addCorner(part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Corner")
}

// addCorner builds the corner-treatment feature into engine fs — shared by Commit and preview.
func (t *SheetMetalCornerTool) addCorner(fs *feature.PartFeatures) *feature.PartFeature {
	size := t.size
	return feature.NewSheetMetalCornerFeatures(fs).Add(&feature.SheetMetalCornerDefinition{
		EdgeKeys: edgeHandleKeys(t.edges), Treatment: feature.CornerChamfer, Size: func() float64 { return size },
	})
}

// DraftFeature returns the unattached corner-treatment feature the viewport previews (#1626).
func (t *SheetMetalCornerTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCorner(fs), nil
	})
}

// SheetMetalCornerSeamTool opens a gap seam where two flanges meet at a corner.
type SheetMetalCornerSeamTool struct {
	edges []EdgeHandle
	gap   float64
	added *feature.PartFeature
}

// NewSheetMetalCornerSeamTool returns a corner-seam tool defaulting to a 0.2 mm gap.
func NewSheetMetalCornerSeamTool() *SheetMetalCornerSeamTool {
	return &SheetMetalCornerSeamTool{gap: 0.02}
}

func (t *SheetMetalCornerSeamTool) Name() string   { return "Sheet Metal Corner Seam" }
func (t *SheetMetalCornerSeamTool) Start(*Session) {}

// AcceptedKinds declares the corner seam picks edges (the corner edges to seam).
func (t *SheetMetalCornerSeamTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectEdge}
}

// Picks reports the picked edges for the unified highlight.
func (t *SheetMetalCornerSeamTool) Picks() []Selectable { return edgeSelectables(t.edges) }
func (t *SheetMetalCornerSeamTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		t.edges = append(t.edges, e)
	}
}
func (t *SheetMetalCornerSeamTool) SetGap(v float64)                   { t.gap = v }
func (t *SheetMetalCornerSeamTool) Gap() float64                       { return t.gap }
func (t *SheetMetalCornerSeamTool) CanCommit() bool                    { return len(t.edges) > 0 && t.gap > 0 }
func (t *SheetMetalCornerSeamTool) Cancel(s *Session)                  {}
func (t *SheetMetalCornerSeamTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalCornerSeamTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal corner seam: pick the seam edges and set a positive gap")
	}
	t.added = t.addCornerSeam(part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Corner Seam")
}

// addCornerSeam builds the corner-seam feature into engine fs — shared by Commit and preview.
func (t *SheetMetalCornerSeamTool) addCornerSeam(fs *feature.PartFeatures) *feature.PartFeature {
	gap := t.gap
	return feature.NewSheetMetalCornerSeamFeatures(fs).Add(&feature.SheetMetalCornerSeamDefinition{
		EdgeKeys: edgeHandleKeys(t.edges), Gap: func() float64 { return gap }, Type: feature.GapSeam,
	})
}

// DraftFeature returns the unattached corner-seam feature the viewport previews (#1626).
func (t *SheetMetalCornerSeamTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCornerSeam(fs), nil
	})
}

// SheetMetalCutTool removes a picked sketch profile through the sheet (through all).
type SheetMetalCutTool struct {
	profile *ProfileHandle
	added   *feature.PartFeature
}

// NewSheetMetalCutTool returns a cut tool awaiting a profile.
func NewSheetMetalCutTool() *SheetMetalCutTool { return &SheetMetalCutTool{} }

func (t *SheetMetalCutTool) Name() string   { return "Sheet Metal Cut" }
func (t *SheetMetalCutTool) Start(*Session) {}

// AcceptedKinds declares the cut picks a closed sketch region (profile).
func (t *SheetMetalCutTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectProfile} }

// Picks reports the picked region for the unified highlight.
func (t *SheetMetalCutTool) Picks() []Selectable { return singlePick(t.profile) }
func (t *SheetMetalCutTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profile = &p
	}
}
func (t *SheetMetalCutTool) CanCommit() bool                    { return t.profile != nil }
func (t *SheetMetalCutTool) Cancel(s *Session)                  {}
func (t *SheetMetalCutTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalCutTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if t.profile == nil {
		return errors.New("sheet-metal cut: pick a closed sketch profile")
	}
	t.added = t.addCut(part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Cut")
}

// addCut builds the sheet-metal cut feature into engine fs — shared by Commit and preview.
func (t *SheetMetalCutTool) addCut(fs *feature.PartFeatures) *feature.PartFeature {
	return feature.NewSheetMetalCutFeatures(fs).Add(&feature.SheetMetalCutDefinition{
		Sketch: t.profile.Sketch, ProfileIndex: t.profile.ProfileIndex,
	})
}

// DraftFeature returns the unattached sheet-metal cut feature the viewport previews (#1626).
func (t *SheetMetalCutTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCut(fs), nil
	})
}

// edgeHandleKeys collects the reference keys of picked edges.
func edgeHandleKeys(edges []EdgeHandle) [][]byte {
	keys := make([][]byte, len(edges))
	for i, e := range edges {
		keys[i] = e.Edge.ReferenceKey()
	}
	return keys
}
