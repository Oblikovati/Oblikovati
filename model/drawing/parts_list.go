// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/types"
)

// Parts lists (M14-F04 PBI-143, #390): a table on the sheet sourced from the referenced assembly's
// parts-only BOM — one row per item with its item number, part number, description and quantity.
// The grid and header are drawing curves; each cell's text is a label. The rows re-read from the
// BOM on recompute, so the list updates with the assembly.

// PartsListRow is one BOM line a parts list shows: the item number, the part metadata and the
// quantity. It is the drawing package's view of a BOM row (the host maps model/bom into it).
type PartsListRow struct {
	Item        int
	PartNumber  string
	Description string
	Quantity    int
}

// bomLookup resolves the drawing's referenced assembly to its parts-only BOM rows.
type bomLookup func() ([]PartsListRow, bool)

// partsListColumns are the table's columns (header text + width in mm), left to right.
var partsListColumns = []struct {
	header string
	width  float64
}{
	{"ITEM", 12}, {"PART NUMBER", 40}, {"DESCRIPTION", 50}, {"QTY", 12},
}

// partsListRowMM is each row's height.
const partsListRowMM = 8.0

// AddPartsList adds a parts list table at (x, y) on the sheet (its top-left corner), sourced from
// the referenced assembly's parts-only BOM. It errors when no BOM resolves (no reference, an
// unwired resolver, or a non-assembly model).
func (as *DrawingAnnotations) AddPartsList(name string, x, y float64) (*DrawingAnnotation, error) {
	if as.bom == nil {
		return nil, fmt.Errorf("drawing: no BOM source for a parts list")
	}
	if _, ok := as.bom(); !ok {
		return nil, fmt.Errorf("drawing: the referenced model has no BOM (reference an assembly)")
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.PartsListAnnotation, x: x, y: y}
	as.recomputePartsList(a)
	as.items = append(as.items, a)
	return a, nil
}

// recomputePartsList re-reads the BOM and rebuilds the table grid + cell text at the list's anchor;
// with no resolvable BOM it clears the table.
func (as *DrawingAnnotations) recomputePartsList(a *DrawingAnnotation) {
	a.curves, a.labels, a.rowCount = nil, nil, 0
	if as.bom == nil {
		return
	}
	rows, ok := as.bom()
	if !ok {
		return
	}
	a.rowCount = len(rows)
	a.curves, a.labels = partsListGeometry(a.x, a.y, rows)
}

// partsListGeometry builds the parts list's grid (drawing curves) and cell text (labels), with
// (x, y) the table's top-left corner. The header row is at the top; data rows grow downward.
func partsListGeometry(x, y float64, rows []PartsListRow) ([]DrawingCurve, []AnnotationLabel) {
	total := 0.0
	for _, c := range partsListColumns {
		total += c.width
	}
	lineCount := len(rows) + 1 // header + one per row
	bottom := y - float64(lineCount)*partsListRowMM
	var curves []DrawingCurve
	// Horizontal grid lines (top, under header, between rows, bottom).
	for i := 0; i <= lineCount; i++ {
		ly := y - float64(i)*partsListRowMM
		curves = append(curves, dimSegment(x, ly, x+total, ly))
	}
	// Vertical grid lines (column dividers + outer left/right).
	cx := x
	for _, c := range partsListColumns {
		curves = append(curves, dimSegment(cx, y, cx, bottom))
		cx += c.width
	}
	curves = append(curves, dimSegment(cx, y, cx, bottom))
	return curves, partsListLabels(x, y, rows)
}

// partsListLabels centres each header and cell value in its column/row.
func partsListLabels(x, y float64, rows []PartsListRow) []AnnotationLabel {
	var labels []AnnotationLabel
	addRow := func(rowTop float64, cells []string) {
		cx := x
		cy := rowTop - partsListRowMM/2
		for i, c := range partsListColumns {
			labels = append(labels, AnnotationLabel{Text: cells[i], X: cx + c.width/2, Y: cy})
			cx += c.width
		}
	}
	headers := make([]string, len(partsListColumns))
	for i, c := range partsListColumns {
		headers[i] = c.header
	}
	addRow(y, headers)
	for i, r := range rows {
		addRow(y-float64(i+1)*partsListRowMM, []string{strconv.Itoa(r.Item), r.PartNumber, r.Description, strconv.Itoa(r.Quantity)})
	}
	return labels
}
