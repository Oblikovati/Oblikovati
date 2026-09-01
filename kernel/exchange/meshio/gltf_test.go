// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// gltfTestMesh builds an tessellate.Mesh from parallel position/normal/index slices —
// the direct fixture shape for the sanitizer and document tests.
func gltfTestMesh(positions []math.Point3, normals []math.Vector3, indices []int) *tessellate.Mesh {
	return &tessellate.Mesh{Positions: positions, Normals: normals, Indices: indices}
}

// gltfBodyMesh wraps a mesh as a BodyMesh record with a stable id/name.
func gltfBodyMesh(id, name string, m *tessellate.Mesh) BodyMesh {
	return BodyMesh{ID: id, Name: name, Mesh: m}
}

// parseGLB is the test-side minimal GLB reader: header, JSON chunk, BIN chunk.
type parsedGLB struct {
	jsonData []byte
	binData  []byte
	jsonLen  int // padded chunkLength
	binLen   int // padded chunkLength
}

func parseGLB(t *testing.T, data []byte) parsedGLB {
	t.Helper()
	if len(data) < 12 {
		t.Fatalf("GLB too short: %d", len(data))
	}
	if got := binary.LittleEndian.Uint32(data[0:4]); got != glbMagic {
		t.Fatalf("magic = %#x, want %#x", got, glbMagic)
	}
	if got := binary.LittleEndian.Uint32(data[4:8]); got != glbVersion {
		t.Fatalf("version = %d, want %d", got, glbVersion)
	}
	if got := binary.LittleEndian.Uint32(data[8:12]); int(got) != len(data) {
		t.Fatalf("header length = %d, want file size %d", got, len(data))
	}
	off := 12
	jsonLen := int(binary.LittleEndian.Uint32(data[off : off+4]))
	if got := binary.LittleEndian.Uint32(data[off+4 : off+8]); got != glbJSON {
		t.Fatalf("first chunk type = %#x, want JSON", got)
	}
	jsonData := data[off+8 : off+8+jsonLen]
	off += 8 + jsonLen
	binLen := int(binary.LittleEndian.Uint32(data[off : off+4]))
	if got := binary.LittleEndian.Uint32(data[off+4 : off+8]); got != glbBIN {
		t.Fatalf("second chunk type = %#x, want BIN", got)
	}
	binData := data[off+8 : off+8+binLen]
	if off+8+binLen != len(data) {
		t.Fatalf("GLB has trailing bytes: consumed %d of %d", off+8+binLen, len(data))
	}
	return parsedGLB{jsonData: jsonData, binData: binData, jsonLen: jsonLen, binLen: binLen}
}

// gltfDoc is the test-side JSON document view.
type gltfDoc struct {
	Asset struct {
		Version   string `json:"version"`
		Generator string `json:"generator"`
	} `json:"asset"`
	Scene  int `json:"scene"`
	Scenes []struct {
		Nodes []int `json:"nodes"`
	} `json:"scenes"`
	Nodes []struct {
		Name string `json:"name"`
		Mesh int    `json:"mesh"`
	} `json:"nodes"`
	Meshes []struct {
		Name       string `json:"name"`
		Primitives []struct {
			Attributes map[string]int `json:"attributes"`
			Indices    int            `json:"indices"`
			Material   int            `json:"material"`
			Mode       int            `json:"mode"`
		} `json:"primitives"`
	} `json:"meshes"`
	Materials []struct {
		Name string `json:"name"`
		PBR  struct {
			BaseColorFactor [4]float64 `json:"baseColorFactor"`
			MetallicFactor  float64    `json:"metallicFactor"`
			RoughnessFactor float64    `json:"roughnessFactor"`
		} `json:"pbrMetallicRoughness"`
	} `json:"materials"`
	Accessors []struct {
		BufferView    int       `json:"bufferView"`
		ComponentType int       `json:"componentType"`
		Count         int       `json:"count"`
		Type          string    `json:"type"`
		Min           []float64 `json:"min"`
		Max           []float64 `json:"max"`
	} `json:"accessors"`
	BufferViews []struct {
		Buffer     int `json:"buffer"`
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
		Target     int `json:"target"`
	} `json:"bufferViews"`
	Buffers []struct {
		ByteLength int `json:"byteLength"`
	} `json:"buffers"`
}

