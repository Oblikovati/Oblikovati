// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// These tests pin the NopSCADlib single-cable-clip recipe at the feature level: three
// extrudes hulled into one solid, then cut by a slot prism and a screw cylinder. Unlike the
// kernel-level TestNopSingleCableClipCSG (whose cut tools span past the body on both ends),
// the live recipe's tools START exactly ON the sketch plane — flush with the hull's bottom
// face — and the slot pokes out through the hull's silhouette. That tangential configuration
// is the hard case for the boolean stack.
//
// buildClip assembles hull + the requested cuts and returns the resulting body.
func buildClip(t *testing.T, slot, hole bool) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	half := func() float64 { return 0.5 }
	deep := func() float64 { return 2.0 }

	sk1 := sketch.NewSketches().Add(sketch.XYPlane())
	clipRect(sk1, -0.8, -0.09, 0.8, 0.09)
	ex.AddByDistanceExtent(sk1, 0, ops.NewBody, half)

	sk2 := sketch.NewSketches().Add(sketch.XYPlane())
	clipRect(sk2, -0.2, -0.45, 0.2, 0.45)
	ex.AddByDistanceExtent(sk2, 0, ops.NewBody, half)

	sk3 := sketch.NewSketches().Add(sketch.XYPlane())
	sk3.Circles().AddByCenterRadius(math.P2(-0.55, 0.62), 0.22)
	ex.AddByDistanceExtent(sk3, 0, ops.NewBody, half)

	NewHullFeatures(fs).Add()

	if slot {
		sk4 := sketch.NewSketches().Add(sketch.XYPlane())
		clipPoly(sk4, clipStadium([2]float64{-0.45, 0.45}, [2]float64{-0.45, 0.05}, 0.18, 10))
		ex.AddByDistanceExtent(sk4, 0, ops.Cut, deep)
	}
	if hole {
		sk5 := sketch.NewSketches().Add(sketch.XYPlane())
		sk5.Circles().AddByCenterRadius(math.P2(0.45, 0.45), 0.15)
		ex.AddByDistanceExtent(sk5, 0, ops.Cut, deep)
	}
	fs.Recompute()
	if n := len(fs.Result()); n != 1 {
		t.Fatalf("result bodies = %d, want 1", n)
	}
	return fs.Result()[0]
}

// TestHullWithSingleFlushCutStaysValid: a hull cut once by a flush-bottomed tool (the slot
// prism or the screw cylinder) must come out a valid closed solid.
func TestHullWithSingleFlushCutStaysValid(t *testing.T) {
	for _, tc := range []struct {
		name       string
		slot, hole bool
	}{
		{"hull-only", false, false},
		{"hull+slot", true, false},
		{"hull+hole", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := buildClip(t, tc.slot, tc.hole)
			if r := ops.Validate(body); !r.Valid {
				t.Errorf("body invalid: %+v", r.Issues)
			}
			if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 || stdmath.IsNaN(v) {
				t.Errorf("volume = %v, want positive", v)
			}
		})
	}
}

// TestHullWithChainedFlushCutsStaysValid is the issue-#137 regression gate: chaining a
// SECOND flush-bottomed cut onto the already-cut hull once produced a non-manifold, open
// shell — visible as openings around the screw hole in the viewport. Root cause: a tool
// wall whose bottom edge lies exactly in the target's bottom plane imprinted that plane
// with a float-wobbled near-duplicate of the coplanar cap edge (and imprinted itself with
// its own boundary), destabilizing the planar boolean's 2D arrangement; the rejected
// result then fell back to the sliver-laden triangle CSG, which the next cut fractured.
// Fixed by filtering boundary-coincident imprints (kernel/brep imprintAll).
func TestHullWithChainedFlushCutsStaysValid(t *testing.T) {
	body := buildClip(t, true, true)
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("chained flush cuts produced an invalid body: %v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 || stdmath.IsNaN(v) {
		t.Errorf("volume = %v, want positive", v)
	}
}

func clipRect(sk *sketch.Sketch, x0, y0, x1, y1 float64) {
	c0 := sk.Points().Add(math.P2(x0, y0))
	c1 := sk.Points().Add(math.P2(x1, y0))
	c2 := sk.Points().Add(math.P2(x1, y1))
	c3 := sk.Points().Add(math.P2(x0, y1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

func clipStadium(a, b [2]float64, radius float64, steps int) [][2]float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	angle := stdmath.Atan2(dy, dx)
	var pts [][2]float64
	for i := 0; i <= steps; i++ {
		th := angle - stdmath.Pi/2 + stdmath.Pi*float64(i)/float64(steps)
		pts = append(pts, [2]float64{b[0] + radius*stdmath.Cos(th), b[1] + radius*stdmath.Sin(th)})
	}
	for i := 0; i <= steps; i++ {
		th := angle + stdmath.Pi/2 + stdmath.Pi*float64(i)/float64(steps)
		pts = append(pts, [2]float64{a[0] + radius*stdmath.Cos(th), a[1] + radius*stdmath.Sin(th)})
	}
	return pts
}

func clipPoly(sk *sketch.Sketch, pts [][2]float64) {
	first := sk.Points().Add(math.P2(pts[0][0], pts[0][1]))
	prev := first
	for _, p := range pts[1:] {
		cur := sk.Points().Add(math.P2(p[0], p[1]))
		sk.Lines().Add(prev, cur)
		prev = cur
	}
	sk.Lines().Add(prev, first)
}
