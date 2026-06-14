// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/cmdline"
)

// builtinActionIDs is the set of reserved built-in action ids the vocabulary may target
// besides registered commands.
func builtinActionIDs() map[string]bool {
	return map[string]bool{
		ActionUndo: true, ActionRedo: true, ActionSave: true, ActionCancel: true,
		ActionCommit: true, ActionToggleVisibility: true,
	}
}

// TestVocabularyActionsAreRealCommands guards CLAUDE.md's "never invent ids": every action
// the built-in AutoCAD vocabulary maps to must be a registered command id or a reserved
// built-in action id, so a typed AutoCAD command always dispatches to something real.
func TestVocabularyActionsAreRealCommands(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	builtins := builtinActionIDs()
	for _, action := range cmdline.DefaultVocabulary().Actions() {
		if _, ok := s.Commands().ByID(action); ok {
			continue
		}
		if builtins[action] {
			continue
		}
		t.Errorf("vocabulary action %q is neither a registered command nor a built-in action", action)
	}
}

// TestVocabularyWordsResolveThroughBindings checks the end-to-end path the command window
// uses: a typed AutoCAD word resolves through the binding engine to its action id.
func TestVocabularyWordsResolveThroughBindings(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	b := s.Bindings()
	cases := map[string]string{
		"LINE":    "Sketch.Line",
		"E":       "Create.Extrude",
		"EXTRUDE": "Create.Extrude",
		"UNDO":    ActionUndo,
	}
	for word, want := range cases {
		if got, ok := b.ResolveAlias(word); !ok || got != want {
			t.Errorf("ResolveAlias(%q) = %q,%v, want %q", word, got, ok, want)
		}
	}
}

// TestUserAliasOverridesVocabulary confirms a user-defined alias wins over the built-in
// AutoCAD vocabulary for the same word.
func TestUserAliasOverridesVocabulary(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	b := s.Bindings()
	if err := b.SetAlias("Create.Revolve", "E"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	if got, ok := b.ResolveAlias("E"); !ok || got != "Create.Revolve" {
		t.Errorf("ResolveAlias(E) = %q,%v, want Create.Revolve (user override wins)", got, ok)
	}
}