func parseGLTFJSON(t *testing.T, data []byte) gltfDoc {
	t.Helper()
	var doc gltfDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal glTF JSON: %v", err)
	}
	return doc
}

// TestGLTFExportUnitsAreMetres: a 6 cm cube exports with POSITION bounds at
// ±0.06 m — the glTF metre contract via the forced FileUnit "m" (R3-3).
func TestGLTFExportUnitsAreMetres(t *testing.T) {
	box := cmBox(t)
	data, tris, warns, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "in"})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	if tris != 12 {
		t.Errorf("triangle count = %d, want 12", tris)
	}
	if len(warns) != 0 {
		t.Errorf("warnings = %v, want none", warns)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	// POSITION accessor is the first accessor of the first primitive.
	pos := doc.Accessors[doc.Meshes[0].Primitives[0].Attributes["POSITION"]]
	if pos.Type != "VEC3" || pos.ComponentType != gltfFloat {
		t.Fatalf("POSITION accessor = %+v, want VEC3 float", pos)
	}
	if len(pos.Min) != 3 || len(pos.Max) != 3 {
		t.Fatalf("POSITION min/max missing: %+v", pos)
	}
	// 6 cm cube → 0.06 m edges; the swizzle maps kernel +Z to glTF −Y.
	wantMin := []float64{0, 0, -0.06}
	wantMax := []float64{0.06, 0.06, 0}
	for c := 0; c < 3; c++ {
		if stdmath.Abs(pos.Min[c]-wantMin[c]) > 1e-6 || stdmath.Abs(pos.Max[c]-wantMax[c]) > 1e-6 {
			t.Errorf("POSITION bounds = min%v max%v, want min%v max%v", pos.Min, pos.Max, wantMin, wantMax)
		}
	}
}

// TestGLTFZeroValueOptionsDefaultUnit: a zero-value TranslationOptions (the
// CHG3-1 ExportBody delegation path and any direct caller passing the zero
// value) must default TargetUnitMM to the kernel's centimetre database unit —
// a 6 cm cube exports with POSITION max 0.06 m, NOT 0.006 m (CHG3-2).
func TestGLTFZeroValueOptionsDefaultUnit(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	pos := doc.Accessors[doc.Meshes[0].Primitives[0].Attributes["POSITION"]]
	if stdmath.Abs(pos.Max[0]-0.06) > 1e-6 {
		t.Errorf("POSITION max X = %v, want 0.06 m (zero-value options must default to the cm database unit)", pos.Max[0])
	}
}

// TestExportBodyGLTFDelegates: ExportBody(FormatGLTF, body, res) delegates to
// the per-body glTF path and returns a valid GLB (magic + nonempty) — the
// CHG3-1 single-body entry point.
func TestExportBodyGLTFDelegates(t *testing.T) {
	box := cmBox(t)
	data, tris, err := ExportBody(types.FormatGLTF, box, types.ResolutionHigh)
	if err != nil {
		t.Fatalf("ExportBody(gltf): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ExportBody(gltf) returned empty data")
	}
	if tris != 12 {
		t.Errorf("triangle count = %d, want 12", tris)
	}
	parseGLB(t, data) // fails on a bad magic/header
}

