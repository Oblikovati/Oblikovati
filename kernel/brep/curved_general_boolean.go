// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General curved∩curved boolean (EPIC Oblikovati/Oblikovati#1403). The bespoke per-pair handlers build the
// result by hand from the SSI imprint loops (a band + caps stitched case by case). This is the GENERAL
// pipeline the half-space/ruled path already proves in miniature, run on a real pair: trim EACH operand's
// curved side by their shared surface-surface-intersection imprint (the same imprint-general
// trimByImprint, #1405), classify kept cells by 3D SOLID MEMBERSHIP (keep(op, isB, insideSolid(other,·)) —
// the same predicate the planar boolean uses) instead of a plane half-space, then weld both kept sides with
// the shared general curvedStitch. No per-pair loop→body constructor.
//
// First migrated pair: cone ∩ cone (the smallest bespoke handler; both operands ruled, so they reuse the
// existing ruledUV plumbing — only the material predicate is new). The bespoke ConeConeIntersect remains as
// a fallback for the configurations this path declines (a full cone reaching its apex, a non-crossing).

// ruledSolidMaterial wraps a ruled side's 3D solid-membership test as a uvSide materialOf builder: a closure
// (not a bound method) so it reads the receiver AFTER trimByImprint shifts the seam — point3 then maps the
// seam-relative (u,v) to the right surface point (#1403, mirroring ruledMaterial).
func ruledSolidMaterial(c *ruledUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool { return c.keptBySolid(uv.X, uv.Y) }
	}
}

// keptBySolid reports whether the band point at (u,v) survives the operation: keep(op, isB, inside) where
// inside is 3D membership of the surface point in the OTHER solid — the general curved∩curved classification
// (#1403). For A∩B this keeps each side's part that lies inside the other solid.
func (c ruledUV) keptBySolid(u, v float64) bool {
	return keep(c.solidOp, c.solidIsB, c.insideOther(c.point3(u, v)))
}

// keepsInsideOther reports whether this side keeps the material INSIDE the other solid (a lens/lobe region)
// rather than OUTSIDE it (a band). It reads the operation's keep table directly — keep-inside iff a point
// inside the other solid survives but one outside does not. For the pinched Steinmetz case this is what tells
// the two lobes (intersect, and a cut's reversed tool bite) apart from the two wrapping bands (cut target,
// join): the lobes must not fire the wrapping-band emission, the bands must (#1403).
func (c ruledUV) keepsInsideOther() bool {
	return keep(c.solidOp, c.solidIsB, true) && !keep(c.solidOp, c.solidIsB, false)
}

// anyKeptVSolid reports whether SOME axial-distance v in the band is kept at azimuth u — the general
// (solid-membership) analogue of keptV's non-empty test, used by wrapsAllU to pick the band orientation
// convention. It samples v because solid membership has no closed-form interval like the linear plane case.
func (c ruledUV) anyKeptVSolid(u float64) bool {
	const n = 96 // dense enough to catch a thin lens where the imprint barely bites the band
	for j := 0; j <= n; j++ {
		v := c.band.vMin + (c.band.vMax-c.band.vMin)*float64(j)/float64(n)
		if c.keptBySolid(u, v) {
			return true
		}
	}
	return false
}

// newRuledUVFrame builds the (u,v) frame of a ruled side WITHOUT a cut plane — the general curved∩curved
// path needs only the geometry (base/axis/ref/radius), not the plane signed-distance coefficients
// (p,q,s,t,uN stay 0; the half-space predicate is replaced by keptBySolid). base is the surface point at
// v=0 (cone apex / cylinder bottom centre), rad(v)=radSlope·v+radConst (#1403).
func newRuledUVFrame(base math.Point3, axis, ref math.Vector3, radSlope, radConst float64, band coneSideBand_) ruledUV {
	return ruledUV{
		base: base, axis: axis, ref: ref, binor: axis.Cross(ref),
		radSlope: radSlope, radConst: radConst, band: band,
	}
}

