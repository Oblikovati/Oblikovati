// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// solveAsymmetricCorner treats a shared corner whose picked edges carry DIFFERENT radii — the case the
// equal-radius sphere/mirror-seam cannot express. A 2-edge corner sharing a face becomes an asymmetric
// miter (the true cyl∩cyl seam of two unequal rolling-ball cylinders, P9/V9); a trihedral corner becomes
// a torus corner patch (A4). A variable-radius edge at a mixed corner is rejected (its faceted end chords
// cannot meet a corner watertight — the same reason the equal-radius miter rejects a varying pick).
func solveAsymmetricCorner(_ *topo.Body, vid uint64, ps []filletPick) (*cornerBlend, *cornerMiter, error) {
	if p := varyingPick(ps); p != nil {
		return nil, nil, fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a mixed-radius corner", p.r0, p.r1)
	}
	v := vertexByID(edgesOf(ps), vid)
	faces := facesAtVertex(v)
	switch {
	case len(ps) == 2 && closedRimPick(ps):
		return nil, nil, nil // a CLOSED rim counted twice is not a corner (see closedRimPick).
	case len(ps) == 2:
		cm, err := solveAsymmetricMiter(v, vid, ps)
		return nil, cm, err
	case len(ps) == 3 && len(faces) == 3:
		// A4 (r10/r5/r5): OCCT builds a TORUS corner patch tangent to the large-radius arm along its
		// outer equator and to the small-radius arms along tube circles — not a sphere. The orthogonal
		// [rB, rS, rS] pattern is solved exactly (fillet_corner_radiustorus.go); anything else (three
		// distinct radii, the big pair sharing the top, a skewed corner) still declines with the radii.
		if cb, ok := solveRadiusTorusCorner(vertexByID(edgesOf(ps), vid), vid, ps); ok {
			return cb, nil, nil
		}
		return nil, nil, fmt.Errorf("fillet: mixed-radius trihedral corner (%d faces, radii %v) needs a torus corner patch — not yet supported", len(faces), cornerRadiiAt(vid, ps))
	default:
		return nil, nil, fmt.Errorf("fillet: mixed-radius corner where %d filleted edges meet a %d-face vertex is not a supported blend (need 2 edges sharing a face, or 3 edges at a trihedral vertex)", len(ps), len(faces))
	}
}

// cornerRadiiAt collects each pick's radius at the shared corner vid, for diagnostics.
func cornerRadiiAt(vid uint64, ps []filletPick) []float64 {
	radii := make([]float64, len(ps))
	for i, p := range ps {
		radii[i] = radiusAtVertex(p, vid)
	}
	return radii
}

// solveAsymmetricMiter builds the seam where two filleted edges of DIFFERENT radii share a face and
// meet at v (P9/V9: a box top corner, r1 on one arm, r0.5 on the other). The equal-radius miter takes a
// shortcut — it samples cyl1 ∩ (the nF1−nF2 mirror plane), valid only because equal cylinders coincide
// on that plane. Unequal cylinders do not, so the asymmetric seam is the TRUE intersection cyl0 ∩ cyl1
// of the two rolling-ball cylinders (both tangent to the shared plane). Every sampled point lies on BOTH
// cylinders, so the two arm faces weld along it watertight, matching OCCT's 8-face P9 solid (area
// 146.393). Curved-contact miters (families B/C) keep their own equal-radius path; this is planar-only.
func solveAsymmetricMiter(v *topo.Vertex, vid uint64, ps []filletPick) (*cornerMiter, error) {
	shared := sharedFace(ps[0].edge, ps[1].edge)
	if shared == nil {
		return nil, fmt.Errorf("fillet: two filleted edges of different radii meeting at a vertex must share a face to miter (none shared)")
	}
	if miterHasCurvedContact(ps, shared) {
		return nil, fmt.Errorf("fillet: an asymmetric-radius miter with a curved contact face is not supported")
	}
	nS, ok := planeNormal(shared)
	if !ok {
		return nil, fmt.Errorf("fillet: asymmetric miter corner's shared face must be planar")
	}
	c0, err := asymMiterCyl(ps[0].edge, shared, nS, v, radiusAtVertex(ps[0], vid))
	if err != nil {
		return nil, err
	}
	c1, err := asymMiterCyl(ps[1].edge, shared, nS, v, radiusAtVertex(ps[1], vid))
	if err != nil {
		return nil, err
	}
	seam, err := sampleAsymmetricMiterSeam(nS, c0, c1)
	if err != nil {
		return nil, err
	}
	return &cornerMiter{vertex: v, shared: shared, sBot: seam[len(seam)-1], seam: seam}, nil
}

