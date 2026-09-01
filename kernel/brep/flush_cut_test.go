// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rectPts returns the CCW corner ring of an axis-aligned rectangle.
func rectPts(x0, y0, x1, y1 float64) []math.Point3 {
	return []math.Point3{math.P3(x0, y0, 0), math.P3(x1, y0, 0), math.P3(x1, y1, 0), math.P3(x0, y1, 0)}
}

// flushCut runs brep.Difference directly and reports the result's validity.
func flushCut(t *testing.T, name string, target, tool *topo.Body) {
	t.Helper()
	body, err := brep.Boolean(brep.Difference, target, tool)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if body == nil {
		t.Fatalf("%s: empty result", name)
	}
	r := ops.Validate(body)
	open := 0
	for _, e := range body.Edges() {
		if len(e.Faces()) == 1 {
			open++
			if open <= 6 {
				vs := e.Vertices()
				t.Logf("%s open edge: %v -> %v", name, vs[0].Point(), vs[1].Point())
			}
		}
	}
	if !r.Valid || !r.Closed || !body.IsSolid() {
		t.Errorf("%s: valid=%v closed=%v solid=%v openEdges=%d issues=%v", name, r.Valid, r.Closed, body.IsSolid(), open, r.Issues)
	}
}

// TestFlushBottomCutInterior: tool bottom coplanar with target bottom, tool strictly inside
// the silhouette, tool taller than the target (pierces the top normally).
func TestFlushBottomCutInterior(t *testing.T) {
	t.Parallel()
	target := prismBody(rectPts(0, 0, 1, 1), 0, 0.5, "box")
	tool := prismBody(rectPts(0.3, 0.3, 0.7, 0.7), 0, 2, "punch")
	flushCut(t, "interior", target, tool)
}

// TestFlushBottomCutCrossingSilhouette: same flush bottom, but the tool sticks out through
// one side wall of the target — the #137 trigger configuration.
func TestFlushBottomCutCrossingSilhouette(t *testing.T) {
	t.Parallel()
	target := prismBody(rectPts(0, 0, 1, 1), 0, 0.5, "box")
	tool := prismBody(rectPts(0.3, 0.6, 1.4, 0.9), 0, 2, "punch") // exits through x=1
	flushCut(t, "crossing", target, tool)
}

// TestNonFlushCutCrossingSilhouette: control — same silhouette crossing but the tool spans
// past the target on both ends. This already works (the kernel clip test relies on it).
func TestNonFlushCutCrossingSilhouette(t *testing.T) {
	t.Parallel()
	target := prismBody(rectPts(0, 0, 1, 1), 0, 0.5, "box")
	tool := prismBody(rectPts(0.3, 0.6, 1.4, 0.9), -0.5, 2, "punch")
	flushCut(t, "control", target, tool)
}

// stadiumPts mirrors the live mcplive slot generator: two 10-step semicircle arcs joined by
// tangent lines, CCW.
func stadiumPts(a, b [2]float64, radius float64, steps int) []math.Point3 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	angle := stdmath.Atan2(dy, dx)
	var pts []math.Point3
	for i := 0; i <= steps; i++ {
		th := angle - stdmath.Pi/2 + stdmath.Pi*float64(i)/float64(steps)
		pts = append(pts, math.P3(b[0]+radius*stdmath.Cos(th), b[1]+radius*stdmath.Sin(th), 0))
	}
	for i := 0; i <= steps; i++ {
		th := angle + stdmath.Pi/2 + stdmath.Pi*float64(i)/float64(steps)
		pts = append(pts, math.P3(a[0]+radius*stdmath.Cos(th), a[1]+radius*stdmath.Sin(th), 0))
	}
	return pts
}

// TestFlushSlotCutOnClipHull is the exact #137 brep call: the clip hull minus the live
// recipe's flush-bottomed stadium slot.
func TestFlushSlotCutOnClipHull(t *testing.T) {
	t.Parallel()
	base := prismBody(rectPts(-0.8, -0.09, 0.8, 0.09), 0, 0.5, "foot")
	post := prismBody(rectPts(-0.2, -0.45, 0.2, 0.45), 0, 0.5, "post")
	top := prismBody(regularPolygonPoints(math.P3(-0.55, 0.62, 0), 0.22, 24, 0), 0, 0.5, "loop")
	hull, err := query.ConvexHullOf("clip-hull", base, post, top)
	if err != nil {
		t.Fatalf("hull: %v", err)
	}
	tool := prismBody(stadiumPts([2]float64{-0.45, 0.45}, [2]float64{-0.45, 0.05}, 0.18, 10), 0, 2, "slot")
	flushCut(t, "clip-slot", hull, tool)
}

// TestNonFlushSlotCutOnClipHull: identical cut, tool extended past the bottom — isolates
// whether the flush z=0 caps are essential to the failure.
func TestNonFlushSlotCutOnClipHull(t *testing.T) {
	t.Parallel()
	base := prismBody(rectPts(-0.8, -0.09, 0.8, 0.09), 0, 0.5, "foot")
	post := prismBody(rectPts(-0.2, -0.45, 0.2, 0.45), 0, 0.5, "post")
	top := prismBody(regularPolygonPoints(math.P3(-0.55, 0.62, 0), 0.22, 24, 0), 0, 0.5, "loop")
	hull, err := query.ConvexHullOf("clip-hull", base, post, top)
	if err != nil {
		t.Fatalf("hull: %v", err)
	}
	tool := prismBody(stadiumPts([2]float64{-0.45, 0.45}, [2]float64{-0.45, 0.05}, 0.18, 10), -0.5, 2, "slot")
	flushCut(t, "clip-slot-nonflush", hull, tool)
}

// TestFlushObliqueCutOnBox: flush bottom + an oblique (rotated square) tool crossing a plain
// box wall — obliquity without the hull/arc complexity. (The tool corner stays clear of the
// y=1 wall: a corner EXACTLY on a target face pinches the solid along a line — a genuine
// non-manifold configuration outside this test's scope.)
func TestFlushObliqueCutOnBox(t *testing.T) {
	t.Parallel()
	target := prismBody(rectPts(0, 0, 1, 1), 0, 0.5, "box")
	tool := prismBody([]math.Point3{
		math.P3(0.7, 0.43, 0), math.P3(1.1, 0.73, 0), math.P3(0.9, 0.98, 0), math.P3(0.5, 0.68, 0),
	}, 0, 2, "punch")
	flushCut(t, "oblique", target, tool)
}

// TestFlushInteriorSlotOnClipHull: flush stadium slot fully INSIDE the hull silhouette.
func TestFlushInteriorSlotOnClipHull(t *testing.T) {
	t.Parallel()
	base := prismBody(rectPts(-0.8, -0.09, 0.8, 0.09), 0, 0.5, "foot")
	post := prismBody(rectPts(-0.2, -0.45, 0.2, 0.45), 0, 0.5, "post")
	top := prismBody(regularPolygonPoints(math.P3(-0.55, 0.62, 0), 0.22, 24, 0), 0, 0.5, "loop")
	hull, err := query.ConvexHullOf("clip-hull", base, post, top)
	if err != nil {
		t.Fatalf("hull: %v", err)
	}
	tool := prismBody(stadiumPts([2]float64{-0.3, 0.2}, [2]float64{-0.3, 0.1}, 0.06, 10), 0, 2, "slot")
	flushCut(t, "clip-slot-interior", hull, tool)
}
