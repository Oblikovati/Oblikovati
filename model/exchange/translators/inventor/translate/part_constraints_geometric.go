// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"math"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// Inventor .ipt part translator — GEOMETRIC constraint application (M48 #2231 split of part.go). Binding
// the decoded value-free relations (geometric, tangent, circle-relation, ground, symmetry) to the emitted
// sketch by coordinate, each self-validated so applying it removes DOF without moving geometry.

// applyGeometricConstraints binds each decoded geometric constraint (horizontal / vertical /
// parallel / perpendicular — ipt.DecodeGeometricConstraints) onto the emitted sketch that holds its
// line(s), by coordinate. These constraints carry no value and are already satisfied by the solved
// geometry, so applying them removes degrees of freedom WITHOUT moving a point — the profile the
// feature consumes is unchanged. A constraint is applied only while the sketch still has free DOF,
// so a redundant one is never piled on to over-constrain it. Reports how many were applied.
func applyGeometricConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, gc := range ipt.DecodeGeometricConstraints(seg) {
		if applyGeoConstraint(def, gc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d geometric constraint(s)", applied)}
}

// applyTangentConstraints binds each decoded line↔circle tangent (ipt.DecodeTangentConstraints)
// onto the sketch that holds both the line and the circle, as an AddTangent. The geometry is
// already tangent (validated at decode — perpendicular distance from centre == radius), so it only
// removes degrees of freedom, never moves a point. DOF-guarded.
func applyTangentConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, tc := range ipt.DecodeTangentConstraints(seg) {
		if applyTangent(def, tc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d tangent constraint(s)", applied)}
}

// applyTangent binds one tangent to the first sketch that holds both its line and its circle.
func applyTangent(def *compdef.PartComponentDefinition, tc ipt.TangentConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		l := lineAtCoords(sk, tc.Line)
		c := circleAtCoord(sk, tc.Center, tc.Radius)
		if l == nil || c == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddTangent(l, c)
		return true
	}
	return false
}

// applyCircleRelations binds each decoded concentric / equal-radius constraint
// (ipt.DecodeCircleRelations) onto the sketch holding both circles. The relation already holds in
// the geometry (validated at decode), so it only removes degrees of freedom. DOF-guarded.
func applyCircleRelations(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, cr := range ipt.DecodeCircleRelations(seg) {
		if applyCircleRelation(def, cr) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d circle relation(s)", applied)}
}

// applyCircleRelation binds one circle relation to the first sketch holding both its circles.
func applyCircleRelation(def *compdef.PartComponentDefinition, cr ipt.CircleRelation) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		c1 := circleAtCoord(sk, cr.C1, cr.R1)
		c2 := circleAtCoord(sk, cr.C2, cr.R2)
		if c1 == nil || c2 == nil || c1 == c2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if cr.Kind == ipt.GeoConcentric {
			sk.GeometricConstraints().AddConcentric(c1, c2)
		} else {
			sk.GeometricConstraints().AddEqualRadius(c1, c2)
		}
		return true
	}
	return false
}

// applyGroundConstraints binds each decoded ground constraint (ipt.DecodeGroundConstraints) onto
// the sketch holding its entity, as an AddGround. Grounding freezes the entity at its current
// position, so it only removes degrees of freedom — no geometry moves. DOF-guarded.
func applyGroundConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, gc := range ipt.DecodeGroundConstraints(seg) {
		if applyGround(def, gc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d ground constraint(s)", applied)}
}

// applyGround grounds one decoded entity on the first sketch that holds it.
func applyGround(def *compdef.PartComponentDefinition, gc ipt.GroundConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		if sk.DegreesOfFreedom() <= 0 {
			continue
		}
		switch gc.Kind {
		case ipt.GroundLine:
			if l := lineAtCoords(sk, gc.Line); l != nil {
				sk.GeometricConstraints().AddGround(l)
				return true
			}
		case ipt.GroundCircle:
			if c := circleAtCoord(sk, gc.Center, gc.Radius); c != nil {
				sk.GeometricConstraints().AddGround(c)
				return true
			}
		case ipt.GroundPoint:
			if p := pointAtCoord(sk, gc.Pt); p != nil {
				sk.GeometricConstraints().AddGround(p)
				return true
			}
		}
	}
	return false
}

