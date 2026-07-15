// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

// TestExtrusionDecode checks the blind extrude depth decodes: the generated part is a rectangle
// extruded 10 mm, so one extrusion with a 0.01 m depth is decoded.
func TestExtrusionDecode(t *testing.T) {
	d, err := Open(readTestdata(t, "extrude_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ext := d.Extrusions()
	if len(ext) != 1 {
		t.Fatalf("got %d extrusions, want 1", len(ext))
	}
	if math.Abs(ext[0].Depth-0.01) > 1e-9 {
		t.Errorf("depth = %g m, want 0.01", ext[0].Depth)
	}
}

// TestNoExtrusion verifies a sketch-only part decodes no extrude features.
func TestNoExtrusion(t *testing.T) {
	d, err := Open(readTestdata(t, "box10_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n := len(d.Extrusions()); n != 0 {
		t.Errorf("got %d extrusions, want 0", n)
	}
}