// newConeUVSolid builds a frustum side's (u,v) model for a general cut whose kept side is decided by the
// other solid's membership oracle `inside` under op (isB marks this operand as the boolean's B). It is the
// plane-free, solid-membership counterpart of newConeUV (#1403).
func newConeUVSolid(cone geom.Cone, band coneSideBand_, op Op, isB bool, inside func(math.Point3) bool) ruledUV {
	c := newRuledUVFrame(cone.Apex, cone.AxisDir.AsVector(), cone.Ref.AsVector(), stdmath.Tan(cone.HalfAngle), 0, band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return c
}

// coneSideSolidSplit trims a frustum side by an SSI imprint, keeping the cells whose surface point lies in
// the other solid (per op). The same general arrangement trim as the half-space cone split (coneSideUVSplit)
// — only the imprint is an SSI loop set and the predicate is solid membership (#1403).
func coneSideSolidSplit(f curvedFace, cone geom.Cone, band coneSideBand_, imprint []geom.Curve3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	c := newConeUVSolid(cone, band, op, isB, inside)
	return trimByImprint(&c, f, cone, imprint, ruledSolidMaterial(&c))
}

// curvedSolidMembership returns a point-in-solid oracle for a primitive curved solid (cone/cylinder
// frustum), or ok=false for a shape it does not handle analytically. A curved solid's faces are curved, so
// the planar ray-cast insideSolid does not apply — each primitive has its own closed-form inside test. This
// is the general curved∩curved classifier's membership stage; it grows one case per primitive (#1403).
func curvedSolidMembership(b *topo.Body) (func(math.Point3) bool, bool) {
	faces := facesOfAny(b)
	if cone, vMin, vMax, ok := coneSolidParams(faces); ok {
		return func(p math.Point3) bool { return pointInsideConeSolid(cone, vMin, vMax, p, false) }, true
	}
	if cyl, base, height, ok := cylinderSolidParams(faces); ok {
		return func(p math.Point3) bool { return pointInsideCylinderSolid(cyl, base, height, p, false) }, true
	}
	return nil, false
}

// pointInsideConeSolid reports whether p is inside a frustum solid: within the apex-distance band
// [vMin, vMax] (between the caps) AND inside the cone radius v·tan(HalfAngle) at that height. When strict
// is false a small model-relative margin keeps an imprint loop (on the surface) from flickering across
// the boundary — the boolean-membership need. When strict is true the test is margin-free (exact
// geometric inside): the point-in-solid CLASSIFIER wants that, having already peeled off the on-surface
// band with its own onTol, so this fast path returns the same verdict the ray-parity path would.
func pointInsideConeSolid(cone geom.Cone, vMin, vMax float64, p math.Point3, strict bool) bool {
	axis := cone.AxisDir.AsVector()
	v := float64(cone.Apex.VectorTo(p).Dot(axis))
	margin := 0.0
	if !strict {
		rim := vMax * stdmath.Tan(cone.HalfAngle)
		margin = geom.ResolutionForSize(rim + vMax).Plane() // model-relative inside-solid margin (#1399)
	}
	if v < vMin+margin || v > vMax-margin {
		return false
	}
	axisPt := cone.Apex.TranslateBy(axis.Scale(math.Scalar(v)))
	rho := float64(axisPt.VectorTo(p).Length())
	return rho < v*stdmath.Tan(cone.HalfAngle)-margin
}

// coneSideFace finds a frustum side face of b: its curvedFace, the geom.Cone, and the rim band. ok=false
// unless b has a cone side bounded by two full-circle rims (a full cone reaching its apex is not the band
// case and defers to the bespoke handler).
func coneSideFace(b *topo.Body) (curvedFace, geom.Cone, coneSideBand_, bool) {
	for _, f := range facesOfAny(b) {
		cone, isCone := f.surface.(geom.Cone)
		if !isCone {
			continue
		}
		if band, ok := coneSideBand(f, cone); ok {
			return f, cone, band, true
		}
	}
	return curvedFace{}, geom.Cone{}, coneSideBand_{}, false
}

// cylinderSideFace finds a cylinder side face of b: its curvedFace, the geom.Cylinder, and the rim band
// (the same two-circle band the half-space path uses, with v the axial distance from the bottom rim).
func cylinderSideFace(b *topo.Body) (curvedFace, geom.Cylinder, coneSideBand_, bool) {
	for _, f := range facesOfAny(b) {
		if _, isCyl := f.surface.(geom.Cylinder); !isCyl {
			continue
		}
		if cyl, band, ok := fullCylinderSideBand(f); ok {
			return f, cyl, band, true
		}
	}
	return curvedFace{}, geom.Cylinder{}, coneSideBand_{}, false
}

// newCylinderUVSolid builds a cylinder side's (u,v) model for a general cut decided by `inside` under op.
// A cylinder is the degenerate cone — constant radius R (radSlope 0, radConst R), v the axial distance from
// the bottom rim centre (band.bottom) — so it reuses the whole ruled solid-membership trim (#1403).
func newCylinderUVSolid(cyl geom.Cylinder, band coneSideBand_, op Op, isB bool, inside func(math.Point3) bool) ruledUV {
	c := newRuledUVFrame(band.bottom, cyl.AxisDir.AsVector(), cyl.Ref.AsVector(), 0, cyl.Radius, band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return c
}

// cylinderSideSolidSplit trims a cylinder side by an SSI imprint, keeping the cells inside the other solid
// (per op) — the cylinder counterpart of coneSideSolidSplit (#1403).
func cylinderSideSolidSplit(f curvedFace, cyl geom.Cylinder, band coneSideBand_, imprint []geom.Curve3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	c := newCylinderUVSolid(cyl, band, op, isB, inside)
	return trimByImprint(&c, f, cyl, imprint, ruledSolidMaterial(&c))
}

// planarCapFaces returns a body's planar (cap) faces as curvedFaces, kept WHOLE — for a clean side-breach
// cut the breach never reaches a cap, so the target's caps survive untouched in the result (#1403).
func planarCapFaces(b *topo.Body) []curvedFace {
	var caps []curvedFace
	for _, f := range facesOfAny(b) {
		if _, isPlane := f.surface.(geom.Plane); isPlane {
			caps = append(caps, f)
		}
	}
	return caps
}

// reverseCurvedFaces flips each tool wall into the CAVITY a Difference carves: the sense flag (so the normal
// points into the void) AND every loop's winding (so the boundary is walked the OTHER way). The tool keeps the
// part INSIDE the target — a tunnel band whose imprint loop is walked the SAME way as the target's own hole —
// so reversing the loop opposes them, the manifold-orientation a watertight cut needs (curvedStitch orients
// each shared edge by its loop traversal, not the face sense). The tunnel is a ruled LOFT band
// (twoClosedRimBandMesh), which lofts rim-to-rim regardless of winding, so the reversal does not change its
// meshed region — only its orientation (#1403/#1476).
func reverseCurvedFaces(faces []curvedFace) []curvedFace {
	out := make([]curvedFace, len(faces))
	for i, f := range faces {
		f.reversed = !f.reversed
		loops := make([]curvedLoop, len(f.loops))
		for j, lp := range f.loops {
			loops[j] = reverseCurvedLoop(lp)
		}
		f.loops = loops
		out[i] = f
	}
	return out
}

// reverseCurvedLoop reverses a loop's traversal: each edge's direction (t0↔t1) and the edge order both flip,
// so the loop walks the opposite way around the same boundary (#1476).
func reverseCurvedLoop(lp curvedLoop) curvedLoop {
	n := len(lp.edges)
	rev := make([]loopEdge, n)
	for i, e := range lp.edges {
		e.t0, e.t1 = e.t1, e.t0
		e.v0, e.v1 = e.v1, e.v0 // keep the exact loop-oriented endpoint carry consistent (ADR-0058)
		rev[n-1-i] = e
	}
	return curvedLoop{edges: rev}
}

// loopsClearOfCaps reports whether every imprint point sits STRICTLY between the side band's two cap levels
// (vMin, vMax) — a clean side-breach where the cut leaves both caps whole. A breach reaching a cap needs the
// planar cap itself trimmed, which this first cut migration defers to the bespoke handler (#1403).
func loopsClearOfCaps(side ruledUV, loops []geom.Curve3) bool {
	margin := geom.ResolutionForSize(side.band.vMax - side.band.vMin).Plane()
	for _, lp := range loops {
		for _, p := range imprintLoopPoints(lp) {
			v := float64(side.base.VectorTo(p).Dot(side.axis))
			if v < side.band.vMin+margin || v > side.band.vMax-margin {
				return false
			}
		}
	}
	return true
}

// Cylinder∩cylinder intersect (including the near-pinch band, #1818) is now built by the unified
// ruledCrossingIntersect (curved_general_crossing.go, ADR-0058 phase 3), which routes the near-pinch fatter
// wall through cylinderLensSplit below as one conditioning branch. The former crossingCylinderIntersectGeneral
// driver collapsed into it.

// cylinderLensSplit trims a cylinder side by the intersection imprint. perLoop trims each loop in a SEPARATE
// arrangement — the #1818 near-pinch fat wall, whose two lens caps sit tip-to-tip at the necks and would fuse
// in a shared arrangement — and concatenates the resulting caps; otherwise it trims by both loops at once (the
// thin wall's single band bounded by both loops, and every non-near-pinch crossing). Each single-loop
// arrangement keeps the loop's clear azimuth gap, so its seam auto-places cleanly with no pinched handling.
func cylinderLensSplit(perLoop bool, f curvedFace, cyl geom.Cylinder, band coneSideBand_, loops []geom.Curve3, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	if !perLoop {
		return cylinderSideSolidSplit(f, cyl, band, loops, Intersection, isB, inside)
	}
	var caps []curvedFace
	for i := range loops {
		lens, _, err := cylinderSideSolidSplit(f, cyl, band, loops[i:i+1], Intersection, isB, inside)
		if err != nil {
			return nil, nil, err
		}
		caps = append(caps, lens...)
	}
	return caps, nil, nil
}

// keptOrNone adapts a side split's (faces, lid, err) to (faces, ok): ok is true only when the split
// succeeded and kept some geometry, so the two-sided drivers read as one short condition (#1403).
func keptOrNone(faces []curvedFace, _ []loopEdge, err error) ([]curvedFace, bool) {
	return faces, err == nil && len(faces) > 0
}

// Cone∩cone and cone∩cylinder intersect are built by the unified ruledConeCrossingIntersect
// (curved_general_crossing.go, ADR-0058 phase 3): the two former per-pair drivers had the identical
// skeleton, so they collapsed into one general driver over curvedSideFace + curvedSideSolidSplit.
