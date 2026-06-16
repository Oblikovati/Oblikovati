// SPDX-License-Identifier: GPL-2.0-only

package display

import "testing"

// TestDefaultOptionsAreValid checks the out-of-the-box app options carry valid enum values.
func TestDefaultOptionsAreValid(t *testing.T) {
	o := DefaultOptions()
	if !o.DisplayQuality.IsValid() {
		t.Errorf("default DisplayQuality %v invalid", o.DisplayQuality)
	}
	if !o.BackFaceCulling.IsValid() || !o.RayTracingQuality.IsValid() || !o.NewWindowProjection.IsValid() {
		t.Errorf("default options carry an invalid enum: %+v", o)
	}
	if o.ViewTransitionTime <= 0 {
		t.Errorf("ViewTransitionTime = %v, want > 0", o.ViewTransitionTime)
	}
}

// TestDefaultSettingsAreValid checks the out-of-the-box per-document settings are valid.
func TestDefaultSettingsAreValid(t *testing.T) {
	s := DefaultSettings()
	if !s.BackgroundType.IsValid() || !s.GroundShadow.IsValid() || !s.ShadowDirection.IsValid() {
		t.Errorf("default settings carry an invalid enum: %+v", s)
	}
	if !s.DisplayModeSource.IsValid() {
		t.Errorf("default DisplayModeSource %v invalid", s.DisplayModeSource)
	}
	if !s.GroundPlane.Visible {
		t.Error("default ground plane should be visible")
	}
}
