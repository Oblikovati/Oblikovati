// SPDX-License-Identifier: GPL-2.0-only

package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.000200.1.1", "0.000200.1.0", true},                                                  // patch bump
		{"0.000200.2.0", "0.000200.1.9", true},                                                  // minor beats a higher patch
		{"0.000200.1.0", "0.000200.2.0", false},                                                 // older minor
		{"0.000200.1.0", "0.000200.1.0", false},                                                 // equal
		{"0.000300.0.1", "0.000200.9.9", true},                                                  // newer API_VERSION wins
		{"0.000200.1.0", "0.000300.0.0", false},                                                 // older API_VERSION
		{"1.000200.0.0", "0.000900.9.9", true},                                                  // MANUAL_MAJOR bump
		{"v0.000200.1.1", "0.000200.1.0", true},                                                 // leading v on latest
		{"0.000200.1.0-nightly.20260615T030000", "0.000200.1.0-nightly.20260614T030000", true},  // later nightly stamp
		{"0.000200.1.0-nightly.20260614T030000", "0.000200.1.0-nightly.20260615T030000", false}, // older stamp
		{"0.000200.1.0-nightly.20260615T030000", "0.000200.1.0-nightly.20260615T030000", false}, // equal nightly
		{"0.000200.2.0-nightly.20260101T000000", "0.000200.1.0-nightly.20260615T030000", true},  // minor beats a newer stamp
		// Compact YYMMDDTHH stamps still compare chronologically by string.
		{"0.000200.1.0-nightly.260616T03", "0.000200.1.0-nightly.260615T09", true},  // next day beats a later hour
		{"0.000200.1.0-nightly.260615T09", "0.000200.1.0-nightly.260615T03", true},  // later hour, same day
		{"0.000200.1.0-nightly.260615T03", "0.000200.1.0-nightly.260615T03", false}, // equal compact stamp
		// The format switch is monotonic: any new compact stamp (26…) sorts above every
		// old long stamp (2026… → "20…"), so the first post-switch nightly reads as newer.
		{"0.000200.1.0-nightly.260622T11", "0.000200.1.0-nightly.20260622T110000", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// TestAPIFieldExceedsInt32 guards the int64 parse: a wide zero-padded API_VERSION
// must compare as int64, not overflow a 32-bit int.
func TestAPIFieldExceedsInt32(t *testing.T) {
	if !IsNewer("0.999999.0.1", "0.999999.0.0") {
		t.Fatal("API_VERSION field must compare as int64, not overflow")
	}
}
