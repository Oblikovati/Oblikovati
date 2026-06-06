// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	stdmath "math"
	"testing"

	"oblikovati/api/types"
	"oblikovati/kernel/brep"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// cylinderBody builds an analytic curved solid cylinder (so tessellation density — the
// resolution knob — varies with quality; a welded planar mesh would not).
func cylinderBody(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return b
}

// asciiCubeSTL is a hand-authored ASCII STL of a unit cube (independent of our encoder)
// — one of the two faces are split into two triangles; written compactly via the helper
// in the test body. We keep it tiny: a single tetrahedron is enough to prove the ASCII
// path decodes, weld and surface-body fallback are covered elsewhere.
const asciiTetraSTL = `solid t
facet normal 0 0 0
 outer loop
  vertex 0 0 0
  vertex 1 0 0
  vertex 0 1 0
 endloop
endfacet
facet normal 0 0 0
 outer loop
  vertex 0 0 0
  vertex 0 1 0
  vertex 0 0 1
 endloop
endfacet
facet normal 0 0 0
 outer loop
  vertex 0 0 0
  vertex 0 0 1
  vertex 1 0 0
 endloop
endfacet
facet normal 0 0 0
 outer loop
  vertex 1 0 0
  vertex 0 0 1
  vertex 0 1 0
 endloop
endfacet
endsolid t
`

func TestDecodeASCIISTLTetrahedronIsWatertightSolid(t *testing.T) {
	raw, err := DecodeSTL([]byte(asciiTetraSTL))
	if err != nil {
		t.Fatalf("DecodeSTL ascii: %v", err)
	}
	body, warns, err := SolidOrSurface(raw, "import:stl#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("hand-authored tetra did not import as a solid; warnings=%v", warns)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("tetra is not valid: %v", r.Issues)
	}
}

func TestBinarySTLRoundTripCubePreservesSolidAndVolume(t *testing.T) {
	src, _, err := SolidOrSurface(cubeSoup(3), "import:src#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build source: %v", err)
	}
	data := EncodeBinarySTL(src, ops.DefaultQuality())
	if !isBinarySTL(data) {
		t.Fatalf("encoded STL is not detected as binary")
	}
	raw, err := DecodeSTL(data)
	if err != nil {
		t.Fatalf("DecodeSTL binary: %v", err)
	}
	body, _, err := SolidOrSurface(raw, "import:rt#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("round-tripped cube is not a solid")
	}
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if want := 27.0; stdmath.Abs(got-want) > 1e-4 { // 3³
		t.Errorf("round-trip volume = %v, want %v", got, want)
	}
}

func TestExportTriangleCountIncreasesWithResolutionForCurvedBody(t *testing.T) {
	cyl := cylinderBody(t)
	low := triCountForFormat(t, cyl, types.ResolutionLow)
	med := triCountForFormat(t, cyl, types.ResolutionMedium)
	high := triCountForFormat(t, cyl, types.ResolutionHigh)
	if low >= med || med >= high {
		t.Errorf("triangle count not strictly increasing low<med<high: %d %d %d", low, med, high)
	}
}

// triCountForFormat exports cyl as STL at res and reports the triangle count written.
func triCountForFormat(t *testing.T, body *topo.Body, res types.MeshResolution) int {
	t.Helper()
	_, tris, err := ExportBody(types.FormatSTL, body, res)
	if err != nil {
		t.Fatalf("ExportBody %s: %v", res, err)
	}
	return tris
}
