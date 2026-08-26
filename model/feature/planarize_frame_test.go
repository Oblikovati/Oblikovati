// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// facetedDiscPolygon replicates sketch.sampleCircle (24 pts, CCW from sketch +X, centered on
// the circle center) — the polygon a direct faceted extrude of a full circle uses. It is the
// reference the analytic→planarized path must reproduce so facet identity stays stable (#129).
func facetedDiscPolygon(cx, cy, r float64) []math.Point2 {
	const n = 24
	out := make([]math.Point2, n)
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		out[i] = math.P2(math.Scalar(cx)+math.Scalar(r*stdmath.Cos(a)), math.Scalar(cy)+math.Scalar(r*stdmath.Sin(a)))
	}
	return out
}

func sortedVertexPoints(pts []math.Point3) []math.Point3 {
	out := append([]math.Point3(nil), pts...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].Z < out[j].Z
	})
	return out
}

// TestPlanarizedDiscMatchesFacetedExtrude pins #129's topology-stable re-faceting: planarizing
// the analytic cylinder built from an extruded circle must yield the SAME vertex order, edge
// order, and positions as a direct faceted extrude of that circle — otherwise downstream
// dress-up (chamfer/fillet/shell) selects a different physical edge (analytic-cylinder re-faceting).
func TestPlanarizedDiscMatchesFacetedExtrude(t *testing.T) {
	plane := sketch.XYPlane()
	circle := &sketch.Circle{Center: &sketch.Point{X: 0, Y: 0}, Radius: 30}
	sp := span{near: 0, far: 10}

	analytic := buildAnalyticCylinder(circle, plane, sp, "f")
	if analytic == nil {
		t.Fatal("buildAnalyticCylinder returned nil for a Ø60 circle")
	}
	got := planarized(analytic, "f")
	if got == analytic {
		t.Fatal("planarized left the analytic cylinder unchanged; expected an N-gon prism")
	}
	want := buildPrism(facetedDiscPolygon(0, 0, 30), plane, sp, 0, "f")

	gv, wv := got.Vertices(), want.Vertices()
	if len(gv) != len(wv) {
		t.Fatalf("vertex count: planarized=%d faceted=%d", len(gv), len(wv))
	}
	// Same SET of vertices (positions), proving identical facet phase and extent.
	gp := sortedVertexPoints(vertexPositions(gv))
	wp := sortedVertexPoints(vertexPositions(wv))
	for i := range gp {
		if gp[i].DistanceTo(wp[i]) > 1e-9 {
			t.Fatalf("vertex %d mismatch: planarized=%v faceted=%v", i, gp[i], wp[i])
		}
	}
	// Same ORDER of vertices (lineage index ⇒ edge[0] identity), proving topology stability.
	for i := range gv {
		if gv[i].Point().DistanceTo(wv[i].Point()) > 1e-9 {
			t.Fatalf("vertex order %d differs: planarized=%v faceted=%v", i, gv[i].Point(), wv[i].Point())
		}
	}
	if len(got.Edges()) != len(want.Edges()) || len(got.Faces()) != len(want.Faces()) {
		t.Fatalf("topology counts: planarized e=%d f=%d, faceted e=%d f=%d",
			len(got.Edges()), len(got.Faces()), len(want.Edges()), len(want.Faces()))
	}
}

func vertexPositions(vs []*topo.Vertex) []math.Point3 {
	out := make([]math.Point3, len(vs))
	for i, v := range vs {
		out[i] = v.Point()
	}
	return out
}
