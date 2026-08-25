// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Non-mutating offset geometry for the live preview. The offset tool shows a ghost of the offset
// following the cursor before the placement click; this builds that ghost as a Recipe (analytic
// lines + arcs) instead of mutating the sketch. It reuses the SAME pure geometry the commit path uses
// (buildOffsetElements + offsetCornerEndpoints, or offsetLine/Circle/Arc), so the preview cannot drift
// from what OK creates.

// OffsetLoopRecipe builds the offset of a connected loop/chain by signed distance d as a Recipe. ok is
// false when the loop cannot offset (an unsupported entity, or a collapse).
func (s *Sketch) OffsetLoopRecipe(path *Path, d float64) (Recipe, bool) {
	ents := path.Entities()
	if len(ents) == 0 {
		return Recipe{}, false
	}
	if r, ok := offsetSingleCircleRecipe(ents, d); ok {
		return r, true
	}
	elems, err := s.buildOffsetElements(ents, d)
	if err != nil {
		return Recipe{}, false
	}
	starts, ends := offsetCornerEndpoints(elems, path.IsClosed())
	return offsetElementsRecipe(elems, starts, ends), true
}

// offsetSingleCircleRecipe mirrors offsetSingleCircle without mutating: a lone circle offsets to a
// concentric circle.
func offsetSingleCircleRecipe(ents []ProfileEntity, d float64) (Recipe, bool) {
	if len(ents) != 1 {
		return Recipe{}, false
	}
	center, r, ok := circleCenterRadius(ents[0].Entity)
	if !ok {
		return Recipe{}, false
	}
	rp := offsetRadius(r, d, !ents[0].Reversed())
	if rp <= 0 {
		return Recipe{}, false
	}
	return CircleRecipe(center, math.Scalar(rp)), true
}

// offsetElementsRecipe assembles the offset primitives (with joined corner endpoints) into a Recipe of
// lines and arcs, mirroring buildOffsetEntities.
func offsetElementsRecipe(elems []offsetElement, starts, ends []math.Point2) Recipe {
	var r Recipe
	add := func(p math.Point2) int { r.Points = append(r.Points, p); return len(r.Points) - 1 }
	for i, el := range elems {
		if el.isLine {
			a, b := add(starts[i]), add(ends[i])
			r.Entities = append(r.Entities, RecipeEntity{Kind: RecipeLine, Points: []int{a, b}})
			continue
		}
		c, a, b := add(el.center), add(starts[i]), add(ends[i])
		r.Entities = append(r.Entities, RecipeEntity{Kind: RecipeArc, Points: []int{c, a, b}, CounterClockwise: el.effCCW})
	}
	return r
}

// OffsetEntityRecipe builds the offset of a single entity by signed distance d as a Recipe: a parallel
// line, or a concentric circle/arc. ok is false for an unsupported entity or a collapse.
func OffsetEntityRecipe(e Entity, d float64) (Recipe, bool) {
	switch v := e.(type) {
	case *Line:
		u, ok := unitVec(v.A.Position().VectorTo(v.B.Position()))
		if !ok {
			return Recipe{}, false
		}
		n := math.V2(-u.Y, u.X).Scale(d)
		return LineRecipe(v.A.Position().TranslateBy(n), v.B.Position().TranslateBy(n)), true
	case *Circle:
		rp := float64(v.Radius) + d
		if rp <= 0 {
			return Recipe{}, false
		}
		return CircleRecipe(v.Center.Position(), math.Scalar(rp)), true
	case *Arc:
		return offsetArcRecipe(v, d)
	default:
		return Recipe{}, false
	}
}

// offsetArcRecipe builds the concentric-arc offset Recipe (endpoints moved radially by r+d).
func offsetArcRecipe(a *Arc, d float64) (Recipe, bool) {
	center := a.Center.Position()
	r := float64(a.Radius())
	nr := r + d
	if nr <= 0 || r == 0 {
		return Recipe{}, false
	}
	scale := nr / r
	start := center.TranslateBy(center.VectorTo(a.Start.Position()).Scale(scale))
	end := center.TranslateBy(center.VectorTo(a.End.Position()).Scale(scale))
	return ArcRecipe(center, start, end, a.CounterClockwise), true
}
