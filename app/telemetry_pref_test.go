// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestTelemetryEnabledDefaultsOn(t *testing.T) {
	if !NewSession().TelemetryEnabled() {
		t.Fatal("telemetry must default to on (opt-out)")
	}
}

func TestSetTelemetryEnabledTogglesAndPersists(t *testing.T) {
	s := NewSession()
	if err := s.SetTelemetryEnabled(false); err != nil {
		t.Fatalf("SetTelemetryEnabled(false): %v", err)
	}
	if s.TelemetryEnabled() {
		t.Error("telemetry should be off after opting out")
	}
	if err := s.SetTelemetryEnabled(true); err != nil {
		t.Fatalf("SetTelemetryEnabled(true): %v", err)
	}
	if !s.TelemetryEnabled() {
		t.Error("telemetry should be on again after re-enabling")
	}
}
