// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"math"

	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// Inventor .ipt part translator — DIMENSIONAL constraint application (M48 #2231 split of part.go).
// Binding the decoded valued dimensions (distance, angle, offset, radius, revolve-radius, axial-length,
// centreline anchor) to the emitted sketch, plus the shared sketch-entity coordinate lookups.

// applyDistanceDimensions binds each decoded point-to-point distance dimension
// (ipt.DecodeDistanceDimensions) onto the emitted sketch holding both endpoints, as a driving
// AddDistance. The value equals the current separation, so it pins DOF without moving geometry. It
// is applied only while the sketch has free DOF, so a redundant dimension can't over-constrain.
func applyDistanceDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, dm := range ipt.DecodeDistanceDimensions(seg) {
		if applyDistanceDim(def, dm) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d distance dimension(s)", applied)}
}

// applyDistanceDim adds one distance dimension to the first sketch containing both endpoints, kept
// only if it does not move geometry (keptWithoutMoving) — the value equals the current separation,
// so a well-decoded dimension is a pure DOF reduction, but the guard makes that a checked invariant
// rather than an assumption.
func applyDistanceDim(def *compdef.PartComponentDefinition, dm ipt.DistanceDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		pa, pb := pointAtCoord(sk, dm.A), pointAtCoord(sk, dm.B)
		if pa == nil || pb == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddDistance(pa, pb, fmt.Sprintf("%g cm", dm.Value))
		})
	}
	return false
}

// applyAngleDimensions binds each decoded angle dimension (ipt.DecodeAngleDimensions) onto the
// sketch holding both its lines, as an AddAngle of their current angle. The value equals the
// present geometric angle, so it pins the angle DOF without moving geometry. Applied only while the
// sketch has free DOF.
func applyAngleDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, ad := range ipt.DecodeAngleDimensions(seg) {
		if applyAngleDim(def, ad) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d angle dimension(s)", applied)}
}

// applyAngleDim adds one angle dimension to the first sketch that holds both its lines — but only
// if it does not move the geometry. An angle dimension pins the UNSIGNED angle between two lines; on
// an under-constrained sketch the solver can satisfy it by rotating/flipping the profile into a
// different configuration, silently corrupting a revolve/extrude profile (observed on a chamfered
// revolve — the profile drifted and the swept volume changed). So it is applied speculatively, the
// sketch is solved, and the dimension is kept only when every point stayed put; otherwise it is
// removed and the points restored. This keeps the correctness-first invariant: a decoded dimension
// is reproduced only when it is a pure degree-of-freedom reduction, never a geometry edit.
func applyAngleDim(def *compdef.PartComponentDefinition, ad ipt.AngleDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		l1, l2 := lineAtCoords(sk, ad.L1), lineAtCoords(sk, ad.L2)
		if l1 == nil || l2 == nil || l1 == l2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddAngle(l1, l2, fmt.Sprintf("%g deg", ad.Degrees))
		})
	}
	return false
}

// applyOffsetDimensions binds each decoded offset (distance-from-line) dimension
// (ipt.DecodeOffsetDimensions) onto the sketch holding its point and reference line, as an
// AddOffsetDim of the current perpendicular distance. Kept only if it does not move geometry.
func applyOffsetDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, od := range ipt.DecodeOffsetDimensions(seg) {
		if applyOffsetDim(def, od) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d offset dimension(s)", applied)}
}

// applyOffsetDim adds one offset dimension to the first sketch that holds its point and line.
func applyOffsetDim(def *compdef.PartComponentDefinition, od ipt.OffsetDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p, l := pointAtCoord(sk, od.Pt), lineAtCoords(sk, od.Line)
		if p == nil || l == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddOffsetDim(p, l, false, fmt.Sprintf("%g cm", od.Value))
		})
	}
	return false
}

