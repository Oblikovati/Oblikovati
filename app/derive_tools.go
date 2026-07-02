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
	dialogTool
	sources  []*doc.Document // open assemblies, captured on Start
	selected int             // index into sources
}

// Start captures the currently-open assemblies as the candidate sources.
func (t *deriveSourceTool) Start(s *Session) { t.sources = s.OpenAssemblies() }

// Pick is unused — the source is chosen from the dialog, not by a viewport pick.

// source returns the chosen source document, or nil when no assembly is open.
func (t *deriveSourceTool) source() *doc.Document {
	if t.selected < 0 || t.selected >= len(t.sources) {
		return nil
	}
	return t.sources[t.selected]
}

// assemblySource returns the chosen source document with its assembly body content — false when
// no source is chosen or (defensively) the chosen document is not an assembly. The gate both
// DraftFeature implementations share (#1626); it subsumes CanCommit.
func (t *deriveSourceTool) assemblySource() (*doc.Document, feature.AssemblyBodySource, bool) {
	d := t.source()
	if d == nil {
		return nil, nil, false
	}
	src, ok := d.Content().(feature.AssemblyBodySource)
	return d, src, ok
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

// DraftFeature implements [PartFeatureTool] (#1626): the derived-assembly feature it would
// commit, built into a scratch engine by the same addDerivedFeature Session.DeriveAssembly uses,
// so the commit gate and preview can evaluate it without touching the part.
func (t *DeriveAssemblyTool) DraftFeature(*Session) (feature.Feature, bool) {
	source, src, ok := t.assemblySource()
	if !ok {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return addDerivedFeature(fs, src, source), nil
	})
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
	_, err := s.ShrinkwrapAssembly(t.source(), t.shrinkwrapDefinition())
	return err
}

// shrinkwrapDefinition materializes the dialog options as the feature definition — shared by
// Commit and DraftFeature so the committed and previewed simplification cannot drift (#1626).
func (t *ShrinkwrapTool) shrinkwrapDefinition() feature.ShrinkwrapDefinition {
	return feature.ShrinkwrapDefinition{
		RemoveStyle:   feature.ShrinkwrapRemoveStyle(t.removeStyle),
		MinPartVolume: t.minPartVolume,
		EnvelopeStyle: feature.ShrinkwrapEnvelopeStyle(t.envelopeStyle),
		PatchHoles:    t.patchHoles,
	}
}

// DraftFeature implements [PartFeatureTool] (#1626): the shrinkwrap feature it would commit,
// built into a scratch engine by the same addShrinkwrapFeature Session.ShrinkwrapAssembly uses,
// so the commit gate and preview can evaluate it without touching the part.
func (t *ShrinkwrapTool) DraftFeature(*Session) (feature.Feature, bool) {
	source, src, ok := t.assemblySource()
	if !ok {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return addShrinkwrapFeature(fs, src, t.shrinkwrapDefinition(), source), nil
	})
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
