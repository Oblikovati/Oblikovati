// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// block53x2 is the shared 5×3×2 cm test solid (database units are cm; reported in mm / g).
func block53x2(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(5, 3, 2), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// TestMassPropertiesOfBox checks volume, surface area, centroid and mass against the analytic
// values for the 5×3×2 cm block, including the default-density path.
func TestMassPropertiesOfBox(t *testing.T) {
	mp := MassPropertiesOf([]*topo.Body{block53x2(t)}, 2.0, types.MassPropertiesMedium) // density 2 g/cm³

	// 50×30×20 mm: volume 30000 mm³, area 2(50·30+50·20+30·20)=6200 mm², centroid (25,15,10) mm.
	approx(t, "volume", mp.VolumeMm3, 30000)
	approx(t, "area", mp.SurfaceAreaMm2, 6200)
	approx(t, "centroidX", mp.CentroidXMm, 25)
	approx(t, "centroidY", mp.CentroidYMm, 15)
	approx(t, "centroidZ", mp.CentroidZMm, 10)
	approx(t, "mass", mp.MassG, 60) // 2 g/cm³ × 30 cm³

	// Inertia (g·mm²) about the centroid: mass m = 60 g, full dims (50,30,20) mm →
	// Ixx = m(30²+20²)/12 = 60·1300/12 = 6500; Iyy = m(50²+20²)/12 = 60·2900/12 = 14500;
	// Izz = m(50²+30²)/12 = 60·3400/12 = 17000. Products zero (axis-aligned about centroid).
	approx(t, "Ixx", mp.InertiaXxGmm2, 6500)
	approx(t, "Iyy", mp.InertiaYyGmm2, 14500)
	approx(t, "Izz", mp.InertiaZzGmm2, 17000)
	approx(t, "Ixy", mp.InertiaXyGmm2, 0)
	// Principal moments equal the (sorted-ascending) diagonal for an axis-aligned box.
	approx(t, "principal0", mp.PrincipalMomentsGmm2[0], 6500)
	approx(t, "principal1", mp.PrincipalMomentsGmm2[1], 14500)
	approx(t, "principal2", mp.PrincipalMomentsGmm2[2], 17000)

	def := MassPropertiesOf([]*topo.Body{block53x2(t)}, 0, types.MassPropertiesMedium) // zero density ⇒ 1 g/cm³
	if def.DensityGCm3 != 1 {
		t.Errorf("default density = %g, want 1", def.DensityGCm3)
	}
	approx(t, "default mass", def.MassG, 30)
}

// TestMassPropertiesCurvedMatchesAnalytic gates the tessellation-driven volume/area accuracy of a
// CURVED body (a cylinder), the case a planar box cannot exercise. Mass properties integrate over
// the tessellated mesh, so a too-coarse facet count under-reports a curved solid's volume — this is
// the −0.64%/curved-feature bias the display default produced against the Inventor analytic oracle.
// Medium (the default) must land within 0.05% and High within 0.01% of the analytic value, so the
// exporter corpus reads true volumes. If this loosens, the drift is back; do not relax the bound
// without re-checking parity.
func TestMassPropertiesCurvedMatchesAnalytic(t *testing.T) {
	const r, h = 2.0, 5.0                                    // cm
	analyticVolMm3 := math.Pi * r * r * h * 1000             // πr²h cm³ → mm³
	analyticAreaMm2 := (2*math.Pi*r*r + 2*math.Pi*r*h) * 100 // (2 caps + side) cm² → mm²
	cyl := func() *topo.Body {
		b, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), r, h)
		if err != nil {
			t.Fatalf("SolidCylinder: %v", err)
		}
		return b
	}
	for _, c := range []struct {
		acc    types.MassPropertiesAccuracy
		relTol float64
	}{
		{types.MassPropertiesMedium, 5e-4}, // default: parity-grade
		{types.MassPropertiesHigh, 1e-4},
	} {
		mp := MassPropertiesOf([]*topo.Body{cyl()}, 1, c.acc)
		if rel := math.Abs(mp.VolumeMm3-analyticVolMm3) / analyticVolMm3; rel > c.relTol {
			t.Errorf("cylinder volume@%s = %.4f mm³, want %.4f (rel err %.4f%% > %.4f%%)",
				c.acc, mp.VolumeMm3, analyticVolMm3, 100*rel, 100*c.relTol)
		}
		if rel := math.Abs(mp.SurfaceAreaMm2-analyticAreaMm2) / analyticAreaMm2; rel > c.relTol {
			t.Errorf("cylinder area@%s = %.4f mm², want %.4f (rel err %.4f%% > %.4f%%)",
				c.acc, mp.SurfaceAreaMm2, analyticAreaMm2, 100*rel, 100*c.relTol)
		}
	}
}

// TestMassPropertiesEmpty: no bodies yields zero properties (and no divide-by-zero centroid).
func TestMassPropertiesEmpty(t *testing.T) {
	mp := MassPropertiesOf(nil, 5, types.MassPropertiesMedium)
	if mp.VolumeMm3 != 0 || mp.MassG != 0 || mp.CentroidXMm != 0 {
		t.Fatalf("empty mass properties = %+v, want all zero", mp)
	}
}

func approx(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6*math.Max(1, math.Abs(want)) {
		t.Errorf("%s = %g, want %g", name, got, want)
	}
}
