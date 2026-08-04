// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The slot recipes. A slot is a centreline thickened by a width and capped at each end, so it
// is rigid once its caps are tangent to its sides and share a radius — parallel sides then
// follow, and stating parallel as well would be redundant.
//
// The centreline persists as construction geometry in both variants: it is what carries the
// length (or radius) dimension, and it is what the user saw while placing the slot.

// StraightSlotRecipe is the centre-to-centre straight slot: two parallel sides capped by a
// semicircular arc at each centre, plus the construction centreline. DOF 5 — centre x,y, angle,
// length and width.
//
//	r := StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2)
func StraightSlotRecipe(c0, c1 math.Point2, width math.Scalar) Recipe {
	a0, a1, b1, b0 := slotSideCorners(c0, c1, width)
	// Points: 0=a0 1=a1 2=b1 3=b0 4=c1 5=c0.
	// Entities: 0=side a, 1=cap at c1, 2=side b, 3=cap at c0, 4=construction centreline.
	return Recipe{
		Points: []math.Point2{a0, a1, b1, b0, c1, c0},
		Entities: []RecipeEntity{
			{Kind: RecipeLine, Points: []int{0, 1}},
			{Kind: RecipeArc, Points: []int{4, 1, 2}},
			{Kind: RecipeLine, Points: []int{2, 3}},
			{Kind: RecipeArc, Points: []int{5, 3, 0}},
			{Kind: RecipeLine, Points: []int{5, 4}, Construction: true},
		},
		Constraints: []RecipeConstraint{
			{Kind: TangentKind, Entities: []int{0, 1}},
			{Kind: TangentKind, Entities: []int{2, 1}},
			{Kind: TangentKind, Entities: []int{2, 3}},
			{Kind: TangentKind, Entities: []int{0, 3}},
			{Kind: EqualRadiusKind, Entities: []int{1, 3}},
		},
		Fields: straightSlotFields(c0, c1, a0, b0, width),
	}
}

// slotSideCorners offsets both centres perpendicular to the centreline by half the width,
// giving the four points where the sides meet the caps.
func slotSideCorners(c0, c1 math.Point2, width math.Scalar) (a0, a1, b1, b0 math.Point2) {
	d := c0.VectorTo(c1)
	du := d.Scale(1 / d.Length()) // non-finite when the centres coincide; Apply rejects it
	half := math.V2(-du.Y, du.X).Scale(float64(width) / 2)
	return c0.TranslateBy(half), c1.TranslateBy(half),
		c1.TranslateBy(half.Negate()), c0.TranslateBy(half.Negate())
}

// straightSlotFields is Length/Angle along the centreline plus the slot Width across it. Width
// dimensions a cap's diameter, which is the slot width by construction and does not duplicate
// what tangency already fixes.
func straightSlotFields(c0, c1, a0, b0 math.Point2, width math.Scalar) []RecipeField {
	along := c0.VectorTo(c1)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: c0.DistanceTo(c1),
			Witness: [2]math.Point2{c0, c1},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{5, 4},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: AlignedDistance,
			},
		},
		steeringAngleField("Angle", stdmath.Atan2(float64(along.Y), float64(along.X)), c0, c1),
		{
			Label: "Width", Unit: FieldLength, Value: float64(width),
			Witness: [2]math.Point2{a0, b0},
			Dim: RecipeDim{
				Kind: DiameterDim, Points: [2]int{0, 0},
				Entity: 3, Entity2: NoRecipeEntity,
			},
		},
	}
}

