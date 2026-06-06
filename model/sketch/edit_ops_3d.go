// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

// This file holds the 3D-sketch editing operations (M22-F08): move, rotate, copy and
// delete over a selection of entities. Transforms apply a [math.Matrix4] to the unique
// defining points of the selection; copy clones the entities over fresh points; delete
// removes an entity, drops its orphaned points, and detaches the constraints/dimensions
// that referenced them.

// entityPoints3D returns the constrainable defining points of a 3D entity (empty for the
// point-free fixed spline / equation curve).
func entityPoints3D(e Entity) []*Point3D {
	switch v := e.(type) {
	case *Point3D:
		return []*Point3D{v}
	case *Line3D:
		return []*Point3D{v.A, v.B}
	case *Circle3D:
		return []*Point3D{v.Center}
	case *Arc3D:
		return []*Point3D{v.Center, v.Start, v.End}
	case *Ellipse3D:
		return []*Point3D{v.Center}
	case *EllipticalArc3D:
		return []*Point3D{v.Center}
	case *HelicalCurve3D:
		return []*Point3D{v.Origin}
	case *Spline3D:
		return v.Points
	default:
		return nil
	}
}

// MoveEntities3D translates the selection's points by v.
func (s *Sketch3D) MoveEntities3D(ents []Entity, v math.Vector3) {
	s.transformPoints3D(ents, math.Translation4(v))
}

// RotateEntities3D rotates the selection's points by angle (radians) about the axis
// through center in direction axis.
func (s *Sketch3D) RotateEntities3D(ents []Entity, center math.Point3, axis math.UnitVector3, angle float64) {
	s.transformPoints3D(ents, math.Rotation4(math.Scalar(angle), axis, center))
}

// transformPoints3D applies m to each unique point of the selection (in place).
func (s *Sketch3D) transformPoints3D(ents []Entity, m math.Matrix4) {
	seen := map[*Point3D]bool{}
	for _, e := range ents {
		for _, p := range entityPoints3D(e) {
			if seen[p] {
				continue
			}
			p.SetPosition(m.TransformPoint(p.Position()))
			seen[p] = true
		}
	}
}

// CopyEntities3D duplicates the selection translated by v, returning the new entities. New
// points are shared across copied entities that shared an original, keeping the copy a
// connected whole.
func (s *Sketch3D) CopyEntities3D(ents []Entity, v math.Vector3) []Entity {
	m := math.Translation4(v)
	pmap := map[*Point3D]*Point3D{}
	mapped := func(p *Point3D) *Point3D {
		if np, ok := pmap[p]; ok {
			return np
		}
		np := s.newPoint3D(m.TransformPoint(p.Position()))
		pmap[p] = np
		return np
	}
	out := make([]Entity, 0, len(ents))
	for _, e := range ents {
		if c := s.cloneEntity3D(e, mapped, m); c != nil {
			out = append(out, c)
		}
	}
	return out
}

// cloneEntity3D duplicates one entity, resolving its points through mapped.
func (s *Sketch3D) cloneEntity3D(e Entity, mapped func(*Point3D) *Point3D, m math.Matrix4) Entity {
	switch v := e.(type) {
	case *Point3D:
		np := mapped(v)
		s.addEntity3D(np)
		return np
	case *Line3D:
		return s.addLine3DPts(mapped(v.A), mapped(v.B))
	case *Circle3D:
		return s.addCircle3DPts(mapped(v.Center), v.Axis, float64(v.Radius))
	case *Arc3D:
		return s.addArc3DPts(mapped(v.Center), mapped(v.Start), mapped(v.End), v.CounterClockwise)
	case *Spline3D:
		return s.addSpline3DPts(mapPoints3D(v.Points, mapped), v.Closed, v.fit)
	default:
		return s.cloneFixedEntity3D(e, m)
	}
}

