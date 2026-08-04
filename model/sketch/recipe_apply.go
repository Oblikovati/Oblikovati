// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Applying a [Recipe]: mint its shared points, build its entities, apply its constraints, and
// (for locked input fields) add its dimensions. Every step is undone on failure, so a rejected
// recipe never leaves orphan geometry behind (#2014).

// Apply materialises a recipe into the sketch and returns the entities and points it created.
// It is atomic: any failure deletes everything it created.
//
// Dimensions are not created here — only the interactive layer knows which input fields the
// user typed into; see [Sketch.ApplyWithFields].
//
//	ents, pts, err := sk.Apply(RectangleRecipe(a, c), types.OverConstrainedApplyDriven)
func (s *Sketch) Apply(r Recipe, behavior types.OverConstrainedDimensionBehavior) ([]Entity, []*Point, error) {
	return s.ApplyWithFields(r, nil, behavior)
}

// ApplyWithFields is [Sketch.Apply] plus a dimension for each locked input field. locked[i] is
// field i's parameter expression (e.g. "10 mm"); an empty string leaves that field
// undimensioned, which is the contract for a field that merely tracked the cursor.
func (s *Sketch) ApplyWithFields(r Recipe, locked []string, behavior types.OverConstrainedDimensionBehavior) ([]Entity, []*Point, error) {
	if err := checkRecipeFinite(r); err != nil {
		return nil, nil, err
	}
	pts := s.mintRecipePoints(r)
	ents, err := s.buildRecipeEntities(r, pts)
	if err != nil {
		return s.rollbackRecipe(ents, err)
	}
	if err := s.applyRecipeConstraints(r, ents, pts); err != nil {
		return s.rollbackRecipe(ents, err)
	}
	if err := s.applyRecipeFields(r, ents, pts, locked, behavior); err != nil {
		return s.rollbackRecipe(ents, err)
	}
	return ents, pts, nil
}

// rollbackRecipe deletes everything a failed apply created and returns the error. It prunes the
// minted points explicitly as well as the entities: a recipe that fails before it builds any
// geometry has already minted its points, and a stray constrainable point would keep inflating
// the sketch's degrees of freedom for ever after.
func (s *Sketch) rollbackRecipe(ents []Entity, err error) ([]Entity, []*Point, error) {
	s.DeleteEntities(ents)
	s.dropConstraintsOnVars(droppedPointVars(s.pruneOrphanPoints()))
	return nil, nil, err
}

// checkRecipeFinite rejects a recipe whose geometry came out non-finite. It is the single guard
// for every degenerate input a shape builder can be handed — coincident slot centres, a
// zero-radius circle, a collinear three-point arc — each of which divides by a zero length and
// so produces ±Inf or NaN coordinates rather than failing outright.
func checkRecipeFinite(r Recipe) error {
	for i, p := range r.Points {
		if isFinitePoint(p) {
			continue
		}
		return fmt.Errorf("recipe point %d is (%v,%v), want finite coordinates: the shape's "+
			"defining points are degenerate (coincident, collinear or zero-sized)", i, p.X, p.Y)
	}
	for i, e := range r.Entities {
		if e.Radius < 0 || math.IsNearZero(float64(e.Radius), math.DefaultTolerance) && needsRadius(e.Kind) {
			return fmt.Errorf("recipe entity %d has radius %v, want a positive radius", i, e.Radius)
		}
	}
	return nil
}

// isFinitePoint reports whether both coordinates are finite.
func isFinitePoint(p math.Point2) bool {
	return !stdmath.IsNaN(float64(p.X)) && !stdmath.IsInf(float64(p.X), 0) &&
		!stdmath.IsNaN(float64(p.Y)) && !stdmath.IsInf(float64(p.Y), 0)
}

// needsRadius reports whether a kind is sized by its Radius field.
func needsRadius(k RecipeEntityKind) bool { return k == RecipeCircle || k == RecipeEllipse }

// mintRecipePoints creates one constrainable point per recipe position, in index order. It
// uses newPoint rather than Points().Add because these are curve endpoints and centres, not
// standalone points: Points().Add would list each as its own entity, which is what every other
// composite in this package (AddStraightSlot, AddPolygon, closedLoopPoints) deliberately
// avoids. A recipe that wants a listed point says so with a RecipePoint entity.
func (s *Sketch) mintRecipePoints(r Recipe) []*Point {
	pts := make([]*Point, len(r.Points))
	for i, p := range r.Points {
		pts[i] = s.newPoint(p)
	}
	return pts
}

// buildRecipeEntities creates every entity in order, sharing the minted points so entities
// naming the same index come out structurally coincident.
func (s *Sketch) buildRecipeEntities(r Recipe, pts []*Point) ([]Entity, error) {
	ents := make([]Entity, 0, len(r.Entities))
	for i, re := range r.Entities {
		e, err := s.buildRecipeEntity(re, pts)
		if err != nil {
			return ents, fmt.Errorf("recipe entity %d: %w", i, err)
		}
		markConstruction(e, re.Construction)
		ents = append(ents, e)
	}
	return ents, nil
}

