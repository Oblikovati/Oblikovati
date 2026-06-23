// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// edgeAlongX traces the world line (0..L in x) on a plane built from axes (UAxis=X), so the edge
// parameter maps to the plane's u with v=0.
func edgeAlongX(length float64) EdgeParam {
	return func(t float64) (u, v float64) { return t * length, 0 }
}

// xyPlane is the z=0 plane with UAxis=X, VAxis=Y (normal +Z), origin at z0.
func xyPlane(t *testing.T, z0 float64) geom.Plane {
	t.Helper()
	p, err := geom.NewPlaneFromAxes(math.P3(0, 0, math.Scalar(z0)), math.V3(1, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	return p
}

func TestCrossEdgeCoplanarIsFullyContinuous(t *testing.T) {
	a, b := xyPlane(t, 0), xyPlane(t, 0)
	rep := CrossEdgeContinuity(a, b, edgeAlongX(5), edgeAlongX(5), 15)
	if rep.MaxGap > 1e-9 || rep.MaxNormalDeg > 1e-9 || rep.MaxCurvDiff > 1e-9 {
		t.Errorf("coplanar seam should be G0/G1/G2 continuous: gap=%g deg=%g curv=%g",
			rep.MaxGap, rep.MaxNormalDeg, rep.MaxCurvDiff)
	}
}

func TestCrossEdgeReportsPositionalGap(t *testing.T) {
	// Two parallel planes offset in z, traced at the same (x,0): a pure G0 gap of δ.
	const delta = 0.25
	a, b := xyPlane(t, 0), xyPlane(t, delta)
	rep := CrossEdgeContinuity(a, b, edgeAlongX(5), edgeAlongX(5), 12)
	if stdmath.Abs(rep.MaxGap-delta) > 1e-9 {
		t.Errorf("MaxGap = %g, want %g", rep.MaxGap, delta)
	}
	if rep.MaxNormalDeg > 1e-9 || rep.MaxCurvDiff > 1e-9 {
		t.Errorf("a pure positional gap should not report G1/G2 deviation: deg=%g curv=%g", rep.MaxNormalDeg, rep.MaxCurvDiff)
	}
}

func TestCrossEdgeReportsTangentBreak(t *testing.T) {
	// A crease: plane A (normal +Z) meets plane B tilted by 30° about the shared x-axis edge.
	const deg = 30.0
	rad := deg * stdmath.Pi / 180
	a := xyPlane(t, 0)
	b, err := geom.NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, math.Scalar(stdmath.Cos(rad)), math.Scalar(stdmath.Sin(rad))))
	if err != nil {
		t.Fatalf("tilted plane: %v", err)
	}
	rep := CrossEdgeContinuity(a, b, edgeAlongX(5), edgeAlongX(5), 12)
	if stdmath.Abs(rep.MaxNormalDeg-deg) > 0.5 {
		t.Errorf("crease G1 angle = %g°, want ~%g°", rep.MaxNormalDeg, deg)
	}
	if rep.MaxGap > 1e-9 || rep.MaxCurvDiff > 1e-9 {
		t.Errorf("two flat faces should report no G0 gap or G2 curvature difference: gap=%g curv=%g", rep.MaxGap, rep.MaxCurvDiff)
	}
}

func TestCrossEdgeReportsCurvatureBreak(t *testing.T) {
	// A plane tangent to a cylinder along the cylinder's top line: G1-continuous (shared tangent
	// plane) but G2-discontinuous — the cylinder bends (1/R) where the plane is flat.
	const r = 2.0
	cyl, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	plane, err := geom.NewPlaneFromAxes(math.P3(0, 0, math.Scalar(r)), math.V3(1, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("tangent plane: %v", err)
	}
	// Top line of the cylinder is (x, 0, r): on the plane u=x,v=0; on the cylinder u=0 (angle), v=x.
	edgePlane := func(t float64) (u, v float64) { return t * 5, 0 }
	edgeCyl := func(t float64) (u, v float64) { return 0, t * 5 }
	rep := CrossEdgeContinuity(plane, cyl, edgePlane, edgeCyl, 12)
	if rep.MaxNormalDeg > 0.01 || rep.MaxGap > 1e-9 {
		t.Errorf("a tangent plane should be G0/G1 continuous with the cylinder: gap=%g deg=%g", rep.MaxGap, rep.MaxNormalDeg)
	}
	if stdmath.Abs(rep.MaxCurvDiff-1/r) > 1e-3 {
		t.Errorf("G2 curvature difference = %g, want ~%g (1/R)", rep.MaxCurvDiff, 1/r)
	}
	if stdmath.Abs(rep.MaxCurvPct-100) > 1 {
		t.Errorf("plane-vs-cylinder curvature is fully discontinuous: pct = %g, want ~100", rep.MaxCurvPct)
	}
}

func TestCrossEdgeIsDeterministic(t *testing.T) {
	a := xyPlane(t, 0)
	b, _ := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 0, 1), 3)
	r1 := CrossEdgeContinuity(a, b, func(t float64) (float64, float64) { return t * 4, 0 }, func(t float64) (float64, float64) { return 0, t * 4 }, 20)
	r2 := CrossEdgeContinuity(a, b, func(t float64) (float64, float64) { return t * 4, 0 }, func(t float64) (float64, float64) { return 0, t * 4 }, 20)
	if r1.MaxCurvDiff != r2.MaxCurvDiff || r1.AvgCurvDiff != r2.AvgCurvDiff || len(r1.Samples) != len(r2.Samples) {
		t.Error("CrossEdgeContinuity must be deterministic")
	}
}
