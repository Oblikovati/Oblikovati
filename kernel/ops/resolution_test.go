// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// The Resolution primitive itself is tested in kernel/geom; here we cover only the
// ops-level constructors that need topo / CSG types geom cannot see.

// TestResolutionForBody covers the body entry point: nil and empty floor to 1, and a
// populated body derives its size from the true RangeBox diagonal.
func TestResolutionForBody(t *testing.T) {
	t.Parallel()
	if got := ResolutionForBody(nil).Size(); got != 1 {
		t.Errorf("ResolutionForBody(nil).Size() = %v, want floor 1", got)
	}
	if got := ResolutionForBody(&topo.Body{}).Size(); got != 1 {
		t.Errorf("ResolutionForBody(empty).Size() = %v, want floor 1", got)
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
	if got := ResolutionForBodies().Size(); got != 1 {
		t.Errorf("ResolutionForBodies().Size() = %v, want floor 1", got)
	}
}

// TestResolutionForTris derives the size from CSG triangles' combined bbox.
func TestResolutionForTris(t *testing.T) {
	t.Parallel()
	if got := resolutionForTris(nil).Size(); got != 1 {
		t.Errorf("resolutionForTris(nil).Size() = %v, want floor 1", got)
	}
	a, _ := newTri(gmath.P3(0, 0, 0), gmath.P3(3, 0, 0), gmath.P3(3, 4, 12))
	if got := resolutionForTris([]tri{a}).Size(); !approxRelOps(got, 13) {
		t.Errorf("resolutionForTris(3-4-12).Size() = %v, want 13", got)
	}
}

func approxRelOps(got, want float64) bool {
	return math.Abs(got-want)/math.Abs(want) < 1e-12
}
