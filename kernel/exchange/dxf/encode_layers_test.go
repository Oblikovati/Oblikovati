// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// TestEncodeNamedLayers checks that entities on custom layers emit a LAYER record per distinct
// layer (plus the default "0"), tag each entity with its layer (group code 8), and that a Text
// entity is written — the surface a flat-pattern export needs (outline / bend / punch layers).
func TestEncodeNamedLayers(t *testing.T) {
	in := &drawing.Drawing{Units: drawing.INSCentimetres, Entities: []drawing.Entity{
		&drawing.Line{Layer: "Outline", Start: [3]float64{0, 0, 0}, End: [3]float64{10, 0, 0}},
		&drawing.Line{Layer: "BendUp", Start: [3]float64{0, 5, 0}, End: [3]float64{10, 5, 0}},
		&drawing.Circle{Layer: "Punches", Center: [3]float64{5, 2, 0}, Radius: 1, Normal: [3]float64{0, 0, 1}},
		&drawing.Text{Layer: "Punches", Position: [3]float64{5, 2, 0}, Height: 0.5, Value: "PUNCH-8"},
	}}
	data, err := Encode(in, R2000)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := string(data)

	// One LAYER record per distinct layer name (group code "2" followed by the name on its own line).
	for _, layer := range []string{"0", "Outline", "BendUp", "Punches"} {
		if !strings.Contains(out, "\n2\n"+layer+"\n") {
			t.Errorf("LAYER table missing a record for %q", layer)
		}
	}
	// Each entity tags its layer (group code 8).
	for _, layer := range []string{"Outline", "BendUp", "Punches"} {
		if !strings.Contains(out, "\n8\n"+layer+"\n") {
			t.Errorf("no entity tagged on layer %q (group code 8)", layer)
		}
	}
	// The TEXT token is written with its string.
	if !strings.Contains(out, "\n0\nTEXT\n") || !strings.Contains(out, "\n1\nPUNCH-8\n") {
		t.Error("TEXT token not emitted")
	}

	// The geometry still decodes (the lines/circle survive; TEXT is not a curve and is skipped).
	dr, _, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := len(dr.Entities); got < 3 {
		t.Errorf("decoded %d curve entities, want >= 3 (the lines + circle)", got)
	}
}
