// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
)

// Drawing notes (M14-F04 PBI-144, #391): a free text note on the sheet, optionally with a leader
// line to the feature it annotates. The note text is a label; the leader is a drawing curve with a
// small arrowhead. Notes are user-supplied markup (the text reuses the annotation tag).

// noteLeaderArrowMM is the leader arrowhead's length.
const noteLeaderArrowMM = 3.0

// AddNote adds a free text note anchored at (x, y), with an optional leader to (leaderX, leaderY) —
// the feature it annotates. A leader of (0, 0) means none. It errors with empty text.
func (as *DrawingAnnotations) AddNote(name string, x, y float64, text string, leaderX, leaderY float64) (*DrawingAnnotation, error) {
	if text == "" {
		return nil, fmt.Errorf("drawing: a note needs text, got %q", text)
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.DrawingNoteAnnotation,
		x: x, y: y, w: leaderX, h: leaderY, tag: text,
	}
	a.curves, a.labels = noteGeometry(x, y, text, leaderX, leaderY)
	as.items = append(as.items, a)
	return a, nil
}

// noteGeometry builds the note's text label and, when a leader target is given, a leader line from
// the note to that point with a small arrowhead at the target.
func noteGeometry(x, y float64, text string, leaderX, leaderY float64) ([]DrawingCurve, []AnnotationLabel) {
	labels := []AnnotationLabel{{Text: text, X: x, Y: y}}
	if leaderX == 0 && leaderY == 0 {
		return nil, labels
	}
	curves := []DrawingCurve{dimSegment(x, y, leaderX, leaderY)}
	return append(curves, noteArrowhead(x, y, leaderX, leaderY)...), labels
}

// noteArrowhead builds the two short barbs of the leader's arrowhead at the target (leaderX, leaderY),
// pointing back along the leader toward the note.
func noteArrowhead(x, y, leaderX, leaderY float64) []DrawingCurve {
	dx, dy := x-leaderX, y-leaderY
	d := math.Hypot(dx, dy)
	if d < 1e-9 {
		return nil
	}
	ux, uy := dx/d, dy/d
	px, py := -uy, ux // perpendicular
	const w = noteLeaderArrowMM * 0.4
	bx, by := leaderX+ux*noteLeaderArrowMM, leaderY+uy*noteLeaderArrowMM
	return []DrawingCurve{
		dimSegment(leaderX, leaderY, bx+px*w, by+py*w),
		dimSegment(leaderX, leaderY, bx-px*w, by-py*w),
	}
}
