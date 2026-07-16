// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"math"
	"testing"
)

// TestExtrudeDirectionOperands pins the direction operands against what the file actually declares:
// real_multipoint_disk (ReelToReel TorquimeterDisk) has six extrudes, three of which say reversed,
// one midplane, and every one names a UNIT direction vector.
//
// The unit-length check is the decode's own oracle: the vector is found by a fixed offset, so a
// wrong offset yields arbitrary magnitudes rather than 1. It reads back |d| = 1 on all 252 of the
// corpus's extrude directions.
func TestExtrudeDirectionOperands(t *testing.T) {
	ex := DecodeExtrudes(openDoc(t, "real_multipoint_disk.ipt"))
	if len(ex) != 6 {
		t.Fatalf("decoded %d extrudes, want 6", len(ex))
	}
	rev, mid := 0, 0
	for i, e := range ex {
		if !e.DirOK {
			t.Errorf("extrude %d: no direction decoded (want a unit vector)", i)
			continue
		}
		l := math.Sqrt(e.Dir[0]*e.Dir[0] + e.Dir[1]*e.Dir[1] + e.Dir[2]*e.Dir[2])
		if math.Abs(l-1) > 1e-6 {
			t.Errorf("extrude %d: |dir| = %g, want 1 (wrong offset?)", i, l)
		}
		if e.Reversed {
			rev++
		}
		if e.Midplane {
			mid++
		}
	}
	if rev != 3 {
		t.Errorf("decoded %d reversed extrudes, want 3 (the file's own count)", rev)
	}
	if mid != 1 {
		t.Errorf("decoded %d midplane extrudes, want 1 (the file's own count)", mid)
	}
}

// TestReversedPairsWithAnAlreadyFlippedDir is the fact that makes `reversed` un-actionable on its
// own: this part's three reversed extrudes each name dir = -Z, so the flip NETS BACK to +Z. Reading
// reversed as "grow the other way" would send them all negative — which is why doing so broke parts
// that were already exact. Whoever changes directionOf must keep this pairing in view.
func TestReversedPairsWithAnAlreadyFlippedDir(t *testing.T) {
	for i, e := range DecodeExtrudes(openDoc(t, "real_multipoint_disk.ipt")) {
		if !e.Reversed || !e.DirOK {
			continue
		}
		if e.Dir[2] >= 0 {
			t.Errorf("extrude %d: reversed with dir.z = %g; expected the file to pair reversed with an already-negative dir", i, e.Dir[2])
		}
		if z := -e.Dir[2]; z <= 0 {
			t.Errorf("extrude %d: effective z = %g, want > 0 (reversed x -Z ⇒ +Z)", i, z)
		}
	}
}
