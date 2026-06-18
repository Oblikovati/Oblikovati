// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
)

// Hole tables (M14-F04 PBI-144, #391): a row per circular edge in a base view, with its X/Y
// position from the view's datum origin (the lower-left-most hole) and its diameter. The table
// re-reads the view's holes on recompute, so it updates with the model.

// holeTableColumns are the hole table's columns.
var holeTableColumns = []gridColumn{
	{"HOLE", 14}, {"X", 22}, {"Y", 22}, {"Ø", 18},
}

// holeRow is one projected hole: its centre on the view plane (cm) and its diameter (mm).
type holeRow struct {
	cx, cy   float64
	diameter float64
}

// AddHoleTable adds a hole table at (x, y) on the sheet listing every circular edge in the named
// base view. It errors when no holes resolve (no model, a non-base view, or no circular edges).
func (as *DrawingAnnotations) AddHoleTable(name, viewName string, x, y float64) (*DrawingAnnotation, error) {
	if _, _, _, err := as.annotationBasis(viewName); err != nil {
		return nil, err
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.HoleTableAnnotation, viewName: viewName, x: x, y: y}
	as.recomputeHoleTable(a)
	if a.rowCount == 0 {
		return nil, fmt.Errorf("drawing: view %q has no circular edges for a hole table", viewName)
	}
	as.items = append(as.items, a)
	return a, nil
}

// recomputeHoleTable re-reads the view's circular edges, measures each hole's position from the
// datum and its diameter, and rebuilds the table; with no resolvable holes it clears it.
func (as *DrawingAnnotations) recomputeHoleTable(a *DrawingAnnotation) {
	a.curves, a.labels, a.rowCount = nil, nil, 0
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	holes := projectedHoles(view, body, basis)
	if len(holes) == 0 {
		return
	}
	a.rowCount = len(holes)
	a.curves, a.labels = holeTableGeometry(a.x, a.y, holes)
}

// projectedHoles collects each distinct hole's centre (on the view plane, cm) and diameter (mm),
// recovered by circle-fitting the view's projected edge loops (so it sees cut holes, not just
// primitive cylinders). Coincident projections (a through-hole's two rims) are listed once — keyed
// on the sheet position like the centre-mark dedup.
func projectedHoles(view *DrawingView, body *topo.Body, basis hlr.View) []holeRow {
	seen := map[string]bool{}
	var out []holeRow
	for _, c := range circlesFromProjection(body, basis) {
		s := view.place(c.center)
		key := fmt.Sprintf("%.1f/%.1f", float64(s.X), float64(s.Y))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, holeRow{cx: float64(c.center.X), cy: float64(c.center.Y), diameter: c.radius * 2 * cmToMM})
	}
	return out
}

// holeTableGeometry builds the hole table's grid + cell text: each hole's X/Y from the datum origin
// (the lower-left-most hole) in millimetres, and its diameter.
func holeTableGeometry(x, y float64, holes []holeRow) ([]DrawingCurve, []AnnotationLabel) {
	datumX, datumY := math.Inf(1), math.Inf(1)
	for _, h := range holes {
		datumX, datumY = math.Min(datumX, h.cx), math.Min(datumY, h.cy)
	}
	cells := make([][]string, len(holes))
	for i, h := range holes {
		cells[i] = []string{
			"H" + strconv.Itoa(i+1),
			holeCoord((h.cx - datumX) * cmToMM),
			holeCoord((h.cy - datumY) * cmToMM),
			"Ø" + holeCoord(h.diameter),
		}
	}
	return gridTableGeometry(x, y, holeTableColumns, cells)
}

// holeCoord formats a millimetre value to two decimals (the hole-table convention).
func holeCoord(mm float64) string { return strconv.FormatFloat(mm, 'f', 2, 64) }
