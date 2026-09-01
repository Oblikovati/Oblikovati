// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Canonical mesh fingerprint (CN6 Minor #2) — a shared, ORDER-INDEPENDENT commutative fingerprint of a
// welded body's Property-quality tessellation plus its volume, for do-no-harm / byte-identity checks across
// worktrees. It combines each triangle's canonical (sorted, quantized) vertex hash by unsigned ADDITION
// (commutative, order-independent, and — unlike XOR — duplicate triangles do not cancel), so face/triangle
// EMISSION ORDER cannot change the fingerprint while a real geometry change does. A CN-C8 change that only
// adds test files must leave every prior-green body bit-identical here; this test pins the fingerprints so a
// future shared-code edit that silently perturbs a green case fails loud. It is deliberately quantized to
// 1e-6·(model scale) so it is robust to last-bit FP noise yet catches any real vertex move.
//
// SCOPE — winding is OUT of fingerprint scope. The full 3-vertex sort makes the hash invariant to a
// triangle's WINDING (and to reflection), and the paired volume is |vol| (massprops.go re-orients each
// facet topologically via consistentOutwardFlips, #1318), so NEITHER the hash NOR the volume detects a
// UNIFORM triangle-winding flip — that is intentional: this kernel does not contract on tessellation
// winding (orientation is recovered topologically), so fingerprint identity means POSITION/topology
// identity, NOT orientation/winding identity. Do not read it as a winding oracle.

// bodyMeshFingerprint returns the order-independent fingerprint of body b: the raw signed-consistent volume,
// triangle count, and the additive fold of every triangle's canonical hash. The volume is kept RAW (not
// snapped to the quantum) so the pin can compare it at a rel tolerance well below the position quantum
// (real re-run noise is ~1e-13); only the per-vertex POSITIONS are quantized, into the hash. Usage:
//
//	fp := bodyMeshFingerprint(caseResultBody(t, "B3"))
func bodyMeshFingerprint(b *topo.Body) meshFingerprint {
	scale := boundingDiag(b)
	quant := 1e-6 * scale
	var sum uint64
	tris := 0
	for _, f := range b.Faces() {
		sum, tris = foldFaceTriangles(f, quant, sum, tris)
	}
	vol := query.BodyGeometryProperties(b, ops.PropertyQuality()).Volume
	return meshFingerprint{Volume: vol, Triangles: tris, Hash: sum}
}

// meshFingerprint is a body's order-independent tessellation signature.
type meshFingerprint struct {
	Volume    float64
	Triangles int
	Hash      uint64
}

// foldFaceTriangles adds each of face f's canonical triangle hashes into sum and counts them.
func foldFaceTriangles(f *topo.Face, quant float64, sum uint64, tris int) (uint64, int) {
	m := tessellate.TessellateFace(f, ops.PropertyQuality())
	if m == nil {
		return sum, tris
	}
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p := m.Positions[m.Indices[i]]
		q := m.Positions[m.Indices[i+1]]
		r := m.Positions[m.Indices[i+2]]
		sum += canonicalTriangleHash(p, q, r, quant)
		tris++
	}
	return sum, tris
}

// canonicalTriangleHash hashes a triangle invariant to vertex WINDING/rotation: it quantizes each vertex,
// sorts the three by their per-vertex hash, then folds them in sorted order — so the same triangle emitted
// with any starting corner hashes identically.
func canonicalTriangleHash(p, q, r math.Point3, quant float64) uint64 {
	h := [3]uint64{vertexHash(p, quant), vertexHash(q, quant), vertexHash(r, quant)}
	sort3(&h)
	acc := uint64(1469598103934665603) // FNV-64 offset basis
	for _, v := range h {
		acc = (acc ^ v) * 1099511628211
	}
	return acc
}

// vertexHash quantizes a point to the model-relative grid and folds its three integer coordinates into a
// 64-bit hash, so last-bit FP noise cannot change the fingerprint but any real vertex move does.
func vertexHash(p math.Point3, quant float64) uint64 {
	acc := uint64(1469598103934665603)
	for _, c := range [3]float64{p.X, p.Y, p.Z} {
		q := uint64(int64(stdmath.Round(c / quant)))
		acc = (acc ^ q) * 1099511628211
	}
	return acc
}

// sort3 sorts a 3-element hash array ascending (a fixed 3-compare network — no allocation).
func sort3(h *[3]uint64) {
	if h[0] > h[1] {
		h[0], h[1] = h[1], h[0]
	}
	if h[1] > h[2] {
		h[1], h[2] = h[2], h[1]
	}
	if h[0] > h[1] {
		h[0], h[1] = h[1], h[0]
	}
}
