// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"errors"
	"testing"
)

func TestCurveLine(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(1.,0.,0.));\n#3=VECTOR('',#2,1.);\n#4=LINE('',#1,#3);")
	mc, err := Curve(g, 4, 1.0)
	if err != nil {
		t.Fatalf("Curve LINE: %v", err)
	}
	if mc.Kind != CurveLine {
		t.Errorf("LINE mapped to kind %d, want CurveLine", mc.Kind)
	}
}

func TestCurveCircleParamsScaled(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=CIRCLE('',#4,3.);")
	mc, err := Curve(g, 5, 2.0)
	if err != nil {
		t.Fatalf("Curve CIRCLE: %v", err)
	}
	if mc.Kind != CurveCircle {
		t.Fatalf("CIRCLE mapped to kind %d, want CurveCircle", mc.Kind)
	}
	if want := 3.0 * 2.0; mc.Circle.Radius != want {
		t.Errorf("circle radius = %g, want %g", mc.Circle.Radius, want)
	}
}

func TestCurveBSplineWithKnots(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(1.,1.,0.));\n"+
		"#4=B_SPLINE_CURVE_WITH_KNOTS('NONE',2,(#1,#2,#3),.UNSPECIFIED.,.F.,.F.,(3,3),(0.,1.),.UNSPECIFIED.);")
	mc, err := Curve(g, 4, 5.0)
	if err != nil {
		t.Fatalf("Curve B_SPLINE_CURVE_WITH_KNOTS: %v", err)
	}
	if mc.Kind != CurveBSpline {
		t.Fatalf("B-spline mapped to kind %d, want CurveBSpline", mc.Kind)
	}
	if mc.BSpline.Degree != 2 || len(mc.BSpline.Ctrl) != 3 || len(mc.BSpline.Knots) != 6 {
		t.Fatalf("unexpected B-spline shape: degree %d ctrl %d knots %d", mc.BSpline.Degree, len(mc.BSpline.Ctrl), len(mc.BSpline.Knots))
	}
	if !near(float64(mc.BSpline.Ctrl[2].X), 5) || !near(float64(mc.BSpline.Ctrl[2].Y), 5) {
		t.Fatalf("scaled control point = %v, want (5,5,0)", mc.BSpline.Ctrl[2])
	}
}

func TestCurveBSplineRejectsMalformedKnots(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(1.,1.,0.));\n"+
		"#4=B_SPLINE_CURVE_WITH_KNOTS('NONE',2,(#1,#2,#3),.UNSPECIFIED.,.F.,.F.,(3),(0.,1.),.UNSPECIFIED.);")
	if _, err := Curve(g, 4, 1.0); err == nil {
		t.Fatal("B_SPLINE_CURVE_WITH_KNOTS accepted mismatched knot multiplicities")
	}
}

func TestCurveUnsupported(t *testing.T) {
	// PARABOLA has no kernel curve analogue (unlike LINE/CIRCLE/ELLIPSE/B-spline/POLYLINE,
	// which geommap now maps), so it is the canonical still-unsupported curve.
	g := graphOf(t, "#1=AXIS2_PLACEMENT_3D('',#2,#3,#4);\n#2=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#3=DIRECTION('',(0.,0.,1.));\n#4=DIRECTION('',(1.,0.,0.));\n#5=PARABOLA('',#1,1.);")
	_, err := Curve(g, 5, 1.0)
	var unsup ErrUnsupportedCurve
	if !errors.As(err, &unsup) {
		t.Fatalf("got %v, want ErrUnsupportedCurve", err)
	}
	if got := unsup.Error(); got != "geommap: unsupported curve PARABOLA (#5)" {
		t.Fatalf("unsupported error = %q", got)
	}
}
