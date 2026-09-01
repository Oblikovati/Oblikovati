// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Arc rim fillets — rounding a PARTIAL circular arc edge where a cylinder meets a perpendicular cap
// (the sharp arc a prior fillet leaves at a box corner). Unlike a full rim (closed circle, a torus
// band), an arc terminates at two vertices where the cylinder runs into a side plane.
//
// The blend is a constant-radius torus over the arc. WHERE that torus sits — which side of the cap, and
// at cylR−r or cylR+r — is solved from the edge's own convexity and the faces' outward normals rather
// than assumed (fillet_arc_seat.go). HOW it terminates at each end is solved too: on the end's radial
// cross-section (a flat SETBACK triangle, or nothing at all when the side face IS that radial plane and
// absorbs it), or — when that cross-section would spill through the side face — on the SPIRIC section
// the side plane cuts from the band (fillet_arc_runout.go).

// FilletCylinderArc rounds a circular ARC edge where a cylinder meets a perpendicular cap and whose ends
// run into side planes (the sharp arc a prior fillet leaves at a box corner) into a torus terminated at
// each end by a setback cap, an absorbing side face, or a run-out onto the side plane.
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
	af, err := resolveArcFillet(b, e, cylF, capF, cyl, pl, r)
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

// IsLoneCurvedAdjacentEdge reports whether the keys are a single edge that borders a CYLINDER and a
// plane — the curved edge a prior fillet leaves (a rim circle, an arc cap, a smooth tangent line, or
// an axial cut). The feature must let the kernel round these on the ANALYTIC body: re-faceting the
// cylinder first destroys the circle/arc the toroidal-band / torus + setback-cap path needs AND the
// G1 classification, so a smooth edge would planarize into the misleading "invalid solid" instead of
// being rejected cleanly. FilletEdges then routes it (rim→band, arc→torus+caps, smooth→reject).
func IsLoneCurvedAdjacentEdge(body *topo.Body, keys [][]byte) bool {
	if len(keys) != 1 {
		return false
	}
	e, ok := body.FindEdgeByKey(keys[0])
	if !ok {
		return false
	}
	_, _, isCylPlane := cylinderPlaneEdge(e)
	return isCylPlane
}

// arcFillet is the solved geometry + the topology an arc fillet replaces and inserts.
type arcFillet struct {
	arcEdge     *topo.Edge
	capF, cylF  *topo.Face
	torusCenter math.Point3
	capCenter   math.Point3
	axisN       math.UnitVector3 // the torus axis: from the torus centre TOWARD the cap plane
	cylR        float64
	majorR, r   float64
	vCyl        float64 // tube angle of the CYLINDER contact: 0 at the cylR−r seat, π at cylR+r
	torus       geom.Torus
	ends        [2]arcEnd
}

// arcEnd is one terminating end of the arc: the surviving corner vertex, the cyl∩side smooth line
// (split where the torus meets it), the side face, the contact azimuths, and the cap-/cyl-tangent points
// the band terminates on. runout is non-nil when the band is terminated ON the side plane instead of on
// this end's radial cross-section (fillet_arc_runout.go), in which case uCyl and uCap differ.
type arcEnd struct {
	rimV       *topo.Vertex
	smoothLine *topo.Edge
	sideF      *topo.Face
	bottomV    *topo.Vertex
	refDir     math.UnitVector3
	uCyl, uCap float64
	uAnchor    float64     // this end's own azimuth, unwrapped along the arc's SPAN (see arcFillet.anchorEnds)
	vc, vt     math.Point3 // cyl-tangent (on the cylinder, receded), cap-tangent (on the cap)
	runout     *geom.SpiricArc
}

// resolveArcFillet validates a cylinder/cap ARC edge whose ends run into side planes, seats the rolling
// ball on the side its convexity demands, and solves the band + its two terminations.
func resolveArcFillet(b *topo.Body, e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane,
	r float64) (*arcFillet, error) {
	if err := checkArcFilletInputs(cyl, pl, r); err != nil {
		return nil, err
	}
	af := &arcFillet{arcEdge: e, capF: capF, cylF: cylF, cylR: cyl.Radius, r: r,
		capCenter: projectOntoAxis(e.StartVertex().Point(), cyl.Origin, cyl.AxisDir)}
	refMid, err := arcMidRadial(e, af.capCenter, cyl.AxisDir)
	if err != nil {
		return nil, err
	}
	seat, err := solveArcBallSeat(b, e, capF, cyl, pl, af.capCenter, refMid, r)
	if err != nil {
		return nil, err
	}
	if err := af.seatBand(seat, refMid, cyl); err != nil {
		return nil, err
	}
	af.anchorEnds(refMid)
	af.resolveTerminal(0)
	af.resolveTerminal(1)
	return af, nil
}

