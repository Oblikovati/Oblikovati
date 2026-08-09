// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// A contour flange's corners are BENDS (#1961). The profile is drawn as a chain of straight
// segments, and the wall was swept with those corners left sharp — geometry no press brake can
// make, and a shape that disagrees with the part it claims to model. Each interior corner is
// rounded to the bend radius, tangent to both segments, before the band is thickened.
//
// The radius comes from the feature's override or the rule, exactly as a plain flange's does.

// contourFacetStep caps a corner arc's facet size, matching the flange bend's smoothness.
const contourFacetStep = flangeFacetStep

// roundProfileCorners replaces each interior vertex of an open profile with an arc of the given
// radius, tangent to both segments. A non-positive radius leaves the profile alone, which is what
// a rule with no bend radius means: sharp corners are then the caller's explicit choice.
func roundProfileCorners(pts []math.Point2, radius float64) ([]math.Point2, error) {
	if radius <= 0 || len(pts) < 3 {
		return pts, nil
	}
	out := []math.Point2{pts[0]}
	for i := 1; i < len(pts)-1; i++ {
		arc, err := cornerArc(pts[i-1], pts[i], pts[i+1], radius)
		if err != nil {
			return nil, err
		}
		out = append(out, arc...)
	}
	return append(out, pts[len(pts)-1]), nil
}

// cornerArc is the arc that rounds the corner at b between segments a→b and b→c, tangent to both.
// The tangent length is radius·tan(half the turn), so a corner that turns further eats more of its
// own segments — which is why a radius that fits one corner can be refused at the next.
func cornerArc(a, b, c math.Point2, radius float64) ([]math.Point2, error) {
	in, out, err := cornerDirections(a, b, c)
	if err != nil {
		return nil, err
	}
	turn := stdmath.Acos(clampUnit(float64(in.Dot(out))))
	if turn < 1e-9 {
		return nil, nil // collinear: no corner to round
	}
	tangent := radius * stdmath.Tan(turn/2)
	if tangent > float64(a.DistanceTo(b))+1e-9 || tangent > float64(b.DistanceTo(c))+1e-9 {
		return nil, fmt.Errorf("sheet-metal contour flange: a bend radius of %g needs %g of straight "+
			"either side of the corner at (%g, %g), and the profile gives %g and %g; shorten the "+
			"radius or lengthen the segments", radius, tangent, float64(b.X), float64(b.Y),
			float64(a.DistanceTo(b)), float64(b.DistanceTo(c)))
	}
	return arcThroughCorner(b, in, out, tangent, turn, radius), nil
}

// cornerDirections returns the unit directions into and out of the corner at b.
func cornerDirections(a, b, c math.Point2) (in, out math.Vector2, err error) {
	in, err = unit2(a.VectorTo(b))
	if err != nil {
		return in, out, err
	}
	out, err = unit2(b.VectorTo(c))
	return in, out, err
}

// arcThroughCorner samples the true tangent arc from where it leaves the incoming segment to where
// it meets the outgoing one, by rotating the radius about the arc's centre.
//
// It is a circular arc and is built as one. A quadratic Bezier through the same three points is a
// parabola, which is close for a gentle corner and visibly not a bend radius for a sharp one — and
// a bend radius is a manufacturing input, not a curve that looks about right.
func arcThroughCorner(b math.Point2, in, out math.Vector2, tangent, turn, radius float64) []math.Point2 {
	start := b.TranslateBy(in.Scale(math.Scalar(-tangent)))
	centre := start.TranslateBy(turnNormal(in, out).Scale(math.Scalar(radius)))
	spoke := centre.VectorTo(start)
	sense := 1.0
	if float64(in.X)*float64(out.Y)-float64(in.Y)*float64(out.X) < 0 {
		sense = -1 // the corner turns clockwise
	}
	steps := int(stdmath.Max(2, stdmath.Round(turn/contourFacetStep)))
	pts := make([]math.Point2, 0, steps+1)
	for k := 0; k <= steps; k++ {
		pts = append(pts, centre.TranslateBy(rotate2(spoke, sense*turn*float64(k)/float64(steps))))
	}
	return pts
}

// turnNormal is the unit direction from the incoming segment toward the arc's centre: the part of
// the outgoing direction perpendicular to the incoming one, which by construction points to the
// side the corner turns.
func turnNormal(in, out math.Vector2) math.Vector2 {
	perp := out.Add(in.Scale(-in.Dot(out)))
	n, err := unit2(perp)
	if err != nil {
		return perp
	}
	return n
}

// unit2 normalises a 2D vector, erroring on a degenerate one.
func unit2(v math.Vector2) (math.Vector2, error) {
	l := float64(v.Length())
	if l < 1e-12 {
		return v, fmt.Errorf("sheet-metal contour flange: profile has a zero-length segment")
	}
	return v.Scale(math.Scalar(1 / l)), nil
}
