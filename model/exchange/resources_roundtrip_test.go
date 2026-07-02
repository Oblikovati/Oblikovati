// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// TestImportedDocumentReopensWithoutSourceFile is the ADR-0031 acceptance test: after importing
// a mesh, the source bytes live IN the document, so the .obk reopens to the same geometry even
// when the original file is gone. It imports a binary STL cube, saves, DELETES the source STL,
// reopens in a fresh workspace, and checks the body is back with the same volume.
func TestImportedDocumentReopensWithoutSourceFile(t *testing.T) {
	dir := t.TempDir()
	stlPath := writeCubeSTL(t, dir, 2) // a binary-STL cube ⇒ a base64 resource
	obkPath := filepath.Join(dir, "part.obk")

	store := persistence.NewPackageStore()
	d, err := doc.NewWorkspace(store, contentset.Default()).Add(doc.Part, obkPath, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	part := d.Content().(*compdef.PartComponentDefinition)
	if res, err := Import(part, stlPath, types.FormatSTL); err != nil || res.BodyCount != 1 {
		t.Fatalf("Import: res=%+v err=%v", res, err)
	}
	wantVol := totalVolume(part)
	if wantVol <= 0 {
		t.Fatalf("imported cube volume = %.4f, want positive", wantVol)
	}
	// A resource must have been embedded (binary STL ⇒ base64), with the source filename as origin.
	if got := part.Resources(); len(got) != 1 {
		t.Fatalf("embedded resources = %d, want 1", len(got))
	}
	for _, r := range part.Resources() {
		if r.Type != "StlFile" || r.Encoding != doc.EncodingBase64 || r.Origin != "cube.stl" {
			t.Errorf("resource = %+v, want StlFile/base64/origin cube.stl", struct{ T, E, O string }{r.Type, r.Encoding, r.Origin})
		}
	}

	if err := doc.NewWorkspace(store, contentset.Default()).Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// The whole point: the source file is gone, yet the document still rebuilds.
	if err := os.Remove(stlPath); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(obkPath, true)
	if err != nil {
		t.Fatalf("Open (source deleted): %v", err)
	}
	rpart := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := rpart.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("reopened body count = %d, want 1 (re-derived from the embedded resource)", len(bodies))
	}
	if got := totalVolume(rpart); math.Abs(got-wantVol)/wantVol > 1e-6 {
		t.Errorf("reopened volume = %.6f, want %.6f (byte-identical re-import)", got, wantVol)
	}
}
