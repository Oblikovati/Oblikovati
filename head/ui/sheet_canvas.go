//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	stdmath "math"
	"unicode/utf8"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/drawing"
)

// The drawing sheet canvas (M14-F01, #384): a 2D view of the active drawing's active
// sheet drawn directly into the viewport panel (ImGui draw-list, no 3D pipeline). It
// renders the sheet face, its border, and the title block with the fields resolved from
// the referenced model's iProperties — so a drawing document shows a real sheet instead
// of an empty 3D viewport.

// Canvas colors (0..1 RGBA): a dark mat, a paper-white sheet, dark ink, and faint rules.
var (
	canvasMat  = [4]float32{0.16, 0.17, 0.19, 1}
	sheetPaper = [4]float32{0.96, 0.96, 0.94, 1}
	sheetInk   = [4]float32{0.10, 0.10, 0.12, 1}
	sheetFaint = [4]float32{0.42, 0.43, 0.47, 1}
	captionInk = [4]float32{0.80, 0.82, 0.86, 1}
	titleLabel = [4]float32{0.35, 0.36, 0.40, 1}
)

// rect is a screen-space rectangle (top-left origin) plus the mm→pixel scale that
// produced it, so nested geometry (border, title block) shares one scale.
type rect struct {
	x, y, w, h float32
	scale      float32
}

// drawSheetCanvas renders the active drawing's active sheet across the viewport panel and
// handles canvas interaction (view placement, selection, right-click) — the drawing
// counterpart of the 3D viewport.
func drawSheetCanvas(s *app.Session, c *drawing.Content) {
	c.SyncViews()    // re-project if the referenced model was recomputed (live associativity)
	drawSheetTabs(c) // a tab per sheet so the user can switch which sheet the canvas shows
	sheet := c.Sheets().Active()
	ox, oy := native.GetCursorScreenPos()
	availW, availH := native.ContentRegionAvail()
	native.InvisibleButton("##sheet-canvas", availW, availH) // reserve the region
	hovered := native.IsItemHovered()
	mx, my := native.MousePos()
	native.DrawQuadFilled(ox, oy, ox+availW, oy, ox+availW, oy+availH, ox, oy+availH, canvasMat)
	if sheet == nil {
		native.DrawText(ox+12, oy+12, "Drawing has no sheet", captionInk)
		return
	}
	face := fitSheet(ox, oy, availW, availH, sheet.WidthMM(), sheet.HeightMM())
	drawSheetFace(face)
	inner := drawSheetBorder(face, sheet)
	drawTitleBlock(inner, sheet)
	drawSheetViews(s, face, sheet)
	drawSheetCaption(ox, oy, sheet, c.ModelReference(), c.Styles().ActiveStandard().String())
	handleSheetCanvasInput(s, face, hovered, mx, my)
}

// prevActiveSheetName tracks the active sheet across frames so a programmatic switch (New Sheet
// on the ribbon, an add-in) force-selects its tab once without fighting the user's tab clicks —
// the same one-frame-force pattern the document tabs use.
var prevActiveSheetName string

// drawSheetTabs renders a tab per sheet at the top of the drawing canvas so the user can switch
// the active sheet (the canvas only shows the active sheet). Clicking a tab activates that sheet;
// New Sheet / Delete Sheet stay on the ribbon. With a single sheet there is nothing to switch, so
// no strip is drawn.
func drawSheetTabs(c *drawing.Content) {
	sheets := c.Sheets()
	if sheets.Count() <= 1 {
		prevActiveSheetName = ""
		return
	}
	active := sheets.Active()
	var cur string
	if active != nil {
		cur = active.Name()
	}
	force := cur != prevActiveSheetName
	prevActiveSheetName = cur
	if native.BeginTabBar("##sheet-tabs") {
		for i := 0; i < sheets.Count(); i++ {
			drawSheetTab(sheets, sheets.Item(i), active, force)
		}
		native.EndTabBar()
	}
}

// drawSheetTab renders one sheet's tab and activates it when the user selects it (but not on the
// force frame, which only mirrors an external switch into ImGui's selection).
func drawSheetTab(sheets *drawing.Sheets, sh, active *drawing.Sheet, force bool) {
	selected := force && active != nil && sh.Name() == active.Name()
	if !native.BeginTabItemSelected(sheetTabLabel(sh), selected) {
		return
	}
	if !force && (active == nil || sh.Name() != active.Name()) {
		_ = sheets.SetActive(sh.Name())
	}
	native.EndTabItem()
}

