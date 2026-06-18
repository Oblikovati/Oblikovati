// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Hole notes (M14-F07, #637): a feature note on each hole in a base view — a leadered diameter
// callout (Ø<d>) whose text is COMPUTED from the hole's circular edge, so it re-resolves when the
// model changes (a hole drilled larger updates its note). Distinct from a plain leader note, whose
// text is fixed. One annotation carries every hole's callout; recompute rebuilds it from the
// current projection (associative to the view, like centre marks).

const (
	holeNoteOffsetMM = 16.0 // how far the callout text sits from the hole centre
	holeNoteGapMM    = 1.5  // leader gap short of the hole rim, so the arrow doesn't touch it
)

// AddHoleNotes adds the hole-note annotation for the named base view: a leadered diameter callout
// per hole. It errors when no holes resolve (no model, a non-base view, or no circular edges).
func (as *DrawingAnnotations) AddHoleNotes(name, viewName string) (*DrawingAnnotation, error) {
	if _, _, _, err := as.annotationBasis(viewName); err != nil {
		return nil, err
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.HoleNoteAnnotation, viewName: viewName}
	as.recomputeHoleNotes(a)
	if a.rowCount == 0 {
		return nil, fmt.Errorf("drawing: view %q has no holes for hole notes", viewName)
	}
	as.items = append(as.items, a)
	return a, nil
}

// recomputeHoleNotes re-reads the view's holes and rebuilds a leadered Ø callout for each; with no
// resolvable holes it clears the annotation.
func (as *DrawingAnnotations) recomputeHoleNotes(a *DrawingAnnotation) {
	a.curves, a.labels, a.rowCount = nil, nil, 0
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, c := range circlesFromProjection(body, basis) {
		s := view.place(c.center)
		key := fmt.Sprintf("%.1f/%.1f", float64(s.X), float64(s.Y))
		if seen[key] {
			continue
		}
		seen[key] = true
		curves, label := holeNoteCallout(float64(s.X), float64(s.Y), c.radius*cmToMM, c.radius*2*cmToMM)
		a.curves = append(a.curves, curves...)
		a.labels = append(a.labels, label)
		a.rowCount++
	}
}

// holeNoteCallout builds one hole's leadered diameter callout: a leader from the text anchor to the
// hole rim (with an arrowhead) and the Ø<d> text label, placed up-right of the hole centre.
func holeNoteCallout(cx, cy, radiusMM, diameterMM float64) ([]DrawingCurve, AnnotationLabel) {
	const k = 0.70710678 // unit diagonal (cos 45°)
	tx, ty := cx+holeNoteOffsetMM*k, cy-holeNoteOffsetMM*k
	// Start the leader on the hole rim nearest the text, stopping a small gap short.
	rx, ry := cx+(radiusMM+holeNoteGapMM)*k, cy-(radiusMM+holeNoteGapMM)*k
	curves := []DrawingCurve{dimSegment(tx, ty, rx, ry)}
	curves = append(curves, noteArrowhead(tx, ty, rx, ry)...)
	return curves, AnnotationLabel{Text: "Ø" + holeCoord(diameterMM), X: tx + 6, Y: ty}
}
