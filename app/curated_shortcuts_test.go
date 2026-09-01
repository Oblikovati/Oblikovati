// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestCuratedDefaultShortcutsResolve pins the shipped Shift+mnemonic shortcuts (#1751 S3): each
// headline sketch/feature tool resolves from its Shift+<letter> chord. Shift is mandatory because
// bare a–z are reserved for command-window typing (S1/S2); the letter mirrors the tool's mnemonic
// so Shift+L is Line, Shift+E is Extrude, etc. Collision-freedom across the whole registry is
// guarded separately by TestStandardCommandsPassCheckDefaults (CheckDefaults).
func TestCuratedDefaultShortcutsResolve(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	want := map[string]string{
		"Shift+L": "Sketch.Line",
		"Shift+C": "Sketch.Circle",
		"Shift+A": "Sketch.Arc",
		"Shift+D": "Sketch.Dimension",
		"Shift+E": "Create.Extrude",
		"Shift+R": "Create.Revolve",
		"Shift+H": "Modify.Hole",
		"Shift+F": "Modify.Fillet",
	}
	for chordStr, id := range want {
		c, err := types.ParseChord(chordStr)
		if err != nil {
			t.Fatalf("ParseChord(%q): %v", chordStr, err)
		}
		got, ok := s.Bindings().ResolveChord(c)
		if !ok || got != id {
			t.Errorf("ResolveChord(%q) = %q,%v; want %q", chordStr, got, ok, id)
		}
	}
}
