// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func TestPatchToFilletFaceCarriesSurfaceLoopsParent(t *testing.T) {
	sph, _ := geom.NewSphere(math.Point3{X: 1, Y: 2, Z: 3}, 5)
	loops := []filletLoop{{pts: []math.Point3{{X: 0}, {X: 1}, {X: 1, Y: 1}}}}
	patch := CornerBlendPatch{Surface: sph, Loops: loops, Kind: BlendKindSphere}
	// Lineage is not ==-comparable (its tokens field is a slice), so build a
	// non-zero value and compare by KeyString — the zero-value Lineage used
	// before this fix serializes to "", which made the parent assertion pass
	// even if patchToFilletFace dropped the argument entirely.
	lin := topo.NewLineage(topo.Tok("blend", "corner", 0))
	f := patchToFilletFace(patch, lin)
	if f.surface != geom.Surface(sph) {
		t.Fatal("surface not carried")
	}
	if len(f.loops) != 1 || len(f.loops[0].pts) != 3 {
		t.Fatalf("loops not carried: %+v", f.loops)
	}
	if f.parent.KeyString() != lin.KeyString() {
		t.Fatalf("parent not carried: got %q, want %q", f.parent.KeyString(), lin.KeyString())
	}
}
