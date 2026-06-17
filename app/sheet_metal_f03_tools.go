// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Sheet-metal F03 modify tools (M13-F03 UI): the interactive Modify-panel commands for the
// rip/lip/punch/cosmetic-bend features. Each picks geometry (an edge, a sketch line, or a
// profile) and commits the feature, mirroring the F02 tools (flange = edge, bend = line, cut =
// profile). Thickness and the default bend radius come from the active rule.

// SheetMetalLipTool folds a stiffening lip (a short flange curled 180° back) onto a picked edge.
type SheetMetalLipTool struct {
	edge      *EdgeHandle
	height    float64 // flange wall height, model units
	returnLen float64 // return wall length after the curl, model units
	angle     float64 // flange bend angle, radians (90° default)
	added     *feature.PartFeature
}

// NewSheetMetalLipTool returns a lip tool defaulting to a 10 mm, 90° flange with a 3 mm return.
func NewSheetMetalLipTool() *SheetMetalLipTool {
	return &SheetMetalLipTool{height: 1.0, returnLen: 0.3, angle: halfPiAngle}
}

func (t *SheetMetalLipTool) Name() string { return "Sheet Metal Lip" }
func (t *SheetMetalLipTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge))
}
func (t *SheetMetalLipTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
func (t *SheetMetalLipTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok {
		t.edge = &e
	}
}

// Lip dimension accessors the property panel edits (lengths in model units, angle in radians).
func (t *SheetMetalLipTool) SetHeight(h float64)       { t.height = h }
func (t *SheetMetalLipTool) Height() float64           { return t.height }
func (t *SheetMetalLipTool) SetReturnLength(l float64) { t.returnLen = l }
func (t *SheetMetalLipTool) ReturnLength() float64     { return t.returnLen }
func (t *SheetMetalLipTool) SetAngle(a float64)        { t.angle = a }
func (t *SheetMetalLipTool) Angle() float64            { return t.angle }

func (t *SheetMetalLipTool) CanCommit() bool {
	return t.edge != nil && t.height > 0 && t.returnLen > 0 && t.angle > 0
}

func (t *SheetMetalLipTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalLipTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal lip: pick an edge and set a positive height/return/angle")
	}
	t.added = t.addLip(part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Lip")
}

func (t *SheetMetalLipTool) addLip(fs *feature.PartFeatures) *feature.PartFeature {
	height, returnLen, angle := t.height, t.returnLen, t.angle
	return feature.NewSheetMetalLipFeatures(fs).Add(&feature.SheetMetalLipDefinition{
		EdgeKey:      t.edge.Edge.ReferenceKey(),
		Height:       func() float64 { return height },
		ReturnLength: func() float64 { return returnLen },
		Angle:        func() float64 { return angle },
	})
}

// DraftFeature returns the unattached sheet-metal lip feature the viewport previews.
func (t *SheetMetalLipTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addLip(fs), nil
	})
}

// SheetMetalRipTool slits the sheet along a picked sketch line, opening a seam of the gap width.
type SheetMetalRipTool struct {
	line  *SketchEntityHandle
	gap   float64 // slit width, model units
	added *feature.PartFeature
}

// NewSheetMetalRipTool returns a rip tool defaulting to a 0.1 mm kerf.
func NewSheetMetalRipTool() *SheetMetalRipTool { return &SheetMetalRipTool{gap: 0.01} }

func (t *SheetMetalRipTool) Name() string { return "Sheet Metal Rip" }
func (t *SheetMetalRipTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
}
func (t *SheetMetalRipTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
func (t *SheetMetalRipTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.line = &h
	}
}
func (t *SheetMetalRipTool) SetGap(g float64)                   { t.gap = g }
func (t *SheetMetalRipTool) Gap() float64                       { return t.gap }
func (t *SheetMetalRipTool) CanCommit() bool                    { return t.line != nil && t.gap > 0 }
func (t *SheetMetalRipTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalRipTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal rip: pick a sketch line and set a positive gap")
	}
	added, err := t.addRip(part, part.Features())
	if err != nil {
		return err
	}
	t.added = added
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Rip")
}

