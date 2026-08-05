// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Dimensions from locked input fields (#2014). The contract from the reference behaviour: a
// field the user typed a value into becomes a persistent driving dimension, and a field left
// tracking the cursor creates nothing.
//
// Redundancy is the real risk here, not arithmetic. A duplicated constraint makes the sketch
// report DOF 0 while the solver settles on a degenerate, self-intersecting configuration that
// extrudes to an empty solid — silently, because a DOF-only check passes. So every dimension is
// trial-added and the redundancy count re-read before it is accepted as driving.

// applyRecipeFields adds a dimension for each locked field. A dimension that raises the
// sketch's redundancy count is handled per behavior: OverConstrainedApplyDriven (the default)
// demotes it to a reference dimension, OverConstrainedApplyDriving keeps it driving, and
// OverConstrainedPrompt falls back to driven here — the interactive layer asks the user before
// it ever reaches this call.
func (s *Sketch) applyRecipeFields(r Recipe, ents []Entity, pts []*Point, locked []string, behavior types.OverConstrainedDimensionBehavior) error {
	for i, f := range r.Fields {
		if !fieldIsDimensionable(f, i, locked) {
			continue
		}
		before := s.AnalyzeConstraints().Redundant
		d, err := s.addRecipeDim(f.Dim, ents, pts, locked[i])
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Label, err)
		}
		d.SetTextPoint(standoffAnchor(r, f)) // where the in-place dimension already stood
		s.resolveRedundantDim(d, before, behavior)
	}
	return nil
}

// fieldIsDimensionable reports whether field i was locked and names something to dimension. A
// field whose Dim.Entity is NoRecipeEntity steers the shape during placement but has nothing to
// measure against — a free-standing shape's rotation has no reference edge — so it is skipped.
func fieldIsDimensionable(f RecipeField, i int, locked []string) bool {
	if i >= len(locked) || locked[i] == "" {
		return false
	}
	if f.Dim.Kind == AngleDim && f.Dim.Entity == NoRecipeEntity {
		return false
	}
	return true
}

// resolveRedundantDim demotes a dimension that introduced redundancy, unless the document asked
// to keep redundant dimensions driving.
func (s *Sketch) resolveRedundantDim(d *DimensionConstraint, before int, behavior types.OverConstrainedDimensionBehavior) {
	if behavior == types.OverConstrainedApplyDriving {
		return
	}
	if s.AnalyzeConstraints().Redundant > before {
		d.SetDriven(true)
	}
}

// addRecipeDim creates the one dimension a locked field names. expression carries its unit
// (e.g. "10 mm") because the parameter engine is unit-strict.
func (s *Sketch) addRecipeDim(dim RecipeDim, ents []Entity, pts []*Point, expression string) (*DimensionConstraint, error) {
	dc := s.DimensionConstraints()
	switch dim.Kind {
	case DistanceDim:
		return s.addRecipeDistanceDim(dc, dim, pts, expression)
	case RadiusDim:
		return s.addRecipeCurveDim(dc.AddRadius, dim, ents, expression)
	case DiameterDim:
		return s.addRecipeCurveDim(dc.AddDiameter, dim, ents, expression)
	case AngleDim:
		return s.addRecipeAngleDim(dc, dim, ents, expression)
	default:
		return nil, fmt.Errorf("unsupported dimension kind %d (want DistanceDim/RadiusDim/DiameterDim/AngleDim)", dim.Kind)
	}
}

// addRecipeDistanceDim dimensions the distance between two of the recipe's points.
func (s *Sketch) addRecipeDistanceDim(dc *DimensionConstraints, dim RecipeDim, pts []*Point, expression string) (*DimensionConstraint, error) {
	a, b := dim.Points[0], dim.Points[1]
	if a < 0 || a >= len(pts) || b < 0 || b >= len(pts) {
		return nil, fmt.Errorf("distance dimension point indices (%d,%d) out of range [0,%d)", a, b, len(pts))
	}
	return dc.AddDistanceOriented(pts[a], pts[b], expression, dim.Orientation)
}

// addRecipeCurveDim dimensions a circular curve's radius or diameter through add.
func (s *Sketch) addRecipeCurveDim(add func(CircularCurve, string) (*DimensionConstraint, error), dim RecipeDim, ents []Entity, expression string) (*DimensionConstraint, error) {
	c, err := recipeCurve(ents, []int{dim.Entity}, 0)
	if err != nil {
		return nil, err
	}
	return add(c, expression)
}

// addRecipeAngleDim dimensions the angle between two of the recipe's lines.
func (s *Sketch) addRecipeAngleDim(dc *DimensionConstraints, dim RecipeDim, ents []Entity, expression string) (*DimensionConstraint, error) {
	l1, l2, err := recipeLinePair(ents, []int{dim.Entity, dim.Entity2})
	if err != nil {
		return nil, err
	}
	return dc.AddAngle(l1, l2, expression)
}

// dimensionStandoff is how far a recipe-created dimension is placed from the geometry it
// measures, in database units (cm). It matches what the in-place preview showed, so the
// annotation does not jump when the shape commits (#2032).
const dimensionStandoff = 0.6

// FieldOutward is the unit direction a field's dimension stands off in: the normal of its
// witness segment pointing AWAY from the shape.
//
// It is shared by the interactive preview and by the dimension the commit creates, so both put
// the annotation on the same side. A rectangle's witness segment IS one of its edges, so without
// this the dimension lands on — and reads as inside — the shape it measures.
//
//	out := sketch.FieldOutward(r, r.Fields[0]) // away from the rectangle
func FieldOutward(r Recipe, f RecipeField) math.Vector2 {
	a, b := f.Witness[0], f.Witness[1]
	edge := a.VectorTo(b)
	if edge.Length() == 0 {
		return math.V2(0, 1) // a degenerate witness still needs a side to stand on
	}
	n := math.V2(-edge.Y, edge.X).Scale(1 / edge.Length())
	centroid, enclosed := recipeCentroid(r)
	if !enclosed {
		return n // an open shape has no inside to point away from
	}
	mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
	if centroid.VectorTo(mid).Dot(n) < 0 {
		return n.Scale(-1) // the normal pointed into the shape
	}
	return n
}

// recipeCentroid averages a recipe's real outline points, reporting false when it encloses
// nothing (a bare point, a single rubber-banding segment).
func recipeCentroid(r Recipe) (math.Point2, bool) {
	pts, _ := RecipeOutline(r)
	if len(pts) < 3 {
		return math.Point2{}, false
	}
	var sum math.Vector2
	for _, p := range pts {
		sum = sum.Add(math.V2(p.X, p.Y))
	}
	return math.P2(0, 0).TranslateBy(sum.Scale(1 / float64(len(pts)))), true
}

// standoffAnchor is where a recipe-created dimension's label sits: the middle of what it measures,
// pushed clear of the shape.
func standoffAnchor(r Recipe, f RecipeField) math.Point2 {
	mid := math.P2((f.Witness[0].X+f.Witness[1].X)/2, (f.Witness[0].Y+f.Witness[1].Y)/2)
	return mid.TranslateBy(FieldOutward(r, f).Scale(dimensionStandoff))
}
