// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// emptySketches is a SketchIndexer with no sketches — an imported body consumes none.
type emptySketches struct{}

func (emptySketches) IndexOf(*sketch.Sketch) (int, bool) { return 0, false }
func (emptySketches) At(int) (*sketch.Sketch, bool)      { return nil, false }

// fakeResources is a named ResourceStore fake: imported file bytes keyed by resource UUID,
// standing in for the document's embedded resource table (ADR-0031).
type fakeResources map[string][]byte

func (f fakeResources) ResourceBytes(id string) ([]byte, bool) {
	b, ok := f[id]
	return b, ok
}

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
	t.Parallel()
	dir := t.TempDir()
	path := writeCubeSTL(t, dir)
	raw := mustRead(t, path)
	body, _, err := meshio.ImportBody(types.FormatSTL, raw, "import:stl#0", 0,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ImportBody: %v", err)
	}
	// The source bytes live in the document resource table, cited by UUID (ADR-0031).
	const resID = "11111111-2222-3333-4444-555555555555"
	store := fakeResources{resID: raw}
	fs := NewPartFeatures(nil)
	fs.SetResourceStore(store)
	NewImportedBodies(fs).Add(body, resID, "stl")

	data, err := fs.MarshalRecipe(emptySketches{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if len(data) != 1 || data[0].Kind != "importedBody" || data[0].Import == nil {
		t.Fatalf("marshaled = %+v, want one importedBody with payload", data)
	}
	if data[0].Import.Resource != resID || data[0].Import.Format != "stl" {
		t.Errorf("import payload = %+v, want resource=%q format=stl", data[0].Import, resID)
	}

	// Rebuild from the recipe (re-derives the body from the embedded resource, NOT from disk —
	// the source file path is never consulted) into a fresh engine and recompute.
	restored := NewPartFeatures(nil)
	restored.SetResourceStore(store)
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
