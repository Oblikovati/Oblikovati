// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// A Recipe is the single declarative description of what one sketch tool creates: its
// geometry, which of it is construction, the constraints that make it rigid, and the
// quantities the user can dimension while placing it. Preview renders a Recipe, Commit applies
// the same Recipe, and the AddConstrained* constructors wrap it — so a shape has exactly one
// definition and a preview can never disagree with the committed result (#2014).
//
// Before this, a polygon had three definitions: AddPolygon built it correctly, PolygonTool
// rebuilt it with no constraints, and the preview approximated it a third way.
//
//	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
//	ents, pts, err := sk.Apply(r, types.OverConstrainedApplyDriven)
type Recipe struct {
	// Points are the shape's defining positions. Entities and Constraints refer to them by
	// index, so two entities naming the same index share one Point and are therefore
	// structurally coincident — no explicit coincident constraint is needed or wanted.
	Points []math.Point2
	// Entities are created in index order; Constraints and Fields refer to them by index.
	Entities []RecipeEntity
	// Constraints are the geometric relations that make the shape rigid.
	Constraints []RecipeConstraint
	// Fields are the dimensionable quantities shown as in-place input boxes while placing.
	Fields []RecipeField
}

// RecipeEntityKind selects which entity a [RecipeEntity] creates.
type RecipeEntityKind uint8

const (
	// RecipeLine is a segment between Points[0] and Points[1].
	RecipeLine RecipeEntityKind = iota
	// RecipeArc is an arc about Points[0] running from Points[1] to Points[2].
	RecipeArc
	// RecipeCircle is a circle about Points[0] with Radius.
	RecipeCircle
	// RecipeEllipse is an ellipse about Points[0] with MajorAxis, Radius and MinorRadius.
	RecipeEllipse
	// RecipeSpline interpolates every point named in Points.
	RecipeSpline
	// RecipePoint is a standalone sketch point at Points[0].
	RecipePoint
)

// RecipeEntity is one entity to create, naming its defining points by index into
// [Recipe.Points].
type RecipeEntity struct {
	Kind         RecipeEntityKind
	Points       []int
	Construction bool
	// CounterClockwise orients a [RecipeArc].
	CounterClockwise bool
	// Radius sizes a [RecipeCircle], or a [RecipeEllipse]'s major radius; MinorRadius and
	// MajorAxis complete an ellipse. Unused by the other kinds.
	Radius      math.Scalar
	MinorRadius math.Scalar
	MajorAxis   math.Vector2
	// FitPoints makes a [RecipeSpline] interpolate its points rather than treat them as a
	// control polygon; Closed joins its last point back to its first.
	FitPoints bool
	Closed    bool
}

// RecipeConstraint is one geometric relation, naming its operands by index into
// [Recipe.Entities] and [Recipe.Points].
type RecipeConstraint struct {
	Kind     ConstraintKind
	Entities []int
	Points   []int
}

// FieldUnit selects how a [RecipeField]'s value is interpreted and formatted.
type FieldUnit uint8

const (
	// FieldLength is a model-unit length.
	FieldLength FieldUnit = iota
	// FieldAngle is an angle in radians.
	FieldAngle
)

// RecipeField is one in-place input box: a labelled quantity the user can type over while
// placing the shape. A field the user types into is locked and becomes the dimension named by
// Dim; a field left tracking the cursor creates nothing.
type RecipeField struct {
	Label string
	Unit  FieldUnit
	// Value is the live measurement at the current cursor — model units, or radians.
	Value float64
	// Witness spans the two positions the dotted extension line connects; the input box is
	// drawn at its midpoint.
	Witness [2]math.Point2
	// Dim names the dimension to create when this field is locked.
	Dim RecipeDim
}

// RecipeDim names the dimension constraint a locked field creates, by index into the recipe's
// points and entities.
type RecipeDim struct {
	Kind DimKind
	// Points are the two point indices a DistanceDim spans.
	Points [2]int
	// Entity is the entity index a RadiusDim/DiameterDim measures, or the first of the two
	// lines an AngleDim spans. -1 means the field steers the shape but has nothing to
	// dimension against (a free-standing shape's rotation has no reference edge).
	Entity int
	// Entity2 is the second line of an AngleDim; -1 when unused.
	Entity2 int
	// Orientation selects what a DistanceDim measures (aligned / horizontal / vertical).
	Orientation DistanceOrientation
}

// NoRecipeEntity marks a [RecipeDim] operand that does not reference an entity.
const NoRecipeEntity = -1
