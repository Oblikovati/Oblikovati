// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// isClosedCircularEdge reports whether e is a full circular rim: closed (its start and end vertex are
// the SAME vertex) AND its geometry is a geom.Circle, or a geom.Arc3d sweeping ~2π (within
// zoneFullCircleTol) back to that shared vertex. The STEP importer never emits geom.Circle — every
// imported full circle round-trips as a full-sweep Arc3d (0 geom.Circle construction sites in
// kernel/exchange/**) — so both forms must count as "a closed circular rim". This is the SOLE such
// predicate in the package: the rim-fillet pick gate (loneRimPick, resolveRim below) and the
// sphere-zone cap fan's fullCircleRimGeom (sphere_zone_mesh.go) both call it, so a full-sweep Arc3d is
// recognized identically everywhere a closed circular edge is classified.
func isClosedCircularEdge(e *topo.Edge) bool {
	if e.StartVertex() != e.EndVertex() {
		return false
	}
	switch c := e.Geometry().(type) {
	case geom.Circle:
		return true
	case geom.Arc3d:
		return stdmath.Abs(stdmath.Abs(c.SweepAngle)-2*stdmath.Pi) < zoneFullCircleTol
	}
	return false
}

// FilletCylinderRim rounds a CONVEX circular rim — a closed circle edge where a cylinder face meets
// a perpendicular planar cap (a boss/cylinder top, a counterbore lip) — into a toroidal band. It is
// the closed-rim curved-adjacent fillet, the case that terminates cleanly (a full circle, no run-out
// corners). The body is rebuilt with the rim replaced by the cap-tangent and cyl-tangent circles, the
// neighbour cap and cylinder receded to them, and the torus between; every other face is copied
// verbatim. Concave rims (a bore lip), non-perpendicular caps, and r ≥ radius are rejected.
func FilletCylinderRim(b *topo.Body, rimKey []byte, r float64) (*topo.Body, error) {
	rim, err := resolveRim(b, rimKey, r)
	if err != nil {
		return nil, err
	}
	return rebuildWithRimFillet(b, rim)
}

// loneRimPick returns the pick when it is the sole, constant-radius selection of a CLOSED circular
// edge between a cylinder and a plane — the rim-fillet trigger. nil otherwise (the ordinary
// plane-plane path runs). A near-rim edge that turns out unbuildable (concave, oblique cap) still
// routes here, so FilletCylinderRim reports the precise reason rather than the misleading planar one.
func loneRimPick(body *topo.Body, picks []EdgeFilletRadii) *EdgeFilletRadii {
	if len(picks) != 1 || picks[0].R0 != picks[0].R1 {
		return nil
	}
	e, ok := body.FindEdgeByKey(picks[0].Key)
	if !ok {
		return nil
	}
	if !isClosedCircularEdge(e) {
		return nil
	}
	if _, _, _, _, err := rimFaces(e); err != nil {
		return nil
	}
	return &picks[0]
}

// rimFillet is the solved geometry + topology of a rim fillet: the two faces and edges it replaces and
// the circles/torus/seam it inserts.
type rimFillet struct {
	cyl      *topo.Face // the cylinder wall
	cap      *topo.Face // the planar cap
	rimEdge  *topo.Edge // the rim circle (cyl ∩ cap), removed
	seamEdge *topo.Edge // the wall's seam edge meeting the rim vertex, re-aimed lower
	rimV     *topo.Vertex
	bottomV  *topo.Vertex // the seam's other end (kept)
	cylTan   geom.Circle  // where the torus meets the developable host (cylinder Rc, or a cone contact circle)
	capTan   geom.Circle  // where the torus meets the cap (radius Rc−r, at the cap)
	torus    geom.Torus
	r        float64
	// seamMid is the on-arc midpoint of the tube seam joining cylTan.PointAt(0) → capTan.PointAt(0)
	// (the point the seam Arc3dByThreePoints passes through). A perpendicular CYLINDER rim's contacts
	// sit at tube v=0 and v=π/2, so it is torus.PointAt(0, π/4) (quarterTube); a CONE rim's contacts sit
	// at other tube angles, so the closed-rim cone solver supplies the true between-contacts midpoint.
	seamMid math.Point3
}