// sheetTabLabel is the ImGui label for a sheet's tab: the sheet name, with a stable "###id" suffix
// (sheet names are unique within a drawing) so the visible text is just the name. See the document
// tabs for why ImGui tab ids must not collide.
func sheetTabLabel(sh *drawing.Sheet) string {
	return sh.Name() + "###sheet-" + sh.Name()
}

// View-curve colors: visible edges solid dark ink, hidden edges dashed in a lighter grey.
var (
	viewVisibleInk = [4]float32{0.10, 0.10, 0.12, 1}
	viewHiddenInk  = [4]float32{0.45, 0.46, 0.50, 1}
	viewSelectInk  = [4]float32{0.20, 0.55, 0.95, 1} // selected-view highlight box
)

// drawSheetViews draws every view's hidden-line curves onto the sheet: visible edges solid,
// hidden edges dashed. Curve coordinates are sheet millimetres (y up from the sheet bottom),
// mapped to the screen through the sheet's fit rectangle. The selected view gets a highlight
// box so canvas selection is visible.
func drawSheetViews(s *app.Session, face rect, sheet *drawing.Sheet) {
	selected := selectedDrawingView(s)
	views := sheet.Views()
	for i := 0; i < views.Count(); i++ {
		v := views.Item(i)
		for _, c := range v.Curves() {
			drawViewCurve(face, c)
		}
		if v == selected {
			drawViewHighlight(face, v)
		}
	}
	drawSheetSketches(face, sheet)
	drawSheetAnnotations(face, sheet)
	drawSheetDimensions(s, face, sheet)
}

// drawSheetSketches strokes the sheet's drawing-sketch curves (2D geometry the user drew in sheet
// space) in the visible-edge ink, under the annotations.
func drawSheetSketches(face rect, sheet *drawing.Sheet) {
	ss := sheet.Sketches()
	for i := 0; i < ss.Count(); i++ {
		for _, c := range ss.Item(i).Curves() {
			ax, ay := curveToScreen(face, c.Start())
			bx, by := curveToScreen(face, c.End())
			native.DrawLine(ax, ay, bx, by, viewVisibleInk, 1.2)
		}
	}
}

// drawSheetDimensions strokes each dimension's glyph (extension/dimension lines, arrowheads) and
// draws its centred value text. The part being dragged (the line or the text) is highlighted so
// the active selection is visible.
func drawSheetDimensions(s *app.Session, face rect, sheet *drawing.Sheet) {
	dragName, dragText, dragging := s.DraggingDimension()
	ds := sheet.Dimensions()
	for i := 0; i < ds.Count(); i++ {
		d := ds.Item(i)
		grabbed := dragging && d.Name() == dragName
		lineInk, thick := viewVisibleInk, float32(1.2)
		if grabbed && !dragText {
			lineInk, thick = viewSelectInk, 1.8
		}
		for _, c := range d.Curves() {
			ax, ay := curveToScreen(face, c.Start())
			bx, by := curveToScreen(face, c.End())
			native.DrawLine(ax, ay, bx, by, lineInk, thick)
		}
		tx, ty := curveToScreen(face, math.P2(math.Scalar(dimAnchorX(d)), math.Scalar(dimAnchorY(d))))
		runes := utf8.RuneCountInString(d.Text()) // centre the text on its (lifted) anchor
		textInk := sheetInk
		if grabbed && dragText {
			textInk = viewSelectInk
			drawTextHighlightBox(tx, ty, runes)
		}
		native.DrawText(tx-float32(runes)*3.2, ty-7, d.Text(), textInk)
	}
}

// drawTextHighlightBox outlines the dragged value text in the highlight colour.
func drawTextHighlightBox(tx, ty float32, runes int) {
	w := float32(runes)*6.4 + 4
	rectOutline(rect{x: tx - w/2, y: ty - 9, w: w, h: 16}, viewSelectInk, 1)
}

// dimAnchorX / dimAnchorY split the dimension's text-anchor for the mm→screen mapping.
func dimAnchorX(d *drawing.DrawingDimension) float64 { x, _ := d.TextAnchorMM(); return x }
func dimAnchorY(d *drawing.DrawingDimension) float64 { _, y := d.TextAnchorMM(); return y }

