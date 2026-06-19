// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/api/types"

// draftingStandardChoice pairs a Standard dropdown label with the standard it selects, in
// dropdown order (index → standard).
type draftingStandardChoice struct {
	label    string
	standard types.DraftingStandard
}

var draftingStandardChoices = []draftingStandardChoice{
	{"ISO (metric)", types.DraftingISO},
	{"ANSI (imperial)", types.DraftingANSI},
}

func draftingStandardIndexOf(std types.DraftingStandard) int {
	for i, c := range draftingStandardChoices {
		if c.standard == std {
			return i
		}
	}
	return 0
}

// DraftingStandardTool sets the active drawing's drafting standard. It is a dialog-only
// tool: the user picks ISO or ANSI and OK re-points the active style preset, so every
// annotation re-renders to that standard.
type DraftingStandardTool struct {
	dialogTool
	index int
}

// NewDraftingStandardTool creates the tool; its selection is set to the drawing's current
// standard on Start.
func NewDraftingStandardTool() *DraftingStandardTool { return &DraftingStandardTool{} }

func (t *DraftingStandardTool) Name() string { return "Drafting Standard" }

// Start pre-selects the drawing's current standard.
func (t *DraftingStandardTool) Start(s *Session) {
	if c, err := ActiveDrawing(s); err == nil {
		t.index = draftingStandardIndexOf(c.Styles().ActiveStandard())
	}
}

func (t *DraftingStandardTool) CanCommit() bool { return true }

// Commit applies the selected standard to the active drawing.
func (t *DraftingStandardTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	idx := t.index
	if idx < 0 || idx >= len(draftingStandardChoices) {
		idx = 0
	}
	c.Styles().SetActiveStandard(draftingStandardChoices[idx].standard)
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the standard dropdown for the property dialog.
func (t *DraftingStandardTool) Params() ToolParams {
	labels := make([]string, len(draftingStandardChoices))
	for i, c := range draftingStandardChoices {
		labels[i] = c.label
	}
	return ToolParams{Choices: []ChoiceParam{
		{Label: "Standard", Options: labels, Get: func() int { return t.index }, Set: func(i int) { t.index = i }},
	}}
}