// markConstruction flags an entity as construction geometry when the recipe asked for it.
func markConstruction(e Entity, construction bool) {
	if !construction {
		return
	}
	if c, ok := e.(interface{ SetConstruction(bool) }); ok {
		c.SetConstruction(true)
	}
}

// buildRecipeEntity creates one entity from its recipe description.
func (s *Sketch) buildRecipeEntity(re RecipeEntity, pts []*Point) (Entity, error) {
	if err := checkPointIndices(re, len(pts)); err != nil {
		return nil, err
	}
	switch re.Kind {
	case RecipeLine:
		return s.Lines().Add(pts[re.Points[0]], pts[re.Points[1]]), nil
	case RecipeArc:
		return s.Arcs().Add(pts[re.Points[0]], pts[re.Points[1]], pts[re.Points[2]], re.CounterClockwise), nil
	case RecipeCircle:
		return s.Circles().Add(pts[re.Points[0]], re.Radius), nil
	case RecipeEllipse:
		return s.Ellipses().AddWithCenter(pts[re.Points[0]], re.MajorAxis, re.Radius, re.MinorRadius), nil
	case RecipeSpline:
		return s.Splines().AddWithPoints(recipePointsAt(pts, re.Points), re.Closed, re.FitPoints), nil
	case RecipePoint:
		p := s.listStandalonePoint(pts[re.Points[0]])
		p.SetCenterPoint(re.CenterPoint)
		return p, nil
	default:
		return nil, fmt.Errorf("unknown entity kind %d (want RecipeLine..RecipePoint)", re.Kind)
	}
}

// recipePointsAt gathers the minted points an entity names, in the order it names them.
func recipePointsAt(pts []*Point, idx []int) []*Point {
	out := make([]*Point, len(idx))
	for i, j := range idx {
		out[i] = pts[j]
	}
	return out
}

// listStandalonePoint promotes an already-minted point to a listed sketch entity, which is what
// a RecipePoint is: a point the user placed in its own right, not a curve endpoint.
func (s *Sketch) listStandalonePoint(p *Point) *Point {
	s.add(p)
	s.points.append(p)
	return p
}

// recipeArity is how many point indices each fixed-arity entity kind needs. RecipeSpline is
// absent because it is variadic.
var recipeArity = map[RecipeEntityKind]int{
	RecipeLine:    2,
	RecipeArc:     3,
	RecipeCircle:  1,
	RecipeEllipse: 1,
	RecipePoint:   1,
}

// checkPointIndices rejects an entity whose point indices are the wrong count or out of range.
func checkPointIndices(re RecipeEntity, n int) error {
	if want, ok := recipeArity[re.Kind]; ok && len(re.Points) != want {
		return fmt.Errorf("kind %d has %d point indices, want %d", re.Kind, len(re.Points), want)
	}
	for _, i := range re.Points {
		if i < 0 || i >= n {
			return fmt.Errorf("point index %d out of range [0,%d)", i, n)
		}
	}
	return nil
}

// applyRecipeConstraints applies every geometric relation, resolving operand indices onto the
// entities and points just created.
func (s *Sketch) applyRecipeConstraints(r Recipe, ents []Entity, pts []*Point) error {
	g := s.GeometricConstraints()
	for i, rc := range r.Constraints {
		if err := checkOperandIndices(rc, len(ents), len(pts)); err != nil {
			return fmt.Errorf("recipe constraint %d (%s): %w", i, rc.Kind, err)
		}
		if err := applyOneRecipeConstraint(g, rc, ents, pts); err != nil {
			return fmt.Errorf("recipe constraint %d: %w", i, err)
		}
	}
	return nil
}

// checkOperandIndices rejects a constraint naming an entity or point outside the recipe.
func checkOperandIndices(rc RecipeConstraint, nEnts, nPts int) error {
	for _, i := range rc.Entities {
		if i < 0 || i >= nEnts {
			return fmt.Errorf("entity index %d out of range [0,%d)", i, nEnts)
		}
	}
	for _, i := range rc.Points {
		if i < 0 || i >= nPts {
			return fmt.Errorf("point index %d out of range [0,%d)", i, nPts)
		}
	}
	return nil
}

// applyOneRecipeConstraint dispatches one relation onto the geometric-constraint factories.
// Only the kinds the shape recipes need are handled; anything else is a programming error and
// is reported rather than silently skipped.
func applyOneRecipeConstraint(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, pts []*Point) error {
	if fn, ok := recipeConstraintAppliers[rc.Kind]; ok {
		return fn(g, rc, ents, pts)
	}
	return fmt.Errorf("unsupported recipe constraint kind %q", rc.Kind)
}

// recipeConstraintApplier applies one constraint kind onto the sketch's factories.
type recipeConstraintApplier func(*GeometricConstraints, RecipeConstraint, []Entity, []*Point) error

