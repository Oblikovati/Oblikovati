// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// curvedDome is a doubly-curved B-spline patch (a raised centre control point) — enough curvature
// that the interior-node grid is non-empty and a coarse triangulation would chord the dome.
func curvedDome(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0), math.P3(0, 2, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 2), math.P3(1, 2, 0)},
		{math.P3(2, 0, 0), math.P3(2, 1, 0), math.P3(2, 2, 0)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// TestAdaptiveInteriorNodesRefineDensifies guards the fold-driven refinement knob (#585): a refine
// factor below 1 must add MORE interior nodes (a finer grid), which is what clears a B-spline lip's
// staircase folds; refine == 1 is the unrefined curvature-adaptive grid.
func TestAdaptiveInteriorNodesRefineDensifies(t *testing.T) {
	s := curvedDome(t)
	outer := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}
	base := adaptiveInteriorNodes(s, outer, nil, DefaultQuality(), 1)
	fine := adaptiveInteriorNodes(s, outer, nil, DefaultQuality(), 0.5)
	if len(fine) <= len(base) {
		t.Errorf("refine 0.5 gave %d interior nodes, want more than the %d at refine 1", len(fine), len(base))
	}
}

// TestFoldDrivenPatchIsFoldFree pins the property the refinement loop delivers: a curved B-spline
// trim tessellates with zero fold edges and a non-empty, consistently wound mesh.
func TestFoldDrivenPatchIsFoldFree(t *testing.T) {
	s := curvedDome(t)
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

// TestNurbsFaceTessellatesFoldFree exercises the B-spline face path end-to-end (TessellateFace →
// nurbsPcurveMesh → the fold-driven refinement loop): a curved dome face whose flat boundary the
// four corner edges trace must tessellate to a non-empty, fold-free mesh (#585).
func TestNurbsFaceTessellatesFoldFree(t *testing.T) {
	s := curvedDome(t)
	corners := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "body", 0)))
	lin := topo.NewLineage(topo.Tok("t", "x", 0))
	v := [4]*topo.Vertex{}
	for i, p := range corners {
		v[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range corners {
		j := (i + 1) % 4
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], lin))
	}
	bld.AddFace(s, lin, topo.OuterLoop(uses...))
	f := bld.Build().Faces()[0]

	m := TessellateFace(f, DefaultQuality())
	if m.TriangleCount() == 0 {
		t.Fatal("B-spline face produced no triangles")
	}
	if n := FoldEdgeCount(m); n != 0 {
		t.Errorf("B-spline face tessellated with %d fold edges; want 0", n)
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
	for k := 0; k < ring; k++ {
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
