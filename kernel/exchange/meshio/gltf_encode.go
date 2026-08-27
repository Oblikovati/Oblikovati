// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	stdmath "math"
)

// glTF JSON document + BIN buffer construction. The document is built with
// ordered structs (R2-10 deterministic ordering) and marshaled with
// encoding/json — UTF-8 without BOM (§2.7; Go escapes non-ASCII, which is
// spec-legal). The BIN buffer is laid out per body: POSITION (float32 VEC3),
// NORMAL (float32 VEC3), then indices (uint16 or uint32), each in its own
// 4-byte-aligned bufferView (one per accessor, no byteStride — §3.6.2.4).

// glTF accessor/bufferView component constants (glTF 2.0 §5.1, §5.11).
const (
	gltfFloat        = 5126
	gltfUint16       = 5123
	gltfUint32       = 5125
	gltfArrayBuffer  = 34962
	gltfElementArray = 34963
)

// gltfJSONDoc is the ordered glTF 2.0 document (asset, scene, nodes, meshes,
// materials, accessors, bufferViews, buffers). Field order is fixed by the
// struct so the JSON is deterministic (R2-10).
type gltfJSONDoc struct {
	Asset       gltfAsset        `json:"asset"`
	Scene       int              `json:"scene"`
	Scenes      []gltfScene      `json:"scenes"`
	Nodes       []gltfNode       `json:"nodes"`
	Meshes      []gltfMesh       `json:"meshes"`
	Materials   []gltfMaterial   `json:"materials"`
	Accessors   []gltfAccessor   `json:"accessors"`
	BufferViews []gltfBufferView `json:"bufferViews"`
	Buffers     []gltfBuffer     `json:"buffers"`
}

type gltfAsset struct {
	Version   string `json:"version"`
	Generator string `json:"generator"`
}

type gltfScene struct {
	Nodes []int `json:"nodes"`
}

type gltfNode struct {
	Name string `json:"name,omitempty"`
	// Mesh must NOT be omitempty: node 0 references mesh 0, and omitempty would
	// drop the reference for the first body (a node with no mesh is invalid for
	// a body node).
	Mesh int `json:"mesh"`
}

type gltfMesh struct {
	Name       string          `json:"name,omitempty"`
	Primitives []gltfPrimitive `json:"primitives"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    int            `json:"indices"`
	Material   int            `json:"material"`
	Mode       int            `json:"mode"`
}

type gltfMaterial struct {
	Name                 string  `json:"name"`
	DoubleSided          bool    `json:"doubleSided"`
	PBRMetallicRoughness gltfPBR `json:"pbrMetallicRoughness"`
}

type gltfPBR struct {
	BaseColorFactor [4]float64 `json:"baseColorFactor"`
	MetallicFactor  float64    `json:"metallicFactor"`
	RoughnessFactor float64    `json:"roughnessFactor"`
}

type gltfAccessor struct {
	BufferView    int       `json:"bufferView"`
	ComponentType int       `json:"componentType"`
	Count         int       `json:"count"`
	Type          string    `json:"type"`
	Min           []float64 `json:"min,omitempty"`
	Max           []float64 `json:"max,omitempty"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	Target     int `json:"target,omitempty"`
}

type gltfBuffer struct {
	ByteLength int `json:"byteLength"`
}

