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
var partsListColumns = []gridColumn{
	{"ITEM", 12}, {"PART NUMBER", 40}, {"DESCRIPTION", 50}, {"QTY", 12},
}

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

// partsListGeometry builds the parts list's grid + cell text from its BOM rows.
func partsListGeometry(x, y float64, rows []PartsListRow) ([]DrawingCurve, []AnnotationLabel) {
	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = []string{strconv.Itoa(r.Item), r.PartNumber, r.Description, strconv.Itoa(r.Quantity)}
	}
	return gridTableGeometry(x, y, partsListColumns, cells)
}