// resolveRim validates the picked edge is a convex cylinder/cap rim and solves the fillet geometry.
func resolveRim(b *topo.Body, rimKey []byte, r float64) (*rimFillet, error) {
	e, ok := b.FindEdgeByKey(rimKey)
	if !ok {
		return nil, fmt.Errorf("fillet: rim edge reference lost: %x", rimKey)
	}
	if !isClosedCircularEdge(e) {
		return nil, fmt.Errorf("fillet: a rim fillet needs a closed circular edge")
	}
	cylF, capF, cyl, pl, err := rimFaces(e)
	if err != nil {
		return nil, err
	}
	if r <= 0 || r >= cyl.Radius {
		return nil, fmt.Errorf("fillet: rim radius %g must be in (0, cylinder radius %g)", r, cyl.Radius)
	}
	if !pl.Normal().AsUnit().IsParallelTo(cyl.AxisDir, 1e-6) {
		return nil, fmt.Errorf("fillet: rim cap plane must be perpendicular to the cylinder axis")
	}
	return solveRim(b, e, cylF, capF, cyl, pl, r)
}

// rimFaces returns the edge's cylinder and plane faces (and their surfaces), erroring on any other pair.
func rimFaces(e *topo.Edge) (cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, cyl, pl, fmt.Errorf("fillet: rim edge bounds %d faces, need 2", len(faces))
	}
	for i := 0; i < 2; i++ {
		c, okc := faces[i].Geometry().(geom.Cylinder)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okc && okp {
			return faces[i], faces[1-i], c, p, nil
		}
	}
	return nil, nil, cyl, pl, fmt.Errorf("fillet: a rim fillet needs a cylinder and a plane face")
}

// solveRim computes the tangent circles, torus, and seam re-aim for a convex rim. The rolling-ball
// centre is r inside the cap (along −capNormal) and at radius Rc−r (inside the cylinder); the band is
// framed so angle 0 sits at the rim vertex, lining the seam up with the wall's existing seam.
func solveRim(b *topo.Body, e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, r float64) (*rimFillet, error) {
	rimV := e.StartVertex()
	capCenter := projectOntoAxis(rimV.Point(), cyl.Origin, cyl.AxisDir)
	inward := pl.Normal().AsUnit().Negate().AsVector() // into the solid, along the axis
	torusCenter := capCenter.TranslateBy(inward.Scale(r))
	majorR := cyl.Radius - r
	ref, err := math.UnitVector3FromVector(perpComponent(capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return nil, fmt.Errorf("fillet: degenerate rim frame")
	}
	probe := torusCenter.TranslateBy(ref.AsVector().Scale(majorR))
	if !PointInsideBody(b, probe) {
		return nil, fmt.Errorf("fillet: concave rim fillet (a bore lip) is not yet supported")
	}
	// The torus axis points OUTWARD along the cap normal, so the tube's v runs cyl-tangent (v=0, the
	// equator) → cap-tangent (v=π/2, toward the cap) regardless of which way the cylinder axis is stored.
	tor, err := geom.NewTorusWithRef(torusCenter, pl.Normal(), ref.AsVector(), majorR, r)
	if err != nil {
		return nil, err
	}
	seamEdge, bottomV := wallSeam(cylF, e, rimV)
	if seamEdge == nil {
		return nil, fmt.Errorf("fillet: cylinder wall has no seam edge at the rim")
	}
	return &rimFillet{
		cyl: cylF, cap: capF, rimEdge: e, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: geom.Circle{Center: torusCenter, Normal: cyl.AxisDir, RefDir: ref, Radius: cyl.Radius},
		capTan: geom.Circle{Center: capCenter, Normal: cyl.AxisDir, RefDir: ref, Radius: majorR},
		torus:  tor, r: r,
		seamMid: tor.PointAt(0, quarterTube), // perpendicular rim: contacts at v=0 (equator) and v=π/2 (cap)
	}, nil
}

// projectOntoAxis returns the point's projection onto the axis line through origin.
func projectOntoAxis(p, origin math.Point3, axis math.UnitVector3) math.Point3 {
	d := origin.VectorTo(p)
	return origin.TranslateBy(axis.AsVector().Scale(d.Dot(axis.AsVector())))
}

// perpComponent returns v with its component along axis removed.
func perpComponent(v math.Vector3, axis math.UnitVector3) math.Vector3 {
	a := axis.AsVector()
	return v.Sub(a.Scale(v.Dot(a)))
}

// wallSeam returns the cylinder face's edge meeting the rim vertex that is NOT the rim circle (its
// axial seam), and the seam's far vertex.
func wallSeam(cylF *topo.Face, rim *topo.Edge, rimV *topo.Vertex) (*topo.Edge, *topo.Vertex) {
	for _, ce := range cylF.Edges() {
		if ce == rim {
			continue
		}
		if ce.StartVertex() == rimV {
			return ce, ce.EndVertex()
		}
		if ce.EndVertex() == rimV {
			return ce, ce.StartVertex()
		}
	}
	return nil, nil
}
