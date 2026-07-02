// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// Derive / Shrinkwrap into the active part (M11-F06, #767): merge an open assembly into the active
// part as one base body — full (Derive) or simplified (Shrinkwrap) — keeping a link to the source
// so the derived body can be updated when the source changes, or frozen by breaking the link. The
// geometry lives in model/feature (the derived-assembly / shrinkwrap features) and the wire surface
// already exists; these are the head/app commands around it. All four mutate the active part and
// record an undo step.

// OpenAssemblies returns the open assembly documents — the candidate sources for a derive or
// shrinkwrap, in workspace order.
func (s *Session) OpenAssemblies() []*doc.Document {
	var out []*doc.Document
	for _, d := range s.workspace.Documents() {
		if _, ok := d.Content().(*compdef.AssemblyComponentDefinition); ok {
			out = append(out, d)
		}
	}
	return out
}

// DeriveAssembly derives the source assembly into the active part as a full base body (every
// component merged) and records the part→source link so it can be updated later. Returns the new
// feature.
func (s *Session) DeriveAssembly(source *doc.Document) (*feature.PartFeature, error) {
	part, src, err := s.deriveTarget(source, "Derive Assembly")
	if err != nil {
		return nil, err
	}
	return s.commitDerive(part, addDerivedFeature(part.Features(), src, source), source, "Derive Assembly"), nil
}

// addDerivedFeature runs the derive construction against engine — shared by Session.DeriveAssembly
// (the part's engine) and DeriveAssemblyTool.DraftFeature (a scratch engine) so the committed and
// previewed features cannot drift (#1626).
func addDerivedFeature(engine *feature.PartFeatures, src feature.AssemblyBodySource, source *doc.Document) *feature.PartFeature {
	return feature.NewDerivedAssemblyComponents(engine).AddDerived(src, deriveSourceLinkOf(source))
}

// ShrinkwrapAssembly derives the source assembly into the active part as a simplified, lightweight
// base body per def (removal / envelope / patch-holes options). Returns the new feature.
func (s *Session) ShrinkwrapAssembly(source *doc.Document, def feature.ShrinkwrapDefinition) (*feature.PartFeature, error) {
	part, src, err := s.deriveTarget(source, "Shrinkwrap")
	if err != nil {
		return nil, err
	}
	return s.commitDerive(part, addShrinkwrapFeature(part.Features(), src, def, source), source, "Shrinkwrap"), nil
}

// addShrinkwrapFeature is addDerivedFeature's shrinkwrap sibling — the one construction shared
// by Session.ShrinkwrapAssembly and ShrinkwrapTool.DraftFeature (#1626).
func addShrinkwrapFeature(engine *feature.PartFeatures, src feature.AssemblyBodySource, def feature.ShrinkwrapDefinition, source *doc.Document) *feature.PartFeature {
	return feature.NewShrinkwrapComponents(engine).AddShrinkwrap(src, def, deriveSourceLinkOf(source))
}

// deriveTarget resolves the active part and validates source is an assembly body source.
func (s *Session) deriveTarget(source *doc.Document, op string) (*compdef.PartComponentDefinition, feature.AssemblyBodySource, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, nil, err
	}
	if source == nil {
		return nil, nil, errors.New(op + ": choose a source assembly first")
	}
	src, ok := source.Content().(feature.AssemblyBodySource)
	if !ok {
		return nil, nil, fmt.Errorf("%s: %q is not an assembly", op, source.DisplayName())
	}
	return part, src, nil
}

// commitDerive names the new derive feature, records the part→source reference edge (so the first
// save snapshots the link), recomputes, and records the undo step.
func (s *Session) commitDerive(part *compdef.PartComponentDefinition, pf *feature.PartFeature, source *doc.Document, label string) *feature.PartFeature {
	pf.SetName(part.Features().UniqueName(pf.Kind()))
	s.ActiveDocument().OpenReference(source.FullDocumentName()) // the source is open; this resolves the edge now
	part.Recompute()
	s.recordEdit(part, label)
	return pf
}

// deriveSourceLinkOf captures the source assembly document's identity for the derive link, so the
// link survives a save and a stale source is detected on reopen (#715).
func deriveSourceLinkOf(source *doc.Document) feature.DeriveSourceLink {
	id := source.FileIdentity()
	return feature.DeriveSourceLink{
		Document:           source.FullDocumentName(),
		InternalName:       id.InternalName,
		DatabaseRevisionID: id.DatabaseRevisionID,
	}
}

// UpdateDerivedFeature re-syncs a derived/shrinkwrap feature to its source's current revision,
// clearing its out-of-date state, and recomputes the part (the browser's Update action).
func (s *Session) UpdateDerivedFeature(f *feature.PartFeature) error {
	part, ds, err := s.deriveStatusOf(f, "update derived")
	if err != nil {
		return err
	}
	ds.AcknowledgeSource(s.sourceRevision(ds.SourceLink().Document))
	part.Recompute()
	s.recordEdit(part, "Update Derived")
	return nil
}

// BreakDerivedLink freezes and severs a derived/shrinkwrap feature's source link, so the part keeps
// the current geometry without further updates (the browser's Break Link action).
func (s *Session) BreakDerivedLink(f *feature.PartFeature) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	breaker, ok := deriveDefinition(f).(interface{ BreakLink() error })
	if !ok {
		return errors.New("break link: not a derived/shrinkwrap feature")
	}
	if err := breaker.BreakLink(); err != nil {
		return fmt.Errorf("break link: %w", err)
	}
	part.Recompute()
	s.recordEdit(part, "Break Link")
	return nil
}

// DerivedFeatureStatus reports a feature's derive drive-state for the browser badge/menu: whether
// it is a derive-family feature at all, whether it is out of date relative to its source, and
// whether it is still linked.
func (s *Session) DerivedFeatureStatus(f *feature.PartFeature) (isDerive, outOfDate, linked bool) {
	ds, ok := deriveDefinition(f).(feature.DeriveStatus)
	if !ok {
		return false, false, false
	}
	return true, ds.OutOfDate(), ds.Linked()
}

// deriveStatusOf resolves the active part and the feature's DeriveStatus, erroring when the feature
// is not a derive-family feature.
func (s *Session) deriveStatusOf(f *feature.PartFeature, op string) (*compdef.PartComponentDefinition, feature.DeriveStatus, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, nil, err
	}
	ds, ok := deriveDefinition(f).(feature.DeriveStatus)
	if !ok {
		return nil, nil, errors.New(op + ": not a derived/shrinkwrap feature")
	}
	return part, ds, nil
}

// sourceRevision returns the source document's current model revision, or "" when it is not open
// (so the drive state cannot be compared).
func (s *Session) sourceRevision(sourceName string) string {
	if sourceName == "" {
		return ""
	}
	d, ok := s.workspace.ByName(sourceName)
	if !ok {
		return ""
	}
	return d.FileIdentity().DatabaseRevisionID
}

// deriveDefinition returns f's underlying feature definition (nil-safe), the value the derive
// drive-state interfaces are asserted against.
func deriveDefinition(f *feature.PartFeature) feature.Feature {
	if f == nil {
		return nil
	}
	return f.Definition()
}