// TestGLBContainerBytes checks the GLB container invariants: magic, version,
// exact header length, JSON-first/BIN-second order, padded chunkLengths, and
// the 0x20/0x00 padding bytes (R3-1).
func TestGLBContainerBytes(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	if glb.jsonLen%4 != 0 || glb.binLen%4 != 0 {
		t.Fatalf("chunk lengths not 4-aligned: json=%d bin=%d", glb.jsonLen, glb.binLen)
	}
	// JSON chunk padded with 0x20, BIN chunk padded with 0x00.
	if len(glb.jsonData) > 0 && glb.jsonData[len(glb.jsonData)-1] != 0x20 {
		t.Errorf("JSON chunk not 0x20-padded: last byte %#x", glb.jsonData[len(glb.jsonData)-1])
	}
	if len(glb.binData) > 0 && glb.binData[len(glb.binData)-1] != 0x00 {
		t.Errorf("BIN chunk not 0x00-padded: last byte %#x", glb.binData[len(glb.binData)-1])
	}
	// buffer.byteLength is the UNPADDED BIN length (R3-1).
	doc := parseGLTFJSON(t, glb.jsonData)
	if len(doc.Buffers) != 1 || doc.Buffers[0].ByteLength > glb.binLen {
		t.Errorf("buffer.byteLength = %+v, want <= padded BIN %d", doc.Buffers, glb.binLen)
	}
}

// TestGLBChunkLengthIsPaddedPayload pins R3-1 with odd-length payloads: the
// chunkLength fields carry the PADDED length (Blender advances by chunkLength
// with no alignment rounding).
func TestGLBChunkLengthIsPaddedPayload(t *testing.T) {
	jsonData := bytes.Repeat([]byte("x"), 123) // odd length
	binData := bytes.Repeat([]byte{0xAB}, 5)   // odd length
	data, err := wrapGLB(jsonData, binData)
	if err != nil {
		t.Fatalf("wrapGLB: %v", err)
	}
	glb := parseGLB(t, data)
	if glb.jsonLen != 124 {
		t.Errorf("JSON chunkLength = %d, want 124 (123 padded)", glb.jsonLen)
	}
	if glb.binLen != 8 {
		t.Errorf("BIN chunkLength = %d, want 8 (5 padded)", glb.binLen)
	}
	if !bytes.Equal(glb.jsonData[:123], jsonData) || glb.jsonData[123] != 0x20 {
		t.Errorf("JSON payload/padding wrong")
	}
	if !bytes.Equal(glb.binData[:5], binData) {
		t.Errorf("BIN payload wrong")
	}
	for i := 5; i < 8; i++ {
		if glb.binData[i] != 0x00 {
			t.Errorf("BIN padding byte %d = %#x, want 0", i, glb.binData[i])
		}
	}
}

