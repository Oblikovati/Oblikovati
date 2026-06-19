// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBuildDrawListMeshColors covers the mesh-diagnostic draw list (normal/per-face and
// per-triangle debug colouring): a box produces triangles, each item carrying per-vertex colours.
func TestBuildDrawListMeshColors(t *testing.T) {
	b := box(2, math.V3(0, 0, 0))
	for _, perTri := range []bool{false, true} {
		list := BuildDrawListMeshColors([]*topo.Body{b}, frontCamera(), ops.DefaultQuality(), perTri)
		if list.Triangles() == 0 {
			t.Fatalf("perTriangle=%v: mesh-colors draw list has no triangles", perTri)
		}
		colored := false
		for _, it := range list.Items {
			if len(it.Colors) == len(it.Positions) && len(it.Colors) > 0 {
				colored = true
			}
		}
		if !colored {
			t.Errorf("perTriangle=%v: no item carried per-vertex debug colours", perTri)
		}
	}
}

// TestMeshDebugColorAndHSV covers the debug-colour generators: hsvToRGB stays in range across the
// hue wheel and successive debug colours differ.
func TestMeshDebugColorAndHSV(t *testing.T) {
	for _, h := range []float64{0, 60, 120, 180, 240, 300, 359} {
		r, g, bl := hsvToRGB(h, 1, 1)
		for _, c := range []float64{r, g, bl} {
			if c < 0 || c > 1 {
				t.Errorf("hsvToRGB(%g,1,1) component %g out of [0,1]", h, c)
			}
		}
	}
	if meshDebugColor(0) == meshDebugColor(3) {
		t.Error("meshDebugColor(0) == meshDebugColor(3), expected distinct hues")
	}
}
