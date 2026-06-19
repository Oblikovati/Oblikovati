// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/doc"
)

// ModelReferenceTool sets the model the active drawing documents — the part or assembly
// whose iProperties fill the title block. It is a dialog-only tool: the user picks one of
// the open part/assembly documents and OK stores the reference.
type ModelReferenceTool struct {
	dialogTool
	candidates []string // full document names of the open part/assembly documents
	selected   int
}

// NewModelReferenceTool creates the tool; its candidate list is captured on Start.
func NewModelReferenceTool() *ModelReferenceTool { return &ModelReferenceTool{} }

func (t *ModelReferenceTool) Name() string { return "Model Reference" }

// Start captures the open models that can be referenced, pre-selecting the drawing's
// current reference if it is among them.
func (t *ModelReferenceTool) Start(s *Session) {
	t.candidates = referenceableModels(s)
	if c, err := ActiveDrawing(s); err == nil {
		for i, name := range t.candidates {
			if name == c.ModelReference() {
				t.selected = i
			}
		}
	}
}

// CanCommit requires at least one open model to reference.
func (t *ModelReferenceTool) CanCommit() bool { return len(t.candidates) > 0 }

// Commit stores the selected model as the drawing's referenced model.
func (t *ModelReferenceTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if t.selected < 0 || t.selected >= len(t.candidates) {
		return fmt.Errorf("drawing: no model selected (%d open models)", len(t.candidates))
	}
	c.SetModelReference(t.candidates[t.selected])
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the open-model dropdown for the property dialog.
func (t *ModelReferenceTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "Model", Options: t.candidates, Get: func() int { return t.selected }, Set: func(i int) { t.selected = i }},
	}}
}

// referenceableModels returns the full document names of the open part and assembly
// documents — the models a drawing can document (drawings and presentations are excluded).
func referenceableModels(s *Session) []string {
	var names []string
	for _, d := range s.workspace.Documents() {
		switch d.DocumentType() {
		case doc.Part, doc.Assembly:
			names = append(names, d.FullDocumentName())
		}
	}
	return names
}
