// SPDX-License-Identifier: GPL-2.0-only

package dwg

// The drawing entity model is format-neutral and lives in kernel/exchange/drawing so the
// DWG and DXF codecs share one model and one Sketch converter (ADR-0018's "define once,
// alias" idiom). These aliases keep every existing dwg call site — type switches,
// struct literals, the decoders and the writer — compiling unchanged; the DWG-specific
// bits (ObjectType codes, bit-stream INSERT decode, block expansion) stay in this package.
import "oblikovati.org/kernel/exchange/drawing"

type (
	Drawing    = drawing.Drawing
	Entity     = drawing.Entity
	Line       = drawing.Line
	Circle     = drawing.Circle
	Arc        = drawing.Arc
	Point      = drawing.Point
	Ellipse    = drawing.Ellipse
	LwPolyline = drawing.LwPolyline
	Spline     = drawing.Spline
	Insert     = drawing.Insert
)

// ScaleEntities and MetersPerUnit are re-exported so the model layer's existing
// dwg.ScaleEntities / dwg.MetersPerUnit references resolve to the shared implementation.
var (
	ScaleEntities = drawing.ScaleEntities
	MetersPerUnit = drawing.MetersPerUnit
)
