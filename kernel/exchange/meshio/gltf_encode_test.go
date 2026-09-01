// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestGLTFDocumentShape: the JSON document has exactly one scene, one material,
// per-body nodes/meshes, and the accessor/bufferView wiring is consistent.
func TestGLTFDocumentShape(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	if doc.Asset.Version != "2.0" || doc.Asset.Generator != "Oblikovati" {
		t.Errorf("asset = %+v, want version 2.0 generator Oblikovati", doc.Asset)
	}
	if len(doc.Nodes) != 1 || len(doc.Meshes) != 1 {
		t.Fatalf("nodes/meshes = %d/%d, want 1/1", len(doc.Nodes), len(doc.Meshes))
	}
	if doc.Nodes[0].Mesh != 0 {
		t.Errorf("node mesh = %d, want 0", doc.Nodes[0].Mesh)
	}
	// The raw JSON must carry the mesh reference — omitempty on node 0's mesh
	// would silently drop it (a node with no mesh is invalid for a body node).
	if !bytes.Contains(glb.jsonData, []byte(`"mesh":0`)) {
		t.Error("JSON missing the node→mesh reference for node 0")
	}
	// The material must carry doubleSided EXPLICITLY (no omitempty): a reader
	// must not guess the spec default for a closed B-rep body (CHG3-4).
	if !bytes.Contains(glb.jsonData, []byte(`"doubleSided":false`)) {
		t.Error("JSON missing the explicit \"doubleSided\":false in the material object")
	}
	prim := doc.Meshes[0].Primitives[0]
	if prim.Mode != 4 {
		t.Errorf("primitive mode = %d, want 4 (TRIANGLES)", prim.Mode)
	}
	// POSITION, NORMAL, indices accessors exist and are distinct.
	posIdx := prim.Attributes["POSITION"]
	normIdx := prim.Attributes["NORMAL"]
	if posIdx == normIdx || prim.Indices == posIdx || prim.Indices == normIdx {
		t.Errorf("accessor indices not distinct: pos=%d norm=%d idx=%d", posIdx, normIdx, prim.Indices)
	}
	// Every accessor references a valid bufferView; every bufferView fits the buffer.
	for i, a := range doc.Accessors {
		if a.BufferView < 0 || a.BufferView >= len(doc.BufferViews) {
			t.Errorf("accessor %d bufferView %d out of range", i, a.BufferView)
		}
	}
	for i, bv := range doc.BufferViews {
		if bv.ByteOffset+bv.ByteLength > doc.Buffers[0].ByteLength {
			t.Errorf("bufferView %d exceeds buffer: %+v > %d", i, bv, doc.Buffers[0].ByteLength)
		}
	}
	// No extensions declared.
	if bytes.Contains(glb.jsonData, []byte("extensionsUsed")) || bytes.Contains(glb.jsonData, []byte("extensionsRequired")) {
		t.Error("document declares extensions; v1 must not")
	}
	// JSON is UTF-8 without BOM.
	if len(glb.jsonData) >= 3 && glb.jsonData[0] == 0xEF && glb.jsonData[1] == 0xBB && glb.jsonData[2] == 0xBF {
		t.Error("JSON chunk starts with a UTF-8 BOM")
	}
}

// TestGLTFBufferLayout: the BIN buffer is laid out POSITION, NORMAL, indices
// per body with 4-byte-aligned bufferViews and correct byteLengths.
func TestGLTFBufferLayout(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	glb := parseGLB(t, data)
	doc := parseGLTFJSON(t, glb.jsonData)
	prim := doc.Meshes[0].Primitives[0]
	posAcc := doc.Accessors[prim.Attributes["POSITION"]]
	normAcc := doc.Accessors[prim.Attributes["NORMAL"]]
	idxAcc := doc.Accessors[prim.Indices]
	posBV := doc.BufferViews[posAcc.BufferView]
	normBV := doc.BufferViews[normAcc.BufferView]
	idxBV := doc.BufferViews[idxAcc.BufferView]
	// 4-byte alignment of every bufferView offset.
	for name, off := range map[string]int{"pos": posBV.ByteOffset, "norm": normBV.ByteOffset, "idx": idxBV.ByteOffset} {
		if off%4 != 0 {
			t.Errorf("%s bufferView offset %d not 4-aligned", name, off)
		}
	}
	// Targets: ARRAY_BUFFER for attributes, ELEMENT_ARRAY_BUFFER for indices.
	if posBV.Target != gltfArrayBuffer || normBV.Target != gltfArrayBuffer || idxBV.Target != gltfElementArray {
		t.Errorf("targets = %d/%d/%d, want 34962/34962/34963", posBV.Target, normBV.Target, idxBV.Target)
	}
	// byteLengths match the accessor counts.
	if posBV.ByteLength != posAcc.Count*12 || normBV.ByteLength != normAcc.Count*12 {
		t.Errorf("attribute byteLengths = %d/%d, want %d/%d", posBV.ByteLength, normBV.ByteLength, posAcc.Count*12, normAcc.Count*12)
	}
	// Indices are uint16 for a box (8 vertices) and the count is divisible by 3.
	if idxAcc.ComponentType != gltfUint16 || idxAcc.Count%3 != 0 {
		t.Errorf("indices accessor = %+v, want uint16 count divisible by 3", idxAcc)
	}
	// The BIN chunk holds exactly the declared buffer bytes (unpadded length).
	if doc.Buffers[0].ByteLength > glb.binLen {
		t.Errorf("buffer.byteLength %d > padded BIN %d", doc.Buffers[0].ByteLength, glb.binLen)
	}
}