// TestGLTFMinMaxExactness: the POSITION min/max are computed from the final
// packed float32 values actually written (R4-3), not the float64 source.
func TestGLTFMinMaxExactness(t *testing.T) {
	// A position that changes when narrowed to float32: 0.1 cm → 0.001 m is
	// not exact in binary; the accessor bounds must match the packed value.
	m := gltfTestMesh(
		[]math.Point3{math.P3(0.1, 0, 0), math.P3(0.1, 0.1, 0), math.P3(0.1, 0, 0.1)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "tri", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	packed := float32(0.1 * 0.01)
	if body.min[0] != packed || body.max[0] != packed {
		t.Errorf("bounds X = [%v,%v], want both %v (the packed value)", body.min[0], body.max[0], packed)
	}
}

// TestGLTFIndexTypeSwitch: ≤65535 vertices → uint16; more → uint32; never the
// component max.
func TestGLTFIndexTypeSwitch(t *testing.T) {
	if ct, n := gltfIndexType(65535); ct != gltfUint16 || n != 2 {
		t.Errorf("gltfIndexType(65535) = %d/%d, want uint16/2", ct, n)
	}
	if ct, n := gltfIndexType(65536); ct != gltfUint32 || n != 4 {
		t.Errorf("gltfIndexType(65536) = %d/%d, want uint32/4", ct, n)
	}
}

// TestGLTFNaNInfPositions: NaN/+Inf/−Inf positions drop their triangles
// (R2-14/R4-3); the surviving geometry exports cleanly.
func TestGLTFNaNInfPositions(t *testing.T) {
	nan := stdmath.NaN()
	inf := stdmath.Inf(1)
	ninf := stdmath.Inf(-1)
	for name, bad := range map[string]float64{"nan": nan, "+inf": inf, "-inf": ninf} {
		t.Run(name, func(t *testing.T) {
			m := gltfTestMesh(
				[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(bad, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
				[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
				[]int{0, 1, 2, 3, 4, 5},
			)
			body, err := sanitizeGLTFBody(gltfBodyMesh("1", "bad", m), 0.01)
			if err != nil {
				t.Fatalf("sanitize: %v", err)
			}
			if len(body.indices) != 3 {
				t.Errorf("indices = %v, want only the good triangle", body.indices)
			}
			if len(body.positions) != 9 {
				t.Errorf("positions = %d floats, want 9 (3 vertices)", len(body.positions))
			}
		})
	}
}

// TestGLTFNaNInfNormals: NaN/+Inf/−Inf normals drop their triangles.
func TestGLTFNaNInfNormals(t *testing.T) {
	nan := stdmath.NaN()
	inf := stdmath.Inf(1)
	ninf := stdmath.Inf(-1)
	for name, bad := range map[string]float64{"nan": nan, "+inf": inf, "-inf": ninf} {
		t.Run(name, func(t *testing.T) {
			m := gltfTestMesh(
				[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
				[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(bad, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 1)},
				[]int{0, 1, 2, 3, 4, 5},
			)
			body, err := sanitizeGLTFBody(gltfBodyMesh("1", "bad", m), 0.01)
			if err != nil {
				t.Fatalf("sanitize: %v", err)
			}
			if len(body.indices) != 3 {
				t.Errorf("indices = %v, want only the good triangle", body.indices)
			}
		})
	}
}

// TestGLTFSharedInvalidVertex: one invalid vertex shared by several triangles
// drops all of them; the remaining geometry is pruned and remapped.
func TestGLTFSharedInvalidVertex(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(stdmath.NaN(), 0, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2, 0, 3, 2, 1, 3, 0},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "shared", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 3 {
		t.Errorf("indices = %v, want only the triangle without the invalid vertex", body.indices)
	}
	for _, idx := range body.indices {
		if int(idx) >= len(body.positions)/3 {
			t.Errorf("index %d out of range after prune", idx)
		}
	}
}

// TestGLTFZeroAreaAndPostQuantizationCollapse: zero-area triangles and
// triangles that collapse only after float32 packing are dropped (R4-4).
func TestGLTFZeroAreaAndPostQuantizationCollapse(t *testing.T) {
	// Triangle 0-1-2 is a valid unit triangle; 3-4-5 is zero-area (collinear);
	// 6-7-8 collapses after float32 packing (near-coincident at 1e-9 cm).
	m := gltfTestMesh(
		[]math.Point3{
			math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0),
			math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0),
			math.P3(0, 0, 0), math.P3(1e-9, 0, 0), math.P3(0, 1e-9, 0),
		},
		[]math.Vector3{
			math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1),
			math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1),
			math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1),
		},
		[]int{0, 1, 2, 3, 4, 5, 6, 7, 8},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "deg", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 3 {
		t.Errorf("indices = %v, want only the valid triangle", body.indices)
	}
}

// TestGLTFAllTrianglesDropped: a mesh whose every triangle is dropped by
// sanitization is a typed error (R2-8).
func TestGLTFAllTrianglesDropped(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 0, 0}, // duplicate indices → degenerate
	)
	_, err := sanitizeGLTFBody(gltfBodyMesh("7", "allbad", m), 0.01)
	if err == nil || !strings.Contains(err.Error(), "no valid triangles") {
		t.Fatalf("err = %v, want a no-valid-triangles error", err)
	}
}

