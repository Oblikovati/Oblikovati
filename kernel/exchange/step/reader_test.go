// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// readFixture loads a hand-authored .step file from testdata, failing on I/O error.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// importOneSolid imports a fixture and returns its single solid body.
func importOneSolid(t *testing.T, name string) *topo.Body {
	t.Helper()
	bodies, warns, err := Reader{}.ImportSolids(readFixture(t, name), exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("import %s: %v", name, err)
	}
	for _, w := range warns {
		t.Logf("warning: %s", w)
	}
	if len(bodies) != 1 {
		t.Fatalf("import %s: got %d bodies, want 1", name, len(bodies))
	}
	return bodies[0]
}

func TestImportCubeIsValidSolid(t *testing.T) {
	body := importOneSolid(t, "cube.step")
	if !body.IsSolid() {
		t.Error("imported cube should be a solid")
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("imported cube is invalid: %+v", r)
	}
	if got := len(body.Faces()); got != 6 {
		t.Errorf("cube has %d faces, want 6", got)
	}
}

func TestImportCubeVolume(t *testing.T) {
	body := importOneSolid(t, "cube.step")
	props := ops.BodyGeometryProperties(body, ops.DefaultQuality())
	const want = 1000.0 // 10mm cube
	if !approx(props.Volume, want, 1e-3) {
		t.Errorf("cube volume = %g mm^3, want %g", props.Volume, want)
	}
}

// approx reports whether got is within relTol (relative) of want.
func approx(got, want, relTol float64) bool {
	d := got - want
	if d < 0 {
		d = -d
	}
	return d <= relTol*want
}

func TestImportCylinderIsValidSolid(t *testing.T) {
	body := importOneSolid(t, "cylinder.step")
	if !body.IsSolid() {
		t.Error("imported cylinder should be a solid")
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("imported cylinder is invalid: %+v", r)
	}
	if got := len(body.Faces()); got != 3 {
		t.Errorf("cylinder has %d faces, want 3", got)
	}
}

func TestImportCylinderVolume(t *testing.T) {
	body := importOneSolid(t, "cylinder.step")
	// Curved-body volume converges with faceting density (the divergence sum is exact
	// only for planar faces), so the gate uses a fine chord/angle tolerance.
	props := ops.BodyGeometryProperties(body, fineQuality())
	const want = 3.141592653589793 * 5.0 * 5.0 * 20.0 // pi r^2 h
	if !approx(props.Volume, want, 5e-3) {
		t.Errorf("cylinder volume = %g mm^3, want %g", props.Volume, want)
	}
}

// fineQuality is a tight faceting used to evaluate curved-body mass properties to
// within the keystone tolerance.
func fineQuality() ops.Quality {
	return ops.Quality{ChordTolerance: 0.002, AngleTolerance: 0.01}
}

func TestImportBoxWithHoleIsValidSolid(t *testing.T) {
	body := importOneSolid(t, "box_hole.step")
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("imported box-with-hole is invalid: %+v", r)
	}
	if got := len(body.Faces()); got != 7 {
		t.Errorf("box-with-hole has %d faces, want 7", got)
	}
}

func TestImportBoxWithHoleVolume(t *testing.T) {
	body := importOneSolid(t, "box_hole.step")
	props := ops.BodyGeometryProperties(body, fineQuality())
	const want = 20.0*20.0*20.0 - 3.141592653589793*5.0*5.0*20.0 // block - bore
	if !approx(props.Volume, want, 5e-3) {
		t.Errorf("box-with-hole volume = %g mm^3, want %g", props.Volume, want)
	}
}

func TestImportCubeSharesEdges(t *testing.T) {
	body := importOneSolid(t, "cube.step")
	// A closed manifold cube has 12 edges, each used by exactly two faces with
	// opposite orientation — the proof the STEP sense triple composed correctly.
	edges := body.Edges()
	if len(edges) != 12 {
		t.Fatalf("cube has %d shared edges, want 12", len(edges))
	}
	for _, e := range edges {
		uses := e.Uses()
		if len(uses) != 2 {
			t.Fatalf("edge %d has %d uses, want 2", e.ID(), len(uses))
		}
		if uses[0].Reversed() == uses[1].Reversed() {
			t.Errorf("edge %d uses are not opposite-oriented", e.ID())
		}
	}
}
