// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati/api/types"
	"oblikovati/kernel/exchange/meshio"
	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// emptySketches is a SketchIndexer with no sketches — an imported body consumes none.
type emptySketches struct{}

func (emptySketches) IndexOf(*sketch.Sketch) (int, bool) { return 0, false }
func (emptySketches) At(int) (*sketch.Sketch, bool)      { return nil, false }

func writeCubeSTL(t *testing.T, dir string) string {
	t.Helper()
	body, _, err := meshio.SolidOrSurface(unitCubeSoup(2), "fixture#0", meshio.DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build cube: %v", err)
	}
	path := filepath.Join(dir, "cube.stl")
	if err := os.WriteFile(path, meshio.EncodeBinarySTL(body, ops.DefaultQuality()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func unitCubeSoup(s float64) meshio.RawMesh {
	v := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s) }
	p := [8]math.Point3{
		v(0, 0, 0), v(1, 0, 0), v(1, 1, 0), v(0, 1, 0),
		v(0, 0, 1), v(1, 0, 1), v(1, 1, 1), v(0, 1, 1),
	}
	quads := [6][4]int{{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {2, 3, 7, 6}, {1, 2, 6, 5}, {0, 4, 7, 3}}
	var m meshio.RawMesh
	for _, q := range quads {
		m.AddTriangle(p[q[0]], p[q[1]], p[q[2]])
		m.AddTriangle(p[q[0]], p[q[2]], p[q[3]])
	}
	return m
}

func TestImportedBodyFeatureRoundTripsViaReImport(t *testing.T) {
	dir := t.TempDir()
	path := writeCubeSTL(t, dir)
	body, _, err := meshio.ImportBody(types.FormatSTL, mustRead(t, path), "import:stl#0", 0)
	if err != nil {
		t.Fatalf("ImportBody: %v", err)
	}
	fs := NewPartFeatures(nil, nil)
	NewImportedBodies(fs).Add(body, path, "stl")

	data, err := fs.MarshalRecipe(emptySketches{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if len(data) != 1 || data[0].Kind != "importedBody" || data[0].Import == nil {
		t.Fatalf("marshaled = %+v, want one importedBody with payload", data)
	}
	if data[0].Import.Path != path || data[0].Import.Format != "stl" {
		t.Errorf("import payload = %+v, want path=%q format=stl", data[0].Import, path)
	}

	// Rebuild from the recipe (re-imports from disk) into a fresh engine and recompute.
	restored := NewPartFeatures(nil, nil)
	if err := restored.ApplyRecipe(data, emptySketches{}, NewWorkGeometry()); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	restored.Recompute()
	bodies := restored.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("restored result = %d bodies; want one solid", len(bodies))
	}
	if r := ops.Validate(bodies[0]); !r.Valid {
		t.Fatalf("restored imported body is not valid: %v", r.Issues)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return data
}
