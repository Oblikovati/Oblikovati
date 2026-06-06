// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	stdmath "math"
	"os"
	"testing"

	"oblikovati/api/types"
	"oblikovati/kernel/ops"
)

func TestDecode3MFHandAuthoredTetrahedronIsWatertightSolid(t *testing.T) {
	data, err := os.ReadFile("testdata/tetra.3mf")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	raw, err := Decode3MF(data)
	if err != nil {
		t.Fatalf("Decode3MF: %v", err)
	}
	body, warns, err := SolidOrSurface(raw, "import:3mf#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("SolidOrSurface: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("hand-authored 3MF tetra did not import as a solid; warnings=%v", warns)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("3MF tetra is not valid: %v", r.Issues)
	}
}

func TestThreeMFRoundTripCubePreservesSolidAndVolume(t *testing.T) {
	src, _, err := SolidOrSurface(cubeSoup(2), "import:src#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("build source: %v", err)
	}
	data, err := Encode3MF(src, ops.DefaultQuality())
	if err != nil {
		t.Fatalf("Encode3MF: %v", err)
	}
	raw, err := Decode3MF(data)
	if err != nil {
		t.Fatalf("Decode3MF: %v", err)
	}
	body, _, err := SolidOrSurface(raw, "import:rt#0", DefaultWeldTolerance)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if !body.IsSolid() {
		t.Fatalf("round-tripped 3MF cube is not a solid")
	}
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if want := 8.0; stdmath.Abs(got-want) > 1e-4 {
		t.Errorf("3MF round-trip volume = %v, want %v", got, want)
	}
}

func TestDecode3MFMissingModelPartErrors(t *testing.T) {
	// An empty (but valid) ZIP has no 3D/3dmodel.model part → a clear error.
	emptyZip := []byte("PK\x05\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	_, err := Decode3MF(emptyZip)
	if err == nil {
		t.Fatalf("expected a missing-model-part error for an empty 3MF")
	}
}

func TestDecode3MFNotAZipErrors(t *testing.T) {
	_, err := Decode3MF([]byte("not a zip"))
	if err == nil {
		t.Fatalf("expected a not-a-ZIP error")
	}
}

func TestExport3MFCurvedResolutionIsMonotonic(t *testing.T) {
	cyl := cylinderBody(t)
	count := func(res types.MeshResolution) int {
		_, n, err := ExportBody(types.Format3MF, cyl, res)
		if err != nil {
			t.Fatalf("ExportBody 3MF %s: %v", res, err)
		}
		return n
	}
	if l, m, h := count(types.ResolutionLow), count(types.ResolutionMedium), count(types.ResolutionHigh); l >= m || m >= h {
		t.Errorf("3MF triangle count not strictly increasing: %d %d %d", l, m, h)
	}
}
