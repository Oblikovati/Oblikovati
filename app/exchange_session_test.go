// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// sessionWithPart returns a session whose active document is an empty part.
func sessionWithPart(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(compdef.NewPartComponentDefinition())
	return s
}

// TestSessionImportExportFile drives the exact methods the head's File ▸ Import / Export menu
// calls: import a STEP solid (format inferred from the extension) into the active part, then
// export the part back to a STEP file and re-import it — the body stays a valid solid throughout.
func TestSessionImportExportFile(t *testing.T) {
	t.Parallel()
	cube := filepath.Join("..", "kernel", "exchange", "step", "testdata", "cube.step")
	s := sessionWithPart(t)
	res, err := s.ImportFile(cube)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if res.BodyCount < 1 || !res.Solid {
		t.Fatalf("import: bodyCount=%d solid=%v, want a solid", res.BodyCount, res.Solid)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if r := ops.Validate(def.SurfaceBodies().Item(0)); !r.Valid || !def.SurfaceBodies().Item(0).IsSolid() {
		t.Fatalf("imported body not a valid solid: %+v", r)
	}

	out := filepath.Join(t.TempDir(), "out.step")
	if _, err := s.ExportFile(out, types.ResolutionMedium); err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	back := sessionWithPart(t)
	if r2, err := back.ImportFile(out); err != nil || !r2.Solid {
		t.Fatalf("re-import exported step: solid=%v err=%v", r2.Solid, err)
	}
}

// TestSessionImportUnknownExtension reports a clear error for an unrecognized file type.
func TestSessionImportUnknownExtension(t *testing.T) {
	t.Parallel()
	if _, err := sessionWithPart(t).ImportFile("model.iges"); err == nil {
		t.Error("ImportFile accepted an unknown extension")
	}
}
