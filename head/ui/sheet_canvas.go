//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/head/internal/native"
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

// drawSheetCanvas renders the active drawing's active sheet across the viewport panel.
// It reads the resolved content directly (display-only in F01; canvas interaction arrives
// with drawing views in F02).
func drawSheetCanvas(c *drawing.Content) {
	sheet := c.Sheets().Active()
	ox, oy := native.GetCursorScreenPos()
	availW, availH := native.ContentRegionAvail()
	native.InvisibleButton("##sheet-canvas", availW, availH) // reserve the region
	native.DrawQuadFilled(ox, oy, ox+availW, oy, ox+availW, oy+availH, ox, oy+availH, canvasMat)
	if sheet == nil {
		native.DrawText(ox+12, oy+12, "Drawing has no sheet", captionInk)
		return
	}
	face := fitSheet(ox, oy, availW, availH, sheet.WidthMM(), sheet.HeightMM())
	drawSheetFace(face)
	inner := drawSheetBorder(face, sheet)
	drawTitleBlock(inner, sheet)
	drawSheetCaption(ox, oy, sheet, c.ModelReference(), c.Styles().ActiveStandard().String())
}

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
