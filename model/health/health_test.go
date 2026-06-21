// SPDX-License-Identifier: GPL-2.0-only

package health

import "testing"

func TestHealthyZeroValue(t *testing.T) {
	var h Health
	if !h.OK() {
		t.Error("zero-value Health is not OK")
	}
	if !Healthy.OK() {
		t.Error("Healthy is not OK")
	}
}

func TestSicken(t *testing.T) {
	h := Sicken("reference lost: face")
	if h.OK() || h.Status != Sick {
		t.Errorf("Sicken status = %v, want sick", h.Status)
	}
	if h.Reason != "reference lost: face" {
		t.Errorf("Sicken reason = %q", h.Reason)
	}
}

func TestWarn(t *testing.T) {
	h := Warn("reference auto-healed via ancestral match")
	if h.OK() || h.Status != Warning {
		t.Errorf("Warn status = %v, want warning", h.Status)
	}
	if h.Reason != "reference auto-healed via ancestral match" {
		t.Errorf("Warn reason = %q", h.Reason)
	}
}

func TestStatusStrings(t *testing.T) {
	cases := map[Status]string{OK: "ok", Warning: "warning", Sick: "sick", Suppressed: "suppressed", Status(9): "unknown"}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Status(%d).String() = %q, want %q", s, got, want)
		}
	}
}
