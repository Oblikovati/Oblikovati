// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// tallyTypes walks every object header in a file and counts them by type name.
func tallyTypes(t *testing.T, name string) (map[string]int, int) {
	t.Helper()
	data := loadTestFile(t, name)
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	omb, err := h.ObjectMapBytes(data)
	if err != nil {
		t.Fatalf("ObjectMapBytes: %v", err)
	}
	od, err := h.ObjectData(data)
	if err != nil {
		t.Fatalf("ObjectData: %v", err)
	}
	refs, err := parseObjectMap(omb)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	headers, errs := WalkObjectHeaders(od, refs, h.Version)
	if len(errs) != 0 {
		t.Fatalf("%d object headers failed to decode (first: %v)", len(errs), errs[0])
	}
	if len(headers) != len(refs) {
		t.Fatalf("decoded %d of %d object headers", len(headers), len(refs))
	}
	tally := map[string]int{}
	for _, o := range headers {
		tally[o.Type.Name()]++
	}
	return tally, len(headers)
}

// TestObjectHeaderTallyMatchesOracle pins the object-header decode (MS size, R2010+
// handle-stream size, BOT/BS type) against dwgread's entity census. Exact counts
// across 100k+ objects prove the per-object positioning and type decoding are
// correct for both the paged (R2018) and flat (R2000) containers.
func TestObjectHeaderTallyMatchesOracle(t *testing.T) {
	cases := []struct {
		file string
		want map[string]int
	}{
		{"testfile-1.dwg", map[string]int{ // AC1032 / R2018
			"LINE": 58062, "LWPOLYLINE": 15525, "SPLINE": 2898, "INSERT": 2100,
			"ARC": 1670, "ELLIPSE": 1271, "CIRCLE": 959, "POINT": 739,
		}},
		{"testfile-2.dwg", map[string]int{ // AC1015 / R2000
			"LINE": 134240, "ARC": 41950, "POINT": 34264, "INSERT": 11465,
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			tally, total := tallyTypes(t, c.file)
			for name, want := range c.want {
				if got := tally[name]; got != want {
					t.Errorf("%s count = %d, want %d (oracle)", name, got, want)
				}
			}
			if total < 1000 {
				t.Errorf("only %d objects decoded", total)
			}
		})
	}
}
