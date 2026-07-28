// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// frustumRimStations is the number of chord stations discretizeEdge gives a circle of radius r at
// PropertyQuality — the closed form the whole defect turns on. adaptiveParams bisects uniformly, so the
// count is the smallest power of two meeting BOTH the chord-sagitta bound r(1−cos(π/N)) ≤ tol and the
// angular bound 2π/N ≤ angleTol.
func frustumRimStations(r float64) int {
	q := PropertyQuality()
	n := 1
	for r*(1-stdmath.Cos(stdmath.Pi/float64(n))) > q.tol() || 2*stdmath.Pi/float64(n) > q.angleTol() {
		n *= 2
	}
	return n
}

// TestFrustumBandTilesEachRimAtItsOwnStations is the regression gate for the shared-rim station-count
// crack (band_rim_stations.go): a plain 100→50 cone frustum solid — the bare J1 / bfuseblend-A2 topology,
// no fillet involved — whose two rims discretize into DIFFERENT station counts (1024 and 512 at
// PropertyQuality, straddling a bisection threshold).
//
// Before the repair the seam-bridged band grid unioned both rims' samples and tiled BOTH at 1024, so the
// r=50 rim carried 1024 chords on the cone and 512 on the cap it shares that rim with: 1536 free edges,
// every one of them a T-junction. The gate asserts the welded body mesh is a CLOSED surface at both gate
// qualities and that the cone face tiles each rim at exactly that rim's own count.
func TestFrustumBandTilesEachRimAtItsOwnStations(t *testing.T) {
	body, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 200), 100, 50, "frustum")
	if err != nil {
		t.Fatalf("frustum fixture: %v", err)
	}
	small, big := frustumRimStations(50), frustumRimStations(100)
	if small != 512 || big != 1024 {
		t.Fatalf("fixture no longer straddles a bisection threshold: rims take %d and %d stations, "+
			"want 512 and 1024 (the whole point is that they DISAGREE by exactly 2×)", small, big)
	}
	for _, q := range []Quality{DefaultQuality(), PropertyQuality()} {
		mesh, _ := TessellateBody(body, q)
		if n := FreeEdgeCount(mesh); n != 0 {
			t.Errorf("frustum body mesh leaks %d free edge(s) at chord tol %g — the two faces sharing a rim "+
				"tiled it differently (a T-junction crack)", n, q.tol())
		}
	}
	if got, want := coneFaceTriangles(t, body), small+big; got != want {
		t.Errorf("cone band meshed to %d triangles, want %d (= %d + %d, one per chord of EACH rim's own "+
			"discretization); a single-station grid would give %d", got, want, small, big, 2*big)
	}
}

// coneFaceTriangles is the triangle count of the body's single geom.Cone face at PropertyQuality.
func coneFaceTriangles(t *testing.T, body *topo.Body) int {
	t.Helper()
	for _, f := range body.Faces() {
		if _, isCone := f.Geometry().(geom.Cone); isCone {
			return TessellateFace(f, PropertyQuality()).TriangleCount()
		}
	}
	t.Fatalf("frustum fixture has no geom.Cone face")
	return 0
}

// TestEqualRimBandKeepsTheGrid pins the other direction: a plain CYLINDER's two rims have the same radius,
// so they discretize identically and the band grid already satisfies the shared-edge invariant. It must
// keep the grid (a full quad strip, 2 triangles per station), not divert to the loft.
func TestEqualRimBandKeepsTheGrid(t *testing.T) {
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 100, 200)
	if err != nil {
		t.Fatalf("cylinder fixture: %v", err)
	}
	n := frustumRimStations(100)
	for _, f := range body.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			continue
		}
		if got := TessellateFace(f, PropertyQuality()).TriangleCount(); got != 2*n {
			t.Fatalf("cylinder wall meshed to %d triangles, want the grid's %d (2 per station at %d "+
				"stations) — equal rims must keep the grid path unchanged", got, 2*n, n)
		}
	}
}
