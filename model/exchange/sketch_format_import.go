// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/model/sketch"
)

// Carrying a drawing's formatting onto the imported sketch (#2015). An imported entity keeps the
// colour, line type and line weight it had in the file, resolved through the layer table, so a
// DWG or DXF looks the same after a round trip instead of coming back monochrome and continuous.
//
// The values land as per-entity overrides rather than as a layer model: the resolved appearance
// is preserved, the layer STRUCTURE is not. An exported entity therefore carries explicit values
// where the original said BYLAYER. That is a deliberate trade — a layer model would mean a
// document-level layer table, a ribbon layer picker and importer work in both formats, for a part
// sketch with no style-management UI.

// applyImportedFormat gives the entities created for one drawing entity the formatting that
// entity had in the file. Entities that resolve to the drawing's defaults get no override, so an
// unformatted import stores nothing.
func applyImportedFormat(sk *sketch.Sketch, dr *drawing.Drawing, e drawing.Entity, made []sketch.Entity) {
	f, ok := importedEntityFormat(dr, e)
	if !ok {
		return
	}
	for _, m := range made {
		sk.SetEntityFormat(m.EntityID(), f)
	}
}

// importedEntityFormat resolves one drawing entity's formatting into a sketch format, reporting
// false when it resolves to the defaults and so needs no override stored.
func importedEntityFormat(dr *drawing.Drawing, e drawing.Entity) (sketch.EntityFormat, bool) {
	color, lineType, weight := dr.ResolveStyle(e.EntityHandle())
	f := sketch.EntityFormat{
		LineType:   sketchLineTypeFor(lineType),
		LineWeight: lineWeightMillimetres(weight),
	}
	if c, ok := aciColor(color); ok {
		f.Color = c
	}
	return f, !f.IsDefault()
}

// lineWeightMillimetres converts a DXF/DWG line weight (hundredths of a millimetre) to
// millimetres. The inherit sentinel and the "by default" values become 0, which is Default.
func lineWeightMillimetres(weight int) float64 {
	if weight <= 0 {
		return 0
	}
	return float64(weight) / 100
}

// sketchLineTypeFor maps a drawing line-type record name onto a sketch line type, matching
// case-insensitively because the name is a user-facing record name rather than an enum. An
// unrecognised or continuous line type is Default: continuous is what an unstyled sketch already
// draws, so storing it as an override would only add noise.
func sketchLineTypeFor(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "DASHED", "DASHED2", "DASHEDX2":
		return string(types.SketchLineDashed)
	case "HIDDEN", "HIDDEN2", "HIDDENX2":
		return string(types.SketchLineHidden)
	case "CENTER", "CENTER2", "CENTERX2":
		return string(types.SketchLineCenter)
	case "PHANTOM", "PHANTOM2", "PHANTOMX2":
		return string(types.SketchLinePhantom)
	default:
		return ""
	}
}

// aciPalette is the AutoCAD Color Index's first nine entries, which is what drawings overwhelmingly
// use: 1–7 are the standard colours and 8–9 the two greys. Index 7 is the drawing's foreground and
// is deliberately absent — it is what an unstyled entity already draws as, so treating it as an
// override would restyle every ordinary entity in the file.
var aciPalette = map[int][3]uint8{
	1: {255, 0, 0},
	2: {255, 255, 0},
	3: {0, 255, 0},
	4: {0, 255, 255},
	5: {0, 0, 255},
	6: {255, 0, 255},
	8: {128, 128, 128},
	9: {192, 192, 192},
}

// aciColor maps a colour index onto a colour, reporting false for the foreground colour and for
// indexes outside the palette — both of which mean "leave it to the sketch defaults" rather than
// guessing a shade.
func aciColor(index int) (types.Color, bool) {
	rgb, ok := aciPalette[index]
	if !ok {
		return types.Color{}, false
	}
	return types.NewColor(rgb[0], rgb[1], rgb[2]), true
}

// --- export ----------------------------------------------------------------

// aciIndexFor maps a colour back onto an AutoCAD Color Index, returning BYLAYER for a colour
// that is not an override or not in the palette — the inverse of aciColor, and lossy in the same
// place: a colour outside the palette exports as inherited rather than as the nearest shade,
// because guessing a neighbour would silently change the drawing.
func aciIndexFor(c types.Color) int {
	if !c.IsOverride() {
		return drawing.ColorByLayer
	}
	for index, rgb := range aciPalette {
		if rgb[0] == c.R && rgb[1] == c.G && rgb[2] == c.B {
			return index
		}
	}
	return drawing.ColorByLayer
}

// drawingLineTypeName maps a sketch line type onto a drawing line-type record name; Default
// becomes the empty name, which the encoder writes as BYLAYER.
func drawingLineTypeName(lineType string) string {
	switch types.SketchLineType(lineType) {
	case types.SketchLineDashed:
		return "DASHED"
	case types.SketchLineHidden:
		return "HIDDEN"
	case types.SketchLineCenter:
		return "CENTER"
	case types.SketchLinePhantom:
		return "PHANTOM"
	default:
		return ""
	}
}

// lineWeightHundredths converts millimetres back to the DXF/DWG line-weight unit; 0 (Default)
// becomes the inherit sentinel.
func lineWeightHundredths(mm float64) int {
	if mm <= 0 {
		return drawing.LineWeightByLayer
	}
	return int(mm*100 + 0.5)
}
