// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The shell's per-face wall overrides over the wire (#1864). Each thickness is an EXPRESSION, so
// the test reads the resolved value off the definition — a request that merely succeeded would
// pass even if the expression never reached the feature.

// lastShellDef returns the definition of the part's most recently added shell.
func lastShellDef(t *testing.T, s *app.Session) *feature.ShellDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if sh, ok := fs.Item(i).Definition().(*feature.ShellFeature); ok {
			return sh.Definition()
		}
	}
	t.Fatal("no shell feature on the part")
	return nil
}

// TestShellFaceThicknessesReachTheDefinition: the override list arrives whole, with each
// expression resolved in model units (cm) — 3 mm is 0.3, not 3.
func TestShellFaceThicknessesReachTheDefinition(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "shell", map[string]any{
		"faceRefs": []string{face}, "thickness": "1 mm",
		"faceThicknesses": []map[string]any{{"faceRef": face, "thickness": "3 mm"}},
	}); err != nil {
		t.Fatalf("shell with faceThicknesses: %v", err)
	}
	fts := lastShellDef(t, s).FaceThicknesses
	if len(fts) != 1 || string(fts[0].FaceKey) != face {
		t.Fatalf("faceThicknesses reached the definition as %+v, want one entry for the picked face", fts)
	}
	if got := fts[0].Thickness(); got < 0.2999 || got > 0.3001 {
		t.Errorf("override thickness resolved to %g cm, want 0.3 (3 mm)", got)
	}
}

// TestShellFaceThicknessRequiresAFace: an entry with no face names nothing, and defaulting it to
// the whole shell would quietly rewrite every wall.
func TestShellFaceThicknessRequiresAFace(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "shell", map[string]any{
		"faceRefs": []string{face}, "thickness": "1 mm",
		"faceThicknesses": []map[string]any{{"thickness": "3 mm"}},
	}); err == nil {
		t.Error("a faceThicknesses entry without a faceRef should be refused")
	}
}

// TestShellWithoutFaceThicknessesIsUniform: omitting the option must leave the shell exactly as
// it was before the option existed.
func TestShellWithoutFaceThicknessesIsUniform(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "shell", map[string]any{
		"faceRefs": []string{face}, "thickness": "1 mm",
	}); err != nil {
		t.Fatalf("plain shell: %v", err)
	}
	if fts := lastShellDef(t, s).FaceThicknesses; len(fts) != 0 {
		t.Errorf("a shell with no overrides carries %+v", fts)
	}
}
