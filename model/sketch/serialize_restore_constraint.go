// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Sketch recipe restore — CONSTRAINTS and DIMENSIONS (M48 #2244 split of serialize_restore.go). Rebuilds
// the geometric constraints and the (basic + advanced) dimensional constraints from their recipe rows,
// re-binding each to the entities it constrains. The shared reference re-binding helpers live in
// serialize_restore.go; the points/entities in serialize_restore_entity.go.

func (r *sketchRestorer) restoreConstraints(constraints []ConstraintData) error {
	for _, cd := range constraints {
		if err := r.restoreConstraint(cd); err != nil {
			return fmt.Errorf("constraint %q: %w", cd.Kind, err)
		}
	}
	return nil
}

// restoreConstraint decodes through the paired codec registry (#1625) — an
// unknown kind is a corrupt-recipe error, named honestly.
func (r *sketchRestorer) restoreConstraint(cd ConstraintData) error {
	codec, ok := constraintCodecs2D[ConstraintKind(cd.Kind)]
	if !ok {
		return fmt.Errorf("unknown constraint kind %q", cd.Kind)
	}
	return codec.decode(r, cd)
}

func (r *sketchRestorer) restoreDimensions(dims []DimensionData) error {
	for _, dd := range dims {
		d, err := r.restoreDimension(dd)
		if err != nil {
			return fmt.Errorf("dimension %q: %w", dd.Kind, err)
		}
		if dd.Driven {
			d.SetDriven(true)
		}
		if dd.Limits != nil {
			d.SetLimits(dd.Limits.Min, dd.Limits.Max)
		}
		if len(dd.TextPoint) == 2 {
			d.SetTextPoint(math.P2(math.Scalar(dd.TextPoint[0]), math.Scalar(dd.TextPoint[1])))
		}
	}
	return nil
}

func (r *sketchRestorer) restoreDimension(dd DimensionData) (*DimensionConstraint, error) {
	dc := r.s.dimCons
	switch dd.Kind {
	case "distance":
		a, err := r.point(dd.Points, 0)
		if err != nil {
			return nil, err
		}
		b, err := r.point(dd.Points, 1)
		if err != nil {
			return nil, err
		}
		o, ok := ParseDistanceOrientation(dd.Orientation)
		if !ok {
			return nil, fmt.Errorf("distance dimension: unknown orientation %q", dd.Orientation)
		}
		return dc.AddDistanceOriented(a, b, dd.Expression, o)
	case "radius":
		// AddRadius accepts any CircularCurve (circle or arc) — an arc radius/diameter
		// dimension is common (esp. from CAD imports), so resolve as a circular curve,
		// not a *Circle. Restoring via r.circle rejected radius-dimensioned arcs and
		// blocked ~40% of real Inventor-exported parts. Symmetric with the write side,
		// which serializes d.refs regardless of circle-vs-arc.
		c, err := r.curve(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddRadius(c, dd.Expression)
	case "diameter":
		c, err := r.curve(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddDiameter(c, dd.Expression)
	case "angle":
		l1, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		l2, err := r.line(dd.Curves, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddAngle(l1, l2, dd.Expression)
	case "arcLength":
		a, err := r.arc(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddArcLength(a, dd.Expression)
	default:
		return r.restoreAdvancedDimension(dd)
	}
}

// restoreAdvancedDimension rebuilds the M21 dimension kinds (offset/three-point-angle/
// ellipse-radius); split out of restoreDimension to keep that switch small.
func (r *sketchRestorer) restoreAdvancedDimension(dd DimensionData) (*DimensionConstraint, error) {
	dc := r.s.dimCons
	switch dd.Kind {
	case "offsetDim":
		p, err := r.point(dd.Points, 0)
		if err != nil {
			return nil, err
		}
		l, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddOffsetDim(p, l, dd.LinearDiameter, dd.Expression)
	case "threePointAngle":
		pts, err := r.points(dd.Points, 3)
		if err != nil {
			return nil, err
		}
		return dc.AddThreePointAngle(pts[0], pts[1], pts[2], dd.Expression)
	case "ellipseRadius":
		e, err := r.ellipse(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddEllipseRadius(e, dd.Expression)
	case "tangentDistance":
		l, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		c, err := r.curve(dd.Curves, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddTangentDistance(l, c, dd.FarSide, dd.LinearDiameter, dd.Expression)
	case "offsetSplineDim":
		o, err := r.offsetSpline(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddOffsetSplineDim(o, dd.Expression)
	default:
		return nil, fmt.Errorf("unknown dimension kind %q", dd.Kind)
	}
}
