// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// cylinderPart returns a session whose active part holds a radius×height cylinder (an extruded
// circle), so the box-select tests run against genuine curved B-rep rim edges.
func cylinderPart(t *testing.T, radius, height float64) *Session {
	t.Helper()
	s := newPartSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(0, 0), radius)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return height })
	def.Recompute()
	return s
}

// curvedEdgeOf returns the first edge of the body whose adaptive sampling has more than two
// points — a genuinely curved edge (the cylinder's circular rim).
func curvedEdgeOf(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if len(tessellate.TessellateEdge(e, ops.DefaultQuality())) > 2 {
			return e
		}
	}
	t.Fatal("no curved edge found on the cylinder")
	return nil
}

// TestPickRegionCurvedEdgeSampledNotEndpoints is the #936 regression: a small rect placed over a
// MID-SPAN point of a curved rim edge — with the edge's vertices outside it — selects the edge on
// a crossing drag. The old endpoint-only outline classified the whole rim by its seam vertex and
// would have missed this; the sampled outline catches it.
func TestPickRegionCurvedEdgeSampledNotEndpoints(t *testing.T) {
	t.Parallel()
	s := cylinderPart(t, 2, 4)
	body := partBodies(s)()[0]
	edge := curvedEdgeOf(t, body)

	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target = math.P3(18, 6, 9), math.P3(0, 0, 2) // oblique, so the rim projects to an ellipse
	p := NewRayPicker(cam, partBodies(s))
	edges := NewSelectionFilter(SelectEdge)

	proj := func(pt math.Point3) (screenPt, bool) {
		sx, sy, ok := renderer.Project(cam, regionNear, regionFar, pt)
		return screenPt{sx, sy}, ok
	}

	// A mid-span sample of the rim (far from the seam vertex), and a tiny rect around it.
	samples := tessellate.TessellateEdge(edge, ops.DefaultQuality())
	mid, ok := proj(samples[len(samples)/2])
	if !ok {
		t.Fatal("mid-span rim sample did not project")
	}
	rect := screenRect{minX: mid.x - 5, minY: mid.y - 5, maxX: mid.x + 5, maxY: mid.y + 5}

	// Fixture guard: the rim edge's own vertices must be OUTSIDE this rect, so a hit can only come
	// from a sampled mid-span point — exactly what the endpoint-only outline missed.
	for _, v := range edge.Vertices() {
		if vp, ok := proj(v.Point()); ok && rect.containsPoint(vp.x, vp.y) {
			t.Fatalf("fixture: a rim vertex %v projected inside the mid-span rect %+v", vp, rect)
		}
	}

	hits := p.PickRegion(rect.minX, rect.minY, rect.maxX, rect.maxY, true, edges)
	if !slices.ContainsFunc(hits, func(h Selectable) bool { eh, ok := h.(EdgeHandle); return ok && eh.Edge == edge }) {
		t.Fatalf("crossing over a curved rim's mid-span did not select it (got %d hits) — sampling regressed", len(hits))
	}
}

// TestPickRegionCurvedEdgeWindowNeedsWholeSpan: a crossing rect over part of the rim selects it,
// but a window rect over that same part does NOT (the rest of the rim is outside) — window-select
// requires every sampled point inside, proving the classification uses the full span.
func TestPickRegionCurvedEdgeWindowNeedsWholeSpan(t *testing.T) {
	t.Parallel()
	s := cylinderPart(t, 2, 4)
	body := partBodies(s)()[0]
	edge := curvedEdgeOf(t, body)

	cam := scene.NewCamera(400, 400)
	cam.Eye, cam.Target = math.P3(18, 6, 9), math.P3(0, 0, 2)
	p := NewRayPicker(cam, partBodies(s))
	edges := NewSelectionFilter(SelectEdge)

	samples := tessellate.TessellateEdge(edge, ops.DefaultQuality())
	midX, _, _ := renderer.Project(cam, regionNear, regionFar, samples[len(samples)/2])
	// A vertical band over part of the rim: tall enough in Y to cover that side, narrow in X so
	// the far side of the rim falls outside it.
	minX, maxX := midX-12, midX+12

	if got := p.PickRegion(minX, -1e6, maxX, 1e6, true, edges); !slices.ContainsFunc(got, func(h Selectable) bool { eh, ok := h.(EdgeHandle); return ok && eh.Edge == edge }) {
		t.Fatalf("crossing over part of the rim should select it, got %d hits", len(got))
	}
	if got := p.PickRegion(minX, -1e6, maxX, 1e6, false, edges); slices.ContainsFunc(got, func(h Selectable) bool { eh, ok := h.(EdgeHandle); return ok && eh.Edge == edge }) {
		t.Error("window over only part of the rim must NOT select it (the rim extends outside the band)")
	}
}
