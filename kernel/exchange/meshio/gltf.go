// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// glTF 2.0 / GLB export (GLB container only in v1 — R1-2). The encoder is
// hand-rolled against the Khronos glTF 2.0 spec (registry.khronos.org/glTF/specs/2.0)
// with zero third-party dependencies, mirroring the STL/OBJ/3MF encoders. The
// invariants it holds are the §8 checklist of impl-inv-4-gltf-spec.md:
// asset.version "2.0"; exactly one scene whose nodes list every body node
// exactly once (R3-2); POSITION accessor min/max computed from the written
// float32s; no NaN/±Inf in buffer data (R4-3); indices uint16 up to 65535
// vertices else uint32, never the component max; one bufferView per accessor,
// 4-byte aligned; an explicit PBR material referenced by every primitive
// (R4-2); GLB chunkLength = PADDED payload length (R3-1); the Z-up→Y-up
// swizzle (x,y,z)→(x,z,−y) baked into POSITION and NORMAL; cm→m via
// opts.FileUnit "m" on a local copy (R3-3).

// gltfErr builds a glTF error naming the offending value (per CLAUDE.md).
func gltfErr(what, value string) error {
	if value == "" {
		return fmt.Errorf("gltf: %s", what)
	}
	return fmt.Errorf("gltf: %s %q", what, value)
}

// ExportBodiesGLTF tessellates the bodies at res and encodes them as a GLB —
// the per-body glTF path (one mesh per body, one primitive per mesh, indexed).
// It forces the file unit to metres (glTF 2.0 §3.4) on a LOCAL copy of opts so
// the caller's options are never mutated (R3-3). A zero TargetUnitMM (the
// zero-value options) defaults to the kernel's centimetre database unit
// (exchange.DBUnitMM, translate.go:78) so a zero-value call exports at the
// same scale as the model layer's explicit options — without the default the
// 6 cm cube would export as 0.006 m instead of 0.06 m (CHG3-2). Warnings
// report skipped empty bodies (R3-5/R3-6).
//
// Example:
//
//	data, tris, warns, err := meshio.ExportBodiesGLTF(bodies, types.ResolutionHigh, opts)
func ExportBodiesGLTF(bodies []*topo.Body, res types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, []string, error) {
	local := opts
	if local.TargetUnitMM == 0 {
		local.TargetUnitMM = exchange.DBUnitMM
	}
	local.FileUnit = "m"
	return encodeGLTFBodies(bodies, QualityFor(res), local)
}

// encodeGLTFBodies is the glTF export core: skip empty bodies with a warning
// (R2-8/R3-5), tessellate the rest once (R4-1), sanitize per body, build the
// JSON document + BIN buffer, and wrap the GLB container. The complete GLB is
// built in memory before any destination write (R4-9).
func encodeGLTFBodies(bodies []*topo.Body, q ops.Quality, opts exchange.TranslationOptions) ([]byte, int, []string, error) {
	warnings, exportable, err := exportableBodies(bodies)
	if err != nil {
		return nil, 0, nil, err
	}
	scale := opts.ExportScale()
	meshes, err := TessellateBodies(exportable, q)
	if err != nil {
		return nil, 0, nil, err
	}
	gb, sanitizeWarnings, err := sanitizeGLTFBodyAll(meshes, scale)
	if err != nil {
		return nil, 0, nil, err
	}
	warnings = append(warnings, sanitizeWarnings...)
	// A body with faces can still tessellate to nothing usable (every triangle
	// dropped as degenerate/non-finite). That is skipped with a warning like an
	// empty body, not a fatal error — but if it leaves no body at all to write,
	// the export has no geometry and errors, mirroring the all-empty case (CHG-8).
	if len(gb) == 0 {
		return nil, 0, nil, errors.New("gltf: no exportable bodies")
	}
	data, tris, err := encodeGLTFBytes(gb)
	if err != nil {
		return nil, 0, nil, err
	}
	return data, tris, warnings, nil
}

// encodeGLTFBytes builds the GLB container from sanitized bodies and totals triangles.
func encodeGLTFBytes(gb []*gltfBody) ([]byte, int, error) {
	jsonData, binData, err := buildGLTFDocument(gb)
	if err != nil {
		return nil, 0, err
	}
	data, err := wrapGLB(jsonData, binData)
	if err != nil {
		return nil, 0, err
	}
	tris := 0
	for _, b := range gb {
		tris += len(b.indices) / 3
	}
	return data, tris, nil
}

