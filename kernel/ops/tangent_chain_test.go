// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestTangentChainBoxHasNoPropagation: every edge of a plain box is an isolated 90° crease, so
// seeding any one yields just that edge and no loop closure.
func TestTangentChainBoxHasNoPropagation(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	keys, closed, err := blend.TangentEdgeChain(box, verticalEdgeKey(t, box), blend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || closed {
		t.Fatalf("box edge chained to %d edges (closed=%v), want 1 and open", len(keys), closed)
	}
}

// TestTangentChainRoundedRimIsClosedLoop is the #1798 repro: filleting the four vertical edges of
// a box rounds the top rim into a tangent-continuous loop of 8 edges (4 straight sides G1 to 4
// corner arcs). Seeding ONE straight side must propagate the whole rim and report closure — the
// "pick one edge → select tangent chain" the chamfer/fillet tools were missing.
func TestTangentChainRoundedRimIsClosedLoop(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	rounded, err := blend.FilletEdges(box, verticalEdgeKeys(t, box), 0.5)
	if err != nil {
		t.Fatalf("fillet setup: %v", err)
	}
	seed := longestTopRimEdge(t, rounded, 2.0)
	keys, closed, err := blend.TangentEdgeChain(rounded, seed, blend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Error("rounded top rim is a loop but chain reported open")
	}
	if len(keys) != 8 {
		t.Fatalf("rounded rim chain = %d edges, want 8 (4 sides + 4 arcs)", len(keys))
	}
}

// TestTangentChainCylinderRimClosesOnOneEdge: a cylinder's rim is a single closed circular edge,
// so it is its own loop — the chain is one edge, closed.
func TestTangentChainCylinderRimClosesOnOneEdge(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	rim := convexEdgeKey(t, cyl)
	keys, closed, err := blend.TangentEdgeChain(cyl, rim, blend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	if !closed || len(keys) < 1 {
		t.Fatalf("cylinder rim chain = %d edges (closed=%v), want a closed loop", len(keys), closed)
	}
}

// verticalEdgeKeys returns every vertical edge (start/end share X,Y) of a box.
func verticalEdgeKeys(t *testing.T, b *topo.Body) [][]byte {
	t.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		t.Fatalf("expected 4 vertical box edges, found %d", len(keys))
	}
	return keys
}

// longestTopRimEdge returns the longest edge whose endpoints both sit at z≈top — a straight
// side of the rounded rim (the straight sides are longer than the quarter-circle corner arcs).
func longestTopRimEdge(t *testing.T, b *topo.Body, top float64) []byte {
	t.Helper()
	var best []byte
	bestLen := 0.0
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(a.Z)-top) > 1e-6 || stdmath.Abs(float64(c.Z)-top) > 1e-6 {
			continue
		}
		if l := float64(a.DistanceTo(c)); l > bestLen {
			best, bestLen = e.ReferenceKey(), l
		}
	}
	if best == nil {
		t.Fatalf("no top-rim edge at z=%g", top)
	}
	return best
}

// convexEdgeKey returns the key of the first convex edge of b.
func convexEdgeKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if blend.ClassifyEdgeConvexity(e) == blend.EdgeConvex {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no convex edge")
	return nil
}