// applySymmetryConstraints binds each decoded symmetry (ipt.DecodeSymmetryConstraints) onto the
// sketch holding both points and the axis line, as an AddSymmetry. The geometry is already
// symmetric (validated at decode — each point reflects onto the other across the axis), so it only
// removes degrees of freedom, never moves a point. DOF-guarded.
func applySymmetryConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, sc := range ipt.DecodeSymmetryConstraints(seg) {
		if applySymmetry(def, sc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d symmetry constraint(s)", applied)}
}

// applySymmetry binds one symmetry to the first sketch that holds both points and its axis line.
func applySymmetry(def *compdef.PartComponentDefinition, sc ipt.SymmetryConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p1 := pointAtCoord(sk, sc.P1)
		p2 := pointAtCoord(sk, sc.P2)
		ax := lineAtCoords(sk, sc.Axis)
		if p1 == nil || p2 == nil || ax == nil || p1 == p2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddSymmetry(p1, p2, ax)
		return true
	}
	return false
}

// circleAtCoord returns the sketch circle whose centre matches c and radius matches r (within
// coincideEps), or nil.
func circleAtCoord(sk *sketch.Sketch, c ipt.Point2D, r float64) *sketch.Circle {
	circles := sk.Circles()
	for i := 0; i < circles.Count(); i++ {
		if q := circles.Item(i); samePt(q.Center, c) && math.Abs(float64(q.Radius)-r) < coincideEps {
			return q
		}
	}
	return nil
}

// applyGeoConstraint applies one geometric constraint to the first sketch containing its geometry.
func applyGeoConstraint(def *compdef.PartComponentDefinition, gc ipt.GeoConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		switch gc.Kind {
		case ipt.GeoHorizontal, ipt.GeoVertical:
			pa, pb := pointAtCoord(sk, gc.L1[0]), pointAtCoord(sk, gc.L1[1])
			if pa == nil || pb == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			if gc.Kind == ipt.GeoHorizontal {
				sk.GeometricConstraints().AddHorizontal(pa, pb)
			} else {
				sk.GeometricConstraints().AddVertical(pa, pb)
			}
			return true
		case ipt.GeoParallel, ipt.GeoPerpendicular, ipt.GeoCollinear, ipt.GeoEqualLength:
			l1, l2 := lineAtCoords(sk, gc.L1), lineAtCoords(sk, gc.L2)
			if l1 == nil || l2 == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			switch gc.Kind {
			case ipt.GeoParallel:
				sk.GeometricConstraints().AddParallel(l1, l2)
			case ipt.GeoPerpendicular:
				sk.GeometricConstraints().AddPerpendicular(l1, l2)
			case ipt.GeoCollinear:
				sk.GeometricConstraints().AddCollinear(l1, l2)
			case ipt.GeoEqualLength:
				sk.GeometricConstraints().AddEqualLength(l1, l2)
			}
			return true
		case ipt.GeoMidpoint:
			// Bind only when a sketch point actually sits at the line's midpoint (the pinned point,
			// whether a standalone point or another line's endpoint at a T-junction). If none does,
			// the constraint isn't reproduced rather than inventing a point.
			l := lineAtCoords(sk, gc.L1)
			p := pointAtCoord(sk, gc.Pt)
			if l == nil || p == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			sk.GeometricConstraints().AddMidpoint(p, l)
			return true
		case ipt.GeoPointOnLine:
			// The pinned vertex lies on the line's interior (validated at decode). Bind it only when
			// both the line and a sketch point at that vertex are present, so it never invents a point.
			l := lineAtCoords(sk, gc.L1)
			p := pointAtCoord(sk, gc.Pt)
			if l == nil || p == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			sk.GeometricConstraints().AddPointOnLine(p, l)
			return true
		}
	}
	return false
}
