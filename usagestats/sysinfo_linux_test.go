// SPDX-License-Identifier: GPL-2.0-only

//go:build linux

package usagestats

import "testing"

func TestParseMemTotalKB(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int64
	}{
		{"typical", "MemFree: 100 kB\nMemTotal:   16327084 kB\nBuffers: 5 kB\n", 16327084},
		{"first line", "MemTotal: 8000 kB\n", 8000},
		{"absent", "MemFree: 100 kB\nSwapTotal: 0 kB\n", 0},
		{"malformed value", "MemTotal: notanumber kB\n", 0},
		{"missing fields", "MemTotal:\n", 0},
		{"empty", "", 0},
	}
	for _, c := range cases {
		if got := parseMemTotalKB([]byte(c.in)); got != c.want {
			t.Errorf("%s: parseMemTotalKB = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestParseCPUModelName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"typical", "processor\t: 0\nmodel name\t: AMD Ryzen 7 5800X\nstepping\t: 0\n", "AMD Ryzen 7 5800X"},
		{"first of many", "model name : Core A\nmodel name : Core B\n", "Core A"},
		{"absent", "processor: 0\nvendor_id: GenuineIntel\n", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		if got := parseCPUModelName([]byte(c.in)); got != c.want {
			t.Errorf("%s: parseCPUModelName = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestLinuxProbesRunOnHost exercises the file-backed probes against the real /proc on the
// Linux CI runner (happy path). A sandbox may legitimately hide values, so this only asserts
// non-negative — the parsing branches are covered by the pure-parser tests above.
func TestLinuxProbesRunOnHost(t *testing.T) {
	if totalRAMBytes() < 0 || homeVolumeCapacityBytes() < 0 {
		t.Error("linux probes returned a negative size")
	}
	_ = cpuModel() // smoke: must not panic
}
