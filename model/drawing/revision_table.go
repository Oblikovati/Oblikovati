// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Revision tables & tags (M14-F04 PBI-144, #391): a revision table records the drawing's change
// history (a row per revision: revision/date/description) in a boxed grid; a revision tag is a
// triangle holding a revision letter, placed on the sheet to flag where that revision changed it.
// Both are user-supplied markup (not model-derived), so their geometry is a pure function of the
// stored fields and they persist verbatim.

// revisionTableColumns are the revision table's columns.
var revisionTableColumns = []gridColumn{
	{"REV", 14}, {"DATE", 28}, {"DESCRIPTION", 60},
}

// revisionTagHalfMM is half the revision tag triangle's width/height.
const revisionTagHalfMM = 6.0

// AddRevisionTable adds a revision table at (x, y) on the sheet listing the given revision rows. It
// errors when no rows are supplied (an empty revision table is meaningless).
func (as *DrawingAnnotations) AddRevisionTable(name string, x, y float64, rows []RevisionRow) (*DrawingAnnotation, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("drawing: a revision table needs at least one revision row, got 0")
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.RevisionTableAnnotation,
		x: x, y: y, revisions: rows, rowCount: len(rows),
	}
	a.curves, a.labels = revisionTableGeometry(x, y, rows)
	as.items = append(as.items, a)
	return a, nil
}

// revisionTableGeometry builds the revision table's grid + cell text from its rows.
func revisionTableGeometry(x, y float64, rows []RevisionRow) ([]DrawingCurve, []AnnotationLabel) {
	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = []string{r.Revision, r.Date, r.Description}
	}
	return gridTableGeometry(x, y, revisionTableColumns, cells)
}

// AddRevisionTag adds a revision tag (a triangle holding the revision letter) centred at (x, y). It
// errors with an empty revision.
func (as *DrawingAnnotations) AddRevisionTag(name string, x, y float64, revision string) (*DrawingAnnotation, error) {
	if revision == "" {
		return nil, fmt.Errorf("drawing: a revision tag needs a revision letter, got %q", revision)
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.RevisionTagAnnotation, x: x, y: y, tag: revision}
	a.curves, a.labels = revisionTagGeometry(x, y, revision)
	as.items = append(as.items, a)
	return a, nil
}

// revisionTagGeometry builds the revision tag's triangle (drawing curves) and the revision letter
// label centred in it. The triangle points up, with the letter on its baseline.
func revisionTagGeometry(x, y float64, revision string) ([]DrawingCurve, []AnnotationLabel) {
	top := [2]float64{x, y - revisionTagHalfMM}
	left := [2]float64{x - revisionTagHalfMM, y + revisionTagHalfMM}
	right := [2]float64{x + revisionTagHalfMM, y + revisionTagHalfMM}
	curves := []DrawingCurve{
		dimSegment(top[0], top[1], left[0], left[1]),
		dimSegment(left[0], left[1], right[0], right[1]),
		dimSegment(right[0], right[1], top[0], top[1]),
	}
	return curves, []AnnotationLabel{{Text: revision, X: x, Y: y + revisionTagHalfMM/2}}
}
