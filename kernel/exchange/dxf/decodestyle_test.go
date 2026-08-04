// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/drawing"
)

// dxfWithStyledLine is a minimal DXF holding one layer with a colour, line type and weight, and
// one line that sits on it and overrides only its colour.
const dxfWithStyledLine = `0
SECTION
2
TABLES
0
LAYER
2
WALLS
62
5
6
DASHED
370
35
0
ENDSEC
0
SECTION
2
ENTITIES
0
LINE
5
2A
8
WALLS
62
1
10
0.0
20
0.0
30
0.0
11
10.0
21
0.0
31
0.0
0
ENDSEC
0
EOF
`

// An entity's formatting must survive import: the layer supplies what the entity does not
// override, and the entity's own colour wins over its layer's. Before #2015 the decoder read
// geometry only and every drawing came back monochrome and continuous.
func TestDecodeResolvesEntityStyleAgainstItsLayer(t *testing.T) {
	dr, _, err := DecodeWithProgress([]byte(dxfWithStyledLine), exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dr.Layers) != 1 || dr.Layers[0].Name != "WALLS" {
		t.Fatalf("layers = %+v, want the one WALLS record", dr.Layers)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(dr.Entities))
	}
	h := dr.Entities[0].EntityHandle()
	color, lineType, weight := dr.ResolveStyle(h)
	if color != 1 {
		t.Errorf("colour = %d, want 1 — the entity's own colour beats its layer's", color)
	}
	if lineType != "DASHED" {
		t.Errorf("line type = %q, want DASHED inherited from the layer", lineType)
	}
	if weight != 35 {
		t.Errorf("weight = %d, want 35 inherited from the layer", weight)
	}
}

// An entity naming a layer the file never defines resolves to the fallback rather than to
// whatever the previous entity had.
func TestResolveStyleOfAnUnknownLayer(t *testing.T) {
	dr := &drawing.Drawing{}
	dr.SetEntityLayer(7, "MISSING")
	color, lineType, weight := dr.ResolveStyle(7)
	if color != 7 || lineType != "CONTINUOUS" || weight != drawing.LineWeightByLayer {
		t.Errorf("resolved = %d/%q/%d, want the white continuous fallback", color, lineType, weight)
	}
}

// BYLAYER and BYBLOCK are inherit sentinels, not colours.
func TestByLayerSentinelInherits(t *testing.T) {
	dr := &drawing.Drawing{Layers: []drawing.Layer{{Name: "L", Color: 3, LineType: "HIDDEN", LineWeight: 50}}}
	dr.SetEntityLayer(1, "L")
	dr.SetStyle(1, drawing.Style{Color: drawing.ColorByLayer, LineWeight: drawing.LineWeightByLayer})
	color, lineType, weight := dr.ResolveStyle(1)
	if color != 3 || lineType != "HIDDEN" || weight != 50 {
		t.Errorf("resolved = %d/%q/%d, want the layer's values", color, lineType, weight)
	}
}
