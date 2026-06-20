// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// affine2 is a 2D affine transform (linear part m + translation t) used by the sketch
// edit operations (move/rotate/mirror). point applies the full transform; dir applies
// only the linear part (for direction vectors like an ellipse's major axis).
type affine2 struct {
	m      [2][2]float64
	tx, ty float64
}

func (a affine2) point(p math.Point2) math.Point2 {
	x, y := float64(p.X), float64(p.Y)
	return math.P2(math.Scalar(a.m[0][0]*x+a.m[0][1]*y+a.tx), math.Scalar(a.m[1][0]*x+a.m[1][1]*y+a.ty))
}

func (a affine2) dir(v math.Vector2) math.Vector2 {
	x, y := float64(v.X), float64(v.Y)
	return math.V2(math.Scalar(a.m[0][0]*x+a.m[0][1]*y), math.Scalar(a.m[1][0]*x+a.m[1][1]*y))
}

// translation moves by a vector.
func translation(v math.Vector2) affine2 {
	return affine2{m: [2][2]float64{{1, 0}, {0, 1}}, tx: float64(v.X), ty: float64(v.Y)}
}

// rotation rotates by angle (radians) about center.
func rotation(center math.Point2, angle float64) affine2 {
	c, s := stdmath.Cos(angle), stdmath.Sin(angle)
	m := [2][2]float64{{c, -s}, {s, c}}
	cx, cy := float64(center.X), float64(center.Y)
	return affine2{m: m, tx: cx - (m[0][0]*cx + m[0][1]*cy), ty: cy - (m[1][0]*cx + m[1][1]*cy)}
}

// reflection mirrors across the line through p with unit direction d.
func reflection(p math.Point2, d math.Vector2) affine2 {
	dx, dy := float64(d.X), float64(d.Y)
	m := [2][2]float64{{2*dx*dx - 1, 2 * dx * dy}, {2 * dx * dy, 2*dy*dy - 1}}
	px, py := float64(p.X), float64(p.Y)
	return affine2{m: m, tx: px - (m[0][0]*px + m[0][1]*py), ty: py - (m[1][0]*px + m[1][1]*py)}
}

// MoveEntities translates a selection in place by a vector.
func (s *Sketch) MoveEntities(ents []Entity, v math.Vector2) {
	s.transformInPlace(ents, translation(v))
}

// RotateEntities rotates a selection in place about center by angle (radians).
func (s *Sketch) RotateEntities(ents []Entity, center math.Point2, angle float64) {
	s.transformInPlace(ents, rotation(center, angle))
}

// CopyEntities duplicates a selection, translating the copies by v; returns the copies.
func (s *Sketch) CopyEntities(ents []Entity, v math.Vector2) []Entity {
	return s.cloneEntities(ents, translation(v))
}

// MirrorEntities reflects a selection across the given line, returning the mirrored copies.
// It errors if the mirror line has zero length.
func (s *Sketch) MirrorEntities(ents []Entity, line *Line) []Entity {
	d, ok := unitVec(line.A.Position().VectorTo(line.B.Position()))
	if !ok {
		return nil
	}
	return s.cloneEntities(ents, reflection(line.A.Position(), d))
}

// MovePoints translates a specific set of points by v — the primitive behind Stretch,
// which moves only the selected vertices and leaves the rest, deforming the geometry that
// shares them (a line with one endpoint stretched grows/angles). A point is moved once
// even if listed twice; nil entries are skipped.
func (s *Sketch) MovePoints(pts []*Point, v math.Vector2) {
	seen := map[*Point]bool{}
	for _, p := range pts {
		if p == nil || seen[p] {
			continue
		}
		p.SetPosition(p.Position().TranslateBy(v))
		seen[p] = true
	}
}

// ScaleEntities scales a selection in place about center by factor, enlarging both the
// point positions and the explicit radii of circles/ellipses (arc radii follow their
// points automatically). A non-positive factor is rejected as a no-op. Scale is not an
// affine2 (which is rigid/reflection only — it would not touch the radius fields).
func (s *Sketch) ScaleEntities(ents []Entity, center math.Point2, factor float64) {
	if factor <= 0 {
		return
	}
	seen := map[*Point]bool{}
	for _, e := range ents {
		for _, p := range entityPoints(e) {
			if !seen[p] {
				p.SetPosition(scaleAbout(center, p.Position(), factor))
				seen[p] = true
			}
		}
		scaleEntityRadius(e, factor)
	}
}

// scaleAbout returns p scaled by f about center.
func scaleAbout(center, p math.Point2, f float64) math.Point2 {
	return math.P2(center.X+math.Scalar(f)*(p.X-center.X), center.Y+math.Scalar(f)*(p.Y-center.Y))
}

