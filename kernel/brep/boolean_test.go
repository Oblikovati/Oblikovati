// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// box builds a solid box [px,px+sx]×… by offsetting the cage vertices.
func box(px, py, pz, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(px, py, pz))
	}
	return subd.ToBody(m, "box")
}

func vol(b *topo.Body) float64 { return query.BodyGeometryProperties(b, ops.DefaultQuality()).Volume }

func checkSolid(t *testing.T, name string, b *topo.Body, want float64) {
	t.Helper()
	if b == nil {
		t.Fatalf("%s: nil body", name)
	}
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("%s: not a valid solid: %+v", name, r)
	}
	if got := vol(b); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("%s volume = %g, want %g", name, got, want)
	}
}

// Non-coplanar overlap (no shared face planes): A=[0,2]³ (8), B=[1,3]×[0.5,2.5]×[0.5,2.5]
// (8); overlap = 1×1.5×1.5 = 2.25.
func overlapping() (a, b *topo.Body) {
	return box(0, 0, 0, 2, 2, 2), box(1, 0.5, 0.5, 2, 2, 2)
}

func TestBRepUnion(t *testing.T) {
	t.Parallel()
	a, b := overlapping()
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "union", res, 8+8-2.25)
}

func TestBRepDifference(t *testing.T) {
	t.Parallel()
	a, b := overlapping()
	res, err := brep.Boolean(brep.Difference, a, b)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "difference", res, 8-2.25)
}

func TestBRepIntersection(t *testing.T) {
	t.Parallel()
	a, b := overlapping()
	res, err := brep.Boolean(brep.Intersection, a, b)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "intersection", res, 2.25)
}

// TestBRepCoplanarUnion fuses two boxes flush along x=2: the shared internal wall (the
// anti-shared coplanar overlap) must vanish, leaving one 4×2×2 = 16 solid.
func TestBRepCoplanarUnion(t *testing.T) {
	t.Parallel()
	a, b := box(0, 0, 0, 2, 2, 2), box(2, 0, 0, 2, 2, 2)
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "coplanar union", res, 16)
}

// TestBRepCoplanarDifferencePocket cuts a tool whose top is flush with A's top (a same-
// normal coplanar overlap) — an open-top square pocket. A's top becomes a frame; the pocket
// region of A's top drops. 8 − 1×1×1 = 7.
func TestBRepCoplanarDifferencePocket(t *testing.T) {
	t.Parallel()
	a := box(0, 0, 0, 2, 2, 2)
	tool := box(0.5, 0.5, 1, 1, 1, 1) // z∈[1,2]; top z=2 coincides with A's top
	res, err := brep.Boolean(brep.Difference, a, tool)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "coplanar pocket", res, 7)
}

// TestBRepCoplanarIntersection intersects boxes sharing all four y/z faces (both span
// [0,2]²) and overlapping x∈[1,2]: the coplanar shared faces must be kept once. 1×2×2 = 4.
func TestBRepCoplanarIntersection(t *testing.T) {
	t.Parallel()
	a, b := box(0, 0, 0, 2, 2, 2), box(1, 0, 0, 2, 2, 2)
	res, err := brep.Boolean(brep.Intersection, a, b)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "coplanar intersection", res, 4)
}

// TestBRepChainedDifference is the headline: a boolean's OUTPUT fed back as a boolean
// input — the exact case the triangle-soup CSG got wrong (returned 0.25). A=[0,2]³ minus
// two disjoint tools, each removing 0.5×1×1 = 0.5 ⇒ 8 − 0.5 − 0.5 = 7.
func TestBRepChainedDifference(t *testing.T) {
	t.Parallel()
	a := box(0, 0, 0, 2, 2, 2)
	t1 := box(-0.25, 0.5, 0.5, 0.75, 1, 1) // x∈[−0.25,0.5]: pokes through the −X face, removes 0.5×1×1
	t2 := box(1.5, 0.5, 0.5, 0.75, 1, 1)   // x∈[1.5,2.25]: pokes through the +X face, removes 0.5×1×1
	step1, err := brep.Boolean(brep.Difference, a, t1)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "chained step1", step1, 8-0.5)
	step2, err := brep.Boolean(brep.Difference, step1, t2)
	if err != nil {
		t.Fatal(err)
	}
	checkSolid(t, "chained step2 (CSG gave 0.25!)", step2, 8-0.5-0.5)
}
