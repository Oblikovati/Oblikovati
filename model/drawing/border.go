// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"strconv"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// BorderDefinition is a reusable border template: the printable-area margins inset
// (millimetres) from the sheet edge on each side, and an optional zone grid (#1989) —
// hZones columns × vZones rows labelled per hLabelMode / vLabelMode. A drawing standard
// owns a set of these.
type BorderDefinition struct {
	name                     string
	left, right, top, bottom float64
	hZones, vZones           int
	hLabelMode, vLabelMode   types.BorderLabelMode
}

// DefaultBorderDefinition is the standard border: 10 mm on three sides and a wider
// 20 mm left (binding) edge — a common drafting default, with no zone grid.
func DefaultBorderDefinition() *BorderDefinition {
	return &BorderDefinition{name: "Default", left: 20, right: 10, top: 10, bottom: 10}
}

// ZonedBorderDefinition is a default-margin border with an hZones×vZones zone grid, the
// horizontal (column) zones labelled per hLabelMode and the vertical (row) zones per
// vLabelMode (#1989).
func ZonedBorderDefinition(hZones, vZones int, hLabelMode, vLabelMode types.BorderLabelMode) *BorderDefinition {
	d := DefaultBorderDefinition()
	d.name = "Zoned"
	d.hZones, d.vZones = hZones, vZones
	d.hLabelMode, d.vLabelMode = hLabelMode, vLabelMode
	return d
}

// Name returns the border definition's name.
func (d *BorderDefinition) Name() string { return d.name }

// Border is a sheet's border — an instance of a [BorderDefinition].
type Border struct {
	def *BorderDefinition
}

func newBorder(def *BorderDefinition) *Border { return &Border{def: def} }

// DefinitionName returns the name of the border definition this border instantiates.
func (b *Border) DefinitionName() string { return b.def.name }

// Margins returns the left, right, top and bottom inset in millimetres
// (contract.DrawingBorder).
func (b *Border) Margins() (left, right, top, bottom float64) {
	return b.def.left, b.def.right, b.def.top, b.def.bottom
}

// ZoneCounts returns the border's horizontal (column) and vertical (row) zone counts; 0,0 when the
// border has no zone grid (#1989).
func (b *Border) ZoneCounts() (h, v int) { return b.def.hZones, b.def.vZones }

// ColumnLabels returns the horizontal zones' labels, left to right, per the horizontal label mode.
func (b *Border) ColumnLabels() []string { return zoneLabels(b.def.hZones, b.def.hLabelMode) }

// RowLabels returns the vertical zones' labels, top to bottom, per the vertical label mode.
func (b *Border) RowLabels() []string { return zoneLabels(b.def.vZones, b.def.vLabelMode) }

// ZoneDivisions returns the interior grid lines dividing the printable area into zones (sheet mm),
// for a sheet of the given width and height; empty when the border has no zone grid (#1989).
func (b *Border) ZoneDivisions(sheetW, sheetH float64) []DrawingCurve {
	x0, y0 := b.def.left, b.def.bottom
	x1, y1 := sheetW-b.def.right, sheetH-b.def.top
	var out []DrawingCurve
	for i := 1; i < b.def.hZones; i++ { // vertical lines between columns
		x := x0 + (x1-x0)*float64(i)/float64(b.def.hZones)
		out = append(out, borderSegment(x, y0, x, y1))
	}
	for j := 1; j < b.def.vZones; j++ { // horizontal lines between rows
		y := y0 + (y1-y0)*float64(j)/float64(b.def.vZones)
		out = append(out, borderSegment(x0, y, x1, y))
	}
	return out
}

// borderSegment builds a visible drawing curve between two sheet points.
func borderSegment(ax, ay, bx, by float64) DrawingCurve {
	return DrawingCurve{A: gmath.P2(gmath.Scalar(ax), gmath.Scalar(ay)), B: gmath.P2(gmath.Scalar(bx), gmath.Scalar(by)), Visible: true}
}

// zoneLabels builds n zone labels for a label mode: "A".."Z","AA".. alphabetical, "1".."n" numeric,
// or empty strings for the none mode (the divisions are still drawn).
func zoneLabels(n int, mode types.BorderLabelMode) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		switch mode {
		case types.NumericBorderLabel:
			out = append(out, strconv.Itoa(i+1))
		case types.NoBorderLabel:
			out = append(out, "")
		default:
			out = append(out, alphaLabel(i))
		}
	}
	return out
}

// alphaLabel returns the spreadsheet-style column label for a zero-based index: A..Z, AA..AZ, ….
func alphaLabel(i int) string {
	label := ""
	for i >= 0 {
		label = string(rune('A'+i%26)) + label
		i = i/26 - 1
	}
	return label
}
