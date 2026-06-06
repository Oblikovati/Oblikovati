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

func TestCurveUnsupported(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n#2=POLYLINE('',(#1,#1));")
	_, err := Curve(g, 2, 1.0)
	var unsup ErrUnsupportedCurve
	if !errors.As(err, &unsup) {
		t.Fatalf("got %v, want ErrUnsupportedCurve", err)
	}
}
