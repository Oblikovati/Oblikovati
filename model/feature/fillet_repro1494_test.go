// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

func cylinderFaces1494(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestEdgeNeedsPlanarizePredicate1494 pins the gate that decides whether the rolling-ball fillet
// re-facets the body: straight-edge-between-planar-faces → no (blend on the analytic body, #1494);
// a curved rim edge or a straight edge bordering a curved face → yes.
func TestEdgeNeedsPlanarizePredicate1494(t *testing.T) {
	// (a) plain box: every vertical edge is straight between two planar faces ⇒ no planarize.
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 3}, {X: 0, Y: 3}},
		sketch.XYPlane(), span{near: 0, far: 5}, 0, "box")
	for _, e := range box.Edges() {
		if edgeNeedsPlanarize(e) {
			t.Fatalf("box edge %v→%v wrongly flagged as needing planarize",
				e.StartVertex().Point(), e.EndVertex().Point())
		}
	}

	// (b) analytic cylinder: the circular rim is a curved edge ⇒ needs planarize; the straight
	// boundary edges where the wall meets a cap border the curved cylinder ⇒ also need planarize.
	fs, _ := extrudedCylinderTopRim(t, 2, 5)
	cyl := fs.Result()[0]
	sawCurved, sawStraightOnCurved := false, false
	for _, e := range cyl.Edges() {
		if _, line := e.Geometry().(geom.Circle); line {
			sawCurved = true
			if !edgeNeedsPlanarize(e) {
				t.Error("cylinder rim (circle) edge must need planarize")
			}
			continue
		}
		sawStraightOnCurved = true
		if !edgeNeedsPlanarize(e) {
			t.Error("straight seam edge bordering the cylinder wall must need planarize")
		}
	}
	if !sawCurved || !sawStraightOnCurved {
		t.Fatalf("cylinder fixture missing edges: curved=%v straightOnCurved=%v", sawCurved, sawStraightOnCurved)
	}
}

// farCornerEdgeKey returns the reference key of the vertical edge of `b` whose foot is farthest
// from (skipX,skipY) — the corner the first fillet did NOT touch.
func farCornerEdgeKey(t *testing.T, b *topo.Body, skipX, skipY float64) []byte {
	t.Helper()
	var key []byte
	best := -1.0
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X != c.X || a.Y != c.Y {
			continue
		}
		if d := stdmath.Hypot(float64(a.X)-skipX, float64(a.Y)-skipY); d > best {
			best, key = d, e.ReferenceKey()
		}
	}
	if key == nil {
		t.Fatal("no vertical edge")
	}
	return key
}

// TestSecondFilletSeparateFeature1494 reproduces bug #1494 at the model layer: a box, a fillet on
// one vertical edge, then — as a SEPARATE feature — a fillet on the diagonally-opposite vertical
// edge whose key was captured from the already-filleted body (exactly as the user picks it in the
// viewport). The whole feature tree then recomputes. The second fillet must produce real geometry
// (a second cylinder face + reduced volume), not a sick browser-only no-op.
func TestSecondFilletSeparateFeature1494(t *testing.T) {
	const r = 0.5
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 3}, {X: 0, Y: 3}},
		sketch.XYPlane(), span{near: 0, far: 5}, 0, "box")

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)

	// First fillet: the vertical edge at corner (0,0).
	edge1 := farCornerEdgeKey(t, box, 4, 3) // farthest from (4,3) ⇒ the (0,0) corner
	f1 := NewDressUpFeatures(fs).AddFillet([][]byte{edge1}, func() float64 { return r })
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("first fillet sick: %+v", f1.Health())
	}
	if n := cylinderFaces1494(fs.Result()[0]); n != 1 {
		t.Fatalf("after first fillet: %d cylinder faces, want 1", n)
	}

	// Pick the opposite corner (4,3) from the DISPLAYED filleted body — its key is owned by the
	// fillet feature (fillet:e#N), which is exactly what the bug report stored.
	edge2 := farCornerEdgeKey(t, fs.Result()[0], 0, 0)
	f2 := NewDressUpFeatures(fs).AddFillet([][]byte{edge2}, func() float64 { return r })
	fs.Recompute()

	if !f2.Health().OK() {
		t.Fatalf("second fillet sick (the #1494 symptom): %+v", f2.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("after second fillet: not a valid solid: %+v", r.Issues)
	}
	if n := cylinderFaces1494(res); n != 2 {
		t.Fatalf("after second fillet: %d cylinder faces, want 2 (second fillet was a no-op)", n)
	}
	// Two rounded vertical edges of length 5: V = 60 − 2·(r²−πr²/4)·5. Matching this analytic
	// frustum-free value also confirms the FIRST fillet's cylinder survived (was not faceted away).
	want := 4*3*5 - 2*(r*r-stdmath.Pi*r*r/4)*5
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 0.02 {
		t.Errorf("two-fillet volume = %g, want ≈%g", got, want)
	}
}
