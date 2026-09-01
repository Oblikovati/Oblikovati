// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"testing"

	"oblikovati.org/math"
)

func TestEarClipConcavePolygon(t *testing.T) {
	t.Parallel()
	// An L-shape (one reflex vertex) forces findEar to skip non-ear vertices.
	l := []math.Point2{
		math.P2(0, 0), math.P2(2, 0), math.P2(2, 1),
		math.P2(1, 1), math.P2(1, 2), math.P2(0, 2),
	}
	tris, complete := earClip(l)
	if !complete {
		t.Fatal("earClip reported an incomplete triangulation for a simple L-shape; want complete")
	}
	if len(tris) != len(l)-2 {
		t.Fatalf("earClip produced %d triangles, want %d", len(tris), len(l)-2)
	}
}

// TestEarClipRefusesDegeneratePolygon is the #3390 regression: on a degenerate polygon (all vertices
// collinear — zero area, no convex ear exists) ear clipping stalls with an un-clippable remainder. It
// must SIGNAL that shortfall (complete=false) rather than presenting a partial stump as a full
// triangulation the caller ships as a clean face.
func TestEarClipRefusesDegeneratePolygon(t *testing.T) {
	t.Parallel()
	// Four collinear points bound no area: findEar finds no convex ear, so the clip cannot complete.
	collinear := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(2, 0), math.P2(3, 0)}
	tris, complete := earClip(collinear)
	if complete {
		t.Fatalf("earClip reported a complete triangulation for a degenerate collinear polygon (%d tris); want a refusal", len(tris))
	}
}

func TestSignedAreaSign(t *testing.T) {
	t.Parallel()
	ccw := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(0, 2)}
	if signedArea(ccw) <= 0 {
		t.Error("CCW loop should have positive signed area")
	}
	cw := []math.Point2{math.P2(0, 0), math.P2(0, 2), math.P2(2, 0)}
	if signedArea(cw) >= 0 {
		t.Error("CW loop should have negative signed area")
	}
}
