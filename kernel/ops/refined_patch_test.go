// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// unitPatch is a flat bilinear B-spline surface over the unit square in z=0 (PointAt(u,v)=(u,v,0)),
// so an L-shaped trim on it has a known 3D area — a controlled stand-in for an imported freeform
// face that exercises refinedTrimmedMesh end to end (ParamAt → CDT → 3D mesh).
func unitPatch(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 0)},
	}
	w := [][]float64{{1, 1}, {1, 1}}
	s, err := geom.NewBSplineSurface(1, 1, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func TestRefinedTrimmedMeshConcavePatch(t *testing.T) {
	s := unitPatch(t)
	// An L-shaped (concave) trim on the patch: the mesh must cover exactly the L, not bridge the
	// notch (the over-count) and not tear (the under-count) — verified by total 3D triangle area.
	outer := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 0.5, 0),
		math.P3(0.5, 0.5, 0), math.P3(0.5, 1, 0), math.P3(0, 1, 0),
	}
	m := refinedTrimmedMesh(s, outer, nil)
	if m.TriangleCount() == 0 {
		t.Fatal("patch produced no triangles")
	}
	var area float64
	bad := 0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		gn := a.VectorTo(b).Cross(a.VectorTo(c))
		area += stdmath.Sqrt(float64(gn.Dot(gn))) / 2
		sn := m.Normals[m.Indices[i]].Add(m.Normals[m.Indices[i+1]]).Add(m.Normals[m.Indices[i+2]])
		if gn.Dot(sn) < 0 {
			bad++
		}
	}
	if stdmath.Abs(area-0.75) > 1e-6 {
		t.Errorf("L-trim mesh area = %g, want 0.75 (notch bridged or torn)", area)
	}
	if bad > 0 {
		t.Errorf("%d triangles wind against their vertex normals", bad)
	}
}
