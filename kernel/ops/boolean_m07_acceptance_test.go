// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M07-F03 PBI-082 acceptance (Oblikovati/Oblikovati#298): the degenerate
// arrangements a robust boolean must survive — operands sharing a coincident
// (flush) face, identical operands, and sliver-thin overlaps — plus
// reference-key continuity at the ops.Boolean level. Every result must be a
// valid manifold solid with the analytically known volume.

// opsBoxNamed is csgBox with a caller-chosen feature name, so two operands carry
// distinct lineages (needed by the key-continuity cases).
func opsBoxNamed(name string, p math.Point3, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(m, name)
}

// requireSolid asserts the result is a valid, closed, manifold solid of the
// expected volume.
func requireSolid(t *testing.T, label string, b *topo.Body, wantVol float64) {
	t.Helper()
	r := ops.Validate(b)
	if !r.Valid || !r.Closed || !r.Manifold {
		t.Fatalf("%s: result not a valid solid: %+v", label, r.Issues)
	}
	if v := csgVolume(b); stdmath.Abs(v-wantVol) > 1e-4 {
		t.Fatalf("%s: volume = %g, want %g", label, v, wantVol)
	}
}

// Two boxes sharing a full coincident face: the join must dissolve the flush
// pair into one solid (volume 2), not leave an internal double wall.
func TestJoinCoincidentFullFace(t *testing.T) {
	a := opsBoxNamed("a", math.P3(0, 0, 0), 1, 1, 1)
	b := opsBoxNamed("b", math.P3(1, 0, 0), 1, 1, 1)
	res, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatal(err)
	}
	requireSolid(t, "flush join", res, 2)
}

// A smaller box flush against a larger face (partial coincidence): join must
// imprint the contact region and stay manifold.
func TestJoinCoincidentPartialFace(t *testing.T) {
	a := opsBoxNamed("a", math.P3(0, 0, 0), 2, 2, 2)
	b := opsBoxNamed("b", math.P3(2, 0.5, 0.5), 1, 1, 1)
	res, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatal(err)
	}
	requireSolid(t, "partial flush join", res, 9)
}

// Identical operands: union and intersection are the operand; difference is empty.
func TestBooleanIdenticalOperands(t *testing.T) {
	a := opsBoxNamed("a", math.P3(0, 0, 0), 1, 1, 1)
	b := opsBoxNamed("b", math.P3(0, 0, 0), 1, 1, 1)
	if res, err := ops.Boolean(ops.Join, a, b); err != nil {
		t.Fatalf("identical join: %v", err)
	} else {
		requireSolid(t, "identical join", res, 1)
	}
	if res, err := ops.Boolean(ops.Intersect, a, b); err != nil {
		t.Fatalf("identical intersect: %v", err)
	} else {
		requireSolid(t, "identical intersect", res, 1)
	}
	if res, err := ops.Boolean(ops.Cut, a, b); err != nil {
		t.Fatalf("identical cut: %v", err)
	} else if v := csgVolume(res); v > 1e-6 {
		t.Fatalf("identical cut volume = %g, want 0", v)
	}
}

// A cut whose tool stops a sliver short of the far wall: the remaining
// sliver-thin wall must survive as valid material, not corrupt the topology.
func TestCutLeavesSliverWall(t *testing.T) {
	const sliver = 1e-4 // thin but above merge tolerance — must survive
	a := opsBoxNamed("a", math.P3(0, 0, 0), 1, 1, 1)
	b := opsBoxNamed("b", math.P3(-0.5, 0.25, 0.25), 0.5+(1-sliver), 0.5, 0.5)
	res, err := ops.Boolean(ops.Cut, a, b)
	if err != nil {
		t.Fatal(err)
	}
	requireSolid(t, "sliver-wall cut", res, 1-0.5*0.5*(1-sliver))
}

// A sliver-wide overlap between the operands: the intersection is a slab of
// near-zero width and must still come out valid (or exactly empty).
func TestIntersectSliverOverlap(t *testing.T) {
	const overlap = 1e-4
	a := opsBoxNamed("a", math.P3(0, 0, 0), 1, 1, 1)
	b := opsBoxNamed("b", math.P3(1-overlap, 0, 0), 1, 1, 1)
	res, err := ops.Boolean(ops.Intersect, a, b)
	if err != nil {
		t.Fatal(err)
	}
	if v := csgVolume(res); v > 1e-6 {
		requireSolid(t, "sliver intersect", res, overlap)
	}
}

// Key continuity at the ops.Boolean level (the feature engine's entry point):
// a face untouched by the operation keeps a rebindable reference key, and the
// result still resolves it after a SECOND edit (a follow-up cut elsewhere) —
// the "rebindable after edits" clause of the acceptance criteria.
func TestBooleanKeyContinuityThroughEdits(t *testing.T) {
	a := opsBoxNamed("base", math.P3(0, 0, 0), 4, 4, 4)
	bottom := faceByNormal(t, a, math.V3(0, 0, -1))
	key := bottom.ReferenceKey()

	joined, err := ops.Boolean(ops.Join, a, opsBoxNamed("boss", math.P3(1, 1, 4), 2, 2, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := joined.FindFaceByKey(key); !ok {
		t.Fatal("bottom-face key lost after the join")
	}
	cutRes, err := ops.Boolean(ops.Cut, joined, opsBoxNamed("hole", math.P3(1.5, 1.5, 3), 1, 1, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cutRes.FindFaceByKey(key); !ok {
		t.Fatal("bottom-face key lost after the follow-up cut")
	}
}

// faceByNormal returns the face whose outward normal points along dir.
func faceByNormal(t *testing.T, b *topo.Body, dir math.Vector3) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if f.Reversed() {
			n = n.Scale(-1)
		}
		if n.Dot(dir) > 0.9 {
			return f
		}
	}
	t.Fatalf("no face with normal %v", dir)
	return nil
}
