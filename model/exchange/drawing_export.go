// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	dmodel "oblikovati.org/model/drawing"
)

// Drawing-document DXF export (M14-F05 PBI-145, #392): a drawing sheet — its views' hidden-line
// curves, border and title block — written to DXF on separate named layers, so the sheet opens
// in any CAD/CAM tool with visible and hidden geometry on the layers it expects. It reuses the
// DXF codec via encodeDXF; only the sheet → neutral-drawing conversion is new here. Coordinates
// are sheet millimetres (the drawing's working unit), written verbatim.

// DrawingExportLayers names the DXF layers a sheet's geometry classes land on. A zero value
// (empty names) resolves to the defaults via withDefaults.
type DrawingExportLayers struct {
	Visible    string // view edges that are visible (solid)
	Hidden     string // view edges hidden behind the solid (dashed)
	Border     string // the sheet border rectangle
	TitleBlock string // the title-block grid and field text
}

// DefaultDrawingExportLayers is the layer scheme used when the caller passes none.
func DefaultDrawingExportLayers() DrawingExportLayers {
	return DrawingExportLayers{Visible: "Visible", Hidden: "Hidden", Border: "Border", TitleBlock: "TitleBlock"}
}

func (l DrawingExportLayers) withDefaults() DrawingExportLayers {
	d := DefaultDrawingExportLayers()
	if l.Visible == "" {
		l.Visible = d.Visible
	}
	if l.Hidden == "" {
		l.Hidden = d.Hidden
	}
	if l.Border == "" {
		l.Border = d.Border
	}
	if l.TitleBlock == "" {
		l.TitleBlock = d.TitleBlock
	}
	return l
}

// SheetToDrawing converts a drawing sheet into the neutral drawing model: every view's edges on
// the Visible/Hidden layer per their classification, the border rectangle, and the title-block
// grid with its resolved field text.
func SheetToDrawing(sheet *dmodel.Sheet, layers DrawingExportLayers) []drawing.Entity {
	layers = layers.withDefaults()
	out := viewEntities(sheet, layers)
	out = append(out, borderEntities(sheet, layers.Border)...)
	return append(out, titleBlockEntities(sheet, layers.TitleBlock)...)
}

// ExportDrawingDXF encodes a sheet as ASCII DXF of the given version, returning the bytes and
// the number of entities written.
//
//	data, n, err := exchange.ExportDrawingDXF(sheet, exchange.DefaultDrawingExportLayers(), types.DXFR2018)
func ExportDrawingDXF(sheet *dmodel.Sheet, layers DrawingExportLayers, version types.DXFVersion) ([]byte, int, error) {
	return encodeDXF(SheetToDrawing(sheet, layers), version, drawing.INSCentimetres)
}

// ExportDrawingDXFFile writes ExportDrawingDXF's output to path.
func ExportDrawingDXFFile(sheet *dmodel.Sheet, path string, layers DrawingExportLayers, version types.DXFVersion) (int, error) {
	return writeDXFFile(SheetToDrawing(sheet, layers), path, version, drawing.INSCentimetres)
}

// viewEntities emits every view's drawing curves as line segments on the visible or hidden
// layer per the curve's classification.
func viewEntities(sheet *dmodel.Sheet, layers DrawingExportLayers) []drawing.Entity {
	var out []drawing.Entity
	views := sheet.Views()
	for i := 0; i < views.Count(); i++ {
		for _, c := range views.Item(i).Curves() {
			layer := layers.Visible
			if !c.IsVisible() {
				layer = layers.Hidden
			}
			out = append(out, &drawing.Line{Layer: layer, Start: point2to3(c.Start()), End: point2to3(c.End())})
		}
	}
	return out
}

// borderEntities emits the sheet border as a rectangle inset from the sheet edge by the border
// margins; a sheet with no border yields nothing.
func borderEntities(sheet *dmodel.Sheet, layer string) []drawing.Entity {
	b := sheet.Border()
	if b == nil {
		return nil
	}
	left, right, top, bottom := b.Margins()
	x0, y0 := left, bottom
	x1, y1 := sheet.WidthMM()-right, sheet.HeightMM()-top
	out := rectEntities(layer, x0, y0, x1, y1)
	return append(out, zoneDivisionEntities(sheet, b, layer)...)
}

