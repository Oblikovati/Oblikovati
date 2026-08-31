// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The shell/wire/query adapters over kernel topology (#628/#629/#630). Void-ness is a property of a
// shell RELATIVE TO ITS BODY (#3483), so the fixture is a block with a fully enclosed cavity: two
// closed shells, exactly one of them a void.

// cavityBlock is a 10³ block with a concentric 4³ cavity — the two-shell fixture.
func cavityBlock(t *testing.T) *topo.Body {
	t.Helper()
	outer, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock outer: %v", err)
	}
	inner, err := brep.SolidBlock(math.P3(3, 3, 3), math.P3(7, 7, 7), "cavity")
	if err != nil {
		t.Fatalf("SolidBlock inner: %v", err)
	}
	body, err := ops.Boolean(ops.Cut, outer, inner)
	if err != nil {
		t.Fatalf("cut cavity: %v", err)
	}
	return body
}

// TestFaceShellsSeparateMaterialFromVoid: the cavity block enumerates two closed shells, exactly one of
// which is a void, and each reports its own region's volume as a magnitude (1000 and 64).
func TestFaceShellsSeparateMaterialFromVoid(t *testing.T) {
	shells := NewFaceShells(cavityBlock(t), ops.DefaultQuality())
	if shells.Count() != 2 {
		t.Fatalf("cavity block has %d shells, want 2", shells.Count())
	}
	voids, volumes := 0, map[float64]bool{}
	for i := range shells.Count() {
		sh := shells.Item(i)
		if !sh.IsClosed() {
			t.Errorf("shell %d is not closed", i)
		}
		if sh.IsVoid() {
			voids++
		}
		volumes[roundTo(sh.Volume(), 1e-6)] = true
		if sh.FaceCount() != 6 || sh.EdgeCount() != 12 {
			t.Errorf("shell %d has %d faces / %d edges, want 6 / 12", i, sh.FaceCount(), sh.EdgeCount())
		}
		if len(sh.ReferenceKey()) == 0 || sh.TransientKey() == 0 {
			t.Errorf("shell %d has no reference key / transient key", i)
		}
	}
	if voids != 1 {
		t.Errorf("cavity block reports %d void shells, want 1", voids)
	}
	if !volumes[1000] || !volumes[64] {
		t.Errorf("shell volumes = %v, want the 1000 outer and the 64 cavity as magnitudes", keysOf(volumes))
	}
}

// TestFaceShellPointContainment maps the kernel verdict onto the frozen wire enum for all three cases.
func TestFaceShellPointContainment(t *testing.T) {
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	sh := NewFaceShell(block.Shells()[0], block, ops.DefaultQuality())
	cases := []struct {
		x, y, z float64
		want    types.Containment
	}{
		{5, 5, 5, types.InsideContainment},
		{5, 5, 0, types.OnContainment},
		{50, 5, 5, types.OutsideContainment},
	}
	for _, c := range cases {
		if got := sh.IsPointInside(c.x, c.y, c.z); got != c.want {
			t.Errorf("IsPointInside(%g,%g,%g) = %v, want %v", c.x, c.y, c.z, got, c.want)
		}
	}
}

// TestBodyQueriesOnCavityBlock: the body-level query surface — containment against the MATERIAL (the
// cavity's interior is outside it), the dihedral counts of a block plus its cavity, and validity.
func TestBodyQueriesOnCavityBlock(t *testing.T) {
	q := NewBodyQueries(cavityBlock(t), ops.DefaultQuality())
	if got := q.IsPointInside(1, 1, 1); got != types.InsideContainment {
		t.Errorf("point in the wall = %v, want inside", got)
	}
	if got := q.IsPointInside(5, 5, 5); got != types.OutsideContainment {
		t.Errorf("point in the cavity = %v, want outside (it is not material)", got)
	}
	if got := q.ConvexEdgeCount(); got != 12 {
		t.Errorf("convex edge count = %d, want the outer block's 12", got)
	}
	if got := q.ConcaveEdgeCount(); got != 12 {
		t.Errorf("concave edge count = %d, want the cavity's 12", got)
	}
	for _, level := range []int{0, 1, 2, 9} {
		if !q.IsEntityValid(level) {
			t.Errorf("IsEntityValid(%d) = false on a valid cavity block", level)
		}
	}
}

// TestBindTransientKeyRoundTrip: a session id resolves to its entity kind and persistent key, and an
// unbound id is refused rather than guessed.
func TestBindTransientKeyRoundTrip(t *testing.T) {
	body := cavityBlock(t)
	q := NewBodyQueries(body, ops.DefaultQuality())
	face := body.Faces()[0]
	kind, key, ok := q.BindTransientKey(face.ID())
	if !ok {
		t.Fatalf("BindTransientKey(%d) did not resolve a face of the body", face.ID())
	}
	if kind != topo.KindFace.String() {
		t.Errorf("bound kind = %q, want %q", kind, topo.KindFace.String())
	}
	if string(key) != string(face.ReferenceKey()) {
		t.Errorf("bound key %x, want the face's own reference key %x", key, face.ReferenceKey())
	}
	if _, _, ok := q.BindTransientKey(1 << 40); ok {
		t.Error("BindTransientKey resolved an id the body never issued")
	}
}

// TestWiresOnSolidBody: a solid block carries no free wire, so the collection is empty rather than
// synthesised from its face loops.
func TestWiresOnSolidBody(t *testing.T) {
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if got := NewWires(block).Count(); got != 0 {
		t.Errorf("solid block reports %d wires, want 0", got)
	}
}

// roundTo snaps v to the nearest multiple of step so exact fixture volumes compare as map keys.
func roundTo(v, step float64) float64 {
	return float64(int64(v/step+0.5)) * step
}

// keysOf lists a set's keys for a failure message.
func keysOf(set map[float64]bool) []float64 {
	out := make([]float64, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