// cloneFixedEntity3D copies the point-free entities (conics, helix, fixed spline) under m.
func (s *Sketch3D) cloneFixedEntity3D(e Entity, m math.Matrix4) Entity {
	switch v := e.(type) {
	case *Ellipse3D:
		return s.addEllipse3DPt(s.newPoint3D(m.TransformPoint(v.Center.Position())), v.Normal, v.MajorAxis, float64(v.MajorRadius), float64(v.MinorRadius))
	case *EllipticalArc3D:
		return s.addEllipticalArc3DPt(s.newPoint3D(m.TransformPoint(v.Center.Position())), v.Normal, v.MajorAxis, float64(v.MajorRadius), float64(v.MinorRadius), v.StartAngle, v.SweepAngle)
	case *HelicalCurve3D:
		return s.addHelix3DPt(s.newPoint3D(m.TransformPoint(v.Origin.Position())), v.Axis, float64(v.StartRadius), v.AxialPerTurn, v.RadialPerTurn, v.Turns, v.Clockwise)
	case *FixedSpline3D:
		return s.AddFixedSpline3D(transformPoint3s(v.Pts, m), v.Closed)
	default:
		return nil // equation curve: no spatial copy (its definition is parametric)
	}
}

// mapPoints3D maps a point slice through the copy point-mapper.
func mapPoints3D(pts []*Point3D, mapped func(*Point3D) *Point3D) []*Point3D {
	out := make([]*Point3D, len(pts))
	for i, p := range pts {
		out[i] = mapped(p)
	}
	return out
}

// transformPoint3s applies m to each point of a slice (for fixed-spline coords).
func transformPoint3s(pts []math.Point3, m math.Matrix4) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = m.TransformPoint(p)
	}
	return out
}

// DeleteEntity3D removes an entity, drops the points no remaining entity references, and
// detaches the constraints/dimensions bound to those dropped points. It reports whether
// the entity was present.
func (s *Sketch3D) DeleteEntity3D(e Entity) bool {
	if !s.removeEntity3D(e) {
		return false
	}
	dropped := s.pruneOrphanPoints3D()
	s.detachConstraints3D(dropped)
	return true
}

// removeEntity3D drops an entity from the geometry list and id index.
func (s *Sketch3D) removeEntity3D(e Entity) bool {
	for i, x := range s.ents {
		if x == e {
			s.ents = append(s.ents[:i], s.ents[i+1:]...)
			delete(s.byID, e.EntityID())
			return true
		}
	}
	return false
}

// pruneOrphanPoints3D rebuilds the solver point list from the surviving entities, dropping
// points no entity references any longer, and returns the dropped points.
func (s *Sketch3D) pruneOrphanPoints3D() []*Point3D {
	live := map[*Point3D]bool{}
	for _, e := range s.ents {
		for _, p := range entityPoints3D(e) {
			live[p] = true
		}
	}
	kept := make([]*Point3D, 0, len(s.pts))
	var dropped []*Point3D
	for _, p := range s.pts {
		if live[p] {
			kept = append(kept, p)
		} else {
			dropped = append(dropped, p)
		}
	}
	s.pts = kept
	return dropped
}

// detachConstraints3D removes the geometric constraints and dimensions whose variables
// point into any of the dropped points (matched by scalar-pointer identity).
func (s *Sketch3D) detachConstraints3D(dropped []*Point3D) {
	if len(dropped) == 0 {
		return
	}
	vars := droppedVarSet3D(dropped)
	for _, c := range s.geomCons.All() {
		if touchesVars3D(c.Variables(), vars) {
			s.geomCons.Delete(c)
		}
	}
	kept := s.dimCons.items[:0]
	for _, d := range s.dimCons.items {
		if !touchesVars3D(d.vars, vars) {
			kept = append(kept, d)
		}
	}
	s.dimCons.items = kept
}

// droppedVarSet3D is the set of scalar-variable addresses of the dropped points.
func droppedVarSet3D(dropped []*Point3D) map[*math.Scalar]bool {
	set := map[*math.Scalar]bool{}
	for _, p := range dropped {
		set[&p.X], set[&p.Y], set[&p.Z] = true, true, true
	}
	return set
}

// touchesVars3D reports whether any of vars is in the dropped set.
func touchesVars3D(vars []*math.Scalar, dropped map[*math.Scalar]bool) bool {
	for _, v := range vars {
		if dropped[v] {
			return true
		}
	}
	return false
}
