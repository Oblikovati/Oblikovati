// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/validate"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
)

// objTetra is a hand-authored OBJ tetrahedron (independent of our encoder), proving the
// v/f decode path and 1-based indices.
const objTetra = `# tetra
v 0 0 0
v 1 0 0
v 0 1 0
v 0 0 1
f 1 3 2
f 1 2 4
f 1 4 3
f 2 3 4
`

func TestDecodeOBJTetrahedronIsWatertightSolid(t *testing.T) {
	raw, err := DecodeOBJ([]byte(objTetra))
	if err != nil {
		t.Fatalf("DecodeOBJ: %v", err)
	}
	body, warns, err := SolidOrSurface(raw, "import:obj#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("hand-authored OBJ tetra did not import as a solid; warnings=%v", warns)
	}
	if r := validate.Validate(body); !r.Valid {
		t.Fatalf("OBJ tetra is not valid: %v", r.Issues)
	}
}

func TestOBJNegativeIndexResolvesRelativeToEnd(t *testing.T) {
	// -1 ⇒ last vertex, -3 ⇒ third-from-last; one face referencing the 3 most-recent verts.
	raw, err := DecodeOBJ([]byte("v 0 0 0\nv 1 0 0\nv 0 1 0\nf -3 -2 -1\n"))
	if err != nil {
		t.Fatalf("DecodeOBJ negative: %v", err)
	}
	if raw.TriangleCount() != 1 {
		t.Fatalf("triangle count = %d, want 1", raw.TriangleCount())
	}
}

func TestOBJRoundTripCubePreservesSolidAndVolume(t *testing.T) {
	src, _, err := SolidOrSurface(cubeSoup(2), "import:src#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build source: %v", err)
	}
	data := EncodeOBJ(tessellateOne(src))
	raw, err := DecodeOBJ(data)
	if err != nil {
		t.Fatalf("DecodeOBJ round-trip: %v", err)
	}
	body, _, err := SolidOrSurface(raw, "import:rt#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("round-tripped OBJ cube is not a solid")
	}
	got := query.BodyGeometryProperties(body, tessellate.DefaultQuality()).Volume
	if want := 8.0; stdmath.Abs(got-want) > 1e-4 {
		t.Errorf("OBJ round-trip volume = %v, want %v", got, want)
	}
}

func TestOBJBadFaceIndexErrors(t *testing.T) {
	_, err := DecodeOBJ([]byte("v 0 0 0\nf 1 2 3\n")) // only one vertex
	if err == nil {
		t.Fatalf("expected an out-of-range face index error")
	}
}

func TestExportOBJCurvedResolutionIsMonotonic(t *testing.T) {
	cyl := cylinderBody(t)
	count := func(res types.MeshResolution) int {
		_, n, err := ExportBody(types.FormatOBJ, cyl, res)
		if err != nil {
			t.Fatalf("ExportBody OBJ %s: %v", res, err)
		}
		return n
	}
	if l, m, h := count(types.ResolutionLow), count(types.ResolutionMedium), count(types.ResolutionHigh); l >= m || m >= h {
		t.Errorf("OBJ triangle count not strictly increasing: %d %d %d", l, m, h)
	}
}
