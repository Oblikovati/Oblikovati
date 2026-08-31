// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Self-intersection — the SAME-SHEET arm (M48/C3, Oblikovati/Oblikovati#3477). It replaces the
// triangle coplanar branch (SAT + Sutherland-Hodgman on projected facets) that used to answer this.
//
// Two faces trimmed out of ONE surface sheet — a doubled wall from a bad import, two copies of the
// same plane, the two halves of a seam-split cylinder — have NO isolated surface-surface intersection
// curve, so the general arm cannot see them at all: geom.SurfaceIntersect reports the pair handled and
// empty ("parallel or coincident: no isolated intersection curve"). Their interpenetration is a
// TRIM-region overlap, and it is decided here on the exact trims, with brep.PointInFaceTrim.
//
// Two rules, both taken at exact probe points on the faces' own boundaries (faceTrimProbes):
//
//  1. PARTIAL overlap — a probe of one face lies STRICTLY inside the other's trim (inside, and clear of
//     that trim's own boundary). This is the corner poking in, the bar crossing a bar, and the face
//     wholly swallowed by another.
//  2. DUPLICATE — every probe of each face lies on the other's trimmed region. Rule 1 cannot see two
//     faces with the SAME trim, because then no probe is strictly inside; but that is the doubled-wall
//     import defect, so it is reported here instead.
//
// Both rules ignore probes on the two faces' shared topology, which is what keeps the legitimate
// populations out: coplanar neighbours meeting along a shared edge (every sheet-metal end cap, #2074),
// and the two halves of a seam-split cylinder, whose common boundary IS the seam.
//
// ★ NAMED DECLINE. A same-sheet overlap that neither reaches a probe nor covers one face entirely —
// two trims whose regions meet only in a lens bounded away from every vertex and edge midpoint — is
// NOT reported. The probe set is exact, but it is finite; widening it means an exact 2D trim-region
// arrangement in the surface's chart, which is a larger piece of work than this slice.

// coincidentTrimOverlap returns a witness where two faces of ONE surface sheet overlap in their trims,
// applying rule 1 (a probe strictly inside the other trim) before rule 2 (mutual full coverage).
func coincidentTrimOverlap(fa, fb *topo.Face, shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	if p, ok := probeStrictlyInside(fa, fb, shared, res); ok {
		return p, true
	}
	if p, ok := probeStrictlyInside(fb, fa, shared, res); ok {
		return p, true
	}
	return duplicateTrimWitness(fa, fb, shared, res)
}

// probeStrictlyInside returns f's first boundary probe that lies strictly inside g's trim and off the
// two faces' shared topology — an overlap that cannot be explained as contact.
func probeStrictlyInside(f, g *topo.Face, shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	for _, p := range faceTrimProbes(f) {
		if shared.holds(p, res.Sew()) {
			continue
		}
		if brep.PointInFaceTrim(g, p) && distanceToFaceBoundary(g, p) > res.Sew() {
			return p, true
		}
	}
	return math.Point3{}, false
}

// duplicateTrimWitness reports the doubled-face defect: each face's whole boundary lies on the other's
// trimmed region, so the two trims are the same region twice. It demands a witness off the shared
// topology, so a pair whose only common ground IS its shared boundary is contact, not a duplicate.
func duplicateTrimWitness(fa, fb *topo.Face, shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	witness, ok := boundaryCoveredBy(fa, fb, shared, res)
	if !ok {
		return math.Point3{}, false
	}
	if _, ok := boundaryCoveredBy(fb, fa, shared, res); !ok {
		return math.Point3{}, false
	}
	return witness, true
}

// boundaryCoveredBy reports whether every probe of f lies on g's trimmed region, returning one probe
// that is off the faces' shared topology (there must be one, or the cover is only the shared boundary).
func boundaryCoveredBy(f, g *topo.Face, shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	var witness math.Point3
	found := false
	for _, p := range faceTrimProbes(f) {
		if !brep.PointOnFace(g, p, res.Sew()) {
			return math.Point3{}, false
		}
		if !found && !shared.holds(p, res.Sew()) {
			witness, found = p, true
		}
	}
	return witness, found
}

// facesShareOneSheet reports whether the two faces are trims of the same surface sheet: each face's
// boundary probes lie on the OTHER face's surface. It is asked both ways so a face whose boundary
// happens to run along a second surface (a planar cap's rim lying on the cylinder it caps) is not
// mistaken for a co-sheet pair.
func facesShareOneSheet(fa, fb *topo.Face, res geom.Resolution) bool {
	return sheetHoldsBoth(fa.Geometry(), fb.Geometry(), faceTrimProbes(fa), faceTrimProbes(fb), res)
}

// sheetHoldsBoth is facesShareOneSheet over probes the caller already has, so a pairwise scan computes
// each face's probes once instead of once per pair.
func sheetHoldsBoth(sa, sb geom.Surface, probesA, probesB []math.Point3, res geom.Resolution) bool {
	return probesLieOnSurface(sb, probesA, res.Sew()) && probesLieOnSurface(sa, probesB, res.Sew())
}

// probesLieOnSurface reports whether every probe sits on s within tol; an empty probe set (a face with
// no edges) is not on anything.
func probesLieOnSurface(s geom.Surface, probes []math.Point3, tol float64) bool {
	if len(probes) == 0 || s == nil {
		return false
	}
	for _, p := range probes {
		if _, _, foot := geom.ClosestPointOnSurface(s, p); float64(foot.DistanceTo(p)) > tol {
			return false
		}
	}
	return true
}

// distanceToFaceBoundary is the exact distance from p to the nearest of f's trim edges — how far
// inside (or outside) the trim a point sits, measured against the edges' own curves rather than any
// discretization of them.
func distanceToFaceBoundary(f *topo.Face, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, e := range f.Edges() {
		best = stdmath.Min(best, brep.EntityDistance(brep.PointSupport(p), brep.CurveSupport(e.Geometry())))
	}
	return best
}