// asymMiterCyl is one arm's rolling-ball cylinder at the corner: axis point cen (v offset into the solid
// by the shared+outer face pair), unit axis along the edge, outer-face outward normal nF, and radius r.
// It reuses miterArmOf's exact frame so an asymmetric arm and an equal-radius arm are the same cylinder.
func asymMiterCyl(e *topo.Edge, shared *topo.Face, nS math.Vector3, v *topo.Vertex, r float64) (miterCyl, error) {
	a, err := miterArmOf(e, shared, nS, v, r)
	if err != nil {
		return miterCyl{}, err
	}
	return miterCyl{cen: a.cen, axis: a.axis, nF: a.nF, r: r}, nil
}

// miterCyl is one arm's rolling-ball cylinder for the asymmetric miter: a point on the axis, the unit
// axis direction, the outer face outward normal, and the radius.
type miterCyl struct {
	cen  math.Point3
	axis math.Vector3
	nF   math.Vector3
	r    float64
}

// sampleAsymmetricMiterSeam samples the true intersection cyl0 ∩ cyl1 from sTop (both fillets meet the
// shared plane) to sBot (the seam runs out where the tighter fillet reaches its own outer face). It walks
// cyl0's rolling-ball contact direction from sTop to sBot and, at each station, solves the axial position
// that places the point on cyl1 — so every returned point lies on BOTH cylinders exactly.
func sampleAsymmetricMiterSeam(nS math.Vector3, c0, c1 miterCyl) ([]math.Point3, error) {
	sTop, sBot, err := asymMiterSeamEnds(nS, c0, c1)
	if err != nil {
		return nil, err
	}
	dTop, dBot := miterContactDir(c0, sTop), miterContactDir(c0, sBot)
	k := seamChordCount(dTop, dBot)
	prev := c0.cen.VectorTo(sTop).Dot(c0.axis) // axial position of sTop on cyl0's axis (continuity seed)
	out := make([]math.Point3, k+1)
	out[0], out[k] = sTop, sBot
	for j := 1; j < k; j++ {
		d := slerpVec(dTop, dBot, float64(j)/float64(k))
		p, lambda, ok := cylContactPointOnCyl(c0, d, c1, prev)
		if !ok {
			return nil, fmt.Errorf("fillet: asymmetric miter seam station %d has no point on both cylinders", j)
		}
		out[j], prev = p, lambda
	}
	return out, nil
}

// asymMiterSeamEnds resolves the seam's two ends: sTop, where both fillets meet the shared plane (the
// intersection of their shared-face tangent lines), and sBot, where the tighter fillet runs out on its
// own outer face (asymMiterTerminus).
func asymMiterSeamEnds(nS math.Vector3, c0, c1 miterCyl) (math.Point3, math.Point3, error) {
	sTop, ok := lineLineIntersect(c0.cen.TranslateBy(nS.Scale(c0.r)), c0.axis, c1.cen.TranslateBy(nS.Scale(c1.r)), c1.axis)
	if !ok {
		return math.Point3{}, math.Point3{}, fmt.Errorf("fillet: asymmetric miter tangent lines do not meet (parallel edges)")
	}
	sBot, ok := asymMiterTerminus(nS, c0, c1)
	if !ok {
		return math.Point3{}, math.Point3{}, fmt.Errorf("fillet: asymmetric miter seam has no valid run-out terminus")
	}
	return sTop, sBot, nil
}

