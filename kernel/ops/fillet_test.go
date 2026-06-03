// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"strings"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
)

// verticalEdgeKey returns a vertical edge (start/end share X,Y) of a box.
func verticalEdgeKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no vertical edge")
	return nil
}

// hasCylinderFace reports whether the body has at least n cylindrical faces.
func hasCylinderFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// filletNotch is the cross-section area a fillet of radius r removes at a convex 90° edge:
// the square corner r² minus the quarter disc πr²/4.
func filletNotch(r float64) float64 { return r*r - stdmath.Pi*r*r/4 }

// TestFilletOneEdge rounds one vertical edge of a 2×2×2 box (r=0.5): a valid solid with one
// cylinder face, volume 8 − (r²−πr²/4)·L (L=2). The headline rolling-ball-fillet acceptance.
func TestFilletOneEdge(t *testing.T) {
	box := shellBox(2, 2, 2)
	res, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("filleted box not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 1 {
		t.Errorf("filleted box has %d cylinder faces, want 1", n)
	}
	// The cylinder face is exact; the measured volume is its faceted approximation, so check
	// at a fine tessellation tolerance where it has converged to the analytic value.
	want := 8 - filletNotch(0.5)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFilletFourVerticalEdges rounds all four vertical edges of a 2×2×2 box in one pass:
// a validated manifold solid with four quarter-cylinder faces of radius 0.5 (the F03
// acceptance). Volume 8 − 4·(r²−πr²/4)·L.
func TestFilletFourVerticalEdges(t *testing.T) {
	box := shellBox(2, 2, 2)
	var keys [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		t.Fatalf("found %d vertical edges, want 4", len(keys))
	}
	res, err := ops.FilletEdges(box, keys, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("filleted box not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 4 {
		t.Errorf("filleted box has %d cylinder faces, want 4", n)
	}
	want := 8 - 4*filletNotch(0.5)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("four-edge fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFilletLostKeyErrors checks a non-convex (concave) edge is rejected rather than
// producing garbage. A box has no concave edges, so a lost key stands in for the error path.
func TestFilletLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.FilletEdges(box, [][]byte{[]byte("ghost")}, 0.5); err == nil {
		t.Error("fillet with a lost key should error")
	}
}

// TestFilletSharedCornerErrors checks that filleting edges meeting at a corner (a fillet-
// fillet corner, not yet supported) returns a clear error rather than a broken solid.
func TestFilletSharedCornerErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	var keys [][]byte
	for _, e := range box.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if (a.X == 0 && a.Y == 0 && a.Z == 0) || (c.X == 0 && c.Y == 0 && c.Z == 0) {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 3 {
		t.Fatalf("found %d edges at the corner, want 3", len(keys))
	}
	_, err := ops.FilletEdges(box, keys, 0.3)
	if err == nil {
		t.Fatal("filleting edges that meet at a corner should error")
	}
	if !strings.Contains(err.Error(), "corner") {
		t.Errorf("error %q should mention the corner-blend limitation", err)
	}
}

// TestFilletRadiusMustBePositive guards the radius.
func TestFilletRadiusMustBePositive(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0); err == nil {
		t.Error("zero radius should error")
	}
}
