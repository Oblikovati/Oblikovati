// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// The 3D-sketch Dimension tool (issue #144) — the 3D counterpart of the 2D Dimension
// tool: pick geometry and the implied dimension is added at its current measured
// value (a spline's length, a circle's radius, a line's length, or the distance
// between two points), ready to be driven via the dimensions list or the API.

// newDimension3DTool builds the 3D dimension tool.
func newDimension3DTool() *ConstraintTool {
	return &ConstraintTool{
		name: "Dimension (3D)", prompt: "Select a spline, a circle, a line, or two points",
		accepts: acceptDimensionable3D, ready: readyDimension3D, apply: entityApply(applyDimension3D),
	}
}

// acceptDimensionable3D admits the 3D entity kinds a dimension can size
// (a spline of either fit flavor qualifies).
func acceptDimensionable3D(e sketch.Entity) bool {
	return entityKindIs(e, sketch.PointKind, sketch.LineKind, sketch.CircleKind,
		sketch.SplineKind, sketch.ControlPointSplineKind)
}

// readyDimension3D is satisfied by a spline, a circle, a line, or two points — the
// single-line case sizes the line itself, mirroring the 2D tool.
func readyDimension3D(ents []sketch.Entity) bool {
	splines, circles, lines, points := dimensionPicks3D(ents)
	return len(splines) >= 1 || len(circles) >= 1 || len(lines) >= 1 || len(points) >= 2
}

// applyDimension3D adds the dimension implied by the picked entities at its current
// measured value in the document's units.
func applyDimension3D(s *Session, ents []sketch.Entity) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("no active 3D sketch")
	}
	if err := addDimension3DFor(sk.DimensionConstraints3D(), s.DocumentUnits(), ents); err != nil {
		return err
	}
	return s.afterConstraint3D()
}

// addDimension3DFor creates the implied dimension: spline length, circle radius, line
// length, or point-point distance (in pick-priority order). Each is seeded at its
// current measured value formatted in the document's units.
func addDimension3DFor(dims *sketch.DimensionConstraints3D, units param.UnitsOfMeasure, ents []sketch.Entity) error {
	splines, circles, lines, points := dimensionPicks3D(ents)
	switch {
	case len(splines) >= 1:
		_, err := dims.AddSplineLength(splines[0], lengthExpr(units, sampledLength3D(splines[0])))
		return err
	case len(circles) >= 1:
		_, err := dims.AddRadius(circles[0], lengthExpr(units, float64(circles[0].Radius)))
		return err
	case len(lines) >= 1:
		_, err := dims.AddLineLength(lines[0], lengthExpr(units, float64(lines[0].Length())))
		return err
	case len(points) >= 2:
		_, err := dims.AddDistance(points[0], points[1], lengthExpr(units, float64(points[0].Position().DistanceTo(points[1].Position()))))
		return err
	default:
		return errNeed("dimension", "a spline, a circle, a line, or two points")
	}
}

// sampledLength3D seeds the spline-length expression from the current sampled
// polyline (the same measure the dimension reads).
func sampledLength3D(sp *sketch.Spline3D) float64 {
	pts := sp.Sample()
	total := 0.0
	for i := 0; i+1 < len(pts); i++ {
		total += float64(pts[i].DistanceTo(pts[i+1]))
	}
	return total
}

// dimensionPicks3D splits the picks into the kinds the dimension chooser weighs.
func dimensionPicks3D(ents []sketch.Entity) (splines []*sketch.Spline3D, circles []*sketch.Circle3D, lines []*sketch.Line3D, points []*sketch.Point3D) {
	for _, e := range ents {
		if v, ok := e.(*sketch.Spline3D); ok {
			splines = append(splines, v)
		}
		if v, ok := e.(*sketch.Circle3D); ok {
			circles = append(circles, v)
		}
		if v, ok := e.(*sketch.Line3D); ok {
			lines = append(lines, v)
		}
		if v, ok := e.(*sketch.Point3D); ok {
			points = append(points, v)
		}
	}
	return splines, circles, lines, points
}
