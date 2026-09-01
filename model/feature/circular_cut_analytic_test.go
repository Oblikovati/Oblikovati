// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// circleSketchAt builds a sketch with one full circle at (cx,cy) of the given radius.
func circleSketchAt(cx, cy, r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Circles().AddByCenterRadius(math.P2(cx, cy), r)
	return s
}

// closedCircleEdgeCount tallies edges whose curve is a full geom.Circle (a single circular rim — the
// edge a user clicks to fillet), as opposed to many short line segments.
func closedCircleEdgeCount(b *topo.Body) int {
	n := 0
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok {
			n++
		}
	}
	return n
}

// TestCircularCutKeepsCircleEdge is the #1472 regression: cutting a cylindrical hole through a box must
// leave a TRUE cylindrical hole wall bounded by single circular edges (the rim a user fillets), not a
// 24-gon faceted prism. The bug was that combine() pre-faceted the analytic cylinder TOOL before the
// boolean because its curved-boolean gate only routed the mirror case (a curved solid cut by a planar
// box). The box−cylinder cut now reaches the exact M2 path (curvedCylindricalHoleCut), which the kernel
// already supported but the model layer never fed an analytic cylinder.
func TestCircularCutKeepsCircleEdge(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)

	// Box: 4×4 rectangle extruded 5 deep (the report's Extrusion1, squared off for a clean volume check).
	ex.AddExtrude(squareSketch(4), []int{0}, ops.NewBody,
		Extent{Type: DistanceExtent, Direction: PositiveDir, Distance: func() float64 { return 5 }}, 0)

	// Circular cut: a r=1 circle centred in the box, drilled through.
	ex.AddExtrude(circleSketchAt(2, 2, 1), []int{0}, ops.Cut,
		Extent{Type: ThroughAllExtent, Direction: SymmetricDir}, 0)

	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("box+cut = %d bodies, want 1", len(bodies))
	}
	body := bodies[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("box+cut is not a valid solid: %+v", r.Issues)
	}

	// 6 box faces + 1 cylindrical hole wall = 7; the wall is bounded by 2 full circular edges (top & bottom).
	if got := cylinderFaceCount(body); got != 1 {
		t.Errorf("hole wall = %d cylinder faces, want 1 (the rim was faceted into a prism — #1472)", got)
	}
	if got := closedCircleEdgeCount(body); got != 2 {
		t.Errorf("hole has %d full-circle edges, want 2 (the circular rim shattered into segments — #1472)", got)
	}
	if got := len(body.Faces()); got != 7 {
		t.Errorf("drilled box has %d faces, want 7 (6 box + 1 cylinder wall)", got)
	}

	// Volume: 4·4·5 box − π·1²·5 cylinder = 80 − 5π.
	want := 4.0*4.0*5.0 - stdmath.Pi*1.0*1.0*5.0
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.01 {
		t.Errorf("drilled box volume = %g, want ≈%g (80 − 5π)", got, want)
	}
}
