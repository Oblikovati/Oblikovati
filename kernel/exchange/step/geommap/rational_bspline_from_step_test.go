// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati/kernel/geom"
)

// TestRationalBSplineCurveFromComplexInstance maps a SolidWorks-style rational b-spline
// curve — a STEP complex instance whose geometry is split across BOUNDED_CURVE /
// B_SPLINE_CURVE / B_SPLINE_CURVE_WITH_KNOTS / RATIONAL_B_SPLINE_CURVE components (no
// per-component name) — and checks the degree, control points, knots, and the non-unit
// weights all land. This is the form a real STEP export uses for every freeform curve.
func TestRationalBSplineCurveFromComplexInstance(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(2.,1.,0.));\n"+
		"#4=CARTESIAN_POINT('',(3.,1.,0.));\n"+
		"#5=( BOUNDED_CURVE() B_SPLINE_CURVE(3,(#1,#2,#3,#4),.UNSPECIFIED.,.F.,.F.) "+
		"B_SPLINE_CURVE_WITH_KNOTS((4,4),(0.,1.),.UNSPECIFIED.) CURVE() "+
		"GEOMETRIC_REPRESENTATION_ITEM() RATIONAL_B_SPLINE_CURVE((1.,0.8,0.8,1.)) "+
		"REPRESENTATION_ITEM('') );")
	mc, err := Curve(g, 5, 10.0)
	if err != nil {
		t.Fatalf("rational B-spline curve: %v", err)
	}
	if mc.Kind != CurveBSpline {
		t.Fatalf("mapped to kind %d, want CurveBSpline", mc.Kind)
	}
	bc := mc.BSpline
	if bc.Degree != 3 || len(bc.Ctrl) != 4 || len(bc.Knots) != 8 || len(bc.Weights) != 4 {
		t.Fatalf("shape: degree %d ctrl %d knots %d weights %d", bc.Degree, len(bc.Ctrl), len(bc.Knots), len(bc.Weights))
	}
	if !near(bc.Weights[1], 0.8) {
		t.Errorf("weight[1] = %v, want 0.8 (rational weights must be read)", bc.Weights[1])
	}
	if !near(float64(bc.Ctrl[3].X), 30) { // scaled by 10
		t.Errorf("scaled control point = %v, want X=30", bc.Ctrl[3])
	}
}

// TestRationalBSplineSurfaceFromComplexInstance maps a rational b-spline surface complex
// instance and checks the degrees, control net, and non-unit weights land.
func TestRationalBSplineSurfaceFromComplexInstance(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(0.,1.,0.));\n"+
		"#4=CARTESIAN_POINT('',(1.,1.,1.));\n"+
		"#5=( BOUNDED_SURFACE() B_SPLINE_SURFACE(1,1,((#1,#2),(#3,#4)),.UNSPECIFIED.,.F.,.F.,.F.) "+
		"B_SPLINE_SURFACE_WITH_KNOTS((2,2),(2,2),(0.,1.),(0.,1.),.UNSPECIFIED.) "+
		"GEOMETRIC_REPRESENTATION_ITEM() RATIONAL_B_SPLINE_SURFACE(((1.,0.9),(0.9,1.))) "+
		"REPRESENTATION_ITEM('') SURFACE() );")
	s, err := Surface(g, 5, 10.0)
	if err != nil {
		t.Fatalf("rational B-spline surface: %v", err)
	}
	bs, ok := s.(geom.BSplineSurface)
	if !ok {
		t.Fatalf("mapped to %T, want BSplineSurface", s)
	}
	if bs.UDegree != 1 || bs.VDegree != 1 || len(bs.Ctrl) != 2 || len(bs.Ctrl[0]) != 2 {
		t.Fatalf("shape: degree %d/%d net %dx%d", bs.UDegree, bs.VDegree, len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	if !near(bs.Weights[0][1], 0.9) {
		t.Errorf("weight[0][1] = %v, want 0.9", bs.Weights[0][1])
	}
	if !near(float64(bs.Ctrl[1][1].Z), 10) { // scaled by 10
		t.Errorf("scaled control point = %v, want Z=10", bs.Ctrl[1][1])
	}
}