// zoneDivisionEntities emits a zoned border's interior grid lines (#1989); nothing for a plain border.
func zoneDivisionEntities(sheet *dmodel.Sheet, b any, layer string) []drawing.Entity {
	bd, ok := b.(*dmodel.Border)
	if !ok {
		return nil
	}
	var out []drawing.Entity
	for _, c := range bd.ZoneDivisions(sheet.WidthMM(), sheet.HeightMM()) {
		out = append(out, &drawing.Line{
			Layer: layer,
			Start: [3]float64{float64(c.Start().X), float64(c.Start().Y), 0},
			End:   [3]float64{float64(c.End().X), float64(c.End().Y), 0},
		})
	}
	return out
}

// titleBlockEntities emits the title block at the border's lower-right: an outlined box with a
// row per resolved field (label left, value right). A sheet with no title block yields nothing.
func titleBlockEntities(sheet *dmodel.Sheet, layer string) []drawing.Entity {
	tb, ok := sheet.TitleBlock().(*dmodel.TitleBlock)
	if !ok {
		return nil
	}
	fields := tb.Fields()
	if len(fields) == 0 {
		return nil
	}
	left, right, top, bottom := borderMargins(sheet)
	const tbWidth, rowH = 80.0, 8.0 // mm
	x0, y0 := titleBlockCorner(tb.Location(), sheet.WidthMM(), sheet.HeightMM(),
		left, right, top, bottom, tbWidth, rowH*float64(len(fields)))
	x1 := x0 + tbWidth
	out := rectEntities(layer, x0, y0, x1, y0+rowH*float64(len(fields)))
	return append(out, titleBlockRows(layer, fields, x0, y0, x1, tbWidth, rowH)...)
}

// titleBlockCorner returns the title block's lower-left origin (x0, y0) for its location, insetting a
// tbWidth×blockH block against the chosen sheet corner within the border margins (#1989).
func titleBlockCorner(loc types.TitleBlockLocation, w, h, left, right, top, bottom, tbWidth, blockH float64) (x0, y0 float64) {
	x0 = left // left-hand corners
	if loc == types.BottomRightTitleBlock || loc == types.TopRightTitleBlock {
		x0 = w - right - tbWidth
	}
	y0 = bottom // bottom corners
	if loc == types.TopLeftTitleBlock || loc == types.TopRightTitleBlock {
		y0 = h - top - blockH
	}
	return x0, y0
}

// titleBlockRows emits each field's row: a divider line (except the first) and the label/value
// text, on the title-block layer.
func titleBlockRows(layer string, fields []dmodel.ResolvedField, x0, y0, x1, tbWidth, rowH float64) []drawing.Entity {
	var out []drawing.Entity
	for i, f := range fields {
		ry := y0 + rowH*float64(i)
		if i > 0 {
			out = append(out, &drawing.Line{Layer: layer, Start: [3]float64{x0, ry, 0}, End: [3]float64{x1, ry, 0}})
		}
		out = append(out, &drawing.Text{Layer: layer, Position: [3]float64{x0 + 2, ry + 2, 0}, Height: 2.5, Value: f.Name})
		out = append(out, &drawing.Text{Layer: layer, Position: [3]float64{x0 + tbWidth*0.45, ry + 2, 0}, Height: 2.5, Value: f.Value})
	}
	return out
}

// borderMargins returns the sheet's border margins, or zeros when it has no border.
func borderMargins(sheet *dmodel.Sheet) (left, right, top, bottom float64) {
	if b := sheet.Border(); b != nil {
		return b.Margins()
	}
	return 0, 0, 0, 0
}

// rectEntities emits the four edges of an axis-aligned rectangle on the layer.
func rectEntities(layer string, x0, y0, x1, y1 float64) []drawing.Entity {
	return []drawing.Entity{
		&drawing.Line{Layer: layer, Start: [3]float64{x0, y0, 0}, End: [3]float64{x1, y0, 0}},
		&drawing.Line{Layer: layer, Start: [3]float64{x1, y0, 0}, End: [3]float64{x1, y1, 0}},
		&drawing.Line{Layer: layer, Start: [3]float64{x1, y1, 0}, End: [3]float64{x0, y1, 0}},
		&drawing.Line{Layer: layer, Start: [3]float64{x0, y1, 0}, End: [3]float64{x0, y0, 0}},
	}
}
