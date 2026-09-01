// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// The per-face analytic oracle (kernel/ops/query) certified on bodies only the general
// boolean driver can build — a curved trim from a real cut, not a primitive.

// TestFaceInteriorPointLandsInsideEveryTrim: the probe a per-face gate classifies must really be on
// the face it came from, on every face of a body with curved trims.
func TestFaceInteriorPointLandsInsideEveryTrim(t *testing.T) {
	t.Parallel()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3, 4.5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := Boolean(Cut, rod, ball)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	for i, f := range res.Faces() {
		p, ok := query.FaceInteriorPoint(f)
		if !ok {
			t.Errorf("face %d (%T) yielded no interior point", i, f.Geometry())
			continue
		}
		if !brep.PointInFaceTrim(f, p) {
			t.Errorf("face %d interior point %v is outside its own trim", i, p)
		}
	}
}

// TestDrilledPlateIntegratesAnalytically is the periodic-band regression (#3453): a plate with a
// hole in it is the most ordinary body in CAD, and its bore wall is a band whose loop crosses the
// parameter seam. Green's u-form cannot close such a loop, so the whole body used to fall back to
// the tessellation; the conjugate v-form integrates it exactly.
func TestDrilledPlateIntegratesAnalytically(t *testing.T) {
	t.Parallel()
	plate, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := Boolean(Cut, plate, drill)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	gp, ok := query.AnalyticGeometryProperties(res)
	if !ok {
		t.Fatal("the drilled plate declined analytic integration; its bore wall is a periodic band")
	}
	want := 10*10*2 - stdmath.Pi*2*2*2
	if analyticRelDiff(gp.Volume, want) > analyticQuadRelTol {
		t.Errorf("drilled plate volume = %.9f, want %.9f", gp.Volume, want)
	}
}

// TestFaceInteriorPointOnSeamWrappingBand is the regression gate for the cone∩box false reject
// (Oblikovati/Oblikovati#3447). The kept cone side is a BAND bounded by two loops that each circle the
// parameter seam a full turn, so neither is a closed polygon in the plane and neither lands in the
// other's branch. query.FaceInteriorPoint used to run the plain even-odd grid on them anyway and returned a
// point in the band the operation DISCARDED — below the cutting plane — which the per-face boolean
// certificate then correctly refused, demoting a correct analytic result to faceted CSG. The probe
// must be on its own face or absent, never on the far side.
func TestFaceInteriorPointOnSeamWrappingBand(t *testing.T) {
	t.Parallel()
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 6, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	box, err := brep.SolidBlock(math.P3(-20, -20, 4), math.P3(20, 20, 30), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	res, err := Boolean(Intersect, cone, box)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	for i, f := range res.Faces() {
		p, ok := query.FaceInteriorPoint(f)
		if !ok {
			continue // a probe the reconstruction cannot place is skipped, never guessed
		}
		if !brep.PointInFaceTrim(f, p) {
			t.Errorf("face %d (%T) interior point %v is outside its own trim", i, f.Geometry(), p)
		}
		if float64(p.Z) < 4 {
			t.Errorf("face %d (%T) interior point %v is below the cutting plane z=4 — outside the intersection", i, f.Geometry(), p)
		}
	}
}

// TestAnalyticShellVolumeSignsTheCavity: a void shell's material-outward normals point INTO the
// cavity, so its signed volume is negative — and now exactly −8, not −8±0.05 as the merged face
// mesh reported.
func TestAnalyticShellVolumeSignsTheCavity(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	var outer, void float64
	for _, sh := range body.Shells() {
		v, ok := query.AnalyticShellVolume(sh)
		if !ok {
			t.Fatalf("shell %d declined analytic integration", sh.Index())
		}
		if v < 0 {
			void = v
		} else {
			outer = v
		}
	}
	if analyticRelDiff(outer, 64) > analyticQuadRelTol {
		t.Errorf("outer shell volume = %.12f, want 64", outer)
	}
	if analyticRelDiff(void, -8) > analyticQuadRelTol {
		t.Errorf("void shell volume = %.12f, want -8", void)
	}
}

// analyticRelDiff is |got-want|/|want|, the scale-free comparison these closed-form checks need.
func analyticRelDiff(got, want float64) float64 {
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// analyticQuadRelTol is what "exact" means here: the adaptive quadrature converges to its own
// relative tolerance, so an analytic value agrees with the closed form to a few ulps of that.
const analyticQuadRelTol = 1e-9
