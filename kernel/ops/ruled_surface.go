// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Ruled surfaces between wire sections (M07-F05, Oblikovati/Oblikovati#628):
// the reference TransientBRep.CreateRuledSurface. Both wires sample at matched
// parameters and the surface is built one ruled span at a time — a planar
// face where the four span corners are coplanar (a wall between matching
// polylines), a bilinear B-spline patch otherwise. Per-span faces keep the
// rails' creases as real edges, so the tessellation follows the surface
// exactly instead of cutting corners.

// ruledSamples is the rail sampling density.
const ruledSamples = 64

// RuledSurfaceBetweenWires builds the ruled surface body between two wires.
//
// Example: surf, err := ops.RuledSurfaceBetweenWires(w1, w2)
func RuledSurfaceBetweenWires(w1, w2 *topo.Wire) (*topo.Body, error) {
	r1, r2 := wireRailPoints(w1, ruledSamples), wireRailPoints(w2, ruledSamples)
	if len(r1) < 2 || len(r2) < 2 {
		return nil, fmt.Errorf("ops.RuledSurfaceBetweenWires: wires sample to %d and %d points; need 2+ each",
			len(r1), len(r2))
	}
	if w1.IsClosed() != w2.IsClosed() {
		return nil, fmt.Errorf("ops.RuledSurfaceBetweenWires: one wire closed, the other open")
	}
	r2 = alignRail(r1, r2, w2.IsClosed())
	if len(r2) != len(r1) {
		r2 = resampleRail(r2, len(r1))
	}
	return ruledSpanBody(r1, r2, w1.IsClosed())
}

// wireRailPoints samples the wire chain at uniform per-edge parameters.
func wireRailPoints(w *topo.Wire, n int) []math.Point3 {
	uses := w.Uses()
	if len(uses) == 0 {
		return nil
	}
	per := n / len(uses)
	if per < 2 {
		per = 2
	}
	var pts []math.Point3
	for _, u := range uses {
		seg := sampleUse(u, per)
		if len(pts) > 0 {
			seg = seg[1:]
		}
		pts = append(pts, seg...)
	}
	return pts
}

func sampleUse(u topo.Use, per int) []math.Point3 {
	c := u.Edge.Geometry()
	lo, hi := c.Domain()
	pts := make([]math.Point3, per+1)
	for i := 0; i <= per; i++ {
		t := float64(i) / float64(per)
		if u.Reversed {
			t = 1 - t
		}
		pts[i] = c.PointAt(lo + (hi-lo)*t)
	}
	return pts
}

// alignRail re-orients (and for closed rails re-phases) rail 2 to minimize
// total ruling length, so the surface doesn't twist.
func alignRail(r1, r2 []math.Point3, closed bool) []math.Point3 {
	rev := reverse3(r2)
	best, bestCost := r2, railCost(r1, r2)
	if c := railCost(r1, rev); c < bestCost {
		best, bestCost = rev, c
	}
	if closed {
		best, _ = bestPhase(r1, best, bestCost)
	}
	return best
}

// bestPhase rotates a closed rail's start to the cheapest phase.
func bestPhase(r1, r2 []math.Point3, cost float64) ([]math.Point3, float64) {
	best, bestCost := r2, cost
	n := len(r2) - 1 // last == first on a closed rail
	for shift := 1; shift < n; shift++ {
		rot := rotateRail(r2, shift)
		if c := railCost(r1, rot); c < bestCost {
			best, bestCost = rot, c
		}
	}
	return best, bestCost
}

func rotateRail(r []math.Point3, shift int) []math.Point3 {
	n := len(r) - 1
	out := make([]math.Point3, len(r))
	for i := 0; i <= n; i++ {
		out[i] = r[(i+shift)%n]
	}
	out[n] = out[0]
	return out
}

func railCost(a, b []math.Point3) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += float64(a[i].DistanceTo(b[i]))
	}
	return sum
}

// resampleRail re-samples a polyline rail to n points by arc length.
func resampleRail(r []math.Point3, n int) []math.Point3 {
	cum := make([]float64, len(r))
	for i := 1; i < len(r); i++ {
		cum[i] = cum[i-1] + float64(r[i-1].DistanceTo(r[i]))
	}
	total := cum[len(cum)-1]
	out := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		out[i] = railPointAt(r, cum, total*float64(i)/float64(n-1))
	}
	return out
}

