// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// weXYuv projects a point to the z=0 plane frame — the uv used by the spill gate on an axis-aligned cap.
func weXYuv(p math.Point3) math.Point2 { return math.P2(float64(p.X), float64(p.Y)) }

// TestDistToCircularSegExact pins the J5 spill-gate fix: a cap face bounded by ONE closed circle used
// to measure its half-extent through a 2-point sample polygon whose chord passes through the centre
// (extent 9.6e-15 — measured on J5 before the fix), spuriously declining a cove that FITS. The exact
// branch reads |R_e − d| from the circle itself.
func TestDistToCircularSegExact(t *testing.T) {
	n, err := math.UnitVector3FromVector(math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := math.UnitVector3FromVector(math.V3(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	circle := geom.Circle{Center: math.P3(0, 0, 0), Normal: n, RefDir: ref, Radius: 156.7}
	seg := endSeg{from: circle.PointAt(0), to: circle.PointAt(0), curve: circle, mid: circle.PointAt(0.5), arc: true}
	d, ok := distToCircularSeg(math.P2(0, 0), seg, weXYuv)
	if !ok || stdmath.Abs(d-156.7) > 1e-12 {
		t.Fatalf("closed-circle seg distance from centre = %.6f ok=%v, want exactly the radius 156.7", d, ok)
	}
	d, ok = distToCircularSeg(math.P2(150, 0), seg, weXYuv)
	if !ok || stdmath.Abs(d-6.7) > 1e-12 {
		t.Fatalf("closed-circle seg distance from (150,0) = %.6f ok=%v, want 6.7", d, ok)
	}
}

// TestDistToCircularSegOpenFallsBack requires an OPEN arc to fall back to the sampled polygon: its
// nearest circle point may lie outside the arc's span, so the exact circle distance would under-read.
func TestDistToCircularSegOpenFallsBack(t *testing.T) {
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 10, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatal(err)
	}
	seg := endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true}
	if _, ok := distToCircularSeg(math.P2(0, 0), seg, weXYuv); ok {
		t.Fatal("an open arc must fall back to the polygon distance (exact circle distance under-reads)")
	}
	poly := segSamplePolygon(seg, weXYuv)
	if len(poly) != 3 {
		t.Fatalf("arc sample polygon has %d points, want from/mid/to", len(poly))
	}
}
