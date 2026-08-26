// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestAdaptiveInteriorNodesRefineDensifies guards the fold-driven refinement knob (#585): a refine
// factor below 1 must add MORE interior nodes (a finer grid), which is what clears a B-spline lip's
// staircase folds; refine == 1 is the unrefined curvature-adaptive grid. domeSurface is the shared
// strongly-curved B-spline fixture from nurbs_interior_test.go.
func TestAdaptiveInteriorNodesRefineDensifies(t *testing.T) {
	s := domeSurface(t)
	outer := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}
	base := interiorNodesOnly(s, outer, nil, DefaultQuality(), 1)
	fine := interiorNodesOnly(s, outer, nil, DefaultQuality(), 0.5)
	if len(fine) <= len(base) {
		t.Errorf("refine 0.5 gave %d interior nodes, want more than the %d at refine 1", len(fine), len(base))
	}
}

// TestFoldDrivenPatchIsFoldFree pins the property the refinement loop delivers: a curved B-spline
// trim tessellates with zero fold edges and a non-empty, consistently wound mesh.
func TestFoldDrivenPatchIsFoldFree(t *testing.T) {
	s := domeSurface(t)
	outerUV := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}
	outer3D := make([]math.Point3, len(outerUV))
	for i, p := range outerUV {
		outer3D[i] = s.PointAt(float64(p.X), float64(p.Y))
	}
	su, sv := metricScale(s)
	m := foldDrivenPatch(s, su, sv, DefaultQuality(), outer3D, outerUV, nil, nil)
	if m == nil || m.TriangleCount() == 0 {
		t.Fatal("foldDrivenPatch produced no mesh")
	}
	if n := FoldEdgeCount(m); n != 0 {
		t.Errorf("foldDrivenPatch left %d fold edges; want 0", n)
	}
}

// TestSpherePoleCapFoldFree guards the sphere pole-cap fix (#585): a cap whose boundary reaches the
// pole (v = π/2, where every u collapses to one 3D point) must tessellate fold-free — the metric
// (u,v) CDT folds the degenerate sliver and repairFolds can't flip it, so metricPatchMesh falls back
// to the watertight boundary triangulation. The mesh must also stay watertight (no tear).
func TestSpherePoleCapFoldFree(t *testing.T) {
	s, err := geom.NewSphere(math.P3(0, 0, 0), 1)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// A small cap: a ring of points just below the pole plus the pole vertex itself.
	const ring = 6
	vEdge := stdmath.Pi/2 - 0.25
	var outerUV []math.Point2
	for k := range ring {
		u := 2 * stdmath.Pi * float64(k) / ring
		outerUV = append(outerUV, math.P2(math.Scalar(u), math.Scalar(vEdge)))
	}
	outerUV = append(outerUV, math.P2(math.Scalar(stdmath.Pi), math.Scalar(stdmath.Pi/2))) // the pole
	outer3D := make([]math.Point3, len(outerUV))
	for i, p := range outerUV {
		outer3D[i] = s.PointAt(float64(p.X), float64(p.Y))
	}
	m := metricPatchMesh(s, DefaultQuality(), outer3D, nil, outerUV, nil)
	if m == nil || m.TriangleCount() == 0 {
		t.Fatal("sphere pole cap produced no mesh")
	}
	if n := FoldEdgeCount(m); n != 0 {
		t.Errorf("sphere pole cap left %d fold edges; want 0", n)
	}
}
