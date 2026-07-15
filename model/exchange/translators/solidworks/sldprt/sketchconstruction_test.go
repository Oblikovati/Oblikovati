// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "testing"

// TestEntityConstructionDecode checks the per-entity construction flags decoded from the trailing
// entity-record table. Each part was generated with its second entity toggled to construction
// geometry in SolidWorks 2026, across the three curve kinds; a plain square has none. The order is
// entity draw order, so the construction entity (drawn second) is the second flag.
func TestEntityConstructionDecode(t *testing.T) {
	cases := []struct {
		file string
		want []bool
	}{
		{"constrcirc_fmtb.sldprt", []bool{false, true}}, // real circle + construction circle
		{"constrline_fmtb.sldprt", []bool{false, true}}, // real line + construction line
		{"constrarc_fmtb.sldprt", []bool{false, true}},  // real line + construction arc
		{"box10_fmtb.sldprt", []bool{false, false, false, false}},
		{"cyl_fmtb.sldprt", []bool{false}},
	}
	for _, c := range cases {
		d, err := Open(readTestdata(t, c.file))
		if err != nil {
			t.Fatalf("Open %s: %v", c.file, err)
		}
		got := d.Sketches()[0].Construction
		if len(got) != len(c.want) {
			t.Errorf("%s: %d construction flags %v, want %v", c.file, len(got), got, c.want)
			continue
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("%s: construction[%d] = %v, want %v (full %v)", c.file, i, got[i], c.want[i], got)
			}
		}
	}
}

// TestLastPointOffset verifies the handle/entity-table boundary: the last cached coordinate sits
// before the entity records, so a region ending in an entity table still reports a point offset that
// precedes it.
func TestLastPointOffset(t *testing.T) {
	if got := lastPointOffset([]byte("no points here")); got != -1 {
		t.Errorf("lastPointOffset with no points = %d, want -1", got)
	}
	// pointMarker (1e 00) then a valid zero coordinate (16 zero bytes) at offset 3.
	region := []byte{0xaa, 0xbb, 0xcc, 0x1e, 0x00, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if got := lastPointOffset(region); got != 3 {
		t.Errorf("lastPointOffset = %d, want 3", got)
	}
}