// buildGLTFDocument assembles the JSON document and the BIN buffer for the
// sanitized bodies. The BIN layout is deterministic: for each body in input
// order, POSITION then NORMAL then indices, each bufferView 4-byte aligned.
// buffer.byteLength is the unpadded BIN data length (R3-1).
func buildGLTFDocument(bodies []*gltfBody) ([]byte, []byte, error) {
	doc := gltfJSONDoc{
		Asset:     gltfAsset{Version: "2.0", Generator: "Oblikovati"},
		Scene:     0,
		Scenes:    []gltfScene{{Nodes: make([]int, len(bodies))}},
		Materials: []gltfMaterial{gltfCADMaterial()},
	}
	names := allocateGLTFNames(bodyNames(bodies))
	var bin bytes.Buffer
	for i, b := range bodies {
		doc.Scenes[0].Nodes[i] = i
		doc.Nodes = append(doc.Nodes, gltfNode{Name: names[i], Mesh: i})
		doc.Meshes = append(doc.Meshes, gltfMesh{Name: names[i], Primitives: []gltfPrimitive{{
			Attributes: map[string]int{"POSITION": len(doc.Accessors), "NORMAL": len(doc.Accessors) + 1},
			Indices:    len(doc.Accessors) + 2,
			Material:   0,
			Mode:       4,
		}}})
		posView := appendGLTFBufferView(&bin, &doc, gltfArrayBuffer)
		bin.Write(f32Bytes(b.positions))
		doc.BufferViews[posView].ByteLength = len(b.positions) * 4
		doc.Accessors = append(doc.Accessors, gltfAccessor{
			BufferView: posView, ComponentType: gltfFloat, Count: len(b.positions) / 3,
			Type: "VEC3", Min: f32s(b.min[:]), Max: f32s(b.max[:]),
		})
		normView := appendGLTFBufferView(&bin, &doc, gltfArrayBuffer)
		bin.Write(f32Bytes(b.normals))
		doc.BufferViews[normView].ByteLength = len(b.normals) * 4
		doc.Accessors = append(doc.Accessors, gltfAccessor{
			BufferView: normView, ComponentType: gltfFloat, Count: len(b.normals) / 3, Type: "VEC3",
		})
		idxView := appendGLTFBufferView(&bin, &doc, gltfElementArray)
		idxType, idxBytes := gltfIndexType(len(b.positions) / 3)
		writeGLTFIndices(&bin, b.indices, idxBytes)
		doc.BufferViews[idxView].ByteLength = len(b.indices) * idxBytes
		doc.Accessors = append(doc.Accessors, gltfAccessor{
			BufferView: idxView, ComponentType: idxType, Count: len(b.indices), Type: "SCALAR",
		})
	}
	doc.Buffers = []gltfBuffer{{ByteLength: bin.Len()}}
	jsonData, err := json.Marshal(&doc)
	if err != nil {
		return nil, nil, fmt.Errorf("gltf: marshal document: %w", err)
	}
	return jsonData, bin.Bytes(), nil
}

// gltfCADMaterial is the single explicit PBR material every primitive
// references (R4-2): neutral grey, metallic 0 (the spec default is 1 —
// chrome), roughness 1, double-sided false (closed B-rep bodies). The
// doubleSided field is emitted EXPLICITLY (no omitempty) so the JSON always
// carries "doubleSided": false — a reader must not guess the spec default
// (CHG3-4).
func gltfCADMaterial() gltfMaterial {
	return gltfMaterial{
		Name:        "CAD_Default",
		DoubleSided: false,
		PBRMetallicRoughness: gltfPBR{
			BaseColorFactor: [4]float64{0.8, 0.8, 0.8, 1.0},
			MetallicFactor:  0.0,
			RoughnessFactor: 1.0,
		},
	}
}

// bodyNames extracts the display names in body order.
func bodyNames(bodies []*gltfBody) []string {
	out := make([]string, len(bodies))
	for i, b := range bodies {
		out[i] = b.name
	}
	return out
}

// allocateGLTFNames makes the exported node/mesh names unique with
// deterministic next-available numbering against the union of original AND
// generated names (R3-11): A, A → A, A#2; a pre-existing A#2 then yields
// A#2#2. Input order is preserved.
func allocateGLTFNames(names []string) []string {
	used := make(map[string]bool, len(names))
	out := make([]string, len(names))
	for i, n := range names {
		candidate := n
		for used[candidate] {
			candidate = candidate + "#2"
		}
		used[candidate] = true
		out[i] = candidate
	}
	return out
}

// appendGLTFBufferView aligns the BIN buffer to 4 bytes, records a bufferView
// for the next payload, and returns its index. The caller writes the payload
// and then sets the bufferView's ByteLength.
func appendGLTFBufferView(bin *bytes.Buffer, doc *gltfJSONDoc, target int) int {
	for bin.Len()%4 != 0 {
		bin.WriteByte(0)
	}
	doc.BufferViews = append(doc.BufferViews, gltfBufferView{Buffer: 0, ByteOffset: bin.Len(), Target: target})
	return len(doc.BufferViews) - 1
}

// gltfIndexType picks the index component type: uint16 up to 65535 vertices,
// else uint32 — never the component max (primitive restart, §3.7.2.1).
func gltfIndexType(vertCount int) (componentType, bytesPerIndex int) {
	if vertCount <= 65535 {
		return gltfUint16, 2
	}
	return gltfUint32, 4
}

// writeGLTFIndices appends the indices as little-endian uint16 or uint32.
func writeGLTFIndices(bin *bytes.Buffer, indices []uint32, bytesPerIndex int) {
	if bytesPerIndex == 2 {
		for _, idx := range indices {
			var b [2]byte
			binary.LittleEndian.PutUint16(b[:], uint16(idx))
			bin.Write(b[:])
		}
		return
	}
	for _, idx := range indices {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], idx)
		bin.Write(b[:])
	}
}

// f32Bytes returns the little-endian float32 byte encoding of v.
func f32Bytes(v []float32) []byte {
	out := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(out[4*i:], stdmath.Float32bits(f))
	}
	return out
}

// f32s converts packed float32 bounds to the JSON float64 representation.
func f32s(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}
