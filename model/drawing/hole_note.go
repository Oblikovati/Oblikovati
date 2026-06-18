// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"strconv"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
)

// Hole notes (M14-F07, #637): a feature note on each hole in a base view — a leadered diameter
// callout (Ø<d>) whose text is COMPUTED from the hole's circular edge, so it re-resolves when the
// model changes (a hole drilled larger updates its note). Distinct from a plain leader note, whose
// text is fixed. One annotation carries every hole's callout; recompute rebuilds it from the
// current projection (associative to the view, like centre marks). A combined quantity mode groups
// holes by diameter into one "<n>x Ø<d>" callout per size.

const (
	holeNoteOffsetMM = 16.0 // how far the callout text sits from the hole centre
	holeNoteGapMM    = 1.5  // leader gap short of the hole rim, so the arrow doesn't touch it
)

// AddHoleNotes adds the hole-note annotation for the named base view: a leadered diameter callout
// per hole (or per distinct diameter, when quantity is combined). An optional format template with
// {d} (diameter) and {n} (count) placeholders overrides the callout text; empty uses the default.
// It errors when no holes resolve (no model, a non-base view, or no circular edges).
func (as *DrawingAnnotations) AddHoleNotes(name, viewName string, quantity types.HoleNoteQuantity, format string) (*DrawingAnnotation, error) {
	if _, _, _, err := as.annotationBasis(viewName); err != nil {
		return nil, err
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.HoleNoteAnnotation, viewName: viewName, holeQuantity: quantity, tag: format}
	as.recomputeHoleNotes(a)
	if a.rowCount == 0 {
		return nil, fmt.Errorf("drawing: view %q has no holes for hole notes", viewName)
	}
	as.items = append(as.items, a)
	return a, nil
}

// formatHoleNote renders a hole callout: the format template with {d} (diameter, 2 decimals) and
// {n} (count) substituted, or the built-in default when the template is empty ("Ø<d>", or
// "<n>x Ø<d>" when count > 1).
func formatHoleNote(format string, diameterMM float64, count int) string {
	d := holeCoord(diameterMM)
	if format == "" {
		if count > 1 {
			return strconv.Itoa(count) + "x Ø" + d
		}
		return "Ø" + d
	}
	out := strings.ReplaceAll(format, "{d}", d)
	return strings.ReplaceAll(out, "{n}", strconv.Itoa(count))
}

// projectedHoleNote is one hole's callout anchor: its centre on the sheet (mm) and radius (mm).
type projectedHoleNote struct {
	sx, sy   float64
	radiusMM float64
}

// recomputeHoleNotes re-reads the view's holes and rebuilds a leadered Ø callout for each (or one
// "<n>x Ø<d>" callout per diameter when combined); with no resolvable holes it clears the note.
func (as *DrawingAnnotations) recomputeHoleNotes(a *DrawingAnnotation) {
	a.curves, a.labels, a.rowCount = nil, nil, 0
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	holes := dedupedHoleNotes(view, body, basis)
	if a.holeQuantity == types.HoleNoteCombined {
		renderCombinedHoleNotes(a, holes, a.tag)
		return
	}
	for _, h := range holes {
		appendHoleNote(a, h, formatHoleNote(a.tag, h.radiusMM*2, 1))
	}
}

// dedupedHoleNotes collects each distinct hole's sheet anchor + radius, fitting the view's
// projection (so cut holes are found) and listing coincident rims once.
func dedupedHoleNotes(view *DrawingView, body *topo.Body, basis hlr.View) []projectedHoleNote {
	seen := map[string]bool{}
	var out []projectedHoleNote
	for _, c := range circlesFromProjection(body, basis) {
		s := view.place(c.center)
		key := fmt.Sprintf("%.1f/%.1f", float64(s.X), float64(s.Y))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, projectedHoleNote{sx: float64(s.X), sy: float64(s.Y), radiusMM: c.radius * cmToMM})
	}
	return out
}

// renderCombinedHoleNotes groups holes by diameter and emits one "<n>x Ø<d>" callout per size
// (or the format template), anchored at the first hole of each group (in encounter order).
func renderCombinedHoleNotes(a *DrawingAnnotation, holes []projectedHoleNote, format string) {
	type group struct {
		first projectedHoleNote
		count int
	}
	order := []string{}
	groups := map[string]*group{}
	for _, h := range holes {
		key := holeCoord(h.radiusMM * 2)
		g, ok := groups[key]
		if !ok {
			g = &group{first: h}
			groups[key] = g
			order = append(order, key)
		}
		g.count++
	}
	for _, key := range order {
		g := groups[key]
		appendHoleNote(a, g.first, formatHoleNote(format, g.first.radiusMM*2, g.count))
	}
}

// appendHoleNote adds one hole's leadered callout (curves + label) to the annotation.
func appendHoleNote(a *DrawingAnnotation, h projectedHoleNote, text string) {
	curves, label := holeNoteCallout(h.sx, h.sy, h.radiusMM, text)
	a.curves = append(a.curves, curves...)
	a.labels = append(a.labels, label)
	a.rowCount++
}

// holeNoteCallout builds one hole's leadered callout: a leader from the text anchor to the hole rim
// (with an arrowhead) and the callout text, placed up-right of the hole centre.
func holeNoteCallout(cx, cy, radiusMM float64, text string) ([]DrawingCurve, AnnotationLabel) {
	const k = 0.70710678 // unit diagonal (cos 45°)
	tx, ty := cx+holeNoteOffsetMM*k, cy-holeNoteOffsetMM*k
	// Start the leader on the hole rim nearest the text, stopping a small gap short.
	rx, ry := cx+(radiusMM+holeNoteGapMM)*k, cy-(radiusMM+holeNoteGapMM)*k
	curves := []DrawingCurve{dimSegment(tx, ty, rx, ry)}
	curves = append(curves, noteArrowhead(tx, ty, rx, ry)...)
	return curves, AnnotationLabel{Text: text, X: tx + 6, Y: ty}
}
