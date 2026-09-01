// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// appTetraSTL is a tiny ASCII STL of a tetrahedron (4 facets, 4 shared corners) for the
// place-mesh tests.
const appTetraSTL = `solid tetra
facet normal 0 0 -1
outer loop
vertex 0 0 0
vertex 1 0 0
vertex 0 1 0
endloop
endfacet
facet normal 0 -1 0
outer loop
vertex 0 0 0
vertex 0 0 1
vertex 1 0 0
endloop
endfacet
facet normal -1 0 0
outer loop
vertex 0 0 0
vertex 0 1 0
vertex 0 0 1
endloop
endfacet
facet normal 1 1 1
outer loop
vertex 1 0 0
vertex 0 0 1
vertex 0 1 0
endloop
endfacet
endsolid tetra
`

// tempSTL writes the tetra STL to a temp file and returns its path.
func tempSTL(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tetra.stl")
	if err := os.WriteFile(path, []byte(appTetraSTL), 0o644); err != nil {
		t.Fatalf("write stl: %v", err)
	}
	return path
}

// TestImportMeshFilePlacesReferenceGeometry places the STL as a mesh feature: the running
// solid passes through unchanged and the mesh exposes its welded facet topology.
func TestImportMeshFilePlacesReferenceGeometry(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	pf, err := s.ImportMeshFile(tempSTL(t))
	if err != nil {
		t.Fatalf("ImportMeshFile: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after place mesh: %d bodies, want 1 (solid passes through)", def.SurfaceBodies().Count())
	}
	mesh := pf.Definition().(*feature.MeshFeature)
	if mesh.Faces().Count() != 4 || mesh.Vertices().Count() != 4 {
		t.Errorf("mesh topology = %d facets / %d verts, want 4 / 4 (welded tetra)",
			mesh.Faces().Count(), mesh.Vertices().Count())
	}
}

// A malformed STL surfaces a precise parse error, not a placed feature.
func TestImportMeshFileRejectsMalformedSTL(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	path := filepath.Join(t.TempDir(), "bad.stl")
	if err := os.WriteFile(path, []byte("solid nope\nfacet normal x y z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportMeshFile(path); err == nil {
		t.Fatal("a malformed STL must error")
	}
}

// TestPlaceMeshCommandArmsFileDialog asserts the ribbon command raises the one-shot
// file-dialog request the head consumes.
func TestPlaceMeshCommandArmsFileDialog(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Mesh.Place"); err != nil {
		t.Fatalf("execute Mesh.Place: %v", err)
	}
	if !s.TakeImportMeshRequest() {
		t.Fatal("Place Mesh did not arm the file-dialog request")
	}
	if s.TakeImportMeshRequest() {
		t.Fatal("the place-mesh request must be one-shot")
	}
}