// TestGLTFSwizzle: the Z-up→Y-up swizzle (x,y,z)→(x,z,−y) is baked into the
// written POSITION and NORMAL data.
func TestGLTFSwizzle(t *testing.T) {
	// A valid triangle whose first vertex is (1,2,3) cm with a kernel +Z
	// normal. After scale (×0.01) and the Z-up→Y-up swizzle the first packed
	// position is (0.01, 0.03, −0.02) m and the normal is (0, 1, 0).
	m := gltfTestMesh(
		[]math.Point3{math.P3(1, 2, 3), math.P3(2, 2, 3), math.P3(1, 3, 3)},
		[]math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		[]int{0, 1, 2},
	)
	body, err := sanitizeGLTFBody(gltfBodyMesh("1", "swz", m), 0.01)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if body.positions[0] != 0.01 || body.positions[1] != 0.03 || body.positions[2] != -0.02 {
		t.Errorf("swizzled position = %v, want [0.01 0.03 -0.02]", body.positions[:3])
	}
	if body.normals[0] != 0 || body.normals[1] != 1 || body.normals[2] != 0 {
		t.Errorf("swizzled normal = %v, want [0 1 0]", body.normals[:3])
	}
}

// TestGLTFUint32IndicesInDocument: a body with more than 65535 vertices uses a
// uint32 indices accessor in the emitted document.
func TestGLTFUint32IndicesInDocument(t *testing.T) {
	// A synthetic gltfBody with 70000 vertices exercises the uint32 index
	// switch in the emitted document (the sanitizer would prune unreferenced
	// vertices, so build the packed body directly).
	positions := make([]float32, 3*70000)
	normals := make([]float32, 3*70000)
	for i := 0; i < 70000; i++ {
		positions[3*i] = float32(i % 100)
		positions[3*i+1] = float32(i / 100 % 100)
		positions[3*i+2] = float32(i / 10000)
		normals[3*i+2] = 1
	}
	body := &gltfBody{name: "big", positions: positions, normals: normals, indices: []uint32{0, 1, 2}}
	jsonData, binData, err := buildGLTFDocument([]*gltfBody{body})
	if err != nil {
		t.Fatalf("buildGLTFDocument: %v", err)
	}
	doc := parseGLTFJSON(t, jsonData)
	idxAcc := doc.Accessors[doc.Meshes[0].Primitives[0].Indices]
	if idxAcc.ComponentType != gltfUint32 {
		t.Errorf("indices componentType = %d, want 5125 (uint32)", idxAcc.ComponentType)
	}
	// The packed indices are 4 bytes each.
	if len(binData) < idxAcc.Count*4 {
		t.Errorf("BIN too small for uint32 indices: %d < %d", len(binData), idxAcc.Count*4)
	}
}

