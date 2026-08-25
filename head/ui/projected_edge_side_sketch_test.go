//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestProjectedModelEdgeOnSideSketchDraws is the #1496 render-layer confirmation: a B-rep edge of a
// block, projected into a sketch hosted on a vertical (side-like) plane, becomes a drawn polyline in
// the viewport. The reported symptom was "Project Geometry does nothing" — once the routing fix lets
// the edge reach the tool and be projected, this proves the result actually renders on a non-XY
// sketch (the same overlay that draws datum projections, exercised here with a model edge + side plane).
func TestProjectedModelEdgeOnSideSketchDraws(t *testing.T) {
	def := boxPart(t)
	body := def.SurfaceBodies().All()[0]

	side := def.Sketches().Add(sketch.XZPlane()) // a vertical sketch standing in for a side face
	edge := projectableEdgeOnto(t, def, body, side.Plane())
	side.ProjectCurve(compdef.NewEdgeRefSource(def, string(edge.ReferenceKey())))

	if got := projectedCurveOverlay(side, nil, nil); len(got) == 0 {
		t.Fatal("a model edge projected onto a side sketch must draw a polyline (#1496)")
	}
}

// boxPart builds a [0,2]x[0,2]x[0,4] solid box on a fresh part and returns its definition.
func boxPart(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(2, 0))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(0, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 4 })
	def.Recompute()
	return def
}

// projectableEdgeOnto returns a box edge whose projection onto plane spans ≥2 distinct sketch points
// (an edge not perpendicular to the plane), so the overlay has a real polyline to draw.
func projectableEdgeOnto(t *testing.T, def *compdef.PartComponentDefinition, body *topo.Body, plane sketch.Plane) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		pts, ok := compdef.NewEdgeRefSource(def, string(e.ReferenceKey())).SamplePoints()
		if !ok || len(pts) < 2 {
			continue
		}
		first := plane.ToSketch(pts[0])
		for _, q := range pts[1:] {
			if !plane.ToSketch(q).IsEqualTo(first, 1e-9) {
				return e
			}
		}
	}
	t.Fatal("no projectable (non-perpendicular) edge found on the box")
	return nil
}
