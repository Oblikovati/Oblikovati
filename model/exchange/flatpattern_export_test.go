// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/dxf"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// flatWithTab develops a side×side base plate with one tab on the y=0 edge (developed length
// tab), at the given gauge — the common tray fixture, with one fold line.
func flatWithTab(t *testing.T, side, thickness, tab float64) *feature.FlatPattern {
	t.Helper()
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.AddRectangleByCorners(math.P2(0, 0), math.P2(side, side))
	tabs := []feature.FlatTab{{A: math.P2(0, 0), B: math.P2(side, 0), Length: tab, Angle: stdmath.Pi / 2}}
	fp, err := feature.BuildFlatPattern(s, 0, thickness, tabs)
	if err != nil {
		t.Fatalf("BuildFlatPattern: %v", err)
	}
	return fp
}

// TestExportFlatPatternDXF develops a tray flat and exports it: the footprint must land on the
// Outline layer and the fold line on the Bend-Up layer, with both layers in the LAYER table —
// the #378 acceptance (flat exports to DXF with bend lines on configured layers).
func TestExportFlatPatternDXF(t *testing.T) {
	fp := flatWithTab(t, 4, 0.2, 1.5)
	data, n, err := ExportFlatPatternDXF(fp, FlatExportLayers{}, types.DXFR2018)
	if err != nil {
		t.Fatalf("ExportFlatPatternDXF: %v", err)
	}
	out := string(data)

	// The two layers used appear as LAYER records.
	for _, layer := range []string{"Outline", "Bend-Up"} {
		if !strings.Contains(out, "\n2\n"+layer+"\n") {
			t.Errorf("LAYER table missing %q", layer)
		}
	}
	// The fold line is drawn on the bend-up layer (this tray's bend folds up).
	if !strings.Contains(out, "\n8\nBend-Up\n") {
		t.Error("fold line not emitted on the Bend-Up layer")
	}
	// Footprint outline: the tray's outer rectangle is at least 4 segments, plus the 1 fold line.
	outline := strings.Count(out, "\n8\nOutline\n")
	if outline < 4 {
		t.Errorf("outline emitted %d segments, want >= 4 (the footprint rectangle)", outline)
	}
	if n != outline+1 {
		t.Errorf("entity count = %d, want %d (outline %d + 1 fold line)", n, outline+1, outline)
	}

	// The export re-decodes to valid geometry (the outline lines + fold line survive).
	dr, _, err := dxf.Decode(data)
	if err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if len(dr.Entities) < 5 {
		t.Errorf("re-decoded %d entities, want >= 5 (outline + fold line)", len(dr.Entities))
	}
}

// TestExportFlatPatternPunches a punch representation exports as a closed outline plus its token
// (a TEXT entity) on the Punches layer — the manufacturing tag from #378's acceptance.
func TestExportFlatPatternPunches(t *testing.T) {
	fp := flatWithTab(t, 4, 0.2, 1.5)
	fp.Punches = []feature.FlatPunch{{
		Outline: []math.Point2{math.P2(1, 1), math.P2(2, 1), math.P2(2, 2), math.P2(1, 2)},
		Token:   "Punch1",
	}}
	data, _, err := ExportFlatPatternDXF(fp, FlatExportLayers{}, types.DXFR2018)
	if err != nil {
		t.Fatalf("ExportFlatPatternDXF: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "\n2\nPunches\n") {
		t.Error("LAYER table missing the Punches layer")
	}
	if !strings.Contains(out, "\n8\nPunches\n") {
		t.Error("no punch geometry on the Punches layer")
	}
	if !strings.Contains(out, "\n0\nTEXT\n") || !strings.Contains(out, "\n1\nPunch1\n") {
		t.Error("punch token TEXT not emitted")
	}
}

// TestFlatExportLayersDefaults the zero layer value resolves to the default scheme.
func TestFlatExportLayersDefaults(t *testing.T) {
	got := FlatExportLayers{}.withDefaults()
	if got != DefaultFlatExportLayers() {
		t.Errorf("withDefaults() = %+v, want the default scheme %+v", got, DefaultFlatExportLayers())
	}
	custom := FlatExportLayers{Outline: "CUT"}.withDefaults()
	if custom.Outline != "CUT" || custom.BendUp != "Bend-Up" {
		t.Errorf("withDefaults kept/filled wrong: %+v", custom)
	}
}
