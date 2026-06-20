// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestBodyPhysicalPropertiesMatchesBox: body.physicalProperties returns one body's geometry
// properties — for the 40×30×20 mm box that is a 24 000 mm³ volume (#1078).
func TestBodyPhysicalPropertiesMatchesBox(t *testing.T) {
	r, s := boxBodySession(t)
	var mp wire.MassPropertiesResult
	call(t, r, s, "body.physicalProperties", `{"bodyIndex":0}`, &mp)

	const wantVol = 40.0 * 30.0 * 20.0 // mm³
	if math.Abs(mp.VolumeMm3-wantVol) > wantVol*0.01 {
		t.Errorf("body volume = %g mm³, want ≈%g", mp.VolumeMm3, wantVol)
	}
	// Centroid of a box from the origin corner sits at its half-extents.
	if math.Abs(mp.CentroidXMm-20) > 0.5 || math.Abs(mp.CentroidYMm-15) > 0.5 || math.Abs(mp.CentroidZMm-10) > 0.5 {
		t.Errorf("centroid = (%g, %g, %g), want ≈(20, 15, 10)", mp.CentroidXMm, mp.CentroidYMm, mp.CentroidZMm)
	}
	if mp.MassG <= 0 {
		t.Errorf("mass = %g g, want a positive mass at the default density", mp.MassG)
	}
}

// TestBodyPhysicalPropertiesHonorsDensity: a supplied density scales the mass linearly.
func TestBodyPhysicalPropertiesHonorsDensity(t *testing.T) {
	r, s := boxBodySession(t)
	var mp wire.MassPropertiesResult
	call(t, r, s, "body.physicalProperties", `{"bodyIndex":0,"densityGCm3":2.5}`, &mp)
	// volume 24 000 mm³ = 24 cm³; mass = 24 cm³ × 2.5 g/cm³ = 60 g.
	if math.Abs(mp.MassG-60) > 0.6 {
		t.Errorf("mass at 2.5 g/cm³ = %g g, want ≈60", mp.MassG)
	}
}

// TestBodyPhysicalPropertiesBadIndexFails: an out-of-range body index is a rejection.
func TestBodyPhysicalPropertiesBadIndexFails(t *testing.T) {
	r, s := boxBodySession(t)
	if _, err := r.Handle(s, "body.physicalProperties", []byte(`{"bodyIndex":7}`)); err == nil {
		t.Error("body.physicalProperties with an out-of-range index should fail")
	}
}

// TestBodyPhysicalPropertiesBadAccuracyFails: an unknown accuracy spelling is a rejection.
func TestBodyPhysicalPropertiesBadAccuracyFails(t *testing.T) {
	r, s := boxBodySession(t)
	if _, err := r.Handle(s, "body.physicalProperties", []byte(`{"bodyIndex":0,"accuracy":"ultra"}`)); err == nil {
		t.Error("body.physicalProperties with an unknown accuracy should fail")
	}
}

// TestBodyListReportsKeyAndStyle: body.list now surfaces each body's persistent reference key
// and assigned color-style name (empty until one is assigned) (#1078).
func TestBodyListReportsKeyAndStyle(t *testing.T) {
	r, s := boxBodySession(t)
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	b := list.Bodies[0]
	if b.Key == "" {
		t.Error("body.list body has an empty Key, want its persistent reference key")
	}
	if b.Style != "" {
		t.Errorf("body Style = %q, want empty before any style is assigned", b.Style)
	}

	if err := s.AssignColorStyleToBody(b.Key, "Steel"); err != nil {
		t.Fatalf("assign color style: %v", err)
	}
	call(t, r, s, "body.list", `{}`, &list)
	if list.Bodies[0].Style != "Steel" {
		t.Errorf("after assigning Steel, body Style = %q, want Steel", list.Bodies[0].Style)
	}
}
