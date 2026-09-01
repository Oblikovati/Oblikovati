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
	t.Parallel()
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
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	b := s.Bindings()
	cases := map[string]string{
		"LINE":       "Sketch.Line",
		"EXT":        "Create.Extrude",
		"EXTRUDE":    "Create.Extrude",
		"FILLET":     "Sketch.Fillet", // 2D
		"FILLETEDGE": "Modify.Fillet", // 3D — distinct word
		"SURFPATCH":  "Surface.Patch",
		"UNION":      "Modify.Combine",  // §2 boolean
		"GCPARALLEL": "Sketch.Parallel", // §2 constraint
		"VIEWBASE":   "Drawing.BaseView",
		"PLACE":      "Assembly.Place",
		"FLANGE":     "SheetMetal.Flange",
		"UNDO":       ActionUndo,
	}
	for word, want := range cases {
		if got, ok := b.ResolveAlias(word); !ok || got != want {
			t.Errorf("ResolveAlias(%q) = %q,%v, want %q", word, got, ok, want)
		}
	}
	// A bare single letter is NOT a static vocabulary word — single letters belong to the
	// keybinding editor (personalised Shift/Control chords), not the command-window list.
	if got, ok := b.ResolveAlias("E"); ok {
		t.Errorf("ResolveAlias(\"E\") = %q,%v, want no resolution (single letters are editor-only)", got, ok)
	}
}

// TestEveryVocabularyWordResolves is the completeness guard: every word in the built-in
// AutoCAD vocabulary must resolve through the binding engine to a real, registered action —
// so no table entry points at a missing or misspelled command id (CLAUDE.md: never invent
// ids; F07 expanded the table broadly).
func TestEveryVocabularyWordResolves(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	b := s.Bindings()
	for _, word := range cmdline.DefaultVocabulary().Words() {
		if _, ok := b.ResolveAlias(word); !ok {
			t.Errorf("vocabulary word %q does not resolve to a registered action", word)
		}
	}
}

// TestUserAliasOverridesVocabulary confirms a user-defined alias wins over the built-in
// AutoCAD vocabulary for the same word.
func TestUserAliasOverridesVocabulary(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	b := s.Bindings()
	// EXTRUDE is a built-in vocabulary word for Create.Extrude; a user alias for the same word
	// must win. (Single-letter aliases are rejected, so the override is a multi-letter word.)
	if err := b.SetAlias("Create.Revolve", "EXTRUDE"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	if got, ok := b.ResolveAlias("EXTRUDE"); !ok || got != "Create.Revolve" {
		t.Errorf("ResolveAlias(EXTRUDE) = %q,%v, want Create.Revolve (user override wins)", got, ok)
	}
}
