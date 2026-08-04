// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// The ADR-2 Step-1 strangler byte-identity golden for the curved-arm trihedral weld, split out of
// fillet_curved_weld_test.go (CLAUDE.md 500-line file cap) so the two files stay independently readable:
// this one owns the "routing through the RailLoop engine changed nothing observable" gate, the other
// owns the setback-rail/host-arc/provenance/volume test bed.

// TestCurvedCornerFace_B3ByteIdentical is the ADR-2 Step-1 strangler gate: routing the corner face
// through the RailLoop engine must leave the clean B3 octant BYTE-for-byte identical to the pre-strangler
// curvedSphereFace — same surface value AND the same boundary loop (points + per-segment curves).
// resolveBlend is used only to VALIDATE the octant resolves to the sphere tier; the emitted face keeps
// the legacy chainSetbackArcs loop and the exact-centre `sphere` (the engine's circumcentre-recovered
// sphere carries ~1e-12 FP noise, so returning it would NOT be byte-identical). This is the surface-swap,
// loop-preserved contract; it FAILS if curvedCornerFace ever emits the engine patch on the octant.
func TestCurvedCornerFace_B3ByteIdentical(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	legacy, ok := curvedSphereFace(w, sphere)
	if !ok {
		t.Fatalf("curvedSphereFace declined the certified B3 corner")
	}
	got, ok := curvedCornerFace(w, sphere, arms, res)
	if !ok {
		t.Fatalf("curvedCornerFace declined the certified B3 octant")
	}
	assertFilletFaceIdentical(t, got, legacy)
}

// assertFilletFaceIdentical asserts two filletFaces are byte-for-byte equal: identical surface value,
// identical loop count, and identical per-loop points and per-segment curves (geom.Arc3d is comparable,
// so == is an exact float-for-float check).
func assertFilletFaceIdentical(t *testing.T, got, want filletFace) {
	t.Helper()
	if got.surface != want.surface {
		t.Fatalf("surface differs: got %v, want %v (octant must keep the exact-centre sphere)", got.surface, want.surface)
	}
	if len(got.loops) != len(want.loops) {
		t.Fatalf("loop count = %d, want %d", len(got.loops), len(want.loops))
	}
	for i := range want.loops {
		assertLoopIdentical(t, got.loops[i], want.loops[i])
	}
}

// assertLoopIdentical asserts two filletLoops have identical point rings and per-segment curves.
func assertLoopIdentical(t *testing.T, got, want filletLoop) {
	t.Helper()
	if len(got.pts) != len(want.pts) || len(got.curves) != len(want.curves) {
		t.Fatalf("loop size differs: pts %d/%d curves %d/%d", len(got.pts), len(want.pts), len(got.curves), len(want.curves))
	}
	for i := range want.pts {
		if got.pts[i] != want.pts[i] {
			t.Fatalf("loop point %d differs: got %v, want %v", i, got.pts[i], want.pts[i])
		}
		if got.curves[i] != want.curves[i] {
			t.Fatalf("loop curve %d differs (segment %d)", i, i)
		}
	}
}
