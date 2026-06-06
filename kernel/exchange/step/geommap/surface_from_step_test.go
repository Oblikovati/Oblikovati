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
}
