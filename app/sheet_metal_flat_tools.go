// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/feature"

// Sheet-metal flat-pattern tools (M13-F04): unfold the part flat (to cut while flat) and
// refold it. Neither picks geometry — they act on the part's recorded bends — so they commit
// immediately when activated and OK'd.

// SheetMetalUnfoldTool flattens every bend of the part (Create Flat Pattern).
type SheetMetalUnfoldTool struct {
	added *feature.PartFeature
}

// NewSheetMetalUnfoldTool returns an unfold tool.
func NewSheetMetalUnfoldTool() *SheetMetalUnfoldTool { return &SheetMetalUnfoldTool{} }

func (t *SheetMetalUnfoldTool) Name() string                       { return "Sheet Metal Unfold" }
func (t *SheetMetalUnfoldTool) Start(*Session)                     {}
func (t *SheetMetalUnfoldTool) Pick(*Session, Selectable)          {}
func (t *SheetMetalUnfoldTool) CanCommit() bool                    { return true }
func (t *SheetMetalUnfoldTool) Cancel(*Session)                    {}
func (t *SheetMetalUnfoldTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalUnfoldTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	pf, err := part.AddUnfold()
	if err != nil {
		return err
	}
	t.added = pf
	return commitSheetMetalFeature(s, part, pf, "Sheet Metal Unfold")
}

// SheetMetalRefoldTool re-folds the bends an earlier unfold flattened.
type SheetMetalRefoldTool struct {
	added *feature.PartFeature
}

// NewSheetMetalRefoldTool returns a refold tool.
func NewSheetMetalRefoldTool() *SheetMetalRefoldTool { return &SheetMetalRefoldTool{} }

func (t *SheetMetalRefoldTool) Name() string                       { return "Sheet Metal Refold" }
func (t *SheetMetalRefoldTool) Start(*Session)                     {}
func (t *SheetMetalRefoldTool) Pick(*Session, Selectable)          {}
func (t *SheetMetalRefoldTool) CanCommit() bool                    { return true }
func (t *SheetMetalRefoldTool) Cancel(*Session)                    {}
func (t *SheetMetalRefoldTool) AddedFeature() *feature.PartFeature { return t.added }

func (t *SheetMetalRefoldTool) Commit(s *Session) error {
	part, err := activeSheetMetalPart(s)
	if err != nil {
		return err
	}
	pf, err := part.AddRefold()
	if err != nil {
		return err
	}
	t.added = pf
	return commitSheetMetalFeature(s, part, pf, "Sheet Metal Refold")
}