// recipeConstraintAppliers is the dispatch table for the constraint kinds the shape recipes
// use. A table keeps each applier short and makes the supported set greppable.
var recipeConstraintAppliers = map[ConstraintKind]recipeConstraintApplier{
	SingleLineHorizontalKind: applyLineHorizontal,
	SingleLineVerticalKind:   applyLineVertical,
	PerpendicularKind:        applyPerpendicular,
	MidpointKind:             applyMidpoint,
	TangentKind:              applyTangent,
	CircularTangentKind:      applyCircularTangent,
	EqualRadiusKind:          applyEqualRadius,
	ConcentricKind:           applyConcentric,
	PointOnCircleKind:        applyPointOnCircle,
	EqualLengthKind:          applyEqualLength,
}

func applyLineHorizontal(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	l, err := recipeLine(ents, rc.Entities, 0)
	if err != nil {
		return err
	}
	g.AddLineHorizontal(l)
	return nil
}

func applyLineVertical(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	l, err := recipeLine(ents, rc.Entities, 0)
	if err != nil {
		return err
	}
	g.AddLineVertical(l)
	return nil
}

func applyPerpendicular(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	l1, l2, err := recipeLinePair(ents, rc.Entities)
	if err != nil {
		return err
	}
	g.AddPerpendicular(l1, l2)
	return nil
}

func applyEqualLength(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	l1, l2, err := recipeLinePair(ents, rc.Entities)
	if err != nil {
		return err
	}
	g.AddEqualLength(l1, l2)
	return nil
}

func applyMidpoint(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, pts []*Point) error {
	l, err := recipeLine(ents, rc.Entities, 0)
	if err != nil {
		return err
	}
	if len(rc.Points) < 1 {
		return fmt.Errorf("midpoint needs 1 point index, got %d", len(rc.Points))
	}
	g.AddMidpoint(pts[rc.Points[0]], l)
	return nil
}

func applyTangent(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	l, err := recipeLine(ents, rc.Entities, 0)
	if err != nil {
		return err
	}
	c, err := recipeCurve(ents, rc.Entities, 1)
	if err != nil {
		return err
	}
	g.AddTangent(l, c)
	return nil
}

func applyCircularTangent(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	c1, c2, err := recipeCurvePair(ents, rc.Entities)
	if err != nil {
		return err
	}
	g.AddCircularTangent(c1, c2)
	return nil
}

func applyEqualRadius(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	c1, c2, err := recipeCurvePair(ents, rc.Entities)
	if err != nil {
		return err
	}
	g.AddEqualRadius(c1, c2)
	return nil
}

func applyConcentric(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, _ []*Point) error {
	c1, c2, err := recipeCurvePair(ents, rc.Entities)
	if err != nil {
		return err
	}
	g.AddConcentric(c1, c2)
	return nil
}

func applyPointOnCircle(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, pts []*Point) error {
	c, err := recipeCurve(ents, rc.Entities, 0)
	if err != nil {
		return err
	}
	if len(rc.Points) < 1 {
		return fmt.Errorf("pointOnCircle needs 1 point index, got %d", len(rc.Points))
	}
	g.AddPointOnCircle(pts[rc.Points[0]], c)
	return nil
}

// recipeLine resolves operand i as a line, reporting the actual type when it is not one.
func recipeLine(ents []Entity, operands []int, i int) (*Line, error) {
	if i >= len(operands) {
		return nil, fmt.Errorf("missing entity operand %d (got %d)", i, len(operands))
	}
	l, ok := ents[operands[i]].(*Line)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want *Line", operands[i], ents[operands[i]])
	}
	return l, nil
}

// recipeCurve resolves operand i as a circular curve (circle or arc).
func recipeCurve(ents []Entity, operands []int, i int) (CircularCurve, error) {
	if i >= len(operands) {
		return nil, fmt.Errorf("missing entity operand %d (got %d)", i, len(operands))
	}
	c, ok := ents[operands[i]].(CircularCurve)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want a circular curve", operands[i], ents[operands[i]])
	}
	return c, nil
}

// recipeLinePair resolves two line operands.
func recipeLinePair(ents []Entity, operands []int) (*Line, *Line, error) {
	l1, err := recipeLine(ents, operands, 0)
	if err != nil {
		return nil, nil, err
	}
	l2, err := recipeLine(ents, operands, 1)
	if err != nil {
		return nil, nil, err
	}
	return l1, l2, nil
}

// recipeCurvePair resolves two circular-curve operands.
func recipeCurvePair(ents []Entity, operands []int) (CircularCurve, CircularCurve, error) {
	c1, err := recipeCurve(ents, operands, 0)
	if err != nil {
		return nil, nil, err
	}
	c2, err := recipeCurve(ents, operands, 1)
	if err != nil {
		return nil, nil, err
	}
	return c1, c2, nil
}