// keptWithoutMoving adds a dimension via add and keeps it only when it is a faithful reproduction:
// it must STRICTLY REDUCE the sketch's degrees of freedom (a redundant dimension that duplicates an
// existing constraint would only over-constrain — e.g. the shaft's offset dim vs its radius dims)
// AND leave every point where it was after a solve (a dimension whose solve admits a different
// configuration, like a two-line angle flip, would silently edit the geometry). If either fails the
// dimension is deleted and the points restored. Reports whether it was kept.
func keptWithoutMoving(sk *sketch.Sketch, add func() (*sketch.DimensionConstraint, error)) bool {
	pts := sk.Points()
	snap := make([]m.Point2, pts.Count())
	for i := 0; i < pts.Count(); i++ {
		snap[i] = pts.Item(i).Position()
	}
	dofBefore := sk.DegreesOfFreedom()
	dim, err := add()
	if err != nil {
		return false
	}
	sk.Solve()
	if sk.DegreesOfFreedom() < dofBefore && !anyPointMoved(pts, snap) {
		return true
	}
	sk.DimensionConstraints().Delete(dim)
	for i := 0; i < pts.Count(); i++ {
		pts.Item(i).SetPosition(snap[i])
	}
	return false
}

// anyPointMoved reports whether any sketch point drifted from its snapshot beyond coincideEps.
func anyPointMoved(pts *sketch.Points, snap []m.Point2) bool {
	for i := 0; i < pts.Count(); i++ {
		if pts.Item(i).Position().DistanceTo(snap[i]) > coincideEps {
			return true
		}
	}
	return false
}

// applyRadiusDimensions binds each decoded radius/diameter dimension (ipt.DecodeRadiusDimensions)
// onto the sketch holding its circle or arc, as an AddRadius of the curve's own radius. Radius and
// diameter are indistinguishable in the file and pin the same DOF, so both apply as a radius
// dimension — the value equals the current radius, so nothing moves. Applied only while the sketch
// has free DOF.
func applyRadiusDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, rd := range ipt.DecodeRadiusDimensions(seg) {
		if applyRadiusDim(def, rd) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d radius/diameter dimension(s)", applied)}
}

// applyRadiusDim adds one radius dimension to the first sketch that holds its circle or arc, kept
// only if it does not move geometry (keptWithoutMoving). An arc's radius has no DOF of its own (it
// is |centre − start|), so pinning it drives the centre/start points — the guard ensures the solve
// pins the radius in place rather than sliding those points to a different arc.
func applyRadiusDim(def *compdef.PartComponentDefinition, rd ipt.RadiusDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		if sk.DegreesOfFreedom() <= 0 {
			continue
		}
		var c sketch.CircularCurve
		if rd.Arc {
			if a := arcAtCoord(sk, rd.Center, rd.Radius); a != nil {
				c = a
			}
		} else if cc := circleAtCoord(sk, rd.Center, rd.Radius); cc != nil {
			c = cc
		}
		if c == nil {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddRadius(c, fmt.Sprintf("%g cm", rd.Radius))
		})
	}
	return false
}

// arcAtCoord returns the sketch arc whose centre matches center and radius matches r (within
// coincideEps), or nil.
func arcAtCoord(sk *sketch.Sketch, center ipt.Point2D, r float64) *sketch.Arc {
	arcs := sk.Arcs()
	for i := 0; i < arcs.Count(); i++ {
		if a := arcs.Item(i); samePt(a.Center, center) && math.Abs(float64(a.Radius())-r) < coincideEps {
			return a
		}
	}
	return nil
}

// applyRevolveRadii binds each decoded revolve radius dimension (ipt.DecodeRevolveRadii) as a
// HORIZONTAL distance from the x=0 centreline to the vertical profile edge at x=V, in the sketch
// holding both (the reunited profile+centreline sketch). The value equals the edge's current x, so
// it pins the radius without moving geometry. Applied only while the sketch has free DOF.
func applyRevolveRadii(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, x := range ipt.DecodeRevolveRadii(seg) {
		if applyRevolveRadius(def, x) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d revolve radius dimension(s)", applied)}
}

// applyRevolveRadius adds one radius dimension: a horizontal distance from a centreline point (x≈0)
// to an edge point (x≈radius) in the first sketch that holds both.
func applyRevolveRadius(def *compdef.PartComponentDefinition, radius float64) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p0, pv := pointAtX(sk, 0), pointAtX(sk, radius)
		if p0 == nil || pv == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if _, err := sk.DimensionConstraints().AddDistanceOriented(p0, pv, fmt.Sprintf("%g cm", radius), sketch.HorizontalDistance); err != nil {
			return false
		}
		return true
	}
	return false
}

