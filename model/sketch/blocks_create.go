// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Create-from-selection (M06-F07, #622): the interactive "make this geometry
// a block" path. The selected entities — with their defining points — move
// out of the sketch into a fresh definition, and one identity-transform
// instance replaces them in place, so the sketch looks unchanged but the
// geometry is now reusable. Points shared with geometry that stays behind
// are cloned for the definition (the kept entities keep theirs); constraints
// and dimensions touching the moved geometry are dropped — a block is a
// rigid group, its internal shape no longer solves.

// CreateFromSelection moves ents into a new definition registered as name
// and returns the definition plus the replacing instance.
func (c *Blocks) CreateFromSelection(reg *BlockDefinitions, name string, ents []Entity) (*BlockDefinition, *BlockInstance, error) {
	if len(ents) == 0 {
		return nil, nil, fmt.Errorf("block %q needs at least one selected entity", name)
	}
	for _, e := range ents {
		if !blockSupportedEntity(e) {
			return nil, nil, fmt.Errorf("block %q cannot include entity %d (%T); blocks hold points, lines, arcs, circles, ellipses and splines",
				name, e.EntityID(), e)
		}
	}
	def, err := reg.Define(name)
	if err != nil {
		return nil, nil, err
	}
	c.s.dropConstraintsTouching(ents)
	for _, e := range ents {
		c.s.moveEntityIntoBlock(def, e)
	}
	return def, c.Insert(def, math.Identity3()), nil
}

// blockSupportedEntity reports whether the entity kind can live in a block.
func blockSupportedEntity(e Entity) bool {
	switch e.(type) {
	case *Point, *Line, *Circle, *Arc, *Ellipse, *EllipticalArc, *Spline, *BlockInstance:
		return true
	}
	return false
}

// moveEntityIntoBlock detaches e from the sketch and hands it (and its
// points) to the definition. A point still referenced by geometry that stays
// behind is cloned for the definition — the kept entities keep theirs (a
// shared rectangle corner must not vanish when one edge becomes a block).
func (s *Sketch) moveEntityIntoBlock(def *BlockDefinition, e Entity) {
	s.deleteEntity(e)
	for _, p := range entityDefiningPoints(e) {
		switch {
		case s.pointInUse(p):
			clone := &Point{id: nextID(), X: p.X, Y: p.Y}
			replaceEntityPoint(e, p, clone)
			def.pts = append(def.pts, clone)
		case !pointAmong(def.pts, p): // shared between two moved entities: move once
			s.removePoint(p)
			s.points.remove(p)
			def.pts = append(def.pts, p)
		}
	}
	_ = def.Add(e) // cycle-free: e came out of a sketch, not a definition
}

// pointInUse reports whether any entity still in the sketch references p.
func (s *Sketch) pointInUse(p *Point) bool {
	for _, e := range s.ents {
		if pointAmong(entityDefiningPoints(e), p) {
			return true
		}
	}
	return false
}

// pointAmong reports whether p is in pts.
func pointAmong(pts []*Point, p *Point) bool {
	for _, x := range pts {
		if x == p {
			return true
		}
	}
	return false
}

// replaceEntityPoint rewires one defining point of a moved entity to its
// definition-owned clone.
func replaceEntityPoint(e Entity, old, clone *Point) {
	swap := func(f **Point) {
		if *f == old {
			*f = clone
		}
	}
	switch t := e.(type) {
	case *Line:
		swap(&t.A)
		swap(&t.B)
	case *Circle:
		swap(&t.Center)
	case *Arc:
		swap(&t.Center)
		swap(&t.Start)
		swap(&t.End)
	default:
		replaceCurvePoint(e, old, clone, swap)
	}
}

// replaceCurvePoint is the conic/spline half of replaceEntityPoint.
func replaceCurvePoint(e Entity, old, clone *Point, swap func(**Point)) {
	switch t := e.(type) {
	case *Ellipse:
		swap(&t.Center)
	case *EllipticalArc:
		swap(&t.Center)
	case *Spline:
		for i := range t.Points {
			if t.Points[i] == old {
				t.Points[i] = clone
			}
		}
	}
}

// entityDefiningPoints returns the points an entity owns, for the supported
// block kinds.
func entityDefiningPoints(e Entity) []*Point {
	switch t := e.(type) {
	case *Point:
		return []*Point{t}
	case *Line:
		return []*Point{t.A, t.B}
	case *Circle:
		return []*Point{t.Center}
	case *Arc:
		return []*Point{t.Center, t.Start, t.End}
	case *Ellipse:
		return []*Point{t.Center}
	case *EllipticalArc:
		return []*Point{t.Center}
	case *Spline:
		return append([]*Point(nil), t.Points...)
	case *SplineHandle:
		return []*Point{t.Anchor, t.End}
	}
	return nil
}

// dropConstraintsTouching removes every geometric constraint and dimension
// that reads any of the moved entities' point variables.
func (s *Sketch) dropConstraintsTouching(ents []Entity) {
	moved := map[*math.Scalar]bool{}
	for _, e := range ents {
		for _, p := range entityDefiningPoints(e) {
			moved[&p.X] = true
			moved[&p.Y] = true
		}
	}
	s.dropConstraintsOnVars(moved)
}

// dropConstraintsOnVars removes every geometric constraint and dimension that reads any
// of the given solver variables (matched by scalar-pointer identity) — the shared half of
// the move-into-block (dropConstraintsTouching) and delete (DeleteEntities) paths.
func (s *Sketch) dropConstraintsOnVars(vars map[*math.Scalar]bool) {
	for _, c := range s.geomCons.All() {
		if constraintTouches(c, vars) {
			s.geomCons.Delete(c)
		}
	}
	for _, d := range append([]*DimensionConstraint(nil), s.dimCons.items...) {
		if constraintTouches(d, vars) {
			s.dimCons.Delete(d)
		}
	}
}

// constraintTouches reports whether the constraint reads any moved variable.
func constraintTouches(c Constraint, moved map[*math.Scalar]bool) bool {
	for _, v := range c.Variables() {
		if moved[v] {
			return true
		}
	}
	return false
}

// PlacementTransform builds the instance transform from the interactive
// placement parameters: insertion point, CCW rotation (radians) about it,
// and a uniform scale (0 ⇒ 1).
func PlacementTransform(position math.Point2, rotation float64, scale float64) math.Matrix3 {
	if scale == 0 {
		scale = 1
	}
	t := math.Translation3(position.AsVector())
	r := math.Rotation3(math.Scalar(rotation), math.P2(0, 0))
	sc := math.Scale3(math.Scalar(scale), math.Scalar(scale))
	return t.Mul(r).Mul(sc)
}