// TestGLTFDegenerateBodySkippedNotFatal: a body whose triangles are all dropped
// by sanitization is skipped with a warning, exactly like an empty body — it
// must NOT fail a multi-body batch. The valid body in the same batch still
// exports (CHG-8: the empty-skip and degenerate-skip behave the same).
func TestGLTFDegenerateBodySkippedNotFatal(t *testing.T) {
	good := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	degenerate := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}, // collinear → zero area
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	gb, warns, err := sanitizeGLTFBodyAll(
		[]BodyMesh{gltfBodyMesh("1", "good", good), gltfBodyMesh("2", "deg", degenerate)}, 0.01)
	if err != nil {
		t.Fatalf("sanitizeGLTFBodyAll: %v", err)
	}
	if len(gb) != 1 {
		t.Fatalf("bodies = %d, want 1 (the degenerate body skipped, the valid body kept)", len(gb))
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "body 2") || !strings.Contains(warns[0], "no valid geometry") {
		t.Errorf("warnings = %v, want one naming body 2 with no valid geometry", warns)
	}
}

// TestGLTFAllBodiesDegenerateErrors: when EVERY body drops to no geometry the
// export has nothing to write and errors like the all-empty case (CHG-8), rather
// than emitting an empty GLB.
func TestGLTFAllBodiesDegenerateErrors(t *testing.T) {
	degenerate := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}, // collinear
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	gb, warns, err := sanitizeGLTFBodyAll([]BodyMesh{gltfBodyMesh("1", "deg", degenerate)}, 0.01)
	if err != nil {
		t.Fatalf("sanitizeGLTFBodyAll: %v", err)
	}
	if len(gb) != 0 {
		t.Fatalf("bodies = %d, want 0 (all degenerate)", len(gb))
	}
	if len(warns) != 1 {
		t.Errorf("warnings = %v, want one for the skipped body", warns)
	}
}

// TestGLTFEmptyBodySkip: an empty body is skipped with a warning; a valid body
// still exports (R3-5/R3-6).
func TestGLTFEmptyBodySkip(t *testing.T) {
	empty := topo.BodyFromShells(topo.NewLineage(topo.Tok("x", "y", 0)), false)
	box := cmBox(t)
	data, tris, warns, err := ExportBodiesGLTF([]*topo.Body{empty, box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	if tris != 12 {
		t.Errorf("triangle count = %d, want 12 (the box)", tris)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "empty") {
		t.Errorf("warnings = %v, want one naming the skipped empty body", warns)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	if len(doc.Nodes) != 1 || len(doc.Scenes[0].Nodes) != 1 {
		t.Errorf("nodes = %d, scene nodes = %d, want 1 each (empty body skipped)", len(doc.Nodes), len(doc.Scenes[0].Nodes))
	}
}

// TestGLTFAllEmpty: all bodies empty before tessellation is a typed error
// (R3-5).
func TestGLTFAllEmpty(t *testing.T) {
	empty := topo.BodyFromShells(topo.NewLineage(topo.Tok("x", "y", 0)), false)
	_, _, _, err := ExportBodiesGLTF([]*topo.Body{empty}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err == nil || !strings.Contains(err.Error(), "no exportable bodies") {
		t.Fatalf("err = %v, want a no-exportable-bodies error", err)
	}
}

// TestGLTFNilBodyTypedError: a nil body in the slice is a typed error naming
// the index, returned BEFORE any dereference — no panic (CHG4-2).
func TestGLTFNilBodyTypedError(t *testing.T) {
	box := cmBox(t)
	_, _, _, err := ExportBodiesGLTF([]*topo.Body{box, nil}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err == nil || !strings.Contains(err.Error(), "gltf: body at index 1 is nil") {
		t.Fatalf("err = %v, want a gltf nil-body error naming index 1", err)
	}
}

// TestTessellateBodiesNilBodyTypedError: a nil body in the slice is a typed
// error naming the index, returned BEFORE any dereference — no panic (CHG4-2).
func TestTessellateBodiesNilBodyTypedError(t *testing.T) {
	box := cmBox(t)
	_, err := TessellateBodies([]*topo.Body{box, nil}, QualityFor(types.ResolutionHigh))
	if err == nil || !strings.Contains(err.Error(), "tessellate: body at index 1 is nil") {
		t.Fatalf("err = %v, want a tessellate nil-body error naming index 1", err)
	}
}

// TestGLTFNormalCountMismatch: len(normals) != len(positions) is a typed error
// (R2-9).
func TestGLTFNormalCountMismatch(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	_, err := sanitizeGLTFBody(gltfBodyMesh("9", "mismatch", m), 0.01)
	if err == nil || !strings.Contains(err.Error(), "normal count") {
		t.Fatalf("err = %v, want a normal-count error", err)
	}
}

// TestGLTFIndexOutOfRange: an index beyond the vertex count is a typed error
// (R3-4).
func TestGLTFIndexOutOfRange(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 3},
	)
	_, err := sanitizeGLTFBody(gltfBodyMesh("10", "oob", m), 0.01)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("err = %v, want an out-of-range error", err)
	}
}

// TestGLTFIndexNotDivBy3: an index count not divisible by 3 is a typed error
// (R3-4).
func TestGLTFIndexNotDivBy3(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1},
	)
	_, err := sanitizeGLTFBody(gltfBodyMesh("11", "nondiv", m), 0.01)
	if err == nil || !strings.Contains(err.Error(), "not a multiple of 3") {
		t.Fatalf("err = %v, want a divisibility error", err)
	}
}