// exportableBodies validates the input slice (nil body -> typed error naming the
// index, change-review CHG4-2) and filters empty bodies, reporting each skip as a
// warning (R3-5/R3-6). An all-empty input is a typed error (R3-5).
func exportableBodies(bodies []*topo.Body) ([]string, []*topo.Body, error) {
	var warnings []string
	exportable := make([]*topo.Body, 0, len(bodies))
	for i, b := range bodies {
		if b == nil {
			return nil, nil, fmt.Errorf("gltf: body at index %d is nil", i)
		}
		if len(b.Faces()) == 0 {
			warnings = append(warnings, fmt.Sprintf("gltf: body %d is empty; skipped", b.ID()))
			continue
		}
		exportable = append(exportable, b)
	}
	if len(exportable) == 0 {
		return nil, nil, errors.New("gltf: no exportable bodies")
	}
	return warnings, exportable, nil
}

// errNoValidTriangles marks a body whose triangles were ALL dropped by
// sanitization (degenerate / non-finite geometry). It is a soft skip — the body
// is warned about and left out, exactly like an empty body — whereas a
// structural mesh-invariant violation (bad normal count, out-of-range index) is
// a hard error that fails the export (CHG-8).
var errNoValidTriangles = errors.New("no valid triangles")

// sanitizeGLTFBodyAll sanitizes each tessellated body record at the given scale.
// A body that yields no valid triangle is skipped with a warning (like an empty
// body), so one all-degenerate body cannot fail an otherwise valid multi-body
// export; any other sanitization failure is a hard error (CHG-8).
func sanitizeGLTFBodyAll(meshes []BodyMesh, scale float64) ([]*gltfBody, []string, error) {
	gb := make([]*gltfBody, 0, len(meshes))
	var warnings []string
	for _, bm := range meshes {
		body, err := sanitizeGLTFBody(bm, scale)
		if errors.Is(err, errNoValidTriangles) {
			warnings = append(warnings, fmt.Sprintf("gltf: body %s has no valid geometry; skipped", bm.ID))
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		gb = append(gb, body)
	}
	return gb, warnings, nil
}

// gltfBody is one body's sanitized, packed glTF data: swizzled/scaled float32
// positions, swizzled unit float32 normals, remapped indices, and the POSITION
// accessor bounds computed from the packed values.
type gltfBody struct {
	name      string
	positions []float32
	normals   []float32
	indices   []uint32
	min       [3]float32
	max       [3]float32
}

// sanitizeGLTFBody validates a body's mesh topology BEFORE any dereference
// (R3-4), packs every vertex through the final output domain — scale, Z-up→Y-up
// swizzle, float32 — checking finiteness there (R4-3), drops triangles with
// invalid or zero-length normals (R4-5), re-checks degeneracy on the packed
// positions (R4-4), prunes unreferenced vertices, and computes the POSITION
// bounds from the surviving packed values.
func sanitizeGLTFBody(bm BodyMesh, scale float64) (*gltfBody, error) {
	m := bm.Mesh
	if err := validateGLTFMesh(bm.ID, m); err != nil {
		return nil, err
	}
	packedPos, packedNorm, valid := packGLTFVertices(m, scale)
	kept := filterGLTFTriangles(m, packedPos, packedNorm, valid)
	if len(kept) == 0 {
		return nil, fmt.Errorf("gltf: body %s has %w", bm.ID, errNoValidTriangles)
	}
	pos, norm, idx := pruneGLTFVertices(packedPos, packedNorm, kept)
	min, max := gltfBounds(pos)
	return &gltfBody{name: bm.Name, positions: pos, normals: norm, indices: idx, min: min, max: max}, nil
}

// validateGLTFMesh checks the mesh's grouping and bounds before any dereference
// (R3-4): positions are VEC3 triples, normals match positions, indices are
// triangle triples, and every index is in range. Violations are typed
// body-specific errors naming the offending value.
func validateGLTFMesh(id string, m *ops.Mesh) error {
	if len(m.Normals) != len(m.Positions) {
		return gltfErr("body "+id+" normal count", fmt.Sprintf("%d != position count %d", len(m.Normals), len(m.Positions)))
	}
	if len(m.Indices)%3 != 0 {
		return gltfErr("body "+id+" index count", fmt.Sprintf("%d (not a multiple of 3)", len(m.Indices)))
	}
	vertCount := len(m.Positions)
	for _, idx := range m.Indices {
		if idx < 0 || idx >= vertCount {
			return gltfErr("body "+id+" index", fmt.Sprintf("%d out of range [0,%d)", idx, vertCount))
		}
	}
	return nil
}

// packGLTFVertices transforms every vertex into the final output domain: the
// position is scaled (cm→m) and swizzled (x,y,z)→(x,z,−y), the normal is
// swizzled only, both narrowed to float32. A vertex is valid when every packed
// component is finite — including float64-finite values that overflow to Inf on
// float32 conversion (R4-3).
func packGLTFVertices(m *ops.Mesh, scale float64) (pos, norm []float32, valid []bool) {
	vertCount := len(m.Positions)
	pos = make([]float32, 3*vertCount)
	norm = make([]float32, 3*vertCount)
	valid = make([]bool, vertCount)
	for i := 0; i < vertCount; i++ {
		p := m.Positions[i]
		n := m.Normals[i]
		px, py, pz := float32(p.X*scale), float32(p.Z*scale), float32(-p.Y*scale)
		nx, ny, nz := float32(n.X), float32(n.Z), float32(-n.Y)
		pos[3*i], pos[3*i+1], pos[3*i+2] = px, py, pz
		norm[3*i], norm[3*i+1], norm[3*i+2] = nx, ny, nz
		valid[i] = finite32(px) && finite32(py) && finite32(pz) &&
			finite32(nx) && finite32(ny) && finite32(nz)
	}
	return pos, norm, valid
}

// finite32 reports whether v is neither NaN nor ±Inf.
func finite32(v float32) bool {
	return !stdmath.IsNaN(float64(v)) && !stdmath.IsInf(float64(v), 0)
}

// filterGLTFTriangles keeps the triangles whose three vertices are valid, whose
// indices are distinct, whose normals are non-zero (R4-5), and whose packed
// positions are non-degenerate (R4-4). The kept indices are returned in
// original order.
func filterGLTFTriangles(m *ops.Mesh, pos, norm []float32, valid []bool) []uint32 {
	kept := make([]uint32, 0, len(m.Indices))
	for t := 0; t+2 < len(m.Indices); t += 3 {
		i, j, k := m.Indices[t], m.Indices[t+1], m.Indices[t+2]
		if !valid[i] || !valid[j] || !valid[k] {
			continue
		}
		if i == j || j == k || k == i {
			continue
		}
		if !normalOK(norm, i) || !normalOK(norm, j) || !normalOK(norm, k) {
			continue
		}
		if degeneratePacked(pos, i, j, k) {
			continue
		}
		kept = append(kept, uint32(i), uint32(j), uint32(k))
	}
	return kept
}

// normalOK reports whether the packed normal at vertex vi is usable: its
// length is computed in float64 (a finite float32 component like 3e38 squares
// to +Inf in float32, which would wrongly read as a zero/Inf length —
// change-review CHG-4), the length is finite and non-zero, and the float32
// normalization of that length is finite and unit-length within 1e-6.
func normalOK(norm []float32, vi int) bool {
	x, y, z := float64(norm[3*vi]), float64(norm[3*vi+1]), float64(norm[3*vi+2])
	l := stdmath.Sqrt(x*x + y*y + z*z)
	if stdmath.IsNaN(l) || stdmath.IsInf(l, 0) || l == 0 {
		return false
	}
	ux, uy, uz := float32(x/l), float32(y/l), float32(z/l)
	if !finite32(ux) || !finite32(uy) || !finite32(uz) {
		return false
	}
	// Re-validate the NARROWED unit vector: float32 rounding can push a
	// normalized component off unit length (CHG-4).
	rl := stdmath.Sqrt(float64(ux)*float64(ux) + float64(uy)*float64(uy) + float64(uz)*float64(uz))
	return stdmath.Abs(rl-1) <= 1e-6
}

// degeneratePacked reports whether the triangle's packed float32 positions are
// degenerate: duplicate positions or geometric area at or below the 1e-12 m²
// epsilon (R4-4 — the check runs again AFTER float32 quantization). The
// difference and cross-product are computed in float64 (CHG2-4): a finite
// float32 component like 3e38 overflows the float32 square-sum to +Inf, which
// would wrongly read as an infinite area. A NON-FINITE area from finite
// inputs is treated as invalid — the triangle cannot be certified
// non-degenerate, so it is dropped (documented rule, CHG2-4).
func degeneratePacked(pos []float32, i, j, k int) bool {
	ax, ay, az := float64(pos[3*i]), float64(pos[3*i+1]), float64(pos[3*i+2])
	bx, by, bz := float64(pos[3*j]), float64(pos[3*j+1]), float64(pos[3*j+2])
	cx, cy, cz := float64(pos[3*k]), float64(pos[3*k+1]), float64(pos[3*k+2])
	if (ax == bx && ay == by && az == bz) ||
		(bx == cx && by == cy && bz == cz) ||
		(cx == ax && cy == ay && cz == az) {
		return true
	}
	ux, uy, uz := bx-ax, by-ay, bz-az
	vx, vy, vz := cx-ax, cy-ay, cz-az
	cx2, cy2, cz2 := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	area := 0.5 * stdmath.Sqrt(cx2*cx2+cy2*cy2+cz2*cz2)
	if stdmath.IsNaN(area) || stdmath.IsInf(area, 0) {
		return true
	}
	return area <= 1e-12
}

// pruneGLTFVertices compacts the packed arrays to the referenced vertices,
// remapping indices, and normalizes non-unit normals in the written data (R4-5;
// kernel normals are unit by construction, so this is defense-in-depth).
func pruneGLTFVertices(pos, norm []float32, kept []uint32) (outPos, outNorm []float32, outIdx []uint32) {
	remap := make(map[int]uint32, len(pos)/3)
	outPos = make([]float32, 0, len(kept))
	outNorm = make([]float32, 0, len(kept))
	outIdx = make([]uint32, 0, len(kept))
	for _, idx := range kept {
		vi := int(idx)
		ni, ok := remap[vi]
		if !ok {
			ni = uint32(len(outPos) / 3)
			remap[vi] = ni
			outPos = append(outPos, pos[3*vi], pos[3*vi+1], pos[3*vi+2])
			u := normalize3(norm[3*vi], norm[3*vi+1], norm[3*vi+2])
			outNorm = append(outNorm, u[0], u[1], u[2])
		}
		outIdx = append(outIdx, ni)
	}
	return outPos, outNorm, outIdx
}

// normalize3 returns the unit vector of (x,y,z), or the zero vector when the
// length is zero (callers drop zero-length normals before normalizing). The
// length is computed in float64 so a finite float32 component that overflows
// the float32 square-sum (e.g. 3e38) still normalizes correctly (CHG-4).
func normalize3(x, y, z float32) [3]float32 {
	l := stdmath.Sqrt(float64(x)*float64(x) + float64(y)*float64(y) + float64(z)*float64(z))
	if l == 0 {
		return [3]float32{}
	}
	return [3]float32{float32(float64(x) / l), float32(float64(y) / l), float32(float64(z) / l)}
}

// gltfBounds returns the per-component min and max of the packed positions.
func gltfBounds(pos []float32) (min, max [3]float32) {
	min = [3]float32{pos[0], pos[1], pos[2]}
	max = min
	for i := 3; i < len(pos); i += 3 {
		for c := 0; c < 3; c++ {
			if pos[i+c] < min[c] {
				min[c] = pos[i+c]
			}
			if pos[i+c] > max[c] {
				max[c] = pos[i+c]
			}
		}
	}
	return min, max
}

// GLB container constants (glTF 2.0 §4.4).
const (
	glbMagic   = 0x46546C67 // "glTF"
	glbVersion = 2
	glbJSON    = 0x4E4F534A // "JSON"
	glbBIN     = 0x004E4942 // "BIN\0"
)

// wrapGLB assembles the GLB container: 12-byte header, JSON chunk padded with
// 0x20, BIN chunk padded with 0x00. The chunkLength fields carry the PADDED
// payload length (R3-1 — Blender's importer advances by chunkLength with no
// alignment rounding); buffer.byteLength (set in the JSON) stays the unpadded
// BIN length. The header length is the exact file size.
func wrapGLB(jsonData, binData []byte) ([]byte, error) {
	jsonPadded, binPadded, total, err := glbLayout(len(jsonData), len(binData))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Grow(total)
	writeUint32(&buf, glbMagic)
	writeUint32(&buf, glbVersion)
	writeUint32(&buf, uint32(total))
	writeUint32(&buf, uint32(jsonPadded))
	writeUint32(&buf, glbJSON)
	buf.Write(padGLB(jsonData, 0x20))
	writeUint32(&buf, uint32(binPadded))
	writeUint32(&buf, glbBIN)
	buf.Write(padGLB(binData, 0x00))
	return buf.Bytes(), nil
}

// glbLayout computes the padded chunk lengths and total GLB size, guarding the
// 32-bit GLB length fields (R1-15): any payload or total over math.MaxUint32 is
// a typed "asset too large" error.
func glbLayout(jsonLen, binLen int) (jsonPadded, binPadded, total int, err error) {
	jsonPadded = (jsonLen + 3) &^ 3
	binPadded = (binLen + 3) &^ 3
	total = 12 + 8 + jsonPadded + 8 + binPadded
	if jsonPadded > stdmath.MaxUint32 || binPadded > stdmath.MaxUint32 || total > stdmath.MaxUint32 {
		return 0, 0, 0, gltfErr("asset too large", fmt.Sprintf("%d bytes", total))
	}
	return jsonPadded, binPadded, total, nil
}

// padGLB returns data padded with pad to a 4-byte boundary (a no-op when
// already aligned).
func padGLB(data []byte, pad byte) []byte {
	if len(data)%4 == 0 {
		return data
	}
	out := make([]byte, len(data)+(4-len(data)%4))
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = pad
	}
	return out
}
