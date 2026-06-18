// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Custom tables (M14-F04 PBI-144, #391): a general-purpose grid with arbitrary column headers and
// rows, for drawing content that the typed tables (parts list, hole table, revision table) don't
// cover. The headers and rows are user-supplied, so they persist verbatim.

// customTableColumnMM is each custom-table column's width (uniform, since the content is arbitrary).
const customTableColumnMM = 36.0

// AddCustomTable adds a general-purpose table at (x, y) on the sheet with the given column headers
// and rows (each row's cells align to the headers). It errors with no headers.
func (as *DrawingAnnotations) AddCustomTable(name string, x, y float64, headers []string, rows [][]string) (*DrawingAnnotation, error) {
	if len(headers) == 0 {
		return nil, fmt.Errorf("drawing: a custom table needs at least one column header, got 0")
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.CustomTableAnnotation,
		x: x, y: y, headers: headers, tableRows: rows, rowCount: len(rows),
	}
	a.curves, a.labels = customTableGeometry(x, y, headers, rows)
	as.items = append(as.items, a)
	return a, nil
}

// customTableGeometry builds the custom table's grid + cell text from its headers and rows, padding
// short rows so every cell aligns to its column.
func customTableGeometry(x, y float64, headers []string, rows [][]string) ([]DrawingCurve, []AnnotationLabel) {
	columns := make([]gridColumn, len(headers))
	for i, h := range headers {
		columns[i] = gridColumn{header: h, width: customTableColumnMM}
	}
	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = padRow(r, len(headers))
	}
	return gridTableGeometry(x, y, columns, cells)
}

// padRow returns row padded with empty cells (or truncated) to exactly n cells, so a ragged input
// still aligns to the column count.
func padRow(row []string, n int) []string {
	out := make([]string, n)
	copy(out, row)
	return out
}