// TestGLTFNameCollisions: duplicate body names get deterministic #2 numbering
// against the union of original and generated names (R3-11).
func TestGLTFNameCollisions(t *testing.T) {
	got := allocateGLTFNames([]string{"A", "A", "A#2", "B"})
	want := []string{"A", "A#2", "A#2#2", "B"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %v, want %v", got, want)
		}
	}
}

// TestGLTFSceneReachability: scenes[0].nodes contains every body node exactly
// once (R3-2).
func TestGLTFSceneReachability(t *testing.T) {
	boxA := cmBox(t)
	boxB := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{boxA, boxB}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	if doc.Scene != 0 || len(doc.Scenes) != 1 {
		t.Fatalf("scene = %d, scenes = %d, want 0/1", doc.Scene, len(doc.Scenes))
	}
	if len(doc.Scenes[0].Nodes) != 2 {
		t.Fatalf("scene nodes = %v, want 2", doc.Scenes[0].Nodes)
	}
	seen := map[int]bool{}
	for _, n := range doc.Scenes[0].Nodes {
		if seen[n] {
			t.Errorf("node %d listed twice in scene", n)
		}
		seen[n] = true
	}
	for i := range doc.Nodes {
		if !seen[i] {
			t.Errorf("body node %d not reachable from the default scene", i)
		}
	}
}

// TestGLTFMaterialReference: every primitive references material 0 and the
// material is the explicit CAD PBR (R4-2).
func TestGLTFMaterialReference(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	if len(doc.Materials) != 1 {
		t.Fatalf("materials = %d, want 1", len(doc.Materials))
	}
	mat := doc.Materials[0]
	if mat.Name != "CAD_Default" || mat.PBR.MetallicFactor != 0 || mat.PBR.RoughnessFactor != 1 {
		t.Errorf("material = %+v, want CAD_Default metallic 0 roughness 1", mat)
	}
	if mat.PBR.BaseColorFactor != [4]float64{0.8, 0.8, 0.8, 1.0} {
		t.Errorf("baseColorFactor = %v, want [0.8 0.8 0.8 1]", mat.PBR.BaseColorFactor)
	}
	referenced := false
	for _, mesh := range doc.Meshes {
		for _, prim := range mesh.Primitives {
			if prim.Material != 0 {
				t.Errorf("primitive material = %d, want 0", prim.Material)
			}
			referenced = true
		}
	}
	if !referenced {
		t.Error("no primitive references material 0")
	}
}

