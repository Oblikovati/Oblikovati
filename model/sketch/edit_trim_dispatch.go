// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// trimmableCurve is the sealed trim capability: each trimmable entity routes
// to its own trim (like definingPoints in drag.go), so the line/circle/arc
// dispatch is stated once here instead of per driver — it used to be copied in
// the wire router and the trim tool (#1624, audit I1).
type trimmableCurve interface {
	trimAt(s *Sketch, pick math.Point2) ([]Entity, error)
}

func (l *Line) trimAt(s *Sketch, pick math.Point2) ([]Entity, error)   { return s.TrimLine(l, pick) }
func (c *Circle) trimAt(s *Sketch, pick math.Point2) ([]Entity, error) { return s.TrimCircle(c, pick) }
func (a *Arc) trimAt(s *Sketch, pick math.Point2) ([]Entity, error)    { return s.TrimArc(a, pick) }

// TrimCurveAt trims the picked curve at pick, removing the picked span up to
// the nearest crossings. Only lines, circles, and arcs are trimmable.
//
//	removed, err := sk.TrimCurveAt(pickedEntity, pickPoint)
func (s *Sketch) TrimCurveAt(e Entity, pick math.Point2) ([]Entity, error) {
	tc, ok := e.(trimmableCurve)
	if !ok {
		return nil, fmt.Errorf("trim: unsupported target %T (want a line, circle, or arc)", e)
	}
	return tc.trimAt(s, pick)
}