// ArcSlotRecipe is the arc-shaped slot: a centreline arc thickened into inner and outer arcs
// about the same centre, capped by a semicircle at each end. DOF 6 — centre x,y, radius, start
// angle, sweep, width.
//
// The inner and outer arcs share the centre point, so they are concentric structurally and an
// explicit concentric constraint would be redundant. Each cap IS tangent to both arcs, though:
// once the touch point is pinned, tangency reads as "centre, touch point and cap centre are
// collinear", so the inner and outer tangencies pin different points and are independent. (In
// the older centre-distance formulation they would have collapsed to the same equation, which
// is why they look redundant at first glance.) Without the inner pair the inner arc's
// endpoints stay angularly free and the slot is 2 DOF short of rigid.
//
// Two constraints that look necessary are deliberately absent, both measured rather than
// assumed. An equal-radius on the two caps adds zero rank at every pose — the four collinear
// tangencies plus the caps' own circularity already force it. And unlike the straight slot,
// this one persists NO construction centreline: an arc entity carries an implicit circularity
// relation, and here that relation is implied by the rest of the shape, so the centreline would
// contribute a permanently redundant equation while anchoring nothing. The radius dimension
// spans centre→start directly instead, which is the same measurement.
func ArcSlotRecipe(center, start, end math.Point2, width math.Scalar, ccw bool) Recipe {
	r := center.DistanceTo(start)
	half := float64(width) / 2
	outS, outE := radialPoint(center, start, r+half), radialPoint(center, end, r+half)
	inE, inS := radialPoint(center, end, r-half), radialPoint(center, start, r-half)
	// Points: 0=centre 1=outS 2=outE 3=inE 4=inS 5=start 6=end.
	// Entities: 0=outer arc, 1=cap at end, 2=inner arc, 3=cap at start.
	return Recipe{
		Points: []math.Point2{center, outS, outE, inE, inS, start, end},
		Entities: []RecipeEntity{
			{Kind: RecipeArc, Points: []int{0, 1, 2}, CounterClockwise: ccw},
			{Kind: RecipeArc, Points: []int{6, 2, 3}, CounterClockwise: ccw},
			{Kind: RecipeArc, Points: []int{0, 3, 4}, CounterClockwise: !ccw},
			{Kind: RecipeArc, Points: []int{5, 4, 1}, CounterClockwise: ccw},
		},
		Constraints: []RecipeConstraint{
			{Kind: CircularTangentKind, Entities: []int{0, 1}},
			{Kind: CircularTangentKind, Entities: []int{0, 3}},
			{Kind: CircularTangentKind, Entities: []int{2, 1}},
			{Kind: CircularTangentKind, Entities: []int{2, 3}},
		},
		Fields: arcSlotFields(center, start, end, r, width),
	}
}

// radialPoint returns the point at distance dist from center along the center→through ray.
func radialPoint(center, through math.Point2, dist float64) math.Point2 {
	v := center.VectorTo(through)
	return center.TranslateBy(v.Scale(dist / v.Length()))
}

// arcSlotFields is Radius/Sweep along the centreline arc plus the slot Width across it.
func arcSlotFields(center, start, end math.Point2, r float64, width math.Scalar) []RecipeField {
	return []RecipeField{
		{
			// The centreline radius, measured centre→start. There is no centreline arc to put
			// a RadiusDim on (see ArcSlotRecipe), and this distance is the same quantity.
			Label: "Radius", Unit: FieldLength, Value: r,
			Witness: [2]math.Point2{center, start},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{0, 5},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: AlignedDistance,
			},
		},
		steeringAngleField("Sweep", arcSweep(center, start, end), start, end),
		{
			Label: "Width", Unit: FieldLength, Value: float64(width),
			Witness: [2]math.Point2{start, radialPoint(center, start, r+float64(width)/2)},
			Dim: RecipeDim{
				Kind: DiameterDim, Points: [2]int{0, 0},
				Entity: 3, Entity2: NoRecipeEntity,
			},
		},
	}
}

// arcSweep is the signed angle swept from start to end about center.
func arcSweep(center, start, end math.Point2) float64 {
	vs, ve := center.VectorTo(start), center.VectorTo(end)
	return stdmath.Atan2(float64(ve.Y), float64(ve.X)) - stdmath.Atan2(float64(vs.Y), float64(vs.X))
}
