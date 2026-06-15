// SPDX-License-Identifier: GPL-2.0-only

package update

import "testing"

func TestDetectChannel(t *testing.T) {
	cases := []struct {
		version string
		want    Channel
	}{
		{"", Dev},
		{"dev", Dev},
		{"0.000200.1.0", Stable},
		{"v0.000200.1.0", Stable}, // a leading v is still a stable release
		{"0.000200.1.0-nightly.20260614T120000", Nightly},
		{"1.000200.4.2-nightly.20260614T120000", Nightly},
	}
	for _, c := range cases {
		if got := DetectChannel(c.version); got != c.want {
			t.Errorf("DetectChannel(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

func TestChannelString(t *testing.T) {
	cases := map[Channel]string{Dev: "dev", Stable: "stable", Nightly: "nightly"}
	for ch, want := range cases {
		if got := ch.String(); got != want {
			t.Errorf("Channel(%d).String() = %q, want %q", ch, got, want)
		}
	}
}
