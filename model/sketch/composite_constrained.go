// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The constrained composite constructors. Each applies the shape's [Recipe], so the interactive
// tools, the api/wire composite path and the live preview all share one definition of what the
// shape is (#2014).
//
// The raw Add* primitives alongside these stay unconstrained on purpose. Importers, pattern
// copies and procedural add-ins that state every constraint themselves depend on that path:
// auto-applied duplicates there make the sketch report a healthy DOF while the solver settles on
// a degenerate, self-intersecting configuration that extrudes to nothing.
//
//	ents, err := sk.AddConstrainedRectangle(math.P2(0, 0), math.P2(10, 8))

// AddConstrainedRectangle creates the rigid axis-aligned two-corner rectangle (DOF 4).
func (s *Sketch) AddConstrainedRectangle(a, c math.Point2) ([]Entity, error) {
	return s.applyShape(RectangleRecipe(a, c))
}

// AddConstrainedThreePointRectangle creates the rigid rotated rectangle (DOF 5).
func (s *Sketch) AddConstrainedThreePointRectangle(base0, base1, height math.Point2) ([]Entity, error) {
	return s.applyShape(ThreePointRectangleRecipe(base0, base1, height))
}

// AddConstrainedCenterRectangle creates the rigid centre-out rectangle with its two construction
// diagonals (DOF 4).
func (s *Sketch) AddConstrainedCenterRectangle(center, corner math.Point2) ([]Entity, error) {
	return s.applyShape(CenterRectangleRecipe(center, corner))
}

// AddConstrainedStraightSlot creates the rigid centre-to-centre slot with its construction
// centreline (DOF 5).
func (s *Sketch) AddConstrainedStraightSlot(c0, c1 math.Point2, width math.Scalar) ([]Entity, error) {
	return s.applyShape(StraightSlotRecipe(c0, c1, width))
}

// AddConstrainedArcSlot creates the rigid arc slot (DOF 6).
func (s *Sketch) AddConstrainedArcSlot(center, start, end math.Point2, width math.Scalar, ccw bool) ([]Entity, error) {
	return s.applyShape(ArcSlotRecipe(center, start, end, width, ccw))
}

// AddConstrainedPolygon creates the rigid regular n-gon with its construction circumcircle
// (DOF 4).
func (s *Sketch) AddConstrainedPolygon(center, through math.Point2, sides int, inscribed bool) ([]Entity, error) {
	return s.applyShape(PolygonRecipe(center, through, sides, inscribed))
}

// applyShape applies a recipe with no locked input fields, under the default over-constrained
// behaviour. The interactive layer uses ApplyWithFields instead, because only it knows which
// fields the user typed a value into.
func (s *Sketch) applyShape(r Recipe) ([]Entity, error) {
	ents, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	return ents, err
}
