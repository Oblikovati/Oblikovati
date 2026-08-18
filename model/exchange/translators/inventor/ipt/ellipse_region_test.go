// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEllipseRegionResolves guards the ellipse-in-region decode: ST3215Bracket's base extrude names
// a region whose boundary includes two trimmed elliptical arcs. Before regionEdgeOf/trimEllipseEdge
// learned the ellipse edge kind, loopAt failed on that loop and regionOf returned nil for the whole
// region — the base declined and the bracket built a 0.023x fragment. Now every extrude resolves its
// region. Corpus-gated (the real part carries the ellipse boundary; no generated fixture does);
// point IPT_CORPUS at the ReelToReel Mechanical directory to enable.
func TestEllipseRegionResolves(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "ST3215Bracket.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	d, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	regions := ExtrudeRegions(d)
	if len(regions) == 0 {
		t.Fatal("no extrude regions decoded")
	}
	// The base (ext 0) region has an outer boundary + 5 holes + a second ellipse-bearing boundary.
	if got := len(regions[0]); got == 0 {
		t.Fatalf("base extrude region has 0 loops — the ellipse boundary edge was dropped")
	}
	// Every extrude must now resolve a non-empty region (the ellipse edges no longer void a loop).
	for i, r := range regions {
		if len(r) == 0 {
			t.Errorf("extrude %d region is empty", i)
		}
	}
}