// TestGLTFSequentialGLTFThenSTL: exporting glTF then STL through the same
// options object leaves the caller's options untouched and scales each format
// correctly (R3-3).
func TestGLTFSequentialGLTFThenSTL(t *testing.T) {
	box := cmBox(t)
	opts := exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM, FileUnit: "in"}
	glb, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh, opts)
	if err != nil {
		t.Fatalf("glTF export: %v", err)
	}
	if opts.FileUnit != "in" {
		t.Fatalf("caller's FileUnit mutated to %q, want in", opts.FileUnit)
	}
	glbParsed := parseGLB(t, glb)
	doc := parseGLTFJSON(t, glbParsed.jsonData)
	pos := doc.Accessors[doc.Meshes[0].Primitives[0].Attributes["POSITION"]]
	if stdmath.Abs(pos.Max[0]-0.06) > 1e-6 {
		t.Errorf("glTF max X = %v, want 0.06 m (metres regardless of document unit)", pos.Max[0])
	}
	stl, _, err := ExportBodies(types.FormatSTL, []*topo.Body{box}, types.ResolutionHigh, opts)
	if err != nil {
		t.Fatalf("STL export: %v", err)
	}
	if opts.FileUnit != "in" {
		t.Fatalf("caller's FileUnit mutated to %q after STL, want in", opts.FileUnit)
	}
	// STL is unitless — millimetre convention: 6 cm cube → 60 mm. The binary
	// STL stores 60.0 as little-endian float32 0x42700000.
	if !bytes.Contains(stl, []byte{0x00, 0x00, 0x70, 0x42}) {
		t.Errorf("STL missing a 60.0 float32 (0x42700000) — wrong scale")
	}
}

// TestGLBSizeOverflowGuard: payloads over the 32-bit GLB length ceiling are a
// typed error (R1-15).
func TestGLBSizeOverflowGuard(t *testing.T) {
	_, _, _, err := glbLayout(stdmath.MaxUint32, 0)
	if err == nil || !strings.Contains(err.Error(), "asset too large") {
		t.Fatalf("err = %v, want an asset-too-large error", err)
	}
	_, _, _, err = glbLayout(0, stdmath.MaxUint32)
	if err == nil || !strings.Contains(err.Error(), "asset too large") {
		t.Fatalf("err = %v, want an asset-too-large error", err)
	}
	if _, _, total, err := glbLayout(4, 4); err != nil || total != 12+8+4+8+4 {
		t.Fatalf("glbLayout(4,4) = %d, %v; want 36, nil", total, err)
	}
}

// TestGLTFNormalUnitLength: every emitted normal is unit length after packing
// (R2-15/R4-5).
func TestGLTFNormalUnitLength(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 2), math.V3(0, 0, 2), math.V3(0, 0, 2)}, // non-unit → normalized
		[]int{0, 1, 2},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "norm", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	for i := 0; i < len(body.normals); i += 3 {
		x, y, z := body.normals[i], body.normals[i+1], body.normals[i+2]
		l := stdmath.Sqrt(float64(x*x + y*y + z*z))
		if stdmath.Abs(l-1) > 1e-6 {
			t.Errorf("normal %d length = %v, want 1", i/3, l)
		}
	}
}

// TestGLTFZeroLengthNormalDropsTriangle: a zero-length normal drops its
// triangle (R4-5).
func TestGLTFZeroLengthNormalDropsTriangle(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2, 3, 4, 5},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "zero", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 3 {
		t.Errorf("indices = %v, want only the triangle with a valid normal", body.indices)
	}
}

// TestGLTFOverflowFloat32: a float64-finite position that overflows to Inf on
// float32 conversion drops its triangle (R4-3).
func TestGLTFOverflowFloat32(t *testing.T) {
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(1e41, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2, 3, 4, 5},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "overflow", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 3 {
		t.Errorf("indices = %v, want only the finite triangle", body.indices)
	}
}

