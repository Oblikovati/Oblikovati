// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// errConvexRimProbeFailed marks solveRim declining the CONVEX R−r tier: its SIGNED seat probe
// (rimBallCapSeat, fillet_rim_cap_side.go) could not verify a ball centre on the side the picked edge's
// convexity requires, or verified one on the VOID side, which is the R+r mirror's business and not this
// tier's. resolveRim catches it via errors.Is and retries with the cap's TRUE (Reversed-aware) outward
// normal via rimWithCapOrientation — first at the SAME R−r (concave=false), then at the CONCAVE R+r
// mirror (concave=true) — instead of rejecting outright. Live corpus rims that take that route: simple/R8
// and simple/W9 (concave boss roots whose R+r cove spills, so the convex round is where they still
// build) and simple/K1 (a bore lip no R−r seat can hold).
var errConvexRimProbeFailed = errors.New("fillet: convex rim material-side probe (R−r) landed outside the body")

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

// FilletCylinderRim rounds a circular rim — a closed circle edge where a cylinder face meets a
// perpendicular planar cap — into a toroidal band. It is the closed-rim curved-adjacent fillet, the
// case that terminates cleanly (a full circle, no run-out corners). Two mirror-image shapes share this
// one entry point: a CONVEX rim (a boss/cylinder top, a counterbore lip — the rolling ball at R−r) and a
// CONCAVE rim (a bore lip, a hole meeting its plate cap — the rolling ball at R+r, resolveRim's fallback
// when the R−r probe lands in the hole void). The body is rebuilt with the rim replaced by the
// cap-tangent and cyl-tangent circles, the neighbour cap and cylinder receded to them, and the torus
// between; every other face is copied verbatim. Non-perpendicular caps and r ≥ radius are rejected.
func FilletCylinderRim(b *topo.Body, rimKey []byte, r float64) (*topo.Body, error) {
	rim, err := resolveRim(b, rimKey, r)
	if err != nil {
		return nil, err
	}
	if rim.concave {
		return rebuildWithConcaveRimFillet(b, rim)
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
	// cylTan/capTan are the two CLOSED contact rails the band is bounded by: where it meets the curved
	// host and where it meets the cap. They are geom.Circle for every analytic-torus rim (the cylinder,
	// cone and sphere hosts), and a closed interpolating BSpline for the ELLIPTIC rim, whose rolling-ball
	// contact loci are non-analytic (fillet_elliptic_rim_canal.go) — hence geom.Curve3, not geom.Circle.
	// The rebuild only ever reads PointAt(0) (the seam frame) and hands the curve to AddEdge verbatim, so
	// widening the type leaves every analytic rim byte-identical.
	cylTan geom.Curve3
	capTan geom.Curve3
	// band is the fillet surface between the two rails: a geom.Torus on every analytic rim, a canal
	// geom.BSplineSurface on the elliptic rim. Widened for the same reason as the rails.
	band geom.Surface
	r    float64
	// concave marks the CONCAVE bore-lip mirror (solveConcaveRim, R+r) so FilletCylinderRim dispatches
	// the rebuild to rebuildWithConcaveRimFillet (the reversed cove-band winding) instead of the CONVEX
	// rebuildWithRimFillet. Zero value (false) for every solveRim result, so the convex path never
	// changes behaviour.
	concave bool
	// seamMid is the on-arc midpoint of the tube seam joining cylTan.PointAt(0) → capTan.PointAt(0)
	// (the point the seam Arc3dByThreePoints passes through). A perpendicular CONVEX cylinder rim's
	// contacts sit at tube v=0 and v=π/2, so it is torus.PointAt(0, π/4) (quarterTube); its CONCAVE
	// mirror's contacts sit at v=π and v=π/2 instead (NewTorusWithRef's Major+Minor·cos v term hits the
	// wall radius R at v=π when major=R+r, not v=0), so it is torus.PointAt(0, 3π/4) (threeQuarterTube);
	// a CONE rim's contacts sit at other tube angles, so the closed-rim cone solver supplies the true
	// between-contacts midpoint.
	seamMid math.Point3
}

// resolveRim validates the picked edge is a cylinder/cap rim and solves the fillet geometry, trying three
// tiers in order and stopping at the first whose seat is verified against the solid:
//  1. solveRim — the CONVEX R−r rim, seated on the side its SIGNED probe verifies (fillet_rim_cap_side.go)
//     rather than on the stored cap normal. It takes I9, U6, U7, complex/B2 (un-Reversed caps, the seat
//     the stored normal happened to agree with) and simple/Z1 (a plain convex rim stored bottom-up, which
//     before the guard failed here and was rebuilt identically by tier 2).
//  2. rimWithCapOrientation(..., R−r, concave=false) — the SAME convex R−r geometry, resolved through the
//     cap's TRUE Reversed-aware outward normal. It now takes the CONCAVE rims tier 1 declines because
//     their ball sits in the material where their convexity says void, yet whose R+r cove SPILLS
//     (simple/R8, simple/W9 — the deep #2012 boss-root weld); the geometry it builds for them is the same
//     convex round tier 1 used to build.
//  3. rimWithCapOrientation(..., R+r, concave=true) — the CONCAVE bore-lip mirror (K1: the plate material
//     is genuinely OUTSIDE the bore, so no R−r seat on either side of the cap holds the ball).
//
// If all three fail, the rim needs more than either mirror (e.g. a non-perpendicular cap) and tier 3's
// honest reject is returned.
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
	if rf, handled, err := concaveBossRim(e, cylF, capF, cyl, r); handled {
		return rf, err
	}
	rf, err := solveRim(b, e, cylF, capF, cyl, pl, r)
	if !errors.Is(err, errConvexRimProbeFailed) {
		return rf, err // success, or a hard error unrelated to the material-side probe
	}
	if rf, err2 := rimWithCapOrientation(b, e, cylF, capF, cyl, pl, r, cyl.Radius-r, false); err2 == nil {
		return rf, nil
	}
	return rimWithCapOrientation(b, e, cylF, capF, cyl, pl, r, cyl.Radius+r, true)
}

