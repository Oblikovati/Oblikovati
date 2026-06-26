// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
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
		return func(p math.Point3) bool { return pointInsideConeSolid(cone, vMin, vMax, p) }, true
	}
	return nil, false
}

// pointInsideConeSolid reports whether p is inside a frustum solid: within the apex-distance band
// [vMin, vMax] (between the caps) AND inside the cone radius v·tan(HalfAngle) at that height. A small
// model-relative margin keeps the imprint loop itself (on the surface) from flickering across the boundary.
func pointInsideConeSolid(cone geom.Cone, vMin, vMax float64, p math.Point3) bool {
	axis := cone.AxisDir.AsVector()
	v := float64(cone.Apex.VectorTo(p).Dot(axis))
	rim := vMax * stdmath.Tan(cone.HalfAngle)
	margin := geom.ResolutionForSize(rim + vMax).Plane() // model-relative inside-solid margin (#1399)
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

// polylineCurves adapts SSI imprint loops (closed polylines) to the []geom.Curve3 trimByImprint consumes.
// Each curve is a *geom.Polyline (a POINTER): the arrangement's run-merge compares edge curves by `==` to
// fuse consecutive same-curve edges, and a geom.Polyline value is uncomparable (it holds a slice), so it
// must be carried by identity. The same pointers are handed to BOTH operand sides, so each loop's edges
// merge and the two sides emit the SAME curve for the shared imprint (#1403).
func polylineCurves(loops []geom.Polyline) []geom.Curve3 {
	out := make([]geom.Curve3, len(loops))
	for i := range loops {
		out[i] = &loops[i]
	}
	return out
}

// ConeConeIntersectGeneral is the exported entry kernel/ops routes cone∩cone intersect through: the GENERAL
// curved∩curved pipeline (#1403), with no bespoke loop→body constructor. It declines (ok=false) for
// configurations outside the wired frustum-crossing case so the caller keeps its fallback.
func ConeConeIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return coneConeIntersectGeneral(a, b, rec)
}

// coneConeIntersectGeneral builds cone ∩ cone through the GENERAL pipeline (#1403): the SSI imprint, then
// trimByImprint on EACH cone side keeping the part inside the other cone, then curvedStitch. The shared
// imprint loops are emitted by both sides referencing the same polyline, so the welder fuses them into a
// watertight body (the rod-wall band between the two loops + the two fat-wall lens caps). ok=false when the
// pair is outside the wired frustum-crossing case, so kernel/ops keeps the bespoke ConeConeIntersect path.
func coneConeIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := coneConeImprint(a, b, rec)
	if !ok || len(loops) == 0 {
		return nil, false
	}
	fa, coneA, bandA, okA := coneSideFace(a)
	fb, coneB, bandB, okB := coneSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, _, errA := coneSideSolidSplit(fa, coneA, bandA, imprint, Intersection, false, insideB)
	keptB, _, errB := coneSideSolidSplit(fb, coneB, bandB, imprint, Intersection, true, insideA)
	if errA != nil || errB != nil || len(keptA) == 0 || len(keptB) == 0 {
		return nil, false
	}
	return curvedStitch(append(keptA, keptB...)), true
}
