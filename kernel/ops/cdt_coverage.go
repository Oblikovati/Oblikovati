// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// COVERAGE CERTIFICATE for a constrained Delaunay triangulation.
//
// A CDT must COVER the domain its loops bound — every point of the domain in exactly one triangle, no
// gap and no overlap. It can fail to, and say nothing: the inside/outside flood either leaves a region
// untriangulated (a HOLE) or spills past a wall it could not realise (a LEAK). constrainedDelaunay
// reports neither; it returns whatever it extracted and the caller ships it. planarTris has guarded
// against exactly this since #1610, on the same AREA proxy (planarAreaMatches) — a triangulation that
// covers a polygon reproduces its shoelace area EXACTLY, so any disagreement beyond float summation
// error IS a coverage defect. The CONFORMANCE re-mesh path never applied it.
//
// WHAT THE CORPUS SWEEP FOUND, and it is not what was expected. Of the 111 conformance re-mesh
// decisions on the shipped bodies of the OCCT blend-parity corpus, 10 are cyl/cone (the rest are
// planes, which take a different mesher); 7 of those 10 fail this certificate — complex/D8's r=24
// corner cylinder 38.94% short of its own (u,v) domain, complex/F2's walls −38.74% / −34.10% /
// −96.99% and one +48.33% LEAK. But the CDT recovers EVERY boundary constraint on all 7 (zero
// unrecovered, no earcut fallback), and the correlation with a different property is perfect: all 7
// non-covering trims have a SELF-INTERSECTING (u,v) boundary polygon and all 3 covering ones have a
// simple polygon, 10 of 10. The mesher is innocent. Its INPUT is malformed — a domain with no
// well-defined area, hence no correct triangulation — and toUVLoops hands it over unchecked, which is
// the defect conformingPlaneMesh already refuses via simpleLoop2D and the cyl/cone path never did.
//
// SO THE CERTIFICATE CHOOSES, IT DOES NOT VETO (see bestConformingPatch). Refusing a non-covering
// re-mesh outright was measured NET HARMFUL: simple/Q5's fillet face is one of the 7, and its
// (incomplete) re-mesh ships 7.6459e6 against DRAWEXE's 8.12117e6 while the mesh it would fall back to
// is 6.5576e6 — refusing would take that face from −5.85% to −19.3% of the oracle. On a domain that
// has no correct triangulation, "more complete" is not available and "closer to the truth" is what is
// left, which is conformingMeshIsFaithful's job. The certificate's job is to say which candidate is
// KNOWN GOOD, and where none is, to make the shortfall travel with the mesh instead of being silent.
//
// In the metric (u,v) of a cyl/cone trim the check is exact for a simple domain: the polygon is known
// (the loops the CDT was handed), so "the triangles' area equals the loops' area" is a complete
// statement of coverage up to the one thing area cannot see — an overlap and a gap of identical area,
// which needs the triangulation to fail in two compensating ways at once. That is the same residual
// hole #1610 accepted, for the same reason.

// cdtCoversLoops reports whether tris covers exactly the domain bounded by loops over pts: loops[0] is
// the outer boundary and the rest are holes (the order every CDT caller in this package builds). A
// false answer means the triangulation is not a triangulation OF this domain.
//
// Example: cdtCoversLoops(b.scaled, loops, tris) == true certifies the interior-refined candidate, so
// bestConformingPatch promotes it over the boundary-only one.
func cdtCoversLoops(pts [][2]float64, loops [][]int, tris [][3]int) bool {
	if len(tris) == 0 || len(loops) == 0 {
		return false // nothing triangulated covers nothing (and forces the caller's decline)
	}
	want := loopedDomainArea(pts, loops)
	if want <= 0 {
		return false // a degenerate or inverted domain: nothing to certify against
	}
	return coverageAreaMatches(trisUnsignedArea(pts, tris), want, indexedPointsResolution(pts).Area())
}

// loopedDomainArea is the area the loops enclose: |outer| minus every hole, by the shoelace formula.
func loopedDomainArea(pts [][2]float64, loops [][]int) float64 {
	total := stdmath.Abs(loopShoelaceArea(pts, loops[0]))
	for _, h := range loops[1:] {
		total -= stdmath.Abs(loopShoelaceArea(pts, h))
	}
	return total
}

// loopShoelaceArea is the signed area of the closed polygon loop indexes into pts.
func loopShoelaceArea(pts [][2]float64, loop []int) float64 {
	var twice float64
	for i := range loop {
		p, q := pts[loop[i]], pts[loop[(i+1)%len(loop)]]
		twice += p[0]*q[1] - q[0]*p[1]
	}
	return twice / 2
}

// indexedPointsResolution builds the model-relative Resolution of a CDT's point set from its
// bounding-box diagonal, so the coverage floor scales with the part (ADR-0042) instead of being an
// absolute epsilon that is ~10% of a µm-scale face's area (#1610).
func indexedPointsResolution(pts [][2]float64) geom.Resolution {
	lo, hi := pts[0], pts[0]
	for _, p := range pts {
		lo = [2]float64{stdmath.Min(lo[0], p[0]), stdmath.Min(lo[1], p[1])}
		hi = [2]float64{stdmath.Max(hi[0], p[0]), stdmath.Max(hi[1], p[1])}
	}
	return geom.ResolutionForSize(stdmath.Hypot(hi[0]-lo[0], hi[1]-lo[1]))
}

// coverageAreaMatches is the shared coverage predicate behind planarAreaMatches (#1610's planar-face
// guard) and cdtCoversLoops (the conformance re-mesh's): the triangulated area must equal the domain
// area to a relative bracket with a model-relative floor. The bracket absorbs float summation error
// only — a covering triangulation reproduces the shoelace area exactly — so it is not a tolerance to
// tune, and every real defect measured on the corpus misses it by 4 to 6 decades.
func coverageAreaMatches(got, want, floor float64) bool {
	return stdmath.Abs(got-want) <= 1e-6*want+floor // tol:numeric (relative area fraction)
}

// CodeTessellateDomainUncovered marks a mesh whose constrained Delaunay did not cover the domain its
// boundary loops bound — a hole left where constraint recovery failed, or a leak past a missing wall.
// The triangulation is still returned (it may be the best available; see bestConformingPatch), but the
// shortfall travels with the mesh instead of being reported as success (Oblikovati#1610's class).
const CodeTessellateDomainUncovered diag.Code = "tessellate.domain-uncovered"

// recordUncoveredDomain records the diagnostic when covers is false, so a partial triangulation is
// never silent. No-op on a covering mesh.
func recordUncoveredDomain(m *Mesh, covers bool) {
	if m == nil || covers {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeTessellateDomainUncovered,
		Severity: diag.Warning,
		Detail:   "constrained Delaunay did not cover the (u,v) domain its boundary loops bound",
	})
}
