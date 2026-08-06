// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

// TestMeshIsSpatial guards the gate that keeps a body-only .ipt's import faithful: Inventor stores a
// FLAT footprint placeholder as the graphics mesh for some parts (e.g. MBK_Keycap: a 6-triangle
// 192×181×0 quad) whose real body lives in the SAB. Importing that would give a wrong flat sheet, so
// a degenerate (non-3-D) mesh must be rejected while a real thin body is kept.
func TestMeshIsSpatial(t *testing.T) {
	flat := meshio.RawMesh{ // a 20×18×0 quad, two triangles — the placeholder shape
		Verts: []m.Point3{m.P3(0, 0, 0), m.P3(2, 0, 0), m.P3(2, 1.8, 0), m.P3(0, 1.8, 0)},
		Tris:  [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	if meshIsSpatial(flat) {
		t.Error("flat placeholder mesh accepted; a z=0 footprint must be rejected")
	}
	thin := meshio.RawMesh{ // a real but thin body (0.29 cm deep, like the light pipe)
		Verts: []m.Point3{m.P3(0, 0, 0), m.P3(2, 0, 0), m.P3(2, 1.8, 0.29), m.P3(0, 1.8, 0.29)},
		Tris:  [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	if !meshIsSpatial(thin) {
		t.Error("real thin body rejected; a mesh with 3-D extent must be kept")
	}
	if meshIsSpatial(meshio.RawMesh{}) {
		t.Error("empty mesh accepted")
	}
}

// TestSoupFromMesh checks that Inventor's decoded display tessellation adapts to a RawMesh
// keeping vertex positions (cm) and triangle indices — the input to the graphics-mesh body
// fallback used for real parts the parametric decode can't rebuild.
func TestSoupFromMesh(t *testing.T) {
	mesh := ipt.Mesh{
		Verts: [][3]float64{{0, 0, 0}, {2, 0, 0}, {0, 3, 0}},
		Tris:  [][3]int{{0, 1, 2}},
	}
	raw := SoupFromMesh(mesh)
	if len(raw.Verts) != 3 || raw.TriangleCount() != 1 {
		t.Fatalf("got %d verts, %d tris; want 3 verts, 1 tri", len(raw.Verts), raw.TriangleCount())
	}
	if v := raw.Verts[1]; float64(v.X) != 2 || float64(v.Y) != 0 || float64(v.Z) != 0 {
		t.Errorf("vertex 1 = %v, want (2,0,0)", v)
	}
}

func readCorpus(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

// TestBoxBodyVolumeMatchesOracle reconstructs the box body from its .ipt and checks
// the kernel-computed volume against Inventor's geometry: 4x2x1 cm = 8 cm^3 = 8000 mm^3.
func TestBoxBodyVolumeMatchesOracle(t *testing.T) {
	body, warns, err := BodyFromIPT(readCorpus(t, "10_box.ipt"))
	if err != nil {
		t.Fatalf("BodyFromIPT: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("reconstructed body is not solid (warnings: %v)", warns)
	}
	mp := analysis.MassPropertiesOf([]*topo.Body{body}, 1, types.MassPropertiesHigh)
	const wantMm3 = 8000.0
	if math.Abs(mp.VolumeMm3-wantMm3) > 1.0 {
		t.Errorf("box volume = %.3f mm^3, want %.1f mm^3", mp.VolumeMm3, wantMm3)
	}
}

// TestCurvedBodyReportsUntessellated documents the current planar-only limit: a body
// with cone/cylinder faces is refused rather than silently wrong.
func TestCurvedBodyReportsUntessellated(t *testing.T) {
	if _, _, err := BodyFromIPT(readCorpus(t, "15_cylinder.ipt")); err == nil {
		t.Errorf("expected an error for the curved cylinder body (not yet tessellated)")
	}
}
