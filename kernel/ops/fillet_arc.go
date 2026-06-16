// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Arc rim fillets — rounding a PARTIAL circular arc edge where a cylinder meets a perpendicular cap
// (the sharp arc a prior fillet leaves at a box corner). Unlike a full rim (closed circle, a torus
// band), an arc terminates at two vertices where the cylinder runs G1-tangent into a side plane.
// The blend is a constant-radius torus over the arc, capped at each end by a flat SETBACK triangle in
// the radial plane (a line on the cap, a line on the side plane, the torus end-arc) — the rolling ball
// stays tangent to the side plane at the ends for any radius, so no run-out taper is needed.

// FilletCylinderArc rounds a convex circular ARC edge where a cylinder meets a perpendicular cap and
// whose ends run G1-tangent into side planes (the sharp arc a prior fillet leaves at a box corner)
// into a torus + two flat setback end-caps.
func FilletCylinderArc(b *topo.Body, arcKey []byte, r float64) (*topo.Body, error) {
	e, ok := b.FindEdgeByKey(arcKey)
	if !ok {
		return nil, fmt.Errorf("fillet: arc edge reference lost: %x", arcKey)
	}
	if _, isArc := e.Geometry().(geom.Arc3d); !isArc {
		return nil, fmt.Errorf("fillet: an arc fillet needs an Arc3d edge")
	}
	cylF, capF, cyl, pl, err := rimFaces(e)
	if err != nil {
		return nil, err
	}
	af, err := resolveArcFillet(e, cylF, capF, cyl, pl, r)
	if err != nil {
		return nil, err
	}
	return rebuildWithArcFillet(b, af)
}

// loneArcPick returns the pick when it is the sole, constant-radius selection of an arc edge between
// a cylinder and a plane — the arc-fillet trigger (a sharp arc a prior fillet left). nil otherwise.
func loneArcPick(body *topo.Body, picks []EdgeFilletRadii) *EdgeFilletRadii {
	if len(picks) != 1 || picks[0].R0 != picks[0].R1 {
		return nil
	}
	e, ok := body.FindEdgeByKey(picks[0].Key)
	if !ok {
		return nil
	}
	if _, isArc := e.Geometry().(geom.Arc3d); !isArc {
		return nil
	}
	if _, _, _, _, err := rimFaces(e); err != nil {
		return nil
	}
	return &picks[0]
}

// arcFillet is the solved geometry + the topology an arc fillet replaces and inserts.
type arcFillet struct {
	arcEdge     *topo.Edge
	capF, cylF  *topo.Face
	torusCenter math.Point3
	capCenter   math.Point3
	axisN       math.UnitVector3 // the cap's outward normal (= the torus axis)
	majorR, r   float64
	torus       geom.Torus
	ends        [2]arcEnd
}

// arcEnd is one terminating end of the arc: the surviving corner vertex, the cyl∩side smooth line
// (split where the torus meets it), the side face, and the cap-/cyl-tangent points the torus lands on.
type arcEnd struct {
	rimV       *topo.Vertex
	smoothLine *topo.Edge
	sideF      *topo.Face
	bottomV    *topo.Vertex
	refDir     math.UnitVector3
	vc, vt     math.Point3 // cyl-tangent (on the cylinder, receded), cap-tangent (on the cap)
}

// resolveArcFillet validates a convex cylinder/cap ARC edge whose ends run into side planes and solves
// the torus + setback geometry.
func resolveArcFillet(e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, r float64) (*arcFillet, error) {
	if r <= 0 || r >= cyl.Radius {
		return nil, fmt.Errorf("fillet: arc radius %g must be in (0, cylinder radius %g)", r, cyl.Radius)
	}
	if !pl.Normal().AsUnit().IsParallelTo(cyl.AxisDir, 1e-6) {
		return nil, fmt.Errorf("fillet: arc cap plane must be perpendicular to the cylinder axis")
	}
	af := &arcFillet{
		arcEdge: e, capF: capF, cylF: cylF, axisN: pl.Normal().AsUnit(),
		capCenter: projectOntoAxis(e.StartVertex().Point(), cyl.Origin, cyl.AxisDir),
		majorR:    cyl.Radius - r, r: r,
	}
	af.torusCenter = af.capCenter.TranslateBy(af.axisN.Negate().AsVector().Scale(r)) // r into the solid along −cap normal
	for i, v := range []*topo.Vertex{e.StartVertex(), e.EndVertex()} {
		end, err := af.solveEnd(v, cyl)
		if err != nil {
			return nil, err
		}
		af.ends[i] = end
	}
	tor, err := geom.NewTorusWithRef(af.torusCenter, af.axisN.AsVector(), af.ends[0].refDir.AsVector(), af.majorR, r)
	if err != nil {
		return nil, err
	}
	af.torus = tor
	return af, nil
}

// solveEnd identifies the end's side face + smooth line and computes the tangent points. The arc ends
// where the cylinder is G1-tangent to a side plane; that vertex carries the arc, a cap∩side edge, and
// the cyl∩side smooth line — the smooth line is the one bordering the cylinder and a NON-cap plane.
func (af *arcFillet) solveEnd(rimV *topo.Vertex, cyl geom.Cylinder) (arcEnd, error) {
	smooth, sideF := cylSideEdgeAt(rimV, af.arcEdge, af.capF)
	if smooth == nil {
		return arcEnd{}, fmt.Errorf("fillet: arc end is not a cylinder/side tangent vertex")
	}
	ref, err := math.UnitVector3FromVector(perpComponent(af.capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return arcEnd{}, fmt.Errorf("fillet: degenerate arc end frame")
	}
	vc := af.torusCenter.TranslateBy(ref.AsVector().Scale(cyl.Radius)) // cyl-tangent: radius Rc, receded axially
	vt := af.capCenter.TranslateBy(ref.AsVector().Scale(af.majorR))    // cap-tangent: radius Rc−r at the cap
	bottom := smooth.StartVertex()
	if bottom == rimV {
		bottom = smooth.EndVertex()
	}
	return arcEnd{rimV: rimV, smoothLine: smooth, sideF: sideF, bottomV: bottom, refDir: ref, vc: vc, vt: vt}, nil
}

// cylSideEdgeAt returns the edge at v bordering the cylinder and a plane OTHER than the cap (the
// cyl∩side smooth tangent line), and that side plane.
func cylSideEdgeAt(v *topo.Vertex, arc *topo.Edge, capF *topo.Face) (*topo.Edge, *topo.Face) {
	for _, e := range v.Edges() {
		if e == arc {
			continue
		}
		var cyl, side *topo.Face
		for _, f := range e.Faces() {
			switch f.Geometry().(type) {
			case geom.Cylinder:
				cyl = f
			case geom.Plane:
				if f != capF {
					side = f
				}
			}
		}
		if cyl != nil && side != nil {
			return e, side
		}
	}
	return nil, nil
}
