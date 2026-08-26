// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Recipes for the shapes that were already well-formed before #2014 — line, circle, arc,
// ellipse, spline, point — plus the polygon, whose correct constrained construction already
// existed in AddPolygon but was bypassed by a duplicate implementation in the interactive tool.
//
// The already-correct shapes get NO constraints here. Their degrees of freedom are their
// intrinsic parameter count (a circle is centre + radius, and that is all a circle is), so
// adding constraints would over-constrain them — a regression, not an improvement.

// LineRecipe is a plain two-point segment. Inference (horizontal / vertical / parallel) is
// applied separately by the interactive layer, which is the part that can see the neighbouring
// geometry a new line should relate to.
func LineRecipe(a, b math.Point2) Recipe {
	return Recipe{
		Points:   []math.Point2{a, b},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
		Fields:   lineFields(a, b),
	}
}

// LineChainRecipe is a run of connected segments through pts, consecutive segments sharing an
// endpoint — what the continuous line tool builds.
//
// Its input fields describe the LAST segment, the one still at the cursor, so the dynamic input
// steers the segment being drawn while the whole chain is described. Fewer than two points is
// not a chain yet and yields an empty recipe.
//
//	r := LineChainRecipe([]math.Point2{a, b, c}) // two segments, sharing b
func LineChainRecipe(pts []math.Point2) Recipe {
	if len(pts) < 2 {
		return Recipe{}
	}
	ents := make([]RecipeEntity, len(pts)-1)
	for i := range ents {
		ents[i] = RecipeEntity{Kind: RecipeLine, Points: []int{i, i + 1}}
	}
	return Recipe{
		Points:   append([]math.Point2(nil), pts...),
		Entities: ents,
		Fields:   chainTailFields(pts),
	}
}

// chainTailFields is the Length/Angle pair for the chain's last segment, with the dimension's
// point indices moved onto that segment — the fields belong to the segment being drawn, not to
// the first one the chain happens to start with.
func chainTailFields(pts []math.Point2) []RecipeField {
	last := len(pts) - 1
	fields := lineFields(pts[last-1], pts[last])
	fields[0].Dim.Points = [2]int{last - 1, last}
	return fields
}

// lineFields is the Length/Angle pair measured from the line's start.
func lineFields(a, b math.Point2) []RecipeField {
	v := a.VectorTo(b)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: a.DistanceTo(b),
			Witness: [2]math.Point2{a, b},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{0, 1},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: AlignedDistance,
			},
		},
		steeringAngleField("Angle", stdmath.Atan2(float64(v.Y), float64(v.X)), a, b),
	}
}

// CircleRecipe is a centre-and-radius circle; its one field dimensions the diameter, which is
// how a circle is conventionally dimensioned.
func CircleRecipe(center math.Point2, radius math.Scalar) Recipe {
	rim := math.P2(center.X+radius, center.Y)
	return Recipe{
		Points:   []math.Point2{center},
		Entities: []RecipeEntity{{Kind: RecipeCircle, Points: []int{0}, Radius: radius}},
		Fields: []RecipeField{{
			Label: "Diameter", Unit: FieldLength, Value: 2 * float64(radius),
			Witness: [2]math.Point2{center, rim},
			Dim: RecipeDim{
				Kind: DiameterDim, Points: [2]int{0, 0},
				Entity: 0, Entity2: NoRecipeEntity,
			},
		}},
	}
}

// ArcRecipe is a centre/start/end arc. Its radius is |centre − start| structurally, so the arc
// carries its own circularity relation and needs no added constraint.
func ArcRecipe(center, start, end math.Point2, ccw bool) Recipe {
	return Recipe{
		Points:   []math.Point2{center, start, end},
		Entities: []RecipeEntity{{Kind: RecipeArc, Points: []int{0, 1, 2}, CounterClockwise: ccw}},
		Fields: []RecipeField{
			{
				Label: "Radius", Unit: FieldLength, Value: center.DistanceTo(start),
				Witness: [2]math.Point2{center, start},
				Dim: RecipeDim{
					Kind: RadiusDim, Points: [2]int{0, 0},
					Entity: 0, Entity2: NoRecipeEntity,
				},
			},
			steeringAngleField("Sweep", arcSweep(center, start, end), start, end),
		},
	}
}

