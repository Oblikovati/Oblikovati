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

// formatHoleNote renders a hole callout: the format template with {d} (diameter, 2 decimals),
// {n} (count) and {thread} (thread designation, empty for plain holes) substituted, or the built-in
// default when the template is empty. The default is the thread designation for a tapped hole and
// "Ø<d>" for a plain hole, prefixed "<n>x " when the callout covers more than one hole (#1995).
func formatHoleNote(format string, diameterMM float64, count int, thread string) string {
	d := holeCoord(diameterMM)
	if format == "" {
		return defaultHoleCallout(d, count, thread)
	}
	out := strings.ReplaceAll(format, "{d}", d)
	out = strings.ReplaceAll(out, "{n}", strconv.Itoa(count))
	return strings.ReplaceAll(out, "{thread}", thread)
}

// defaultHoleCallout is the built-in callout text: the thread designation for a tapped hole, else
// "Ø<d>", with an "<n>x " count prefix when the callout groups more than one hole.
func defaultHoleCallout(diameter string, count int, thread string) string {
	callout := "Ø" + diameter
	if thread != "" {
		callout = thread
	}
	if count > 1 {
		return strconv.Itoa(count) + "x " + callout
	}
	return callout
}

// projectedHoleNote is one hole's callout anchor: its centre on the sheet (mm), radius (mm), and
// thread designation when the hole is tapped ("" for a plain hole).
type projectedHoleNote struct {
	sx, sy   float64
	radiusMM float64
	thread   string
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
	a.threadCount = countThreadedHoles(holes)
	if a.holeQuantity == types.HoleNoteCombined {
		renderCombinedHoleNotes(a, holes, a.tag)
		return
	}
	for _, h := range holes {
		appendHoleNote(a, h, formatHoleNote(a.tag, h.radiusMM*2, 1, h.thread))
	}
}

// countThreadedHoles reports how many of the recovered holes are tapped (carry a thread designation).
func countThreadedHoles(holes []projectedHoleNote) int {
	n := 0
	for _, h := range holes {
		if h.thread != "" {
			n++
		}
	}
	return n
}

// dedupedHoleNotes collects each distinct hole's sheet anchor, radius and thread designation, fitting
// the view's projection (so cut holes are found) and listing coincident rims once. A hole coaxial
// with a machined-thread face is tagged with that thread's designation (#1995).
func dedupedHoleNotes(view *DrawingView, body *topo.Body, basis hlr.View) []projectedHoleNote {
	threads := threadCalloutsFrom(body, basis)
	seen := map[string]bool{}
	var out []projectedHoleNote
	for _, c := range circlesFromProjection(body, basis) {
		s := view.place(c.center)
		key := fmt.Sprintf("%.1f/%.1f", float64(s.X), float64(s.Y))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, projectedHoleNote{
			sx: float64(s.X), sy: float64(s.Y), radiusMM: c.radius * cmToMM,
			thread: threadAt(threads, c.center, c.radius),
		})
	}
	return out
}

// renderCombinedHoleNotes groups holes by callout and emits one "<n>x <callout>" note per group
// (or the format template), anchored at the first hole of each group (in encounter order). Threaded
// holes group by their designation (so "M6x1" and "M8x1.25" stay apart), plain holes by diameter.
func renderCombinedHoleNotes(a *DrawingAnnotation, holes []projectedHoleNote, format string) {
	type group struct {
		first projectedHoleNote
		count int
	}
	order := []string{}
	groups := map[string]*group{}
	for _, h := range holes {
		key := combinedHoleKey(h)
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
		appendHoleNote(a, g.first, formatHoleNote(format, g.first.radiusMM*2, g.count, g.first.thread))
	}
}

// combinedHoleKey groups a hole for the combined quantity mode: by thread designation when tapped,
// else by diameter — so like tapped holes merge and never fold in with a plain hole of equal bore.
func combinedHoleKey(h projectedHoleNote) string {
	if h.thread != "" {
		return "T:" + h.thread
	}
	return "D:" + holeCoord(h.radiusMM*2)
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
