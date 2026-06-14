// SPDX-License-Identifier: GPL-2.0-only

package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.0.20260615030000", "0.0.20260614120000", true},                 // later nightly timestamp
		{"0.0.20260614120000", "0.0.20260615030000", false},                // older
		{"0.0.20260614120000", "0.0.20260614120000", false},                // equal
		{"0.1.20260101000000", "0.0.20260615030000", true},                 // minor bump beats a newer timestamp
		{"1.0.20260101000000", "0.9.20260615030000", true},                 // major bump
		{"v0.0.20260615030000", "0.0.20260614120000", true},                // leading v on latest
		{"0.0.20260615030000-nightly", "0.0.20260614120000-nightly", true}, // prerelease ignored
		{"0.0.20260615030000-nightly", "0.0.20260615030000-nightly", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// TestPatchTimestampExceedsInt32 guards the int64 parse: a 14-digit timestamp PATCH
// overflows a 32-bit int, which would corrupt the comparison on a 32-bit build.
func TestPatchTimestampExceedsInt32(t *testing.T) {
	if !IsNewer("0.0.20260101000001", "0.0.20260101000000") {
		t.Fatal("timestamp PATCH must compare as int64, not overflow")
	}
}