func (t *SheetMetalRipTool) addRip(part *compdef.PartComponentDefinition, fs *feature.PartFeatures) (*feature.PartFeature, error) {
	sk, idx, ok := lineHandleInPart(part, t.line.Entity)
	if !ok {
		return nil, errors.New("sheet-metal rip: the pick is not a sketch line")
	}
	gap := t.gap
	return feature.NewSheetMetalRipFeatures(fs).Add(&feature.SheetMetalRipDefinition{
		Sketch: sk, LineIndex: idx, Gap: func() float64 { return gap },
	}), nil
}

// DraftFeature returns the unattached sheet-metal rip feature the viewport previews.
func (t *SheetMetalRipTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addRip(part, fs)
	})
}

// SheetMetalPunchTool stamps every closed profile of the picked profile's sketch through the
// sheet (a die pattern from one sketch).
type SheetMetalPunchTool struct {
	profile *ProfileHandle
	added   *feature.PartFeature
}

// NewSheetMetalPunchTool returns a punch tool awaiting a profile.
func NewSheetMetalPunchTool() *SheetMetalPunchTool { return &SheetMetalPunchTool{} }

func (t *SheetMetalPunchTool) Name() string { return "Sheet Metal Punch" }
func (t *SheetMetalPunchTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile))
}
func (t *SheetMetalPunchTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
func (t *SheetMetalPunchTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profile = &p
	}
}
func (t *SheetMetalPunchTool) CanCommit() bool                    { return t.profile != nil }
func (t *SheetMetalPunchTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalPunchTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if t.profile == nil {
		return errors.New("sheet-metal punch: pick a closed sketch profile (all profiles of its sketch are punched)")
	}
	t.added = t.addPunch(part.Features())
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Punch")
}

func (t *SheetMetalPunchTool) addPunch(fs *feature.PartFeatures) *feature.PartFeature {
	return feature.NewSheetMetalPunchFeatures(fs).Add(&feature.SheetMetalPunchDefinition{
		Sketch: t.profile.Sketch,
	})
}

// DraftFeature returns the unattached sheet-metal punch feature the viewport previews.
func (t *SheetMetalPunchTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addPunch(fs), nil
	})
}

// SheetMetalCosmeticBendTool marks a cosmetic bend line (no fold) on a picked sketch line.
type SheetMetalCosmeticBendTool struct {
	line  *SketchEntityHandle
	angle float64 // bend angle, radians (90° default)
	added *feature.PartFeature
}

// NewSheetMetalCosmeticBendTool returns a cosmetic-bend tool defaulting to 90°.
func NewSheetMetalCosmeticBendTool() *SheetMetalCosmeticBendTool {
	return &SheetMetalCosmeticBendTool{angle: halfPiAngle}
}

func (t *SheetMetalCosmeticBendTool) Name() string { return "Sheet Metal Cosmetic Bend" }
func (t *SheetMetalCosmeticBendTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
}
func (t *SheetMetalCosmeticBendTool) Cancel(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter())
}
func (t *SheetMetalCosmeticBendTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.line = &h
	}
}
func (t *SheetMetalCosmeticBendTool) SetAngle(a float64)                 { t.angle = a }
func (t *SheetMetalCosmeticBendTool) Angle() float64                     { return t.angle }
func (t *SheetMetalCosmeticBendTool) CanCommit() bool                    { return t.line != nil && t.angle > 0 }
func (t *SheetMetalCosmeticBendTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalCosmeticBendTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	if !t.CanCommit() {
		return errors.New("sheet-metal cosmetic bend: pick a sketch line and set a positive angle")
	}
	sk, idx, ok := lineHandleInPart(part, t.line.Entity)
	if !ok {
		return errors.New("sheet-metal cosmetic bend: the pick is not a sketch line")
	}
	angle := t.angle
	t.added = feature.NewSheetMetalCosmeticBendFeatures(part.Features()).Add(&feature.SheetMetalCosmeticBendDefinition{
		Sketch: sk, LineIndex: idx, Angle: func() float64 { return angle },
	})
	return commitSheetMetalFeature(s, part, t.added, "Sheet Metal Cosmetic Bend")
}
