// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
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
// BOTH walls analytic. Before ADR-0056 the analytic path had no handler for
// cylinder ∪ cocylindrical-arc-prism, so it faceted the whole join (74 planar faces, 0
// cylinders) and the two walls' mismatched facet grids showed as a visible seam. The
// The general per-face pipeline builds both walls on their exact cylinder surface, so they
// re-tessellate against one surface (aligned grids, no seam) — asserted here as analytic
// cylinder faces surviving the join plus the exact stacked volume.
func TestPistonHeadCocylindricalJoinKeepsAnalyticWalls(t *testing.T) {
	t.Parallel()
	// #2167 at the feature level: an EXTRUDE-built full cylinder JOINED to a stacked D-prism whose
	// arc wall is cocylindrical must keep BOTH walls analytic, merged to ONE analytic wall (cyl==1),
	// so they re-tessellate against a single surface with no seam. The
	// robustness layers this extrude-built shape needed are all closed. The reconstruction engine that
	// first delivered this row is gone (ADR-0061 stage 7) and the general per-face pipeline builds it,
	// so there is no kill-switch left to skip on: the row is unconditional.
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
	// ONE analytic wall: the full cylinder wall and the D's cocylindrical arc wall lie on one surface,
	// and the run they share bounds nothing, so it dissolves and the two are one face (ADR-0061 stage
	// 5). The faceted bug left ZERO cylinders, which is what #2167 was; this row was pinned at 2 while
	// the merge was outstanding, and converting it is what landing the merge means.
	if got := cylinderFaceCount(body); got != 1 {
		t.Fatalf("piston-head join has %d analytic cylinder faces, want 1 (faceted gave 0, #2167)", got)
	}
	// Exact stacked volume: cylinder + the D-segment prism (disc minus the minor segment
	// the chord cuts). A 24-gon faceting under-reports this by ~1%.
	minorSeg := 0.5 * r * r * (2*theta - stdmath.Sin(2*theta))
	dArea := stdmath.Pi*r*r - minorSeg
	analytic := stdmath.Pi*r*r*h1 + dArea*h2
	if v := query.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; relErr(v, analytic) > 5e-3 {
		t.Fatalf("piston-head join volume = %g, want %g — faceted, not the analytic union", v, analytic)
	}
}