// checkArcFilletInputs rejects a non-positive radius and a cap plane that is not perpendicular to the
// cylinder axis. The old blanket `r >= cyl.Radius` rejection is NOT here: it is a CONVEX-tier fact (only
// the cylR−r seat needs r < cylR) and now lives in arcBallSeats, which simply does not offer a seat with
// a non-positive majorR.
func checkArcFilletInputs(cyl geom.Cylinder, pl geom.Plane, r float64) error {
	if r <= 0 {
		return fmt.Errorf("fillet: arc radius %g must be > 0 (cylinder radius %g)", r, cyl.Radius)
	}
	if !pl.Normal().AsUnit().IsParallelTo(cyl.AxisDir, 1e-6) {
		return fmt.Errorf("fillet: arc cap plane must be perpendicular to the cylinder axis")
	}
	return nil
}

// seatBand places the torus on the solved seat and solves both ends' side faces. The torus axis points
// from the band's centre TOWARD the cap plane, so the cap contact is at v = π/2 on either seat and the
// cylinder contact falls at v = 0 (cylR−r) or v = π (cylR+r).
func (af *arcFillet) seatBand(seat arcBallSeat, refMid math.UnitVector3, cyl geom.Cylinder) error {
	af.majorR = seat.majorR
	af.torusCenter = af.capCenter.TranslateBy(seat.capSide.Scale(af.r))
	axis, err := math.UnitVector3FromVector(seat.capSide.Negate())
	if err != nil {
		return fmt.Errorf("fillet: degenerate arc cap normal %v", seat.capSide)
	}
	af.axisN = axis
	af.vCyl = 0
	if seat.majorR > cyl.Radius {
		af.vCyl = stdmath.Pi
	}
	for i, v := range []*topo.Vertex{af.arcEdge.StartVertex(), af.arcEdge.EndVertex()} {
		end, err := af.solveEnd(v, cyl)
		if err != nil {
			return err
		}
		af.ends[i] = end
	}
	// The band's angle-zero reference is the arc midpoint's OPPOSITE radial, so the torus's periodic u
	// seam sits diametrically away from the band. Framing on an END (solveRim's convention, which lines a
	// closed band's seam up with the wall's own) puts the seam ON the band boundary, and the face mesher
	// charts in ParamAt's [0, 2π): one boundary sample a single ulp the wrong side of it wraps a whole
	// turn and the chart polygon claims the rest of the torus — measured on simple/W2 as a 0.418 band
	// meshing to 4.9146. An arc band has no seam edge to line up with, so it is free to move.
	tor, err := geom.NewTorusWithRef(af.torusCenter, axis.AsVector(), refMid.AsVector().Negate(), af.majorR, af.r)
	af.torus = tor
	return err
}

// anchorEnds fixes each end's azimuth on the SPAN THE ARC ACTUALLY COVERS. atan2 alone cannot: it returns
// (−π, π], so a band wider than a half turn (simple/H6's 270° revolve) reads its far end as the SHORT way
// round and charts a third of the band it should. The arc's own midpoint settles it — the far end sits
// twice as far along as the midpoint — and every azimuth of that end is then unwrapped to its anchor.
func (af *arcFillet) anchorEnds(refMid math.UnitVector3) {
	u0 := af.torusAzimuth(af.ends[0].refDir) // 0 by construction: the torus Ref IS end 0's radial
	uMid := unwrapNear(af.torusAzimuth(refMid), u0)
	af.ends[0].uAnchor, af.ends[1].uAnchor = u0, u0+2*(uMid-u0)
}

// unwrapNear returns a shifted by whole turns into (ref−π, ref+π].
func unwrapNear(a, ref float64) float64 {
	return ref + stdmath.Remainder(a-ref, 2*stdmath.Pi)
}

// solveEnd identifies the end's side face + smooth line and its radial direction. The arc ends where the
// cylinder runs into a side plane; that vertex carries the arc, a cap∩side edge, and the cyl∩side smooth
// line — the smooth line is the one bordering the cylinder and a NON-cap plane. The contact points
// themselves are fixed later, by resolveTerminal, because a run-out end does not put them on this radial.
func (af *arcFillet) solveEnd(rimV *topo.Vertex, cyl geom.Cylinder) (arcEnd, error) {
	smooth, sideF := cylSideEdgeAt(rimV, af.arcEdge, af.capF)
	if smooth == nil {
		return arcEnd{}, fmt.Errorf("fillet: arc end is not a cylinder/side tangent vertex")
	}
	ref, err := math.UnitVector3FromVector(perpComponent(af.capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return arcEnd{}, fmt.Errorf("fillet: degenerate arc end frame")
	}
	bottom := smooth.StartVertex()
	if bottom == rimV {
		bottom = smooth.EndVertex()
	}
	return arcEnd{rimV: rimV, smoothLine: smooth, sideF: sideF, bottomV: bottom, refDir: ref}, nil
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