// seamChordCount picks the chord count spanning the seam's contact-direction arc (from dTop to dBot),
// matched to filletChordsPerTurn, with a floor of 4 so a short seam still reads as an arc.
func seamChordCount(dTop, dBot math.Vector3) int {
	w := stdmath.Acos(math.Clamp(dTop.Dot(dBot), -1, 1))
	k := int(stdmath.Ceil(w / (2 * stdmath.Pi / filletChordsPerTurn)))
	if k < 4 {
		return 4
	}
	return k
}

// asymMiterTerminus finds sBot: the seam runs out where ONE fillet first reaches its outer face. Each arm
// contributes a candidate — the point where that arm's outer-face tangent line pierces the OTHER cylinder
// — and the true terminus is the candidate that keeps the other arm's contact direction inside its
// rolling-ball wedge (nS→nF). The tighter (smaller-wedge) fillet runs out first, so exactly one holds.
func asymMiterTerminus(nS math.Vector3, c0, c1 miterCyl) (math.Point3, bool) {
	if q, ok := outerTangentOnCyl(c1, c0); ok && inMiterWedge(miterContactDir(c0, q), nS, c0.nF) {
		return q, true // c1 reaches its outer face first; c0 still inside its wedge
	}
	if q, ok := outerTangentOnCyl(c0, c1); ok && inMiterWedge(miterContactDir(c1, q), nS, c1.nF) {
		return q, true // c0 reaches its outer face first; c1 still inside its wedge
	}
	return math.Point3{}, false
}

// outerTangentOnCyl returns the point where arm a's tangent line to its own outer face (the ruling where
// cyl a touches nF) pierces cyl b, choosing the root nearer a's shared-face tangent point (the corner
// side, not the far crossing). The returned point lies on both cylinders — a seam-terminus candidate.
func outerTangentOnCyl(a, b miterCyl) (math.Point3, bool) {
	lp := a.cen.TranslateBy(a.nF.Scale(a.r)) // a's ruling of contact with its outer face
	roots := cylLineRoots(b, lp, a.axis)
	if len(roots) == 0 {
		return math.Point3{}, false
	}
	ref := a.cen // corner-side reference: the axis point, nearest the shared vertex
	best, bestD := lp.TranslateBy(a.axis.Scale(roots[0])), stdmath.Inf(1)
	for _, t := range roots {
		p := lp.TranslateBy(a.axis.Scale(t))
		if d := p.DistanceSquaredTo(ref); d < bestD {
			best, bestD = p, d
		}
	}
	return best, true
}

// miterContactDir is the unit rolling-ball contact direction of cylinder c at a point p on its surface:
// the outward radial from the axis to p.
func miterContactDir(c miterCyl, p math.Point3) math.Vector3 {
	w := c.cen.VectorTo(p)
	radial := w.Sub(c.axis.Scale(w.Dot(c.axis)))
	u, err := math.UnitVector3FromVector(radial)
	if err != nil {
		return radial
	}
	return u.AsVector()
}

// cylContactPointOnCyl places the point that sits on cyl c0 at contact direction d AND on cyl c1. The
// point family is base + λ·axis0 (base = c0.cen + r0·d, on cyl0 for every λ); requiring it on cyl1 is a
// quadratic in λ, (1−cc²)λ² + 2[(w0·u0)−cc(w0·u1)]λ + [|w0|²−(w0·u1)²−r1²] = 0 with cc = u0·u1. The root
// nearest prev keeps the seam on one continuous branch. Returns the point and its λ.
func cylContactPointOnCyl(c0 miterCyl, d math.Vector3, c1 miterCyl, prev float64) (math.Point3, float64, bool) {
	base := c0.cen.TranslateBy(d.Scale(c0.r))
	w0 := c1.cen.VectorTo(base)
	u0, u1 := c0.axis, c1.axis
	cc := u0.Dot(u1)
	a := 1 - cc*cc
	b := 2 * (w0.Dot(u0) - cc*w0.Dot(u1))
	cee := w0.LengthSquared() - w0.Dot(u1)*w0.Dot(u1) - c1.r*c1.r
	lambda, ok := nearestSeamRoot(a, b, cee, prev)
	if !ok {
		return math.Point3{}, 0, false
	}
	return base.TranslateBy(u0.Scale(lambda)), lambda, true
}

