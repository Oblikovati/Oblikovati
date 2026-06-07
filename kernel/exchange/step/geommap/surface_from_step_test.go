// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"errors"
	"testing"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
)

// graphOf parses DATA statements into an EntityGraph for mapper tests.
func graphOf(t *testing.T, stmts string) *part21.EntityGraph {
	t.Helper()
	src := "ISO-10303-21;\nHEADER;\nFILE_DESCRIPTION((''),'');\n" +
		"FILE_NAME('','',(''),(''),'','','');\nFILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));\n" +
		"ENDSEC;\nDATA;\n" + stmts + "\nENDSEC;\nEND-ISO-10303-21;\n"
	f, err := part21.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f.Graph
}

func TestCartesianPointScaling(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(1.,2.,3.));")
	p, err := CartesianPoint(g, 1, 25.4) // inches → mm
	if err != nil {
		t.Fatalf("CartesianPoint: %v", err)
	}
	if !near(p.X, 25.4) || !near(p.Y, 50.8) || !near(p.Z, 76.2) {
		t.Errorf("scaled point = %v, want (25.4,50.8,76.2)", p)
	}
}

func TestDirectionRejectsZero(t *testing.T) {
	g := graphOf(t, "#1=DIRECTION('',(0.,0.,0.));")
	if _, err := Direction(g, 1); err == nil {
		t.Error("zero-length direction should error")
	}
}

func TestPlacementDefaultsAndOrthogonality(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,1.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);")
	f, err := Placement(g, 4, 1.0)
	if err != nil {
		t.Fatalf("Placement: %v", err)
	}
	if dot := f.AxisZ.Dot(f.AxisX); dot > 1e-12 || dot < -1e-12 {
		t.Errorf("AxisX·AxisZ = %g, want orthogonal (0)", dot)
	}
}

func TestPlacementDefaultsAndParallelRefFallback(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(1.,2.,3.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n"+
		"#3=AXIS2_PLACEMENT_3D('',#1,$,$);\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#2);")
	f, err := Placement(g, 3, 2.0)
	if err != nil {
		t.Fatalf("Placement defaults: %v", err)
	}
	if !near(float64(f.Origin.X), 2) || !near(float64(f.Origin.Y), 4) || !near(float64(f.Origin.Z), 6) {
		t.Fatalf("scaled default placement origin = %v", f.Origin)
	}
	f, err = Placement(g, 4, 1.0)
	if err != nil {
		t.Fatalf("Placement parallel ref: %v", err)
	}
	if f.AxisX.LengthSquared() == 0 || !near(f.AxisZ.Dot(f.AxisX), 0) {
		t.Fatalf("parallel ref fallback AxisZ·AxisX = %g AxisX=%v", f.AxisZ.Dot(f.AxisX), f.AxisX)
	}
}