// rimFaces returns the edge's cylinder and plane faces (and their surfaces), erroring on any other pair.
func rimFaces(e *topo.Edge) (cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, cyl, pl, fmt.Errorf("fillet: rim edge bounds %d faces, need 2", len(faces))
	}
	for i := range 2 {
		c, okc := faces[i].Geometry().(geom.Cylinder)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okc && okp {
			return faces[i], faces[1-i], c, p, nil
		}
	}
	return nil, nil, cyl, pl, fmt.Errorf("fillet: a rim fillet needs a cylinder and a plane face")
}

// solveRim computes the tangent circles, torus, and seam re-aim for a convex rim. The rolling-ball
// centre is r off the cap on the side the SIGNED seat probe verifies (rimBallCapSeat — never the stored
// plane normal, see fillet_rim_cap_side.go) and at radius Rc−r (inside the cylinder); the band is framed
// so angle 0 sits at the rim vertex, lining the seam up with the wall's existing seam.
func solveRim(b *topo.Body, e *topo.Edge, cylF, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane, r float64) (*rimFillet, error) {
	rimV := e.StartVertex()
	capCenter := projectOntoAxis(rimV.Point(), cyl.Origin, cyl.AxisDir)
	majorR := cyl.Radius - r
	ref, err := math.UnitVector3FromVector(perpComponent(capCenter.VectorTo(rimV.Point()), cyl.AxisDir))
	if err != nil {
		return nil, fmt.Errorf("fillet: degenerate rim frame")
	}
	seat, err := rimBallCapSeat(b, e, capF, pl, capCenter, ref, majorR, r)
	if err != nil {
		return nil, err // errConvexRimProbeFailed-wrapped: resolveRim's cap-orientation ladder takes it
	}
	if !seat.inMaterial {
		// Tier 1 is the CONVEX tier: R−r and the quarterTube seam are the convex derivation, so a seat the
		// probe verifies on the VOID side belongs to the ladder's R+r mirror, not here. This subsumes the
		// old unsigned PointInsideBody probe exactly — it asked the SAME point of the SAME tessellation,
		// and rimBallCapSeat has already established the point's side — at one tessellation instead of two.
		return nil, errConvexRimProbeFailed // resolveRim falls back to the CONCAVE R+r mirror
	}
	torusCenter := capCenter.TranslateBy(seat.toCentre.Scale(r))
	// The torus axis points OUTWARD along the cap normal, so the tube's v runs cyl-tangent (v=0, the
	// equator) → cap-tangent (v=π/2, toward the cap) regardless of which way the cylinder axis is stored.
	tor, err := geom.NewTorusWithRef(torusCenter, seat.bandAxis, ref.AsVector(), majorR, r)
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
		band:   tor, r: r,
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
