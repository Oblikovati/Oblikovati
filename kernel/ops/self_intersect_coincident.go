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
//     that trim's own boundary), and the overlap is a REGION rather than a graze: either both
//     boundaries cross into each other, or one face is wholly swallowed by the other. A boundary that
//     merely dips a fraction of a millimetre into its neighbour at one corner and comes straight back
//     out — which a sheet-metal lip's end cap does (#2086) — encloses no area and is not an overlap.
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
// applying rule 1 (an overlap REGION) before rule 2 (mutual full coverage).
//
// ★ IT DOES NOT READ THE TWO FACES' ORIENTATIONS, and that is deliberate. Two coincident faces that
// FACE each other look, in any neighbourhood of the shared patch, exactly like two solids resting on
// one another: material on both sides in both cases. What separates a wall folded through itself from a
// block stacked on a block is global, not local, so orientation cannot decide it — an earlier revision
// of this arm tried, and it retired the sheet-metal lip fold (#2086) that must be caught. The mesh
// detector this replaced reported both as well, so the parity is deliberate too: two coincident,
// overlapping faces are reported whichever way they face.
func coincidentTrimOverlap(fa, fb *topo.Face, shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	ab, ba := probeIncursion(fa, fb, shared, res), probeIncursion(fb, fa, shared, res)
	if ab.any && (ba.any || ab.all) {
		return ab.witness, true
	}
	if ba.any && ba.all {
		return ba.witness, true
	}
	return duplicateTrimWitness(fa, fb, shared, res)
}

// trimIncursion is how one face's boundary sits inside another's trim: a witness strictly inside it,
// whether any probe is, and whether EVERY probe is (which is containment).
type trimIncursion struct {
	witness  math.Point3
	any, all bool
}

// probeIncursion measures f's boundary against g's trim.
func probeIncursion(f, g *topo.Face, shared sharedContact, res geom.Resolution) trimIncursion {
	inc := trimIncursion{all: true}
	for _, p := range faceTrimProbes(f) {
		if !strictlyInsideTrim(g, p, res.Sew()) {
			inc.all = false
			continue
		}
		if !inc.any && !shared.holds(p, res.Sew()) {
			inc.witness, inc.any = p, true
		}
	}
	return inc
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

// strictlyInsideTrim reports whether p lies inside f's trimmed region AND clear of its trim boundary by
// tol — the strict form of brep.PointInFaceTrim, which is inclusive and so accepts the boundary itself.
// The boundary is exactly where a face TOUCHING another one meets it, so every contact decision in this
// detector needs the strict form.
func strictlyInsideTrim(f *topo.Face, p math.Point3, tol float64) bool {
	return brep.PointInFaceTrim(f, p) && distanceToFaceBoundary(f, p) > tol
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
