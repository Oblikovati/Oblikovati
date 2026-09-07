// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// The Resolution primitive itself is tested in kernel/geom; here we cover only the
// ops-level constructors that need topo / CSG types geom cannot see.

// TestResolutionForBody covers the body entry point: nil and empty floor to the degeneracy floor,
// and a populated body derives its size from the true RangeBox diagonal. The floor is read from geom
// (a degenerate size floors to it by definition) rather than repeated as a literal here.
func TestResolutionForBody(t *testing.T) {
	t.Parallel()
	floor := geom.ResolutionForSize(0).Size()
	if got := ResolutionForBody(nil).Size(); got != floor {
		t.Errorf("ResolutionForBody(nil).Size() = %v, want floor %v", got, floor)
	}
	if got := ResolutionForBody(&topo.Body{}).Size(); got != floor {
		t.Errorf("ResolutionForBody(empty).Size() = %v, want floor %v", got, floor)
	}
	box := subd.ToBody(subd.Box(3, 4, 12), "box") // 3-4-12 box: diagonal = 13
	if got := ResolutionForBody(box).Size(); !approxRelOps(got, 13) {
		t.Errorf("ResolutionForBody(3×4×12 box).Size() = %v, want 13", got)
	}
}

// TestResolutionForBodies takes the largest operand's size, so a boolean's tolerance
// suits the bigger body rather than a tiny tool.
func TestResolutionForBodies(t *testing.T) {
	t.Parallel()
	small := subd.ToBody(subd.Box(1, 1, 1), "s")
	big := subd.ToBody(subd.Box(3, 4, 12), "b") // diagonal 13
	if got := ResolutionForBodies(small, big).Size(); !approxRelOps(got, 13) {
		t.Errorf("ResolutionForBodies(small,big).Size() = %v, want 13 (largest)", got)
	}
	if got, floor := ResolutionForBodies().Size(), geom.ResolutionForSize(0).Size(); got != floor {
		t.Errorf("ResolutionForBodies().Size() = %v, want floor %v", got, floor)
	}
}

// TestResolutionForTris derives the size from CSG triangles' combined bbox.
func TestResolutionForTris(t *testing.T) {
	t.Parallel()
	if got, floor := ResolutionForTris(nil).Size(), geom.ResolutionForSize(0).Size(); got != floor {
		t.Errorf("ResolutionForTris(nil).Size() = %v, want floor %v", got, floor)
	}
	a, _ := mesh.NewTri(gmath.P3(0, 0, 0), gmath.P3(3, 0, 0), gmath.P3(3, 4, 12))
	if got := ResolutionForTris([]mesh.Tri{a}).Size(); !approxRelOps(got, 13) {
		t.Errorf("ResolutionForTris(3-4-12).Size() = %v, want 13", got)
	}
}

func approxRelOps(got, want float64) bool {
	return math.Abs(got-want)/math.Abs(want) < 1e-12
}
