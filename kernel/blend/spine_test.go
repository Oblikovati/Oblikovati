// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/blend"
	opsblend "oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// spineBox builds a validated solid box for spine fixtures.
func spineBox(sx, sy, sz float64) *topo.Body { return subd.ToBody(subd.Box(sx, sy, sz), "box") }

// TestSpineSingleLineEdge: a lone straight edge is its own guideline — length equals the edge
// length, endpoints map to abscissa 0 and Length, and the tangent is constant.
func TestSpineSingleLineEdge(t *testing.T) {
	box := spineBox(2, 2, 2)
	e := box.Edges()[0] // every box edge is a straight line segment
	sp, err := blend.NewSpine([]*topo.Edge{e}, false)
	if err != nil {
		t.Fatal(err)
	}
	wantLen := float64(e.StartVertex().Point().DistanceTo(e.EndVertex().Point()))
	if stdmath.Abs(sp.Length()-wantLen) > 1e-9 {
		t.Fatalf("spine length = %g, want %g", sp.Length(), wantLen)
	}
	if !sp.IsSingleEdge() {
		t.Error("single-edge spine should report the known-part fast path")
	}
	t0, tL := sp.TangentAt(0), sp.TangentAt(sp.Length())
	if float64(t0.AngleTo(tL)) > 1e-9 {
		t.Error("straight-edge spine tangent is not constant")
	}
}

// TestSpineRoundedRimLoop concatenates the 8-edge tangent loop of a rounded box top rim into one
// guideline: total length = 4 straight sides (len 1) + 4 quarter-arcs (r=0.5) = 4 + π, closed,
// and the start point rejoins the end point.
func TestSpineRoundedRimLoop(t *testing.T) {
	box := spineBox(2, 2, 2)
	rounded, err := opsblend.FilletEdges(box, boxVerticalEdgeKeys(t, box), 0.5)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	edges, closed := rimChainEdges(t, rounded, 2.0)
	sp, err := blend.NewSpine(edges, closed)
	if err != nil {
		t.Fatal(err)
	}
	if !sp.IsClosed() {
		t.Error("rounded rim spine should be closed")
	}
	const want = 4 + stdmath.Pi
	if stdmath.Abs(sp.Length()-want) > 1e-4 {
		t.Fatalf("rim spine length = %g, want %g", sp.Length(), want)
	}
	if d := sp.PointAt(0).DistanceTo(sp.PointAt(sp.Length())); float64(d) > 1e-6 {
		t.Errorf("closed spine endpoints differ by %g", d)
	}
}

// boxVerticalEdgeKeys returns the four vertical edges of a box.
func boxVerticalEdgeKeys(t *testing.T, b *topo.Body) [][]byte {
	t.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// rimChainEdges expands the top-rim tangent loop and resolves it to ordered edges.
func rimChainEdges(t *testing.T, b *topo.Body, top float64) ([]*topo.Edge, bool) {
	t.Helper()
	var seed []byte
	best := 0.0
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(a.Z)-top) > 1e-6 || stdmath.Abs(float64(c.Z)-top) > 1e-6 {
			continue
		}
		if l := float64(a.DistanceTo(c)); l > best {
			seed, best = e.ReferenceKey(), l
		}
	}
	keys, closed, err := opsblend.TangentEdgeChain(b, seed, opsblend.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	edges := make([]*topo.Edge, len(keys))
	for i, k := range keys {
		e, ok := b.FindEdgeByKey(k)
		if !ok {
			t.Fatalf("chain key %d not resolvable", i)
		}
		edges[i] = e
	}
	return edges, closed
}