func railPointAt(r []math.Point3, cum []float64, s float64) math.Point3 {
	for i := 1; i < len(cum); i++ {
		if s <= cum[i] {
			span := cum[i] - cum[i-1]
			if span == 0 {
				return r[i]
			}
			f := math.Scalar((s - cum[i-1]) / span)
			return r[i-1].TranslateBy(r[i-1].VectorTo(r[i]).Scale(f))
		}
	}
	return r[len(r)-1]
}

// ruledSpanBody assembles the per-span faces with shared rail and ruling edges.
func ruledSpanBody(r1, r2 []math.Point3, closed bool) (*topo.Body, error) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("ruled", "body", 0)))
	n := len(r1)
	m := n // distinct sample columns
	if closed {
		m = n - 1 // last == first
	}
	v1, v2 := railVertices(bld, r1, m, 0), railVertices(bld, r2, m, 1)
	rulings := make([]*topo.Edge, m)
	for i := 0; i < m; i++ {
		rulings[i] = bld.AddEdge(geom.NewLineSegment(r1[i], r2[i]), v1[i], v2[i],
			topo.NewLineage(topo.Tok("ruled", "ruling", i)))
	}
	for i := 0; i < n-1; i++ {
		j := (i + 1) % m
		addRuledSpanFace(bld, i, spanCorners{
			a: r1[i], b: r1[(i+1)%m], c: r2[(i+1)%m], d: r2[i],
			va: v1[i], vb: v1[j], vc: v2[j], vd: v2[i],
			rulL: rulings[i], rulR: rulings[j],
		})
	}
	return bld.Build(), nil
}

func railVertices(bld *topo.Builder, r []math.Point3, m, rail int) []*topo.Vertex {
	out := make([]*topo.Vertex, m)
	for i := 0; i < m; i++ {
		out[i] = bld.AddVertex(r[i], topo.NewLineage(topo.Tok("ruled", fmt.Sprintf("rail%d-vertex", rail), i)))
	}
	return out
}

// spanCorners is one ruled span: corners a→b along rail 1, d→c along rail 2,
// rulings a–d (left) and b–c (right).
type spanCorners struct {
	a, b, c, d     math.Point3
	va, vb, vc, vd *topo.Vertex
	rulL, rulR     *topo.Edge
}

// addRuledSpanFace adds one span as a planar face when its corners are
// coplanar, else a 2×2 bilinear B-spline patch.
func addRuledSpanFace(bld *topo.Builder, i int, sc spanCorners) {
	e1 := bld.AddEdge(geom.NewLineSegment(sc.a, sc.b), sc.va, sc.vb, topo.NewLineage(topo.Tok("ruled", "rail1", i)))
	e2 := bld.AddEdge(geom.NewLineSegment(sc.d, sc.c), sc.vd, sc.vc, topo.NewLineage(topo.Tok("ruled", "rail2", i)))
	uses := []topo.Use{topo.Fwd(e1), topo.Fwd(sc.rulR), topo.Rev(e2), topo.Rev(sc.rulL)}
	surf, err := spanSurface(sc)
	if err != nil {
		return // a fully degenerate span (both rails stalled) carries no area
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok("ruled", "face", i)), topo.OuterLoop(uses...))
}

// spanSurface picks the span's surface: plane for coplanar corners (the
// dominant case between matching polylines), bilinear patch otherwise.
func spanSurface(sc spanCorners) (geom.Surface, error) {
	n := sc.a.VectorTo(sc.b).Cross(sc.a.VectorTo(sc.d))
	if l := float64(n.Length()); l > 0 {
		dev := stdmath.Abs(float64(sc.a.VectorTo(sc.c).Dot(n))) / l
		// Coplanarity is model-relative (#1399): scaled by the span corners' own extent.
		if dev < ResolutionForPoints([]math.Point3{sc.a, sc.b, sc.c, sc.d}).Weld() {
			return geom.NewPlane(sc.a, n)
		}
	}
	ctrl := [][]math.Point3{{sc.a, sc.d}, {sc.b, sc.c}}
	weights := [][]float64{{1, 1}, {1, 1}}
	return geom.NewBSplineSurface(1, 1, ctrl, weights, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
}
