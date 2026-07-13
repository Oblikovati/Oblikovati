// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
)

// TestLinearExtrusionEllipsePerpendicular maps a SURFACE_OF_LINEAR_EXTRUSION of an ELLIPSE swept
// along its own normal (the non-oblique base case) to a right elliptical cylinder that recovers the
// profile's semi-axes exactly.
func TestLinearExtrusionEllipsePerpendicular(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n#3=DIRECTION('',(1.,0.,0.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=ELLIPSE('',#4,150.,100.);\n"+
		"#6=DIRECTION('',(0.,0.,1.));\n#7=VECTOR('',#6,1.);\n"+
		"#8=SURFACE_OF_LINEAR_EXTRUSION('',#5,#7);")
	s, err := Surface(g, 8, 1.0)
	if err != nil {
		t.Fatalf("Surface: %v", err)
	}
	cyl, ok := s.(geom.EllipticalCylinder)
	if !ok {
		t.Fatalf("surface type = %T, want geom.EllipticalCylinder", s)
	}
	if !near(cyl.MajorRadius, 150) || !near(cyl.MinorRadius, 100) {
		t.Errorf("radii = (%g, %g), want (150, 100)", cyl.MajorRadius, cyl.MinorRadius)
	}
	if a := cyl.AxisDir.AsVector(); !near(a.Z, 1) {
		t.Errorf("axis = %v, want +Z", a)
	}
}

// TestLinearExtrusionCircleOblique maps an obliquely-swept CIRCLE (STEP case U3: circle normal X,
// sweep in the XY plane at 45°) to a right elliptical cylinder. The area invariant π·majorR·minorR
// = π·r²·|d·n| and the foreshortened minor radius r·|d·n| are the correctness oracles.
func TestLinearExtrusionCircleOblique(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(1.,0.,0.));\n#3=DIRECTION('',(0.,0.,1.));\n"+
		"#4=AXIS2_PLACEMENT_3D('',#1,#2,#3);\n#5=CIRCLE('',#4,12.);\n"+
		"#6=DIRECTION('',(0.707106781187,-0.707106781187,0.));\n#7=VECTOR('',#6,1.);\n"+
		"#8=SURFACE_OF_LINEAR_EXTRUSION('',#5,#7);")
	s, err := Surface(g, 8, 1.0)
	if err != nil {
		t.Fatalf("Surface: %v", err)
	}
	cyl, ok := s.(geom.EllipticalCylinder)
	if !ok {
		t.Fatalf("surface type = %T, want geom.EllipticalCylinder", s)
	}
	dn := stdmath.Abs(0.707106781187) // |d·n|, n = X
	if !near(cyl.MajorRadius, 12) {
		t.Errorf("major radius = %g, want unchanged r = 12", cyl.MajorRadius)
	}
	if !near(cyl.MinorRadius, 12*dn) {
		t.Errorf("minor radius = %g, want foreshortened r·|d·n| = %g", cyl.MinorRadius, 12*dn)
	}
}

// TestLinearExtrusionUnsupportedProfile pins the honest fallback: a SURFACE_OF_LINEAR_EXTRUSION whose
// profile is not a conic (here a LINE) is out of scope and returns ErrUnsupportedSurface so the face
// is skipped rather than mis-built.
func TestLinearExtrusionUnsupportedProfile(t *testing.T) {
	g := graphOf(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(1.,0.,0.));\n#3=VECTOR('',#2,1.);\n#4=LINE('',#1,#3);\n"+
		"#5=DIRECTION('',(0.,0.,1.));\n#6=VECTOR('',#5,1.);\n"+
		"#7=SURFACE_OF_LINEAR_EXTRUSION('',#4,#6);")
	if _, err := Surface(g, 7, 1.0); err == nil {
		t.Fatal("expected ErrUnsupportedSurface for a non-conic extrusion profile, got nil")
	}
}
