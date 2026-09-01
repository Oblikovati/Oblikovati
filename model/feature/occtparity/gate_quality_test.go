// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// gateQuality names one tessellation quality a structural mesh gate is evaluated at.
type gateQuality struct {
	name string
	q    ops.Quality
}

// gateQualities returns every quality this corpus's exact-target mesh invariants must hold at.
//
// The corpus scores AREA at PropertyQuality — correctly, since that is the tolerance every property
// readout uses and the OCCT oracles are calibrated there. But fold-freeness is not an area: it is an
// EXACT, sampling-independent property of the mesher, and until this slice every one of the corpus's
// per-face fold gates sampled PropertyQuality alone. #1510 showed what that hides in the other
// direction — a covering mesh that folded only at PropertyQuality kept a DefaultQuality-only gate green
// over a body with 12 free edges — so the same blind spot applies to a Property-only gate: it tests one
// faceting, not the mesher. DefaultQuality is the coarser sampling and costs little to add.
func gateQualities() []gateQuality {
	return []gateQuality{
		{"default", ops.DefaultQuality()},
		{"property", ops.PropertyQuality()},
	}
}

// assertFaceFoldFreeAtEveryQuality fails if a face's mesh carries a fold edge at ANY gate quality.
// propMesh is the caller's already-computed PropertyQuality mesh, reused so the sweep costs only the
// extra samplings; pass nil to have the helper mesh every quality itself.
//
// Example:
//
//	m := tessellate.TessellateFace(f, ops.PropertyQuality())
//	area := ops.MeshArea(m) // areas stay pinned at PropertyQuality
//	assertFaceFoldFreeAtEveryQuality(t, "D9", f, m)
func assertFaceFoldFreeAtEveryQuality(t *testing.T, name string, f *topo.Face, propMesh *ops.Mesh) {
	t.Helper()
	for _, gq := range gateQualities() {
		m := propMesh
		if m == nil || gq.q != ops.PropertyQuality() {
			m = tessellate.TessellateFace(f, gq.q)
		}
		if m == nil {
			t.Fatalf("%s %T face did not tessellate at %s quality", name, f.Geometry(), gq.name)
		}
		if n := ops.FoldEdgeCount(m); n != 0 {
			t.Fatalf("%s %T face has %d fold edges at %s quality; want 0 — a fold is exact and "+
				"sampling-independent, so it must be absent at every faceting%s",
				name, f.Geometry(), n, gq.name, foldDiagnostics(m))
		}
	}
}

// foldDiagnostics describes each folding edge and the RELATIVE area of the two triangles meeting
// there, so a failure says WHICH kind of fold it is rather than only that one exists.
//
// The distinction it exists to settle: a fold between two well-formed triangles is a real geometry
// defect, but a fold reported against a near-degenerate SLIVER is a measurement artefact — such a
// triangle's normal direction is dominated by rounding, so "opposed" carries no information.
// ops.FoldEdgeCount cannot tell them apart (normalsOppose rejects only an EXACTLY zero normal, while
// the repair path beside it uses a real degenerateNormal predicate). This was added because
// TestK1Z1TessellationFoldGate/Z1 reports one fold on macOS/arm64 and none on amd64 (PR #2013), and
// no arm64 hardware was available to inspect it directly — the diagnostic makes CI itself the
// instrument. DO NOT relax the gate on the strength of a guess: read this output first.
func foldDiagnostics(m *ops.Mesh) string {
	edges := ops.FoldEdges(m)
	if len(edges) == 0 {
		return ""
	}
	total := ops.MeshArea(m)
	var b strings.Builder
	for _, e := range edges {
		fmt.Fprintf(&b, "\n  fold edge v%d-v%d at %v..%v", e[0], e[1], m.Positions[e[0]], m.Positions[e[1]])
		for ti, t := range trianglesOnEdge(m, e) {
			a := triangleArea(m, t)
			fmt.Fprintf(&b, "\n    tri%d area=%.6g (%.3g%% of face %.6g)", ti, a, 100*a/total, total)
		}
	}
	return b.String()
}

// trianglesOnEdge returns the indices of the triangles incident to a mesh edge.
func trianglesOnEdge(m *ops.Mesh, e [2]int) []int {
	var out []int
	for t := 0; t+2 < len(m.Indices); t += 3 {
		hits := 0
		for k := range 3 {
			if v := m.Indices[t+k]; v == e[0] || v == e[1] {
				hits++
			}
		}
		if hits == 2 {
			out = append(out, t/3)
		}
	}
	return out
}

// triangleArea is triangle t's area (half the cross-product magnitude of two of its edges).
func triangleArea(m *ops.Mesh, t int) float64 {
	p0, p1, p2 := m.Positions[m.Indices[3*t]], m.Positions[m.Indices[3*t+1]], m.Positions[m.Indices[3*t+2]]
	return 0.5 * float64(p0.VectorTo(p1).Cross(p0.VectorTo(p2)).Length())
}
