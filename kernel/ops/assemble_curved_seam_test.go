// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// fullCircle is a closed seam arc: start==end over its [0,2π] domain (a torus/cylinder seam).
func fullCircle(t *testing.T) geom.Curve3 {
	t.Helper()
	a, err := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, 2*math.Pi)
	if err != nil {
		t.Fatalf("fullCircle: %v", err)
	}
	return a
}

// microArc is a short open arc: its endpoints are far apart (a spuriously-welded arc would
// look like this — same welded vertex index but a curve that does NOT return to its start).
func microArc(t *testing.T) geom.Curve3 {
	t.Helper()
	a, err := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, 0.1)
	if err != nil {
		t.Fatalf("microArc: %v", err)
	}
	return a
}

// TestIsClosedSeam pins the closure gate: only a==b AND a curve that returns to its start
// within the weld tolerance is a seam. A micro-arc, a nil curve, or an open pair is not.
func TestIsClosedSeam(t *testing.T) {
	const weld = 1e-3
	cases := []struct {
		name  string
		a, b  int
		curve geom.Curve3
		want  bool
	}{
		{"full circle at one vertex", 0, 0, fullCircle(t), true},
		{"micro-arc endpoints apart", 0, 0, microArc(t), false},
		{"nil curve (zero-length line)", 0, 0, nil, false},
		{"open pair", 0, 1, geom.NewLineSegment(m.P3(0, 0, 0), m.P3(1, 0, 0)), false},
	}
	for _, c := range cases {
		if got := isClosedSeam(c.a, c.b, c.curve, weld); got != c.want {
			t.Errorf("isClosedSeam(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// newSeamCatalog builds a minimal edgeCatalog over the given welded points for use() tests.
func newSeamCatalog(pts []m.Point3) *edgeCatalog {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("t", "body", 0)))
	tv := make([]*topo.Vertex, len(pts))
	for i, p := range pts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("t", "v", i)))
	}
	return &edgeCatalog{bld: bld, verts: pts, tv: tv, edges: map[seamEdgeKey]edgeRec{}, classes: map[[2]int]int{}, tag: "t", weld: 1e-3}
}

// TestUseClosedSeamFlipsSecondUse is the core regression: a closed seam edge (both endpoints
// welded to one vertex) must get OPPOSITE Reversed on its two coedges — the manifold invariant
// ops.Validate checks — even though the welded vertex order (rec.from!=a) cannot distinguish
// them. Covers both "used twice by one periodic face" and "shared by two faces": both present
// to use() as the same key requested twice. Regression for the B1 inconsistent-orientation bug.
func TestUseClosedSeamFlipsSecondUse(t *testing.T) {
	ec := newSeamCatalog([]m.Point3{m.P3(5, 0, 0)})
	arc := fullCircle(t)
	u0 := ec.use(0, 0, arc, 1)
	u1 := ec.use(0, 0, arc, 1)
	if u0.Edge != u1.Edge {
		t.Fatalf("closed seam: two uses got different edges %p %p", u0.Edge, u1.Edge)
	}
	if u0.Reversed {
		t.Errorf("closed seam: first use Reversed = true, want false")
	}
	if !u1.Reversed {
		t.Errorf("closed seam: second use Reversed = false, want true (antiparallel coedge)")
	}
}

// TestUseOpenEdgeUnchanged guards the baseline: an ordinary open edge still derives its second
// use's sense from the welded vertex order (rec.from!=a), untouched by the closed-edge branch.
func TestUseOpenEdgeUnchanged(t *testing.T) {
	ec := newSeamCatalog([]m.Point3{m.P3(0, 0, 0), m.P3(10, 0, 0)})
	line := geom.NewLineSegment(m.P3(0, 0, 0), m.P3(10, 0, 0))
	u0 := ec.use(0, 1, line, 0) // first face traverses 0→1
	u1 := ec.use(1, 0, line, 0) // second face traverses 1→0 (the manifold opposite)
	if u0.Reversed {
		t.Errorf("open edge: first use Reversed = true, want false")
	}
	if !u1.Reversed {
		t.Errorf("open edge: second use (1→0) Reversed = false, want true")
	}
}

// TestUseSpuriousSelfEdgeNotFlipped is the fail-fast guard: an a==b segment whose curve does
// NOT close (a micro-arc that only spuriously welded to one vertex) must NOT be treated as a
// seam. Its second use stays Reversed=false, so both uses agree and ops.Validate rejects the
// body LOUD — surfacing the upstream weld defect instead of laundering it into a valid-looking
// topological ghost.
func TestUseSpuriousSelfEdgeNotFlipped(t *testing.T) {
	ec := newSeamCatalog([]m.Point3{m.P3(5, 0, 0)})
	arc := microArc(t) // endpoints ~0.5 apart, >> weld
	u0 := ec.use(0, 0, arc, 0)
	u1 := ec.use(0, 0, arc, 0)
	if u0.Reversed || u1.Reversed {
		t.Errorf("spurious self-edge: uses Reversed = (%v,%v), want (false,false) so Validate fails loud", u0.Reversed, u1.Reversed)
	}
}