// TestGLTFHugeNormalComponent: a normal component of magnitude ~3e38 is finite
// as a float32 but its float32 square-sum overflows to +Inf (CHG-4). The
// float64 length/normalize path keeps the triangle and writes a unit normal —
// no zero or Inf normal is emitted.
func TestGLTFHugeNormalComponent(t *testing.T) {
	huge := 3e38
	m := gltfTestMesh(
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(huge, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2, 3, 4, 5},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "huge", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 6 {
		t.Fatalf("indices = %v, want both triangles kept (float64 length is finite)", body.indices)
	}
	for i := 0; i < len(body.normals); i += 3 {
		x, y, z := body.normals[i], body.normals[i+1], body.normals[i+2]
		if stdmath.IsNaN(float64(x)) || stdmath.IsInf(float64(x), 0) ||
			stdmath.IsNaN(float64(y)) || stdmath.IsInf(float64(y), 0) ||
			stdmath.IsNaN(float64(z)) || stdmath.IsInf(float64(z), 0) {
			t.Fatalf("normal %d = [%v %v %v], want finite", i/3, x, y, z)
		}
		l := stdmath.Sqrt(float64(x)*float64(x) + float64(y)*float64(y) + float64(z)*float64(z))
		if stdmath.Abs(l-1) > 1e-6 {
			t.Errorf("normal %d length = %v, want 1 (unit)", i/3, l)
		}
	}
}

// TestGLTFHugeCollinearTriangleDropped: the CHG2-4 regression fixture —
// A=(-3e38,0,0), B=(3e38,0,0), C=(0,0,0) are finite float64 positions that
// pack to finite float32 values, but the float32 difference B−A overflows to
// +Inf. The post-pack degeneracy check computes the area cross-product in
// float64, so the collinear triangle has area 0 and is dropped; the old
// float32 path produced a NaN area (Inf×0) that read as non-degenerate and
// kept the triangle. No NaN may appear in any accessor.
func TestGLTFHugeCollinearTriangleDropped(t *testing.T) {
	huge := 3e38
	// Direct pin of the documented rule: the huge collinear triangle is
	// degenerate (area 0 in float64) and must be dropped.
	pos := []float32{float32(-huge), 0, 0, float32(huge), 0, 0, 0, 0, 0}
	if !degeneratePacked(pos, 0, 1, 2) {
		t.Fatal("degeneratePacked(huge collinear) = false, want true (dropped)")
	}
	// End-to-end: the huge collinear triangle plus a valid unit triangle.
	// Only the valid triangle survives; every emitted accessor value is finite.
	m := gltfTestMesh(
		[]math.Point3{
			math.P3(-huge, 0, 0), math.P3(huge, 0, 0), math.P3(0, 0, 0),
			math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0),
		},
		[]math.Vector3{
			math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1),
			math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1),
		},
		[]int{0, 1, 2, 3, 4, 5},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "hugecollinear", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if len(body.indices) != 3 {
		t.Fatalf("indices = %v, want only the valid triangle", body.indices)
	}
	for i := 0; i < len(body.positions); i++ {
		if stdmath.IsNaN(float64(body.positions[i])) || stdmath.IsInf(float64(body.positions[i]), 0) {
			t.Fatalf("position %d = %v, want finite", i, body.positions[i])
		}
	}
	for i := 0; i < len(body.normals); i++ {
		if stdmath.IsNaN(float64(body.normals[i])) || stdmath.IsInf(float64(body.normals[i]), 0) {
			t.Fatalf("normal %d = %v, want finite", i, body.normals[i])
		}
	}
	for _, c := range body.min {
		if stdmath.IsNaN(float64(c)) || stdmath.IsInf(float64(c), 0) {
			t.Fatalf("min component = %v, want finite", c)
		}
	}
	for _, c := range body.max {
		if stdmath.IsNaN(float64(c)) || stdmath.IsInf(float64(c), 0) {
			t.Fatalf("max component = %v, want finite", c)
		}
	}
}

// TestGLTFDeterministicOrdering: two exports of the same bodies produce
// byte-identical GLBs (R2-10).
func TestGLTFDeterministicOrdering(t *testing.T) {
	boxA := cmBox(t)
	boxB := cmBox(t)
	opts := exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM}
	first, _, _, err := ExportBodiesGLTF([]*topo.Body{boxA, boxB}, types.ResolutionHigh, opts)
	if err != nil {
		t.Fatalf("first export: %v", err)
	}
	second, _, _, err := ExportBodiesGLTF([]*topo.Body{boxA, boxB}, types.ResolutionHigh, opts)
	if err != nil {
		t.Fatalf("second export: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("two exports of the same bodies differ — ordering is not deterministic")
	}
}
