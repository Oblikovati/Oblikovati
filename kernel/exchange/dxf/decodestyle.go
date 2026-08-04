// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"strconv"

	"oblikovati.org/kernel/exchange/drawing"
)

// Layer table and per-entity formatting (#2015). Before this the decoder read geometry only:
// every imported entity lost its colour, line type and line weight, and the layer table was not
// parsed at all — so a DWG or DXF round trip came back monochrome and continuous.
//
// The values are what the sketch Format panel's three lists expose, which is the whole reason
// the panel names DWG interoperability.

// DXF group codes for entity formatting.
const (
	groupLayerName  = 8   // the entity's (or table record's) layer name
	groupColorIndex = 62  // AutoCAD Color Index; 256 = BYLAYER, 0 = BYBLOCK
	groupLineType   = 6   // line-type record name
	groupLineWeight = 370 // hundredths of a millimetre; -1 = BYLAYER
)

// decodeLayerTable reads the LAYER records from a TABLES section, returning the drawing's layer
// table. A TABLES section with no LAYER table yields none, and entities then resolve against the
// default fallback.
func decodeLayerTable(pairs []pair) []drawing.Layer {
	var layers []drawing.Layer
	for _, g := range splitEntities(pairs) {
		if g.name != "LAYER" {
			continue
		}
		if l, ok := decodeLayerRecord(indexByCode(g.body)); ok {
			layers = append(layers, l)
		}
	}
	return layers
}

// decodeLayerRecord reads one LAYER table record. A record with no name is skipped: it cannot be
// referenced, so keeping it would only shadow a later well-formed record of the same index.
func decodeLayerRecord(m map[int]pair) (drawing.Layer, bool) {
	name, ok := m[2]
	if !ok || name.value == "" {
		return drawing.Layer{}, false
	}
	return drawing.Layer{
		Name:       name.value,
		Color:      colorIndexOf(m, defaultLayerColorIndex),
		LineType:   lineTypeOf(m, "CONTINUOUS"),
		LineWeight: lineWeightOf(m),
	}, true
}

// defaultLayerColorIndex is ACI 7, the colour a layer record without a colour shows as.
const defaultLayerColorIndex = 7

// entityStyle reads one entity's explicit formatting. Every field defaults to its inherit
// sentinel, so an entity that names none resolves entirely to its layer.
func entityStyle(m map[int]pair) drawing.Style {
	return drawing.Style{
		Color:      colorIndexOf(m, drawing.ColorByLayer),
		LineType:   lineTypeOf(m, ""),
		LineWeight: lineWeightOf(m),
	}
}

// layerNameOf reads an entity's layer, defaulting to the layer every drawing has.
func layerNameOf(m map[int]pair) string {
	if p, ok := m[groupLayerName]; ok && p.value != "" {
		return p.value
	}
	return drawing.DefaultLayerName
}

// colorIndexOf reads the colour index, or fallback when absent or malformed. A negative index
// means the layer is off in AutoCAD; its magnitude is still the colour, so the sign is dropped.
func colorIndexOf(m map[int]pair, fallback int) int {
	p, ok := m[groupColorIndex]
	if !ok {
		return fallback
	}
	v, err := strconv.Atoi(p.value)
	if err != nil {
		return fallback
	}
	if v < 0 {
		return -v
	}
	return v
}

// lineTypeOf reads the line-type name, or fallback when absent.
func lineTypeOf(m map[int]pair, fallback string) string {
	if p, ok := m[groupLineType]; ok && p.value != "" {
		return p.value
	}
	return fallback
}

// lineWeightOf reads the line weight in hundredths of a millimetre, defaulting to BYLAYER.
func lineWeightOf(m map[int]pair) int {
	p, ok := m[groupLineWeight]
	if !ok {
		return drawing.LineWeightByLayer
	}
	v, err := strconv.Atoi(p.value)
	if err != nil {
		return drawing.LineWeightByLayer
	}
	return v
}

// recordEntityStyle stores a decoded entity's explicit formatting and its layer against its
// handle, so the drawing can resolve the two together later.
//
// An entity whose source carried no handle cannot be keyed and keeps its layer's formatting. That
// is the correct fallback rather than a silent wrong colour: a handle-less entity is one the file
// never referenced, so it has no identity to hang an override on.
func recordEntityStyle(dr *drawing.Drawing, e drawing.Entity, m map[int]pair) {
	h := e.EntityHandle()
	if h == 0 {
		return
	}
	dr.SetStyle(h, entityStyle(m))
	dr.SetEntityLayer(h, layerNameOf(m))
}