func TestSurfacePlane(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=PLANE('',#4);")
	s, err := Surface(g, 5, 1.0)
	if err != nil {
		t.Fatalf("Surface PLANE: %v", err)
	}
	if _, ok := s.(geom.Plane); !ok {
		t.Errorf("PLANE mapped to %T, want geom.Plane", s)
	}
}

func TestSurfaceCylinderRadiusScaled(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=CYLINDRICAL_SURFACE('',#4,2.);")
	s, err := Surface(g, 5, 25.4)
	if err != nil {
		t.Fatalf("Surface CYLINDER: %v", err)
	}
	cyl, ok := s.(geom.Cylinder)
	if !ok {
		t.Fatalf("mapped to %T, want geom.Cylinder", s)
	}
	if want := 2.0 * 25.4; !near(cyl.Radius, want) {
		t.Errorf("cylinder radius = %g, want %g", cyl.Radius, want)
	}
}

func TestAnalyticSurfaceVariantsFromStep(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n"+
		"#5=CONICAL_SURFACE('',#4,2.,0.7853981633974483);\n"+
		"#6=SPHERICAL_SURFACE('',#4,3.);\n"+
		"#7=TOROIDAL_SURFACE('',#4,4.,1.);")
	for _, tc := range []struct {
		id   int
		want any
	}{
		{5, geom.Cone{}},
		{6, geom.Sphere{}},
		{7, geom.Torus{}},
	} {
		s, err := Surface(g, tc.id, 2.0)
		if err != nil {
			t.Fatalf("Surface #%d: %v", tc.id, err)
		}
		switch tc.want.(type) {
		case geom.Cone:
			if _, ok := s.(geom.Cone); !ok {
				t.Fatalf("#%d mapped to %T, want Cone", tc.id, s)
			}
		case geom.Sphere:
			sp, ok := s.(geom.Sphere)
			if !ok || !near(sp.Radius, 6) {
				t.Fatalf("#%d mapped to %T radius %v, want Sphere radius 6", tc.id, s, sp.Radius)
			}
		case geom.Torus:
			to, ok := s.(geom.Torus)
			if !ok || !near(to.MajorRadius, 8) || !near(to.MinorRadius, 2) {
				t.Fatalf("#%d mapped to %T radii %v/%v, want Torus 8/2", tc.id, s, to.MajorRadius, to.MinorRadius)
			}
		}
	}
}

func TestBSplineSurfaceWithKnotsFromStep(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(0.,1.,0.));\n"+
		"#4=CARTESIAN_POINT('',(1.,1.,1.));\n"+
		"#5=B_SPLINE_SURFACE_WITH_KNOTS(1,1,((#1,#2),(#3,#4)),.UNSPECIFIED.,.F.,.F.,.F.,(2,2),(2,2),(0.,1.),(0.,1.),.UNSPECIFIED.);")
	s, err := Surface(g, 5, 10.0)
	if err != nil {
		t.Fatalf("Surface B_SPLINE_SURFACE_WITH_KNOTS: %v", err)
	}
	bs, ok := s.(geom.BSplineSurface)
	if !ok {
		t.Fatalf("mapped to %T, want BSplineSurface", s)
	}
	if bs.UDegree != 1 || bs.VDegree != 1 || len(bs.Ctrl) != 2 || len(bs.Ctrl[0]) != 2 {
		t.Fatalf("unexpected bspline shape: degree %d/%d net %dx%d", bs.UDegree, bs.VDegree, len(bs.Ctrl), len(bs.Ctrl[0]))
	}
	if !near(float64(bs.Ctrl[1][1].X), 10) || !near(float64(bs.Ctrl[1][1].Y), 10) || !near(float64(bs.Ctrl[1][1].Z), 10) {
		t.Fatalf("scaled control point = %v, want (10,10,10)", bs.Ctrl[1][1])
	}
}

func TestBSplineSurfaceFromStepRejectsMalformedKnots(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=CARTESIAN_POINT('',(1.,0.,0.));\n"+
		"#3=CARTESIAN_POINT('',(0.,1.,0.));\n"+
		"#4=CARTESIAN_POINT('',(1.,1.,1.));\n"+
		"#5=B_SPLINE_SURFACE_WITH_KNOTS(1,1,((#1,#2),(#3,#4)),.UNSPECIFIED.,.F.,.F.,.F.,(2),(2,2),(0.,1.),(0.,1.),.UNSPECIFIED.);")
	if _, err := Surface(g, 5, 1.0); err == nil {
		t.Fatal("B_SPLINE_SURFACE_WITH_KNOTS accepted mismatched u knot multiplicities")
	}
}

// near reports whether a and b are within 1e-9.
func near(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

func TestSurfaceUnsupportedReportsKind(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=AXIS2_PLACEMENT_3D('',#1,$,$);\n#3=SURFACE_OF_REVOLUTION('',#2,#2);")
	_, err := Surface(g, 3, 1.0)
	var unsup ErrUnsupportedSurface
	if !errors.As(err, &unsup) {
		t.Fatalf("got %v, want ErrUnsupportedSurface", err)
	}
	if unsup.Keyword != "SURFACE_OF_REVOLUTION" {
		t.Errorf("unsupported keyword = %q, want SURFACE_OF_REVOLUTION", unsup.Keyword)
	}
	if got := unsup.Error(); got != "geommap: unsupported surface SURFACE_OF_REVOLUTION (#3)" {
		t.Fatalf("unsupported error = %q", got)
	}
}
