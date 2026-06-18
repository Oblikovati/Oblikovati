// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestAnalysisMassPropertiesOverWire drives the mass-properties surface: the box-part fixture
// (a 40×30×50 mm block) reports its volume/area/mass through the live router→model→kernel stack.
func TestAnalysisMassPropertiesOverWire(t *testing.T) {
	r, s := boxPartSession(t)

	var mp wire.MassPropertiesResult
	call(t, r, s, "analysis.massProperties", `{"densityGCm3":2}`, &mp)

	// 40×30×50 mm = 60000 mm³; surface area 2(40·30+40·50+30·50) = 9400 mm²; mass = 2 g/cm³ × 60 cm³ = 120 g.
	if math.Abs(mp.VolumeMm3-60000) > 1 {
		t.Errorf("volume = %g mm³, want 60000", mp.VolumeMm3)
	}
	if math.Abs(mp.SurfaceAreaMm2-9400) > 1 {
		t.Errorf("surface area = %g mm², want 9400", mp.SurfaceAreaMm2)
	}
	if math.Abs(mp.MassG-120) > 1e-3 {
		t.Errorf("mass = %g g, want 120", mp.MassG)
	}
	// Inertia is populated; principal moments positive and sorted ascending.
	if mp.InertiaXxGmm2 <= 0 || mp.PrincipalMomentsGmm2[0] <= 0 || mp.PrincipalMomentsGmm2[0] > mp.PrincipalMomentsGmm2[2] {
		t.Errorf("inertia = %+v, want positive Ixx + ascending principal moments", mp)
	}

	// A bad accuracy errors; with no active part, the method errors.
	if _, err := r.Handle(s, "analysis.massProperties", []byte(`{"accuracy":"bogus"}`)); err == nil {
		t.Error("massProperties with a bad accuracy = ok, want error")
	}
	br, bs := New(opregistry.Default()), app.NewSession()
	if _, err := br.Handle(bs, "analysis.massProperties", []byte(`{}`)); err == nil {
		t.Error("massProperties with no active part = ok, want error")
	}
}
