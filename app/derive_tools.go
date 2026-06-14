// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// The Derive and Shrinkwrap tools (#767) gather a source assembly (and, for shrinkwrap, the
// simplification options) and merge it into the active part as a base body on Commit. They read
// the open assemblies on Start and present them — plus the shrinkwrap options — through the generic
// tool-param dialog (Params), so no head dialog is needed. The geometry is model/feature; these are
// interaction shells driven headlessly here.

// deriveSourceTool collects the source-assembly choice shared by Derive and Shrinkwrap.
type deriveSourceTool struct {
	sources  []*doc.Document // open assemblies, captured on Start
	selected int             // index into sources
}

// Start captures the currently-open assemblies as the candidate sources.
func (t *deriveSourceTool) Start(s *Session) { t.sources = s.OpenAssemblies() }

// Pick is unused — the source is chosen from the dialog, not by a viewport pick.
func (t *deriveSourceTool) Pick(*Session, Selectable) {}
func (t *deriveSourceTool) Cancel(*Session)           {}

// source returns the chosen source document, or nil when no assembly is open.
func (t *deriveSourceTool) source() *doc.Document {
	if t.selected < 0 || t.selected >= len(t.sources) {
		return nil
	}
	return t.sources[t.selected]
}

// sourceChoice is the source-assembly chooser param (the open assemblies by display name).
func (t *deriveSourceTool) sourceChoice() ChoiceParam {
	names := make([]string, len(t.sources))
	for i, d := range t.sources {
		names[i] = d.DisplayName()
	}
	return ChoiceParam{Label: "Source assembly", Options: names,
		Get: func() int { return t.selected }, Set: func(i int) { t.selected = i }}
}

// --- Derive Assembly ------------------------------------------------------

// DeriveAssemblyTool merges the chosen source assembly into the active part as a full base body.
type DeriveAssemblyTool struct{ deriveSourceTool }

// NewDeriveAssemblyTool returns an idle Derive tool.
func NewDeriveAssemblyTool() *DeriveAssemblyTool { return &DeriveAssemblyTool{} }
func (t *DeriveAssemblyTool) Name() string       { return "Derive Assembly" }
func (t *DeriveAssemblyTool) Prompt(*Session) string {
	return "Choose the source assembly to derive, then OK."
}
func (t *DeriveAssemblyTool) CanCommit() bool { return t.source() != nil }

func (t *DeriveAssemblyTool) Commit(s *Session) error {
	_, err := s.DeriveAssembly(t.source())
	return err
}

func (t *DeriveAssemblyTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{t.sourceChoice()}}
}

// --- Shrinkwrap -----------------------------------------------------------

// ShrinkwrapTool merges the chosen source assembly into the active part as a simplified, lightweight
// base body: drop small parts, replace the rest with bounding-box envelopes, and/or patch internal
// voids first.
type ShrinkwrapTool struct {
	deriveSourceTool
	removeStyle   int     // 0 none, 1 small parts
	minPartVolume float64 // threshold for "small parts" (units³)
	envelopeStyle int     // 0 none, 1 per-part box, 2 whole box
	patchHoles    bool
}

// NewShrinkwrapTool returns a Shrinkwrap tool with the plain include-all defaults.
func NewShrinkwrapTool() *ShrinkwrapTool { return &ShrinkwrapTool{} }
func (t *ShrinkwrapTool) Name() string   { return "Shrinkwrap" }
func (t *ShrinkwrapTool) Prompt(*Session) string {
	return "Choose the source assembly and simplification options, then OK."
}
func (t *ShrinkwrapTool) CanCommit() bool { return t.source() != nil }

func (t *ShrinkwrapTool) Commit(s *Session) error {
	def := feature.ShrinkwrapDefinition{
		RemoveStyle:   feature.ShrinkwrapRemoveStyle(t.removeStyle),
		MinPartVolume: t.minPartVolume,
		EnvelopeStyle: feature.ShrinkwrapEnvelopeStyle(t.envelopeStyle),
		PatchHoles:    t.patchHoles,
	}
	_, err := s.ShrinkwrapAssembly(t.source(), def)
	return err
}

func (t *ShrinkwrapTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{
			t.sourceChoice(),
			{Label: "Remove", Options: []string{"Keep all", "Small parts"},
				Get: func() int { return t.removeStyle }, Set: func(i int) { t.removeStyle = i }},
			{Label: "Envelope", Options: []string{"Real geometry", "Per-part box", "Whole box"},
				Get: func() int { return t.envelopeStyle }, Set: func(i int) { t.envelopeStyle = i }},
		},
		Floats: []FloatParam{
			{"Min part volume", func() float64 { return t.minPartVolume }, func(v float64) { t.minPartVolume = v }},
		},
		Bools: []BoolParam{
			{"Patch holes", func() bool { return t.patchHoles }, func(b bool) { t.patchHoles = b }},
		},
	}
}
