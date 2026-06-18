// SPDX-License-Identifier: GPL-2.0-only

package drawing

// Drawing tables (M14-F04, #382): a grid of cells on the sheet — a header row plus one data row
// each — shared by the parts list (BOM-sourced) and the hole table (geometry-sourced). The grid
// lines are drawing curves; each header/cell value is a centred text label.

// gridColumn is one column of a drawing table: its header text and its width in millimetres.
type gridColumn struct {
	header string
	width  float64
}

// tableRowMM is each table row's height.
const tableRowMM = 8.0

// gridTableGeometry builds a table's grid (drawing curves) and its header + cell text (labels), with
// (x, y) the table's top-left corner; the header row is at the top and data rows grow downward. Each
// row of cells must have one value per column.
func gridTableGeometry(x, y float64, columns []gridColumn, cells [][]string) ([]DrawingCurve, []AnnotationLabel) {
	total := 0.0
	for _, c := range columns {
		total += c.width
	}
	lineCount := len(cells) + 1 // header + one per data row
	bottom := y - float64(lineCount)*tableRowMM
	var curves []DrawingCurve
	for i := 0; i <= lineCount; i++ { // horizontal grid lines
		ly := y - float64(i)*tableRowMM
		curves = append(curves, dimSegment(x, ly, x+total, ly))
	}
	cx := x
	for _, c := range columns { // vertical dividers + outer left/right
		curves = append(curves, dimSegment(cx, y, cx, bottom))
		cx += c.width
	}
	curves = append(curves, dimSegment(cx, y, cx, bottom))
	return curves, gridTableLabels(x, y, columns, cells)
}

// gridTableLabels centres each header and cell value in its column/row.
func gridTableLabels(x, y float64, columns []gridColumn, cells [][]string) []AnnotationLabel {
	var labels []AnnotationLabel
	addRow := func(rowTop float64, values []string) {
		cx := x
		cy := rowTop - tableRowMM/2
		for i, c := range columns {
			if i < len(values) {
				labels = append(labels, AnnotationLabel{Text: values[i], X: cx + c.width/2, Y: cy})
			}
			cx += c.width
		}
	}
	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = c.header
	}
	addRow(y, headers)
	for i, row := range cells {
		addRow(y-float64(i+1)*tableRowMM, row)
	}
	return labels
}
