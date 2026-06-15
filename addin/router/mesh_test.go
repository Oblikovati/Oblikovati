// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The #700 acceptance over the wire: the mesh op places an ASCII STL as reference
// geometry through features.add (it must not consume the running solid).

// routerTetraSTL is a tiny ASCII STL of a tetrahedron for the wire test.
const routerTetraSTL = `solid tetra
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

func TestMeshFeatureOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	path := filepath.Join(t.TempDir(), "tetra.stl")
	if err := os.WriteFile(path, []byte(routerTetraSTL), 0o644); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", fmt.Sprintf(`{"kind":"mesh","args":{"path":%q}}`, path), &res)
	if res.Bodies != 0 {
		t.Fatalf("mesh reference geometry changed the body count to %d, want 0 (empty part passes through)", res.Bodies)
	}
}

// TestMeshSolidOverWire imports the same STL with solid:true and confirms the mesh is
// converted to a validated solid body (M20-F15 #492).
func TestMeshSolidOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	path := filepath.Join(t.TempDir(), "tetra.stl")
	if err := os.WriteFile(path, []byte(routerTetraSTL), 0o644); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", fmt.Sprintf(`{"kind":"mesh","args":{"path":%q,"solid":true}}`, path), &res)
	if res.Bodies != 1 {
		t.Fatalf("mesh-solid body count = %d, want 1 (converted solid)", res.Bodies)
	}
	var v struct {
		Valid bool `json:"valid"`
	}
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatal("converted mesh-solid body is not valid")
	}
}

func TestMeshFeatureOverWireRejectsMissingFile(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"mesh","args":{"path":"/nonexistent/x.stl"}}`)); err == nil {
		t.Fatal("a missing STL path must error")
	}
}
