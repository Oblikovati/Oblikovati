// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/ops"
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
//	m := ops.TessellateFace(f, ops.PropertyQuality())
//	area := ops.MeshArea(m) // areas stay pinned at PropertyQuality
//	assertFaceFoldFreeAtEveryQuality(t, "D9", f, m)
func assertFaceFoldFreeAtEveryQuality(t *testing.T, name string, f *topo.Face, propMesh *ops.Mesh) {
	t.Helper()
	for _, gq := range gateQualities() {
		m := propMesh
		if m == nil || gq.q != ops.PropertyQuality() {
			m = ops.TessellateFace(f, gq.q)
		}
		if m == nil {
			t.Fatalf("%s %T face did not tessellate at %s quality", name, f.Geometry(), gq.name)
		}
		if n := ops.FoldEdgeCount(m); n != 0 {
			t.Fatalf("%s %T face has %d fold edges at %s quality; want 0 — a fold is exact and "+
				"sampling-independent, so it must be absent at every faceting", name, f.Geometry(), n, gq.name)
		}
	}
}
