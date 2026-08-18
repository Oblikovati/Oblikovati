// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Resolving the geometry a hole placement is measured against (Oblikovati#1861). Each helper takes
// the running body and the recorded reference — a lineage key, or a geometric descriptor for an
// author that could not mint one — and returns the geometry, or an error naming what was lost. The
// error path matters as much as the happy one: a placement whose anchor has gone is the feature
// going sick, which the engine reports, not a hole quietly drilled somewhere else.

// HoleFaceRef names the planar face a placement starts its bore on, by lineage key or by geometry.
// It is the pair [HoleDefinition] has always carried, named so every placement resolves a face the
// same way.
type HoleFaceRef struct {
	Key  []byte
	Geom *topo.GeometricFaceRef
}

// resolve finds the face on the running body and the direction that drills INTO it (against the
// outward normal).
func (r HoleFaceRef) resolve(body *topo.Body) (*topo.Face, math.UnitVector3, error) {
	face, ok := resolveHoleFace(body, r, nil)
	if !ok {
		return nil, math.UnitVector3{}, errors.New("hole: placement face reference lost")
	}
	into, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0).Scale(-1))
	if err != nil {
		return nil, math.UnitVector3{}, errors.New("hole: placement face has no normal")
	}
	return face, into, nil
}

// circularEdgeCentre is the centre of the circle a concentric placement is aligned with. Only a
// circular edge names an axis; a straight or spline edge does not, so it is refused rather than
// approximated by, say, its midpoint.
func circularEdgeCentre(body *topo.Body, key []byte, ref *topo.GeometricEdgeRef) (math.Point3, error) {
	edge, err := resolveHoleEdge(body, key, ref, "concentric reference")
	if err != nil {
		return math.Point3{}, err
	}
	switch c := edge.Geometry().(type) {
	case geom.Circle:
		return c.Center, nil
	case geom.Arc3d:
		return c.Center, nil
	default:
		return math.Point3{}, fmt.Errorf("hole: the concentric reference edge is a %s, not a circle or arc — "+
			"only a circular edge names an axis to be concentric with", curveKindName(edge.Geometry()))
	}
}

// resolveHoleEdge finds a referenced edge on the running body: by lineage key, else by the
// geometric descriptor an external author records instead (ADR-0040).
func resolveHoleEdge(body *topo.Body, key []byte, ref *topo.GeometricEdgeRef, what string) (*topo.Edge, error) {
	if len(key) > 0 {
		if e, ok := body.FindEdgeByKey(key); ok {
			return e, nil
		}
		return nil, fmt.Errorf("hole: %s edge reference lost", what)
	}
	if ref == nil {
		return nil, fmt.Errorf("hole: %s needs an edge reference", what)
	}
	if e, ok := body.FindEdgeByGeometry(*ref, geomEdgeBindTol); ok {
		return e, nil
	}
	return nil, fmt.Errorf("hole: %s edge was not found on the current body", what)
}

// faceLine is a straight line lying in a placement face's plane: a point on it and its direction.
type faceLine struct {
	origin math.Point3
	dir    math.Vector3
}

// offsetLineOnFace is the locus of points at distance d from a reference edge, measured in the
// placement face's plane. The offset runs TOWARD the face's interior — the two edges a linear
// placement measures from bound the face, so offsetting away from it would name a point off the
// part, and which side the caller meant is never in doubt.
func offsetLineOnFace(body *topo.Body, face *topo.Face, key []byte, ref *topo.GeometricEdgeRef,
	d float64, what string) (faceLine, error) {
	edge, err := resolveHoleEdge(body, key, ref, what)
	if err != nil {
		return faceLine{}, err
	}
	seg, ok := edge.Geometry().(geom.LineSegment)
	if !ok {
		return faceLine{}, fmt.Errorf("hole: linear placement's %s is a %s; it must be a straight edge to measure an offset from",
			what, curveKindName(edge.Geometry()))
	}
	dir := seg.StartPoint.VectorTo(seg.EndPoint)
	inward, err := inwardNormalOnFace(face, seg.StartPoint, dir)
	if err != nil {
		return faceLine{}, fmt.Errorf("hole: linear placement's %s: %w", what, err)
	}
	return faceLine{origin: seg.StartPoint.TranslateBy(inward.Scale(math.Scalar(d))), dir: dir}, nil
}

// inwardNormalOnFace is the in-plane unit direction perpendicular to an edge, pointing at the
// face's interior (taken as its centroid).
func inwardNormalOnFace(face *topo.Face, on math.Point3, along math.Vector3) (math.Vector3, error) {
	n := face.Geometry().NormalAt(0, 0)
	perp := n.Cross(along)
	if perp.Length() <= math.DefaultTolerance {
		return math.Vector3{}, errors.New("the edge runs along the face normal, so no in-plane offset direction exists")
	}
	unit := perp.AsUnit().AsVector()
	if on.VectorTo(centroidOf(faceVertexPoints(face))).Dot(unit) < 0 {
		return unit.Scale(-1), nil
	}
	return unit, nil
}

// crossOnPlane intersects two lines known to lie in the same plane, whose normal is n. Parallel
// reference edges name no point, so they are an error rather than a far-away intersection.
func crossOnPlane(a, b faceLine, n math.Vector3) (math.Point3, error) {
	den := a.dir.Cross(b.dir).Dot(n)
	if den > -math.DefaultTolerance && den < math.DefaultTolerance {
		return math.Point3{}, errors.New("hole: the linear placement's two reference edges are parallel, so they cross nowhere")
	}
	t := a.origin.VectorTo(b.origin).Cross(b.dir).Dot(n) / den
	return a.origin.TranslateBy(a.dir.Scale(t)), nil
}

// curveKindName names an edge's analytic kind for an error message, falling back to a plain label
// for a curve that does not declare one.
func curveKindName(c geom.Curve3) string {
	if k, ok := c.(geom.KindedCurve); ok {
		return k.Kind().String()
	}
	return "curve"
}
