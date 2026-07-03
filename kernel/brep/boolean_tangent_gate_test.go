// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestExactTangentIsValidGuards covers the exact-result gate's rejection branches (#1600): a nil
// result and a non-solid (surface) body are both rejected, so the caller falls back to the recorded
// nudge rather than shipping a body that cannot be a closed 2-manifold.
func TestExactTangentIsValidGuards(t *testing.T) {
	if exactTangentIsValid(nil) {
		t.Error("nil result must be rejected by the exact-tangent gate")
	}
	surf := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "surf", 0))).Build()
	if exactTangentIsValid(surf) {
		t.Error("a non-solid body must be rejected by the exact-tangent gate")
	}
}

// TestExactTangentIsValidAcceptsSolid confirms the gate ACCEPTS a plain valid solid (every edge
// used twice, χ admissible) — the common flush/box tangency the boolean now ships exactly.
func TestExactTangentIsValidAcceptsSolid(t *testing.T) {
	blk, err := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "blk")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if !exactTangentIsValid(blk) {
		t.Error("a valid closed solid must pass the exact-tangent gate")
	}
}