// drawSheetAnnotations strokes the sheet's annotation curves (CoG markers, revision clouds) in
// the accent ink, over the views.
func drawSheetAnnotations(face rect, sheet *drawing.Sheet) {
	an := sheet.Annotations()
	for i := 0; i < an.Count(); i++ {
		a := an.Item(i)
		for _, c := range a.Curves() {
			ax, ay := curveToScreen(face, c.Start())
			bx, by := curveToScreen(face, c.End())
			native.DrawLine(ax, ay, bx, by, viewSelectInk, 1.4)
		}
		for _, l := range a.Labels() { // FCF tolerance / datum text, centred on its anchor
			tx, ty := curveToScreen(face, math.P2(math.Scalar(l.X), math.Scalar(l.Y)))
			native.DrawText(tx-float32(utf8.RuneCountInString(l.Text))*3.2, ty-7, l.Text, sheetInk)
		}
	}
}

// selectedDrawingView returns the currently selected view (canvas/browser), or nil.
func selectedDrawingView(s *app.Session) *drawing.DrawingView {
	if h, ok := s.Selection().First().(app.DrawingViewHandle); ok {
		return h.View
	}
	return nil
}

// drawViewHighlight outlines the selected view's bounds in the highlight color.
func drawViewHighlight(face rect, v *drawing.DrawingView) {
	minX, minY, maxX, maxY, ok := v.BoundsMM()
	if !ok {
		return
	}
	const pad = 3 // mm breathing room around the geometry
	x0, y0 := curveToScreen(face, math.P2(math.Scalar(minX-pad), math.Scalar(maxY+pad)))
	x1, y1 := curveToScreen(face, math.P2(math.Scalar(maxX+pad), math.Scalar(minY-pad)))
	native.DrawLine(x0, y0, x1, y0, viewSelectInk, 1)
	native.DrawLine(x1, y0, x1, y1, viewSelectInk, 1)
	native.DrawLine(x1, y1, x0, y1, viewSelectInk, 1)
	native.DrawLine(x0, y1, x0, y0, viewSelectInk, 1)
}

// drawViewCurve strokes one drawing curve in the style of its kind: a section cut outline bold,
// hatch lines thin and faint, and model edges solid (visible) or dashed (hidden).
func drawViewCurve(face rect, c drawing.DrawingCurve) {
	ax, ay := curveToScreen(face, c.Start())
	bx, by := curveToScreen(face, c.End())
	switch c.Kind() {
	case types.DrawingSectionCurve:
		native.DrawLine(ax, ay, bx, by, viewVisibleInk, 2.2)
	case types.DrawingHatchCurve:
		native.DrawLine(ax, ay, bx, by, sheetFaint, 1)
	case types.DrawingBreakCurve:
		native.DrawLine(ax, ay, bx, by, viewVisibleInk, 1)
	default:
		if c.IsVisible() {
			native.DrawLine(ax, ay, bx, by, viewVisibleInk, 1.4)
		} else {
			drawDashed(ax, ay, bx, by, viewHiddenInk)
		}
	}
}

// curveToScreen maps a sheet-millimetre point (y up) to screen pixels within the sheet face.
func curveToScreen(face rect, p math.Point2) (float32, float32) {
	return face.x + float32(p.X)*face.scale, face.y + face.h - float32(p.Y)*face.scale
}

// drawDashed strokes a dashed line (a hidden edge) as a run of short on/off segments.
func drawDashed(ax, ay, bx, by float32, c [4]float32) {
	const dash = 4 // pixels on, then off
	dx, dy := bx-ax, by-ay
	length := float32(stdSqrt(float64(dx*dx + dy*dy)))
	if length < 1e-3 {
		return
	}
	ux, uy := dx/length, dy/length
	for t := float32(0); t < length; t += 2 * dash {
		s1 := t + dash
		if s1 > length {
			s1 = length
		}
		native.DrawLine(ax+ux*t, ay+uy*t, ax+ux*s1, ay+uy*s1, c, 1.2)
	}
}

func stdSqrt(v float64) float64 { return stdmath.Sqrt(v) }

