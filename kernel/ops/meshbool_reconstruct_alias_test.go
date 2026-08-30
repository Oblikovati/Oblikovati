// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestReconstructionMergedFaceKeepsBothParentKeys pins ADR-0057 multi-parent identity end to end:
// unioning two boxes stacked along z merges each pair of coplanar side faces into one, and the merged
// face must resolve from BOTH parents' reference keys — so a pick on either operand's +x face survives
// the merge. Without the alias wiring only the representative parent's key would resolve.
func TestReconstructionMergedFaceKeepsBothParentKeys(t *testing.T) {
	a, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := brep.SolidBlock(math.P3(0, 0, 2), math.P3(2, 2, 4), "b")
	if err != nil {
		t.Fatal(err)
	}
	keyA := plusXFaceKey(t, a) // a's +x wall (z in [0,2])
	keyB := plusXFaceKey(t, b) // b's +x wall (z in [2,4]) — coplanar with a's, merges under union

	res, ok := reconstructBoolean(a, b, meshbool.Union, DefaultQuality())
	if !ok || !Validate(res).ValidSolid() {
		t.Fatalf("union did not reconstruct to a valid solid (ok=%v)", ok)
	}

	fromA := res.FacesByKey(keyA)
	fromB := res.FacesByKey(keyB)
	if len(fromA) != 1 {
		t.Fatalf("a's +x key resolved %d faces in the union, want 1", len(fromA))
	}
	if len(fromB) != 1 {
		t.Fatalf("b's +x key resolved %d faces in the union, want 1 (alias-wired merged parent)", len(fromB))
	}
	if fromA[0] != fromB[0] {
		t.Fatalf("both parents' +x keys must resolve to the SAME merged face; got distinct faces")
	}
}

// plusXFaceKey returns the reference key of the body's planar face whose outward normal is +x.
func plusXFaceKey(t *testing.T, body *topo.Body) []byte {
	t.Helper()
	for _, f := range body.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if f.Reversed() {
			n = n.Scale(-1)
		}
		if n.X > 0.99 && absf(n.Y) < 0.01 && absf(n.Z) < 0.01 {
			return f.ReferenceKey()
		}
	}
	t.Fatalf("no +x face found")
	return nil
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
