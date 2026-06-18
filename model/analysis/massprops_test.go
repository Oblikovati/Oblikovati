// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

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
	mp := MassPropertiesOf([]*topo.Body{block53x2(t)}, 2.0) // density 2 g/cm³

	// 50×30×20 mm: volume 30000 mm³, area 2(50·30+50·20+30·20)=6200 mm², centroid (25,15,10) mm.
	approx(t, "volume", mp.VolumeMm3, 30000)
	approx(t, "area", mp.SurfaceAreaMm2, 6200)
	approx(t, "centroidX", mp.CentroidXMm, 25)
	approx(t, "centroidY", mp.CentroidYMm, 15)
	approx(t, "centroidZ", mp.CentroidZMm, 10)
	approx(t, "mass", mp.MassG, 60) // 2 g/cm³ × 30 cm³

	def := MassPropertiesOf([]*topo.Body{block53x2(t)}, 0) // zero density ⇒ 1 g/cm³
	if def.DensityGCm3 != 1 {
		t.Errorf("default density = %g, want 1", def.DensityGCm3)
	}
	approx(t, "default mass", def.MassG, 30)
}

// TestMassPropertiesEmpty: no bodies yields zero properties (and no divide-by-zero centroid).
func TestMassPropertiesEmpty(t *testing.T) {
	mp := MassPropertiesOf(nil, 5)
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