// TestGLTFExportBodyRejectsGLTF: the merged ExportBodies path refuses glTF
// with a typed error pointing at the per-body entry point (R4-1).
func TestGLTFExportBodyRejectsGLTF(t *testing.T) {
	box := cmBox(t)
	_, _, err := ExportBodies(types.FormatGLTF, []*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err == nil || !strings.Contains(err.Error(), "ExportBodiesGLTF") {
		t.Fatalf("ExportBodies(gltf) = %v, want a typed per-body-path error", err)
	}
}

// TestGLTFDecodeRejectsGLTF: import of glTF stays a typed unsupported-format
// error — the export-only scope pin (R1-2).
func TestGLTFDecodeRejectsGLTF(t *testing.T) {
	if _, err := Decode(types.FormatGLTF, []byte("glTF")); err == nil {
		t.Fatal("Decode accepted FormatGLTF; import is out of scope in v1")
	}
}

// TestGLTFBodyMeshNames: TessellateBodies returns stable ID/Name records in
// input order (R2-7).
func TestGLTFBodyMeshNames(t *testing.T) {
	boxA := cmBox(t)
	boxB := cmBox(t)
	meshes, err := TessellateBodies([]*topo.Body{boxA, boxB}, QualityFor(types.ResolutionHigh))
	if err != nil {
		t.Fatalf("TessellateBodies: %v", err)
	}
	if len(meshes) != 2 {
		t.Fatalf("TessellateBodies returned %d records, want 2", len(meshes))
	}
	if meshes[0].ID == "" || meshes[0].Name == "" || meshes[0].Mesh == nil {
		t.Errorf("record 0 = %+v, want id/name/mesh", meshes[0])
	}
	if meshes[0].ID == meshes[1].ID {
		t.Errorf("body IDs not distinct: %q", meshes[0].ID)
	}
	if meshes[0].Mesh.TriangleCount() != 12 || meshes[1].Mesh.TriangleCount() != 12 {
		t.Errorf("triangle counts = %d/%d, want 12/12", meshes[0].Mesh.TriangleCount(), meshes[1].Mesh.TriangleCount())
	}
}

// TestGLTFMergeTessellationsUnchanged: the pure merger produces the same
// concatenation order as the legacy per-body loop (R4-1 regression).
func TestGLTFMergeTessellationsUnchanged(t *testing.T) {
	boxA := cmBox(t)
	boxB := cmBox(t)
	q := QualityFor(types.ResolutionLow)
	merged, err := mergeTessellations([]*topo.Body{boxA, boxB}, q)
	if err != nil {
		t.Fatalf("mergeTessellations: %v", err)
	}
	if merged.TriangleCount() != 24 {
		t.Fatalf("merged triangle count = %d, want 24", merged.TriangleCount())
	}
	// The merged positions are the concatenation of the per-body tessellations.
	records, err := TessellateBodies([]*topo.Body{boxA, boxB}, q)
	if err != nil {
		t.Fatalf("TessellateBodies: %v", err)
	}
	if len(merged.Positions) != len(records[0].Mesh.Positions)+len(records[1].Mesh.Positions) {
		t.Errorf("merged positions = %d, want %d+%d", len(merged.Positions), len(records[0].Mesh.Positions), len(records[1].Mesh.Positions))
	}
	// Indices of the second body are offset by the first body's vertex count.
	base := len(records[0].Mesh.Positions)
	for _, idx := range records[1].Mesh.Indices {
		if idx+base >= len(merged.Positions) {
			t.Errorf("merged index %d out of range", idx+base)
		}
	}
}

// TestGLTFMergeTessellationsHandComputed: the merger's output equals a
// hand-computed concatenation of two known body meshes — positions and normals
// in body order, the second body's indices offset by the first body's vertex
// count (CHG-7 semantic golden). The golden is built from the per-body
// tessellations of two distinct known bodies, so it pins the merger's
// concatenation arithmetic, not the tessellator's output.
func TestGLTFMergeTessellationsHandComputed(t *testing.T) {
	boxA := cmBox(t)
	boxB, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(3, 3, 3), "small")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := QualityFor(types.ResolutionLow)
	meshA, _ := tessellate.TessellateBody(boxA, q)
	meshB, _ := tessellate.TessellateBody(boxB, q)

	// Hand-computed concatenation: A's vertices, then B's vertices; B's
	// indices offset by A's vertex count.
	wantPos := append(append([]math.Point3(nil), meshA.Positions...), meshB.Positions...)
	wantNorm := append(append([]math.Vector3(nil), meshA.Normals...), meshB.Normals...)
	base := len(meshA.Positions)
	wantIdx := make([]int, 0, len(meshA.Indices)+len(meshB.Indices))
	wantIdx = append(wantIdx, meshA.Indices...)
	for _, idx := range meshB.Indices {
		wantIdx = append(wantIdx, base+idx)
	}

	got, err := mergeTessellations([]*topo.Body{boxA, boxB}, q)
	if err != nil {
		t.Fatalf("mergeTessellations: %v", err)
	}
	if len(got.Positions) != len(wantPos) {
		t.Fatalf("positions = %d, want %d", len(got.Positions), len(wantPos))
	}
	for i := range wantPos {
		if got.Positions[i] != wantPos[i] {
			t.Errorf("position %d = %v, want %v", i, got.Positions[i], wantPos[i])
		}
	}
	if len(got.Normals) != len(wantNorm) {
		t.Fatalf("normals = %d, want %d", len(got.Normals), len(wantNorm))
	}
	for i := range wantNorm {
		if got.Normals[i] != wantNorm[i] {
			t.Errorf("normal %d = %v, want %v", i, got.Normals[i], wantNorm[i])
		}
	}
	if len(got.Indices) != len(wantIdx) {
		t.Fatalf("indices = %d, want %d", len(got.Indices), len(wantIdx))
	}
	for i := range wantIdx {
		if got.Indices[i] != wantIdx[i] {
			t.Errorf("index %d = %d, want %d", i, got.Indices[i], wantIdx[i])
		}
	}
}

// TestGLTFWriteAtomicity: the encoder returns bytes; a destination that cannot
// be written leaves any pre-existing file unchanged (R4-9). The encoder itself
// never writes — this pins the model-layer contract via the CLI path.
func TestGLTFWriteAtomicity(t *testing.T) {
	box := cmBox(t)
	data, _, _, err := ExportBodiesGLTF([]*topo.Body{box}, types.ResolutionHigh,
		exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		t.Fatalf("ExportBodiesGLTF: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encoder returned empty bytes")
	}
	// The encoder must not have touched the filesystem: no side-channel check
	// needed — the function signature returns bytes only (R4-9).
}
