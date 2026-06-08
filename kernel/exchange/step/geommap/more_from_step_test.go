// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati/kernel/geom"
)

// A SURFACE_CURVE/SEAM_CURVE/INTERSECTION_CURVE/TRIMMED_CURVE is a carrier: Curve must unwrap
// it to its basis 3D curve at parameter 1 (OpenCASCADE emits every edge this way).
func TestCurveSurfaceCurveUnwrapsBasis(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=CIRCLE('',#4,3.);\n"+
		"#6=SURFACE_CURVE('',#5,(),.PCURVE_S1.);")
	mc, err := Curve(g, 6, 2.0)
	if err != nil {
		t.Fatalf("Curve SURFACE_CURVE: %v", err)
	}
	if mc.Kind != CurveCircle || mc.Circle.Radius != 6.0 {
		t.Fatalf("SURFACE_CURVE did not unwrap to a scaled circle: kind=%d r=%g", mc.Kind, mc.Circle.Radius)
	}
}

func TestCurveTrimmedCurveUnwrapsBasis(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(1.,0.,0.));\n#3=VECTOR('',#2,1.);\n#4=LINE('',#1,#3);\n"+
		"#5=TRIMMED_CURVE('',#4,(PARAMETER_VALUE(0.)),(PARAMETER_VALUE(1.)),.T.,.PARAMETER.);")
	mc, err := Curve(g, 5, 1.0)
	if err != nil {
		t.Fatalf("Curve TRIMMED_CURVE: %v", err)
	}
	if mc.Kind != CurveLine {
		t.Fatalf("TRIMMED_CURVE did not unwrap to its basis line: kind=%d", mc.Kind)
	}
}

func TestCurveEllipseParamsScaled(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=ELLIPSE('',#4,4.,2.);")
	mc, err := Curve(g, 5, 3.0)
	if err != nil {
		t.Fatalf("Curve ELLIPSE: %v", err)
	}
	if mc.Kind != CurveEllipse {
		t.Fatalf("ELLIPSE mapped to kind %d, want CurveEllipse", mc.Kind)
	}
	if mc.Ellipse.Major != 12.0 || mc.Ellipse.Minor != 6.0 {
		t.Errorf("ellipse semi-axes = (%g,%g), want (12,6)", mc.Ellipse.Major, mc.Ellipse.Minor)
	}
}

func TestCurvePlainBSplineImplicitKnots(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n#3=CARTESIAN_POINT('',(1.,1.,0.));\n"+
		"#4=B_SPLINE_CURVE('',2,(#1,#2,#3),.UNSPECIFIED.,.F.,.F.);")
	mc, err := Curve(g, 4, 1.0)
	if err != nil {
		t.Fatalf("Curve B_SPLINE_CURVE: %v", err)
	}
	// degree 2, 3 control points → clamped knots [0,0,0,1,1,1] (n+degree+1 = 6).
	if mc.Kind != CurveBSpline || mc.BSpline.Degree != 2 || len(mc.BSpline.Knots) != 6 {
		t.Fatalf("plain B-spline shape: kind=%d degree=%d knots=%d", mc.Kind, mc.BSpline.Degree, len(mc.BSpline.Knots))
	}
}

func TestCurvePolyline(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n#3=CARTESIAN_POINT('',(1.,1.,0.));\n"+
		"#4=POLYLINE('',(#1,#2,#3));")
	mc, err := Curve(g, 4, 1.0)
	if err != nil {
		t.Fatalf("Curve POLYLINE: %v", err)
	}
	if mc.Kind != CurvePolyline {
		t.Fatalf("POLYLINE mapped to kind %d, want CurvePolyline", mc.Kind)
	}
}

func TestImplicitKnotsForms(t *testing.T) {
	cases := []struct {
		form        string
		degree, n   int
		wantLen     int
		wantClamped bool // first knot repeated degree+1 times
	}{
		{"BEZIER_CURVE", 3, 4, 8, true},
		{"QUASI_UNIFORM_CURVE", 2, 5, 8, true},
		{"B_SPLINE_SURFACE", 2, 5, 8, true},
		{"UNIFORM_CURVE", 2, 5, 8, false},
	}
	for _, c := range cases {
		k := implicitKnots(c.form, c.degree, c.n)
		if len(k) != c.wantLen {
			t.Errorf("%s: knots len %d, want %d (%v)", c.form, len(k), c.wantLen, k)
		}
		clamped := k[0] == k[c.degree]
		if clamped != c.wantClamped {
			t.Errorf("%s: clamped=%v, want %v (%v)", c.form, clamped, c.wantClamped, k)
		}
	}
}

func TestSurfacePlainBSplineImplicitKnots(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(0.,1.,0.));\n#4=CARTESIAN_POINT('',(1.,1.,1.));\n"+
		"#5=B_SPLINE_SURFACE('',1,1,((#1,#2),(#3,#4)),.UNSPECIFIED.,.F.,.F.,.F.);")
	s, err := Surface(g, 5, 1.0)
	if err != nil {
		t.Fatalf("Surface B_SPLINE_SURFACE: %v", err)
	}
	if s == nil {
		t.Fatal("plain B-spline surface mapped to nil")
	}
}

func TestSurfaceRectangularTrimmedUnwrapsBasis(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=PLANE('',#4);\n"+
		"#6=RECTANGULAR_TRIMMED_SURFACE('',#5,0.,1.,0.,1.,.T.,.T.);")
	s, err := Surface(g, 6, 1.0)
	if err != nil {
		t.Fatalf("Surface RECTANGULAR_TRIMMED_SURFACE: %v", err)
	}
	if _, ok := s.(geom.Plane); !ok {
		t.Fatalf("RECTANGULAR_TRIMMED_SURFACE unwrapped to %T, want geom.Plane (its basis)", s)
	}
}
