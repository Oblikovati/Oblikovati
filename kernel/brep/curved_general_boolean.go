// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

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

// newCylinderUVSolid builds a cylinder side's (u,v) model for a general cut decided by `inside` under op.
// A cylinder is the degenerate cone — constant radius R (radSlope 0, radConst R), v the axial distance from
// the bottom rim centre (band.bottom) — so it reuses the whole ruled solid-membership trim (#1403).
func newCylinderUVSolid(cyl geom.Cylinder, band coneSideBand_, op Op, isB bool, inside func(math.Point3) bool) ruledUV {
	c := newRuledUVFrame(band.bottom, cyl.AxisDir.AsVector(), cyl.Ref.AsVector(), 0, cyl.Radius, band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return c
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

// Cone∩cone and cone∩cylinder intersect are built by the unified ruledConeCrossingIntersect
// (curved_general_crossing.go, ADR-0058 phase 3): the two former per-pair drivers had the identical
// skeleton, so they collapsed into one general driver over curvedSideFace + curvedSideSolidSplit.
