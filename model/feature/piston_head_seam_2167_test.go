// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// dProfileSketchOnPlaneZ builds a "D" profile on the plane at z: a major circular arc of
// radius r closed by a chord. The arc shares the main cylinder's radius, so an extrude
// from it has a wall COCYLINDRICAL with the cylinder below — the #2167 piston-head shape.
// theta is the half-angle the chord subtends at the centre (the flat sits at x = r·cosθ).
func dProfileSketchOnPlaneZ(z, r, theta float64) *sketch.Sketch {
	ux, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	uy, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	pl, _ := sketch.NewPlane(math.P3(0, 0, z), ux, uy)
	s := sketch.NewSketches().Add(pl)
	ax, ay := r*stdmath.Cos(theta), r*stdmath.Sin(theta)
	a := s.Points().Add(math.P2(ax, -ay))
	b := s.Points().Add(math.P2(ax, ay))
	s.Lines().Add(a, b)                                     // the chord (the flat of the D)
	s.Arcs().Add(s.Points().Add(math.P2(0, 0)), b, a, true) // major arc B→A CCW around the back
	return s
}

// TestPistonHeadCocylindricalJoinKeepsAnalyticWalls is the #2167 regression: a full
// cylinder JOINED to a stacked D-profile prism whose arc wall is cocylindrical must keep
// BOTH walls analytic. Before ADR-0054 the analytic path had no handler for
// cylinder ∪ cocylindrical-arc-prism, so it faceted the whole join (74 planar faces, 0
// cylinders) and the two walls' mismatched facet grids showed as a visible seam. The
// provenance reconstruction rebuilds both walls on their exact cylinder surface, so they
// re-tessellate against one surface (aligned grids, no seam) — asserted here as analytic
// cylinder faces surviving the join plus the exact stacked volume.
func TestPistonHeadCocylindricalJoinKeepsAnalyticWalls(t *testing.T) {
	// ADR-0054 target, not yet closed. Reconstruction rebuilds the two cocylindrical
	// walls analytically (cyl=2), but the mesh boolean produces a zero-volume rim-sliver
	// membrane for this cocylindrical cap-on-wall case: the cylinder's top cap tessellates
	// to the true rim (radius R) while the D-prism's arc is inscribed (below R), so the
	// thin ring between them is kept as an opposite-facing membrane the tessellations do
	// not conform. The soup is 2-manifold and correct-volume, yet it groups to a
	// non-manifold B-rep. The fix is a conforming (canonical absolute-angle) tessellation
	// of the shared cocylindrical circle — a tessellation-path change validated on its own.
	// The recovery adopts reconstruction only when it validates, so #2167 stays faceted
	// today: no regression, no fix. See ADR-0054 §Status and the kernel-level twin
	// ops.TestReconstructCocylindricalCapOnWall. Un-skip when the tessellation fix lands.
	t.Skip("ADR-0054/#2167: cocylindrical cap-on-wall needs conforming tessellation (rim-sliver membrane)")
	const r, theta, h1, h2 = 3.0, 0.6, 6.0, 4.0
	fs := NewPartFeatures(nil)
	ex := NewExtrudeFeatures(fs)
	ex.AddByDistanceExtent(circleSketchAt(0, 0, r), 0, ops.NewBody, func() float64 { return h1 })           // Ø6 z[0,6]
	ex.AddByDistanceExtent(dProfileSketchOnPlaneZ(h1, r, theta), 0, ops.Join, func() float64 { return h2 }) // D z[6,10]
	fs.Recompute()

	if n := len(fs.Result()); n != 1 {
		t.Fatalf("piston-head join = %d bodies, want 1", n)
	}
	body := fs.Result()[0]
	if v := ops.Validate(body); !v.Valid || !body.IsSolid() {
		t.Fatalf("piston-head join is not a valid solid: %+v", v)
	}
	// Both walls analytic: the full cylinder wall and the D's cocylindrical arc wall. The
	// faceted bug left zero. (Reconstruction keeps them as two cocylindrical faces.)
	if got := cylinderFaceCount(body); got < 2 {
		t.Fatalf("piston-head join has %d analytic cylinder walls, want >=2 (faceted, #2167)", got)
	}
	// Exact stacked volume: cylinder + the D-segment prism (disc minus the minor segment
	// the chord cuts). A 24-gon faceting under-reports this by ~1%.
	minorSeg := 0.5 * r * r * (2*theta - stdmath.Sin(2*theta))
	dArea := stdmath.Pi*r*r - minorSeg
	analytic := stdmath.Pi*r*r*h1 + dArea*h2
	if v := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; relErr(v, analytic) > 5e-3 {
		t.Fatalf("piston-head join volume = %g, want %g — faceted, not the analytic union", v, analytic)
	}
}
