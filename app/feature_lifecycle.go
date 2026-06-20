// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Session actions for the lifecycle of a placed feature (issue #140): rename,
// explicit suppression, history reorder, and committing an in-place definition
// edit. Like the browser actions (app/browser_actions.go) each records an undo
// edit and recomputes when the change affects geometry; they are the seam the
// addin router's features.* methods call into.

// activePartFeature resolves the active part and guards that f belongs to it, naming
// the action for errors (e.g. "RenameFeature").
func activePartFeature(s *Session, f *feature.PartFeature, action string) (*compdef.PartComponentDefinition, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("app: %s with nil feature", action)
	}
	if got, ok := part.Features().ByID(f.ID()); !ok || got != f {
		return nil, fmt.Errorf("app: %s: feature %q not in active part", action, f.Name())
	}
	return part, nil
}

// RenameFeature sets a feature's display name (the id stays stable). The name must
// be non-empty and not already used by another feature — Inventor enforces unique
// browser names, and ByName lookups would otherwise turn ambiguous.
func (s *Session) RenameFeature(f *feature.PartFeature, name string) error {
	part, err := activePartFeature(s, f, "RenameFeature")
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("app: RenameFeature: name must be non-empty")
	}
	if other, ok := part.Features().ByName(name); ok && other != f {
		return fmt.Errorf("app: RenameFeature: name %q is already used by another feature", name)
	}
	f.SetName(name)
	s.recordEdit(part, "Rename Feature")
	return nil
}

// SetFeatureSuppressed sets (not toggles) explicit suppression and recomputes.
// Setting the current state is a no-op, so a replicated or retried call is
// idempotent and records no spurious undo edit.
func (s *Session) SetFeatureSuppressed(f *feature.PartFeature, suppressed bool) error {
	part, err := activePartFeature(s, f, "SetFeatureSuppressed")
	if err != nil {
		return err
	}
	if f.Suppressed() == suppressed {
		return nil
	}
	f.SetSuppressed(suppressed)
	part.Recompute()
	s.recordEdit(part, suppressLabel(suppressed))
	return nil
}

func suppressLabel(suppressed bool) string {
	if suppressed {
		return "Suppress Feature"
	}
	return "Unsuppress Feature"
}

// ReorderFeature moves a feature to a new history index and recomputes. The model
// rejects a move that would place the feature before one it depends on.
func (s *Session) ReorderFeature(f *feature.PartFeature, newIndex int) error {
	part, err := activePartFeature(s, f, "ReorderFeature")
	if err != nil {
		return err
	}
	if err := part.Features().Reorder(f, newIndex); err != nil {
		return err
	}
	part.Recompute()
	s.recordEdit(part, "Reorder Feature")
	return nil
}

// CommitFeatureEdit recomputes after the caller mutated f's definition through its
// [feature.Editable] / [feature.ReferenceEditable] surface, and records the undo
// edit — the commit seam model/feature/editable.go points feature editors at.
func (s *Session) CommitFeatureEdit(f *feature.PartFeature) error {
	part, err := activePartFeature(s, f, "CommitFeatureEdit")
	if err != nil {
		return err
	}
	part.Features().MarkDirty(f)
	part.Recompute()
	s.recordEdit(part, "Edit Feature")
	s.EmitFeatureLifecycle(FeatureEdited, f) // featureEdited (#1085)
	return nil
}