// cylLineRoots returns the real parameters t at which the line lp+t·ld meets cylinder c (dist-to-axis =
// r). It is the same quadratic as cylContactPointOnCyl seen from the line, kept separate so the terminus
// search reads naturally.
func cylLineRoots(c miterCyl, lp math.Point3, ld math.Vector3) []float64 {
	e := c.cen.VectorTo(lp)
	la := ld.Dot(c.axis)
	ea := e.Dot(c.axis)
	a := ld.LengthSquared() - la*la
	b := 2 * (e.Dot(ld) - ea*la)
	cee := e.LengthSquared() - ea*ea - c.r*c.r
	return quadraticRoots(a, b, cee)
}

// inMiterWedge reports whether the unit direction d lies in the rolling-ball wedge spanned from the
// shared-face normal nS to the outer-face normal nF — i.e. d = α·nS + β·nF with α,β ≥ 0 (the three are
// coplanar, both ⊥ the arm axis). A terminus is valid only when the OTHER arm's contact is still in-wedge.
func inMiterWedge(d, nS, nF math.Vector3) bool {
	// Solve the 2×2 Gram system [nS·nS nS·nF; nF·nS nF·nF][α;β] = [d·nS; d·nF].
	a, bdot, c := nS.Dot(nS), nS.Dot(nF), nF.Dot(nF)
	det := a*c - bdot*bdot
	if stdmath.Abs(det) < 1e-12 {
		return false
	}
	rs, rf := d.Dot(nS), d.Dot(nF)
	alpha := (c*rs - bdot*rf) / det
	beta := (a*rf - bdot*rs) / det
	const wedgeTol = -1e-9
	return alpha >= wedgeTol && beta >= wedgeTol
}

// lineLineIntersect returns the intersection of two coplanar, non-parallel lines p0+s·d0 and p1+t·d1.
// s = ((p1−p0)×d1)·(d0×d1)/|d0×d1|². ok is false when the directions are parallel (no unique meet).
func lineLineIntersect(p0 math.Point3, d0 math.Vector3, p1 math.Point3, d1 math.Vector3) (math.Point3, bool) {
	cross := d0.Cross(d1)
	den := cross.LengthSquared()
	if den < 1e-18 {
		return math.Point3{}, false
	}
	s := p0.VectorTo(p1).Cross(d1).Dot(cross) / den
	return p0.TranslateBy(d0.Scale(s)), true
}

// quadraticRoots returns the real roots of a·x²+b·x+c (0, 1, or 2), degrading to the linear root when
// a≈0. Used by the asymmetric-miter cylinder/line intersections.
func quadraticRoots(a, b, c float64) []float64 {
	if stdmath.Abs(a) < 1e-14 {
		if stdmath.Abs(b) < 1e-14 {
			return nil
		}
		return []float64{-c / b}
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return nil
	}
	sq := stdmath.Sqrt(disc)
	return []float64{(-b + sq) / (2 * a), (-b - sq) / (2 * a)}
}

// nearestSeamRoot returns the real root of a·x²+b·x+c closest to ref (for branch continuity along
// the seam). ok is false when there is no real root.
func nearestSeamRoot(a, b, c, ref float64) (float64, bool) {
	roots := quadraticRoots(a, b, c)
	if len(roots) == 0 {
		return 0, false
	}
	best, bestD := roots[0], stdmath.Abs(roots[0]-ref)
	for _, r := range roots[1:] {
		if d := stdmath.Abs(r - ref); d < bestD {
			best, bestD = r, d
		}
	}
	return best, true
}