// EllipseRecipe is a centre/axis/two-radii ellipse.
func EllipseRecipe(center math.Point2, majorAxis math.Vector2, majorR, minorR math.Scalar) Recipe {
	minorDir := math.V2(-majorAxis.Y, majorAxis.X)
	return Recipe{
		Points: []math.Point2{center},
		Entities: []RecipeEntity{{
			Kind: RecipeEllipse, Points: []int{0},
			MajorAxis: majorAxis, Radius: majorR, MinorRadius: minorR,
		}},
		Fields: []RecipeField{
			{
				Label: "Major", Unit: FieldLength, Value: float64(majorR),
				Witness: [2]math.Point2{center, center.TranslateBy(majorAxis.Scale(float64(majorR)))},
				Dim: RecipeDim{
					Kind: EllipseRadiusDim, Points: [2]int{0, 0},
					Entity: 0, Entity2: NoRecipeEntity,
				},
			},
			{
				// The minor radius has no dedicated dimension kind, so it steers the shape
				// during placement without creating one.
				Label: "Minor", Unit: FieldLength, Value: float64(minorR),
				Witness: [2]math.Point2{center, center.TranslateBy(minorDir.Scale(float64(minorR)))},
				Dim: RecipeDim{
					Kind: EllipseRadiusDim, Points: [2]int{0, 0},
					Entity: NoRecipeEntity, Entity2: NoRecipeEntity,
				},
			},
		},
	}
}

// SplineRecipe interpolates the given points when fit is true, or treats them as a control
// polygon when false. Every point stays free (DOF 2n), which is the curve's parameterisation.
func SplineRecipe(pts []math.Point2, fit bool) Recipe {
	idx := make([]int, len(pts))
	for i := range pts {
		idx[i] = i
	}
	return Recipe{
		Points:   append([]math.Point2(nil), pts...),
		Entities: []RecipeEntity{{Kind: RecipeSpline, Points: idx, FitPoints: fit}},
	}
}

// PointRecipe is a standalone sketch point — the one recipe whose point is listed as an entity
// in its own right.
func PointRecipe(p math.Point2) Recipe {
	return Recipe{
		Points:   []math.Point2{p},
		Entities: []RecipeEntity{{Kind: RecipePoint, Points: []int{0}}},
	}
}

// PolygonRecipe is the regular n-gon: vertices pinned to a shared construction circumcircle
// with equal consecutive edges, which makes it rigid — equal chords on one circle are equally
// spaced, so the shape is determined by centre, circumradius and rotation. DOF 4.
//
// This mirrors AddPolygon, the already-correct implementation the interactive polygon tool
// bypassed with its own unconstrained rebuild (#2014).
func PolygonRecipe(center, through math.Point2, sides int, inscribed bool) Recipe {
	v := center.VectorTo(through)
	r := v.Length()
	verts := polygonVertices(center, stdmath.Atan2(float64(v.Y), float64(v.X)), r, sides, inscribed)
	centerIdx := len(verts)
	ents := append(closedLoopEntities(sides), RecipeEntity{
		Kind: RecipeCircle, Points: []int{centerIdx},
		Radius: math.Scalar(center.DistanceTo(verts[0])), Construction: true,
	})
	return Recipe{
		Points:      append(append([]math.Point2(nil), verts...), center),
		Entities:    ents,
		Constraints: polygonConstraints(sides),
		Fields:      polygonFields(center, verts[0], sides),
	}
}

// polygonConstraints pins every vertex to the circumcircle (the entity after the n edges) and
// equalises consecutive edges. The last edge needs no equal-length row: the loop closes, so
// n−1 equalities already make all n edges equal.
func polygonConstraints(sides int) []RecipeConstraint {
	cons := make([]RecipeConstraint, 0, 2*sides)
	for i := range sides {
		cons = append(cons, RecipeConstraint{Kind: PointOnCircleKind, Points: []int{i}, Entities: []int{sides}})
	}
	for i := 0; i+1 < sides; i++ {
		cons = append(cons, RecipeConstraint{Kind: EqualLengthKind, Entities: []int{i, i + 1}})
	}
	return cons
}

// polygonFields dimensions the circumscribed diameter and steers the rotation. The circumcircle
// is the last entity, after the n edges.
func polygonFields(center, vertex math.Point2, sides int) []RecipeField {
	v := center.VectorTo(vertex)
	return []RecipeField{
		{
			Label: "Diameter", Unit: FieldLength, Value: 2 * center.DistanceTo(vertex),
			Witness: [2]math.Point2{center, vertex},
			Dim: RecipeDim{
				Kind: DiameterDim, Points: [2]int{0, 0},
				Entity: sides, Entity2: NoRecipeEntity,
			},
		},
		steeringAngleField("Angle", stdmath.Atan2(float64(v.Y), float64(v.X)), center, vertex),
	}
}
