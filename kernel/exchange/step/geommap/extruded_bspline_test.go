// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// bsplineExtrusionGraph is a SURFACE_OF_LINEAR_EXTRUSION whose profile is a degree-1 B-spline (a
// segment from (0,0,0) to (10,0,0)) swept along +Z — the minimal B-spline-profile extrusion.
func bsplineExtrusionGraph(t *testing.T) (profile geom.BSplineCurve, dir math.Vector3) {
	t.Helper()
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n#2=CARTESIAN_POINT('',(10.,0.,0.));\n"+
		"#3=B_SPLINE_CURVE_WITH_KNOTS('',1,(#1,#2),.UNSPECIFIED.,.F.,.F.,(2,2),(0.,1.),.PIECEWISE_BEZIER_KNOTS.);\n"+
		"#4=DIRECTION('',(0.,0.,1.));\n#5=VECTOR('',#4,1.);\n"+
		"#6=SURFACE_OF_LINEAR_EXTRUSION('',#3,#5);")
	profile, dir, ok, err := LinearExtrusionBSpline(g, 6, 1.0)
	if err != nil || !ok {
		t.Fatalf("LinearExtrusionBSpline ok=%v err=%v, want a detected B-spline extrusion", ok, err)
	}
	return profile, dir
}

// TestLinearExtrusionBSplineDetect confirms a B-spline-profile extrusion is detected (unlike the conic
// profile, which stays on the elliptical-cylinder path) and its profile and sweep direction returned.
func TestLinearExtrusionBSplineDetect(t *testing.T) {
	profile, dir := bsplineExtrusionGraph(t)
	if len(profile.Ctrl) != 2 {
		t.Fatalf("profile control points = %d, want 2", len(profile.Ctrl))
	}
	if a := dir; !near(a.X, 0) || !near(a.Y, 0) || !near(a.Z, 1) {
		t.Errorf("sweep direction = %v, want +Z", dir)
	}
}

// TestLinearExtrusionBSplineConicSkips pins that a conic profile is NOT taken by the B-spline path
// (ok=false), so the exact elliptical-cylinder mapping is untouched (do-no-harm).
func TestLinearExtrusionBSplineConicSkips(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n#2=DIRECTION('',(0.,0.,1.));\n"+
		"#3=DIRECTION('',(1.,0.,0.));\n#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=CIRCLE('',#4,12.);\n"+
		"#6=DIRECTION('',(0.,0.,1.));\n#7=VECTOR('',#6,1.);\n#8=SURFACE_OF_LINEAR_EXTRUSION('',#5,#7);")
	if _, _, ok, err := LinearExtrusionBSpline(g, 8, 1.0); ok || err != nil {
		t.Fatalf("conic profile: ok=%v err=%v, want ok=false (stays on elliptical-cylinder path)", ok, err)
	}
}

// TestNewExtrudedBSplineSurface checks the extrusion is the exact ruled surface S(u,v)=C(u)+v·d: the
// v=lo row reproduces the profile and v=hi row is the profile translated by (hi-lo)·d.
func TestNewExtrudedBSplineSurface(t *testing.T) {
	profile, dir := bsplineExtrusionGraph(t)
	s, err := NewExtrudedBSplineSurface(profile, dir, 0, 5)
	if err != nil {
		t.Fatalf("NewExtrudedBSplineSurface: %v", err)
	}
	if _, isBSpline := s.(geom.BSplineSurface); !isBSpline {
		t.Fatalf("surface type = %T, want geom.BSplineSurface", s)
	}
	base := s.PointAt(0.5, 0) // midpoint of the profile segment, v=0
	if !near(base.X, 5) || !near(base.Y, 0) || !near(base.Z, 0) {
		t.Errorf("S(0.5,0) = %v, want profile midpoint (5,0,0)", base)
	}
	top := s.PointAt(0.5, 5) // swept 5 along +Z
	if !near(top.X, 5) || !near(top.Y, 0) || !near(top.Z, 5) {
		t.Errorf("S(0.5,5) = %v, want (5,0,5)", top)
	}
}

// TestNewExtrudedBSplineSurfaceRangeGuard rejects a non-increasing sweep range rather than building a
// degenerate patch (the caller then warns and skips the face honestly).
func TestNewExtrudedBSplineSurfaceRangeGuard(t *testing.T) {
	profile, dir := bsplineExtrusionGraph(t)
	if _, err := NewExtrudedBSplineSurface(profile, dir, 3, 3); err == nil {
		t.Fatal("expected error for a non-increasing sweep range [3,3], got nil")
	}
}