// scaleEntityRadius scales the explicit radius fields not derived from points.
func scaleEntityRadius(e Entity, f float64) {
	switch v := e.(type) {
	case *Circle:
		v.Radius *= math.Scalar(f)
	case *Ellipse:
		v.MajorRadius *= math.Scalar(f)
		v.MinorRadius *= math.Scalar(f)
	case *EllipticalArc:
		v.MajorRadius *= math.Scalar(f)
		v.MinorRadius *= math.Scalar(f)
	}
}

// transformInPlace applies a to the unique points of the selection (and any ellipse axis
// directions), mutating the existing geometry.
func (s *Sketch) transformInPlace(ents []Entity, a affine2) {
	seen := map[*Point]bool{}
	for _, e := range ents {
		for _, p := range entityPoints(e) {
			if !seen[p] {
				p.SetPosition(a.point(p.Position()))
				seen[p] = true
			}
		}
		transformEntityDir(e, a)
	}
}

// cloneEntities duplicates the selection under transform a, sharing new points across
// entities that shared originals so the copy stays a connected whole.
func (s *Sketch) cloneEntities(ents []Entity, a affine2) []Entity {
	out, _ := s.cloneEntitiesMapped(ents, a)
	return out
}

// cloneEntitiesMapped is cloneEntities that also returns the seed→clone point map, so
// a caller (a parametric pattern) can link each clone point back to its seed.
func (s *Sketch) cloneEntitiesMapped(ents []Entity, a affine2) ([]Entity, map[*Point]*Point) {
	out, pmap, _ := s.cloneEntitiesFull(ents, a)
	return out, pmap
}

// cloneEntitiesFull is cloneEntitiesMapped that additionally returns the seed→clone entity
// map, so the constraint/dimension carry-over (#1083) can decide which relations are wholly
// inside the copied set and remap their operands. Points are tracked separately (a constraint
// references a point that may be shared by several entities, so the point map is the finer key).
func (s *Sketch) cloneEntitiesFull(ents []Entity, a affine2) ([]Entity, map[*Point]*Point, map[Entity]Entity) {
	pmap := map[*Point]*Point{}
	emap := map[Entity]Entity{}
	mapped := func(p *Point) *Point {
		if np, ok := pmap[p]; ok {
			return np
		}
		np := s.newPoint(a.point(p.Position()))
		pmap[p] = np
		return np
	}
	out := make([]Entity, 0, len(ents))
	for _, e := range ents {
		if c := s.cloneEntity(e, mapped, a); c != nil {
			emap[e] = c
			out = append(out, c)
		}
	}
	return out, pmap, emap
}

// cloneEntity duplicates one entity, resolving its points through mapped and transforming
// any direction vectors with a.
func (s *Sketch) cloneEntity(e Entity, mapped func(*Point) *Point, a affine2) Entity {
	switch v := e.(type) {
	case *Line:
		return s.lines.Add(mapped(v.A), mapped(v.B))
	case *Circle:
		return s.circles.Add(mapped(v.Center), v.Radius)
	case *Arc:
		return s.arcs.Add(mapped(v.Center), mapped(v.Start), mapped(v.End), v.CounterClockwise)
	case *Ellipse:
		return s.ellipses.AddWithCenter(mapped(v.Center), a.dir(v.MajorAxis), v.MajorRadius, v.MinorRadius)
	case *EllipticalArc:
		return s.ellArcs.AddWithCenter(mapped(v.Center), a.dir(v.MajorAxis), v.MajorRadius, v.MinorRadius, v.StartAngle, v.EndAngle)
	case *Spline:
		pts := make([]*Point, len(v.Points))
		for i, p := range v.Points {
			pts[i] = mapped(p)
		}
		return s.splines.AddWithPoints(pts, v.Closed, v.fit)
	case *Point:
		return s.points.Add(mapped(v).Position())
	default:
		return nil
	}
}

// entityPoints returns the points an entity owns (for in-place transforms).
func entityPoints(e Entity) []*Point {
	switch v := e.(type) {
	case *Line:
		return []*Point{v.A, v.B}
	case *Circle:
		return []*Point{v.Center}
	case *Arc:
		return []*Point{v.Center, v.Start, v.End}
	case *Ellipse:
		return []*Point{v.Center}
	case *EllipticalArc:
		return []*Point{v.Center}
	case *Spline:
		return v.Points
	case *Point:
		return []*Point{v}
	default:
		return nil
	}
}

// transformEntityDir rotates/reflects an entity's intrinsic direction vectors in place
// (only ellipses/elliptical arcs carry one).
func transformEntityDir(e Entity, a affine2) {
	switch v := e.(type) {
	case *Ellipse:
		v.MajorAxis = a.dir(v.MajorAxis)
	case *EllipticalArc:
		v.MajorAxis = a.dir(v.MajorAxis)
	}
}