// fitSheet centers the sheet (given in mm) inside the panel with padding and returns its
// screen rectangle and the mm→pixel scale.
func fitSheet(ox, oy, availW, availH float32, sheetWmm, sheetHmm float64) rect {
	const pad = 40
	scale := minf(float64(availW-2*pad)/sheetWmm, float64(availH-2*pad)/sheetHmm)
	if scale <= 0 {
		scale = 1
	}
	w := float32(sheetWmm * scale)
	h := float32(sheetHmm * scale)
	return rect{x: ox + (availW-w)/2, y: oy + (availH-h)/2, w: w, h: h, scale: float32(scale)}
}

// drawSheetFace paints the paper-white sheet and outlines it.
func drawSheetFace(r rect) {
	native.DrawQuadFilled(r.x, r.y, r.x+r.w, r.y, r.x+r.w, r.y+r.h, r.x, r.y+r.h, sheetPaper)
	rectOutline(r, sheetInk, 1.5)
}

// drawSheetBorder insets the sheet by its border margins (mm) and outlines the printable
// area, returning that inner rectangle for the title block. With no border it returns the
// full face.
func drawSheetBorder(face rect, sheet *drawing.Sheet) rect {
	b := sheet.Border()
	if b == nil {
		return face
	}
	left, right, top, bottom := b.Margins()
	inner := rect{
		x:     face.x + float32(left)*face.scale,
		y:     face.y + float32(top)*face.scale,
		w:     face.w - float32(left+right)*face.scale,
		h:     face.h - float32(top+bottom)*face.scale,
		scale: face.scale,
	}
	rectOutline(inner, sheetInk, 1)
	return inner
}

// drawTitleBlock draws the title block at the inner area's lower-right, one row per
// resolved field (label on the left, value on the right).
func drawTitleBlock(inner rect, sheet *drawing.Sheet) {
	tb, ok := sheet.TitleBlock().(*drawing.TitleBlock)
	if !ok {
		return
	}
	fields := tb.Fields()
	if len(fields) == 0 {
		return
	}
	box := titleBlockRect(inner, len(fields))
	native.DrawQuadFilled(box.x, box.y, box.x+box.w, box.y, box.x+box.w, box.y+box.h, box.x, box.y+box.h, sheetPaper)
	rectOutline(box, sheetInk, 1.5)
	rowH := box.h / float32(len(fields))
	for i, f := range fields {
		ry := box.y + float32(i)*rowH
		if i > 0 {
			native.DrawLine(box.x, ry, box.x+box.w, ry, sheetFaint, 1)
		}
		native.DrawText(box.x+4, ry+2, f.Name, titleLabel)
		native.DrawText(box.x+box.w*0.42, ry+2, f.Value, sheetInk)
	}
}

// titleBlockRect places the title block: 70 mm wide (clamped to the inner width and a
// readable minimum) and a readable row height, anchored to the inner lower-right corner.
func titleBlockRect(inner rect, rows int) rect {
	w := clampf(70*inner.scale, 150, inner.w)
	rowH := maxf(7*inner.scale, 14)
	h := rowH * float32(rows)
	return rect{x: inner.x + inner.w - w, y: inner.y + inner.h - h, w: w, h: h, scale: inner.scale}
}

// drawSheetCaption labels the canvas (top-left) with the sheet, the referenced model, and
// the active drafting standard.
func drawSheetCaption(ox, oy float32, sheet *drawing.Sheet, modelRef, standard string) {
	caption := fmt.Sprintf("%s — %s %s (%.0f×%.0f mm)",
		sheet.Name(), sheet.Size(), sheet.Orientation(), sheet.WidthMM(), sheet.HeightMM())
	native.DrawText(ox+12, oy+10, caption, captionInk)
	model := modelRef
	if model == "" {
		model = "(no model referenced)"
	}
	native.DrawText(ox+12, oy+28, "Model: "+model+"    Standard: "+standard, captionInk)
}

// rectOutline strokes a rectangle's four edges.
func rectOutline(r rect, c [4]float32, th float32) {
	native.DrawLine(r.x, r.y, r.x+r.w, r.y, c, th)
	native.DrawLine(r.x+r.w, r.y, r.x+r.w, r.y+r.h, c, th)
	native.DrawLine(r.x+r.w, r.y+r.h, r.x, r.y+r.h, c, th)
	native.DrawLine(r.x, r.y+r.h, r.x, r.y, c, th)
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
