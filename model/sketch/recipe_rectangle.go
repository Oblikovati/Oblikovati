// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The three rectangle recipes. Each is rigid: a committed rectangle stays a rectangle when a
// corner is dragged. The pre-#2014 tools emitted a bare four-line loop with no constraints at
// all (DOF 8), so dragging a corner sheared it into an arbitrary quadrilateral.

// RectangleRecipe is the axis-aligned two-corner rectangle: four lines over four shared
// corners, held square by a horizontal on each horizontal edge and a vertical on each vertical
// edge. DOF 4 — corner x,y plus width and height.
//
//	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
func RectangleRecipe(a, c math.Point2) Recipe {
	b, d := math.P2(c.X, a.Y), math.P2(a.X, c.Y)
	return Recipe{
		Points:   []math.Point2{a, b, c, d},
		Entities: closedLoopEntities(4),
		Constraints: []RecipeConstraint{
			{Kind: SingleLineHorizontalKind, Entities: []int{0}},
			{Kind: SingleLineHorizontalKind, Entities: []int{2}},
			{Kind: SingleLineVerticalKind, Entities: []int{1}},
			{Kind: SingleLineVerticalKind, Entities: []int{3}},
		},
		Fields: rectangleFields(a, b, c),
	}
}

// closedLoopEntities returns n lines joining consecutive point indices in a closed ring.
func closedLoopEntities(n int) []RecipeEntity {
	ents := make([]RecipeEntity, n)
	for i := range ents {
		ents[i] = RecipeEntity{Kind: RecipeLine, Points: []int{i, (i + 1) % n}}
	}
	return ents
}

// rectangleFields is the Width/Height input pair, each witnessed along the edge it measures.
// The values are absolute so a rectangle dragged right-to-left still reports a positive width.
func rectangleFields(a, b, c math.Point2) []RecipeField {
	return []RecipeField{
		{
			Label: "Width", Unit: FieldLength, Value: stdmath.Abs(float64(c.X - a.X)),
			Witness: [2]math.Point2{a, b},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{0, 1},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: HorizontalDistance,
			},
		},
		{
			Label: "Height", Unit: FieldLength, Value: stdmath.Abs(float64(c.Y - a.Y)),
			Witness: [2]math.Point2{b, c},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{1, 2},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: VerticalDistance,
			},
		},
	}
}

// ThreePointRectangleRecipe is the rotated rectangle: a base edge from base0 to base1, then a
// height taken as the perpendicular distance of the height point from that base. Three
// perpendicular constraints round the loop keep it square — a fourth would be redundant, since
// the loop closes. DOF 5 — corner x,y, base angle, length and width.
func ThreePointRectangleRecipe(base0, base1, height math.Point2) Recipe {
	c2, c3 := offsetBaseEdge(base0, base1, height)
	return Recipe{
		Points:   []math.Point2{base0, base1, c2, c3},
		Entities: closedLoopEntities(4),
		Constraints: []RecipeConstraint{
			{Kind: PerpendicularKind, Entities: []int{0, 1}},
			{Kind: PerpendicularKind, Entities: []int{1, 2}},
			{Kind: PerpendicularKind, Entities: []int{2, 3}},
		},
		Fields: threePointRectangleFields(base0, base1, c2),
	}
}

// offsetBaseEdge returns the two far corners: the base edge translated by the perpendicular
// depth of the height point, which is how a three-point rectangle reads its width.
func offsetBaseEdge(base0, base1, height math.Point2) (math.Point2, math.Point2) {
	along := base0.VectorTo(base1)
	perp := math.V2(-along.Y, along.X)
	if perp.Length() == 0 {
		return base1, base0 // degenerate base; the recipe is rejected downstream
	}
	n := perp.Scale(1 / perp.Length())
	depth := n.Scale(base1.VectorTo(height).Dot(n))
	return base1.TranslateBy(depth), base0.TranslateBy(depth)
}

// threePointRectangleFields is Length/Angle for the base edge plus Width for the offset. The
// Angle field steers the shape but names no entity to dimension against: a free-standing
// rectangle has no reference edge to measure its rotation from.
func threePointRectangleFields(base0, base1, c2 math.Point2) []RecipeField {
	along := base0.VectorTo(base1)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: base0.DistanceTo(base1),
			Witness: [2]math.Point2{base0, base1},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{0, 1},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: AlignedDistance,
			},
		},
		steeringAngleField("Angle", stdmath.Atan2(float64(along.Y), float64(along.X)), base0, base1),
		{
			Label: "Width", Unit: FieldLength, Value: base1.DistanceTo(c2),
			Witness: [2]math.Point2{base1, c2},
			Dim: RecipeDim{
				Kind: DistanceDim, Points: [2]int{1, 2},
				Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: AlignedDistance,
			},
		},
	}
}

// steeringAngleField is an angle input that steers the shape during placement but creates no
// dimension, because the shape has no reference edge to measure the angle against.
func steeringAngleField(label string, radians float64, from, to math.Point2) RecipeField {
	return RecipeField{
		Label: label, Unit: FieldAngle, Value: radians,
		Witness: [2]math.Point2{from, to},
		Dim: RecipeDim{
			Kind: AngleDim, Points: [2]int{0, 0},
			Entity: NoRecipeEntity, Entity2: NoRecipeEntity,
		},
	}
}

// CenterRectangleRecipe is the centre-out rectangle: four corners around a centre point,
// squared by horizontal/vertical constraints, with the centre pinned as the midpoint of one
// construction diagonal. Both diagonals persist as construction geometry — the user drew them,
// and they are what anchors the centre. DOF 4 — centre x,y plus width and height.
func CenterRectangleRecipe(center, corner math.Point2) Recipe {
	a, b, c, d := cornersAboutCenter(center, corner)
	ents := append(closedLoopEntities(4),
		RecipeEntity{Kind: RecipeLine, Points: []int{0, 2}, Construction: true},
		RecipeEntity{Kind: RecipeLine, Points: []int{1, 3}, Construction: true},
	)
	return Recipe{
		Points:   []math.Point2{a, b, c, d, center},
		Entities: ents,
		Constraints: []RecipeConstraint{
			{Kind: SingleLineHorizontalKind, Entities: []int{0}},
			{Kind: SingleLineHorizontalKind, Entities: []int{2}},
			{Kind: SingleLineVerticalKind, Entities: []int{1}},
			{Kind: SingleLineVerticalKind, Entities: []int{3}},
			{Kind: MidpointKind, Points: []int{4}, Entities: []int{4}},
		},
		Fields: rectangleFields(a, b, c),
	}
}

// cornersAboutCenter mirrors corner through center to give the four corners in ring order.
func cornersAboutCenter(center, corner math.Point2) (math.Point2, math.Point2, math.Point2, math.Point2) {
	dx, dy := corner.X-center.X, corner.Y-center.Y
	return math.P2(center.X-dx, center.Y-dy),
		math.P2(center.X+dx, center.Y-dy),
		math.P2(center.X+dx, center.Y+dy),
		math.P2(center.X-dx, center.Y+dy)
}
