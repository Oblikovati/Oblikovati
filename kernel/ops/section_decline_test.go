// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// A face whose surface∩plane section the intersector cannot resolve has its section SLICED FROM ITS
// MESH — a chord approximation whose vertices are facet corners, on the wrong side of "no modelling
// decision reads tessellated data". It rode out of here in silence. geom.SurfaceIntersectDeclining now
// names the gate that refused, and SectionWithPlane carries it on the body the caller gets
// (Oblikovati/Oblikovati#3525).

// TestAMeshedSectionFaceSaysSoOnTheBody: an axis-parallel plane through a cone's apex region is the
// live case — the axis-parallel hyperbola is degenerate there, the marcher finds no curve, and the
// cone's wall falls to its mesh.
func TestAMeshedSectionFaceSaysSoOnTheBody(t *testing.T) {
	t.Parallel()
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 6), 3, 0.2, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	sec, err := SectionWithPlane(cone, math.P3(0.5, 0, 1), math.V3(1, 0, 0), DefaultQuality())
	if err != nil {
		t.Fatalf("SectionWithPlane: %v", err)
	}
	d := onlyDiagWithCode(t, sec.BuildDiagnostics(), CodeSectionFaceMeshed)
	if d.Severity != diag.Defect {
		t.Errorf("a meshed section face is %v; the curve is a facet chord where an exact one belongs", d.Severity)
	}
	for _, want := range []string{"geom.Cone", "cone", "sliced from the face's MESH", "no closed form claims"} {
		if !strings.Contains(d.Detail, want) {
			t.Errorf("the meshed-section defect does not name %q: %s", want, d.Detail)
		}
	}
}

// TestAnExactSectionCarriesNoDefect: the same cone cut PERPENDICULAR to its axis is an exact circle,
// and a diagnostic that fired on the ordinary case would be noise.
func TestAnExactSectionCarriesNoDefect(t *testing.T) {
	t.Parallel()
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 6), 3, 0.2, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	sec, err := SectionWithPlane(cone, math.P3(0, 0, 3), math.V3(0, 0, 1), DefaultQuality())
	if err != nil {
		t.Fatalf("SectionWithPlane: %v", err)
	}
	if got := sec.BuildDiagnostics(); len(got) != 0 {
		t.Errorf("an exact analytic section reports %v, want nothing", got)
	}
}

// onlyDiagWithCode returns the first diagnostic carrying code, failing when there is none.
func onlyDiagWithCode(t *testing.T, ds []diag.Diagnostic, code diag.Code) diag.Diagnostic {
	t.Helper()
	for _, d := range ds {
		if d.Code == code {
			return d
		}
	}
	t.Fatalf("no %s diagnostic; the body carries %v", code, ds)
	return diag.Diagnostic{}
}