// applyAxialLengths binds each decoded axial step-length dimension (ipt.DecodeAxialLengths) as a
// VERTICAL distance between the two horizontal profile edges it spans, in the sketch holding both.
// The value equals the edges' current separation, so it pins the step length without moving
// geometry. Applied only while the sketch has free DOF.
func applyAxialLengths(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, ax := range ipt.DecodeAxialLengths(seg) {
		if applyAxialLength(def, ax) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d axial length dimension(s)", applied)}
}

// applyAxialLength adds one vertical distance between a point at y≈Y1 and a point at y≈Y2.
func applyAxialLength(def *compdef.PartComponentDefinition, ax ipt.AxialLength) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p1, p2 := pointAtY(sk, ax.Y1), pointAtY(sk, ax.Y2)
		if p1 == nil || p2 == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if _, err := sk.DimensionConstraints().AddDistanceOriented(p1, p2, fmt.Sprintf("%g cm", ax.Value), sketch.VerticalDistance); err != nil {
			return false
		}
		return true
	}
	return false
}

// applyCentrelineAnchor fixes a revolve sketch's point at the sketch origin (0,0). A revolve's
// centreline runs from the origin, and geometry drawn AT the sketch origin is coincident with the
// origin — a fixed reference in every CAD sketch — so pinning that point reproduces the anchoring
// the file leaves implicit (there is no explicit fix node; the axis line carries no vertical/fix
// constraint of its own). Fixing a point at its current position never moves geometry, and it is
// applied only while the sketch has free DOF. Gated to revolves so it only ever pins a centreline
// origin, not an incidental origin-touching corner of an extrude profile.
func applyCentrelineAnchor(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok || !ipt.HasRevolve(seg) {
		return nil
	}
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p := pointAtCoord(sk, ipt.Point2D{X: 0, Y: 0})
		if p == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddFix(p)
		return []string{"anchored the centreline to the sketch origin"}
	}
	return nil
}

// pointAtY returns a sketch point whose Y coordinate is within coincideEps of y, or nil.
func pointAtY(sk *sketch.Sketch, y float64) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.Y)-y) < coincideEps {
			return q
		}
	}
	return nil
}

// pointAtX returns a sketch point whose X coordinate is within coincideEps of x, or nil.
func pointAtX(sk *sketch.Sketch, x float64) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.X)-x) < coincideEps {
			return q
		}
	}
	return nil
}

// pointAtCoord returns the sketch point at coordinate p (within coincideEps), or nil.
func pointAtCoord(sk *sketch.Sketch, p ipt.Point2D) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.X)-p.X) < coincideEps && math.Abs(float64(q.Y)-p.Y) < coincideEps {
			return q
		}
	}
	return nil
}

// lineAtCoords returns the sketch line whose endpoints match e (in either order), or nil.
func lineAtCoords(sk *sketch.Sketch, e [2]ipt.Point2D) *sketch.Line {
	lines := sk.Lines()
	for i := 0; i < lines.Count(); i++ {
		l := lines.Item(i)
		if (samePt(l.A, e[0]) && samePt(l.B, e[1])) || (samePt(l.A, e[1]) && samePt(l.B, e[0])) {
			return l
		}
	}
	return nil
}

// samePt reports whether a sketch point sits at coordinate p (within coincideEps).
func samePt(q *sketch.Point, p ipt.Point2D) bool {
	return math.Abs(float64(q.X)-p.X) < coincideEps && math.Abs(float64(q.Y)-p.Y) < coincideEps
}

// sharedPoints returns a coordinate→Point allocator over one sketch: the first time a coordinate
// is seen it mints a sketch Point; a later corner within coincideEps of it reuses the same Point.
// This makes touching profile corners structurally coincident (the original's endpoint coincidence
// constraints), so the rebuilt sketch has the same DOF instead of independent duplicated endpoints.
func sharedPoints(sk *sketch.Sketch) func(ipt.Point2D) *sketch.Point {
	type cached struct {
		p  ipt.Point2D
		pt *sketch.Point
	}
	var cache []cached
	return func(p ipt.Point2D) *sketch.Point {
		for _, e := range cache {
			if math.Abs(e.p.X-p.X) < coincideEps && math.Abs(e.p.Y-p.Y) < coincideEps {
				return e.pt
			}
		}
		pt := sk.Points().Add(m.P2(p.X, p.Y))
		cache = append(cache, cached{p, pt})
		return pt
	}
}

// coincideEps is the coordinate tolerance (cm) below which two profile corners are treated as one
// coincident sketch point.
const coincideEps = 1e-6
