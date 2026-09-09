// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The DEGENERATE OVERLAP: two faces of the two operands lying on the SAME surface, so their contact is
// a two-dimensional region rather than a curve (ADR-0045). The planar boolean has always had it — two
// flush faces are covered once, from one side, by the ON/ON table — and the rule is not about planes.
// Two coaxial cylinders of one radius overlap exactly so, and the mixed pipeline declined them on the
// only evidence it looked for: an intersector asked for a crossing between two identical surfaces
// answers that it cannot, and "cannot" was read as "unsupported" (ADR-0061 stage 4).
//
// The rule has two halves, and they are the same two the crossing case has: an IMPRINT, which for a
// coincident pair is the other face's own boundary rather than a section curve, and a CLASSIFICATION,
// which is the ON/ON table rather than the membership oracle.

// coincidentCover reports whether a point of f's surface is covered by a face of the other operand
// lying on the SAME surface, and if so whether that face's outward normal agrees with f's — shared
// contact — or opposes it. It is the general form of the coplanar cover: identity of the surfaces
// through geom.SurfacesCoincide, containment through each face's own trim.
func coincidentCover(f curvedFace, p math.Point3, others []curvedFace) (covered, sameNormal bool) {
	res := geom.ResolutionForBox(faceLoopBox(f))
	for _, o := range others {
		if !geom.SurfacesCoincide(f.surface, o.surface, res) || !faceHoldsPoint(o, p) {
			continue
		}
		return true, float64(outwardNormalAt(f, p).Dot(outwardNormalAt(o, p))) > 0
	}
	return false, false
}

// faceHoldsPoint is a face's trim containment at a point already on its surface: the exact conic/polygon
// test for a planar face (the verdict the planar boolean has always used, unchanged), the developed
// (u,v) trim for every other surface.
func faceHoldsPoint(f curvedFace, p math.Point3) bool {
	if _, isPlane := planeOf(f); isPlane {
		return faceContainsExact(f, p)
	}
	return pointInTrimUV(f, p)
}

// outwardNormalAt is the face's OUTWARD normal at a point of its surface — the surface normal, turned
// round for a reversed face, whose surface normal points into the material.
func outwardNormalAt(f curvedFace, p math.Point3) math.Vector3 {
	n := geom.SurfaceNormalAt(f.surface, p)
	if f.reversed {
		return n.Scale(-1)
	}
	return n
}

// coincidentKeepAt is the keep test every chart shares: a point covered by a face of the other operand
// on the same surface follows the ON/ON table (coplanarKeep — the shared region is emitted once, as A's
// copy); every other point follows the keep table over the membership oracle. Asking the oracle there
// instead is what a coincident overlap makes meaningless: the point is ON the other solid's boundary,
// where inside and outside are not the question.
func coincidentKeepAt(f curvedFace, others []curvedFace, other insideOracle, op Op, isB bool) func(math.Point3) bool {
	return func(pt math.Point3) bool {
		if covered, same := coincidentCover(f, pt, others); covered {
			return coplanarKeep(op, isB, same)
		}
		return keep(op, isB, other.inside(pt))
	}
}

// coincidentWallImprint is the shared imprint of two walls on ONE ruled surface: each band's rim
// circles, kept where they fall strictly inside the other's band. That is the whole of the contact —
// two coincident surfaces cross nowhere, and what divides them is where one band ends inside the other.
// A rim on the receiving chart's OWN frame is dropped there by ruledFaceUV.admits, so returning both
// sides' rims together keeps the "solve once, write to both" rule the crossing pairs follow.
func coincidentWallImprint(ra, rb ruledSide) []geom.Curve3 {
	out := rimsInsideBand(ra, rb)
	return append(out, rimsInsideBand(rb, ra)...)
}

// rimsInsideBand returns the rim circles of `of` that lie strictly inside `in`'s band. The rims are
// built from the band's own world anchors and radii rather than read off it: a band's v is measured in
// its OWN frame, so two coincident walls both report [0, h] and only the world points place them
// against one another.
func rimsInsideBand(of, in ruledSide) []geom.Curve3 {
	var out []geom.Curve3
	for _, rim := range [2]struct {
		at math.Point3
		r  float64
	}{{of.band.bottom, of.band.rBot}, {of.band.top, of.band.rTop}} {
		v := bandV(rim.at, in.axis, in.band)
		if inside, _ := bandPlacement(v, v, in.band); !inside || rim.r <= 0 {
			continue
		}
		// The receiving chart's own Ref fixes the circle's angle 0, so the imprint and the frame it is
		// arranged against read one azimuth.
		out = append(out, geom.Circle{Center: rim.at, Normal: in.frame.Axis.AsUnit(),
			RefDir: in.frame.Ref.AsUnit(), Radius: rim.r})
	}
	return out
}

// sectionOnFaceBoundary reports whether a section curve runs ALONG one of the receiving face's own
// boundary edges. It is that face's half of the rule sectionOnWallEdge states for the wall: the section
// a coaxial cylinder's wall cuts from the plane of the cap that closes the other IS that cap's own rim,
// and no face is split by its own boundary (ADR-0061 stage 4).
//
// Every sample of the section must lie on ONE edge's own span. A section that merely touches the
// boundary, or that runs past its end, is a genuine imprint and is left to the island rule — which is
// why the edge's span is asked about and not just the conic it lies on.
func sectionOnFaceBoundary(cv geom.Curve3, f curvedFace, res geom.Resolution) bool {
	for _, l := range f.loops {
		for _, e := range l.edges {
			if curveRunsAlongEdge(cv, e, res) {
				return true
			}
		}
	}
	return false
}

// curveRunsAlongEdge reports whether every sample of cv lies on the edge's own parameter span.
func curveRunsAlongEdge(cv geom.Curve3, e loopEdge, res geom.Resolution) bool {
	lo, hi := cv.Domain()
	for i := 0; i <= sectionContactSamples; i++ {
		p := cv.PointAt(lo + (hi-lo)*float64(i)/sectionContactSamples)
		if _, ok := curveParamWithin(e.curve, e.t0, e.t1, p, res); !ok {
			return false
		}
	}
	return true
}

// sectionContactSamples is how densely a section is walked when asking whether it runs along an edge.
// The two are analytic curves and the question is metric, so this only has to be dense enough that a
// section leaving the edge is caught at some station; a section that never leaves it agrees everywhere.
const sectionContactSamples = 16

// edgesRunTogether reports whether two loop edges trace the same stretch of the same curve.
func edgesRunTogether(a, b loopEdge, res geom.Resolution) bool {
	sa, sb := geom.SubCurve(a.curve, a.t0, a.t1), geom.SubCurve(b.curve, b.t0, b.t1)
	return curveRunsAlongEdge(sa, b, res) && curveRunsAlongEdge(sb, a, res)
}
