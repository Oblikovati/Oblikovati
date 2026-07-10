// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"errors"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Curve-derived datum points that need real curve∩surface geometry (#1842): the point where a
// curve pierces a surface entity (AddByCurveAndEntity), and the length-weighted centroid of a set
// of edges (AddAtCentroid). The intersection math is grounded on OCCT IntAna_IntConicQuad
// (line/circle ∩ plane); the tests cross-check against the oracle harness.

// curveEntityPointDef is the point where a curve meets a planar entity (Inventor
// AddByCurveAndEntity). proximity, when non-nil, selects the intersection nearest it — needed for a
// circular edge, which can cross a plane at two points. It goes sick when the curve misses the plane.
type curveEntityPointDef struct {
	curve     WorkRef
	entity    WorkRef
	proximity *math.Point3 // solution-selection point; nil = first intersection
}

func (d curveEntityPointDef) kindName() string { return "curve-and-entity" }
func (d curveEntityPointDef) refs() []WorkRef  { return []WorkRef{d.curve, d.entity} }
func (d curveEntityPointDef) eval(r workResolver) (math.Point3, error) {
	e, err := r.edge(d.curve)
	if err != nil {
		return math.Point3{}, err
	}
	pl, err := r.plane(d.entity)
	if err != nil {
		return math.Point3{}, err
	}
	hits, err := curvePlaneHits(e, pl)
	if err != nil {
		return math.Point3{}, err
	}
	return nearestHit(hits, d.proximity)
}

// curvePlaneHits returns the 0-2 points where an edge's carrier curve crosses a plane. A straight
// edge yields at most one; a circle yields zero, one (tangent), or two.
func curvePlaneHits(e *topo.Edge, pl sketch.Plane) ([]math.Point3, error) {
	switch g := e.Geometry().(type) {
	case geom.Line, geom.LineSegment:
		o, dir, err := edgeLine(e)
		if err != nil {
			return nil, err
		}
		p, ok := linePlanePoint(o, dir, pl)
		if !ok {
			return nil, errors.New("the curve is parallel to the entity; no intersection")
		}
		return []math.Point3{p}, nil
	case geom.Circle:
		return circlePlanePoints(g, pl), nil
	default:
		return nil, fmt.Errorf("curve-and-entity: unsupported curve geometry %T", g)
	}
}

// linePlanePoint returns where the infinite line through o with unit direction dir meets plane pl;
// ok is false when the line is parallel to the plane (denominator ~0).
func linePlanePoint(o math.Point3, dir math.UnitVector3, pl sketch.Plane) (math.Point3, bool) {
	n := pl.Normal().AsVector()
	d := dir.AsVector()
	denom := d.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point3{}, false
	}
	t := o.VectorTo(pl.Origin()).Dot(n) / denom
	return o.TranslateBy(d.Scale(t)), true
}

// circlePlanePoints returns the points where a full circle meets a plane. The circle's plane and the
// target plane meet in a line (unless parallel); the circle crosses that line where the line's
// distance to the circle centre is ≤ the radius — two points in general, one when tangent, none when
// the line clears the circle. Grounded on OCCT IntAna_IntConicQuad(circle, plane).
func circlePlanePoints(c geom.Circle, target sketch.Plane) []math.Point3 {
	bi, err := math.UnitVector3FromVector(c.Normal.Cross(c.RefDir))
	if err != nil {
		return nil
	}
	cPlane, err := sketch.NewPlane(c.Center, c.RefDir, bi)
	if err != nil {
		return nil
	}
	o, dir, err := planeIntersectionLine(cPlane, target)
	if err != nil {
		return nil // circle plane parallel to the target: no proper crossing
	}
	dv := dir.AsVector()
	foot := o.TranslateBy(dv.Scale(o.VectorTo(c.Center).Dot(dv)))
	gap := float64(foot.DistanceTo(c.Center))
	half := c.Radius*c.Radius - gap*gap
	if half < -float64(math.DefaultTolerance) {
		return nil // the line clears the circle
	}
	if half <= float64(math.DefaultTolerance) {
		return []math.Point3{foot} // tangent: single point
	}
	h := stdmath.Sqrt(half)
	return []math.Point3{foot.TranslateBy(dv.Scale(h)), foot.TranslateBy(dv.Scale(-h))}
}

// nearestHit picks the intersection to keep: the one nearest proximity, or the first when proximity
// is nil. An empty hit set means the curve missed the entity (→ healthy=false).
func nearestHit(hits []math.Point3, proximity *math.Point3) (math.Point3, error) {
	if len(hits) == 0 {
		return math.Point3{}, errors.New("the curve does not meet the entity")
	}
	if proximity == nil {
		return hits[0], nil
	}
	best, bestDist := hits[0], hits[0].DistanceTo(*proximity)
	for _, h := range hits[1:] {
		if dist := h.DistanceTo(*proximity); dist < bestDist {
			best, bestDist = h, dist
		}
	}
	return best, nil
}

// AddByCurveAndEntity creates the datum point where a curve pierces a planar entity (#1842).
// proximity may be nil (take the first solution) or a point selecting the nearest intersection.
func (c *WorkPoints) AddByCurveAndEntity(curve, entity WorkRef, proximity *math.Point3) *WorkPoint {
	return c.addUser(curveEntityPointDef{curve: curve, entity: entity, proximity: proximity})
}

// centroidPointDef is the length-weighted centroid of a set of edges (Inventor AddAtCentroid): the
// mean of each edge's chord midpoint weighted by its chord length — exact for straight edges (the
// documented case), the chord approximation for curved ones (matching edgeChordMidpoint). It goes
// sick when no referenced edge resolves to a non-zero length.
type centroidPointDef struct{ edges []WorkRef }

func (d centroidPointDef) kindName() string { return "centroid" }
func (d centroidPointDef) refs() []WorkRef  { return d.edges }
func (d centroidPointDef) eval(r workResolver) (math.Point3, error) {
	var weighted math.Vector3
	var total float64
	for _, ref := range d.edges {
		e, err := r.edge(ref)
		if err != nil {
			continue // skip an edge that no longer resolves; unhealthy only if none do
		}
		length := float64(e.StartVertex().Point().DistanceTo(e.EndVertex().Point()))
		weighted = weighted.Add(edgeChordMidpoint(e).AsVector().Scale(length))
		total += length
	}
	if total <= float64(math.DefaultTolerance) {
		return math.Point3{}, errors.New("centroid: no referenced edge resolved to a non-zero length")
	}
	return weighted.Scale(1 / total).AsPoint(), nil
}

// AddAtCentroid creates the datum point at the length-weighted centroid of the given edges (#1842).
func (c *WorkPoints) AddAtCentroid(edges ...WorkRef) *WorkPoint {
	return c.addUser(centroidPointDef{edges: append([]WorkRef(nil), edges...)})
}
