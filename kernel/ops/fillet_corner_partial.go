// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial planar corner (OCCT tests/blend tolblend_simple/D3 E6 E7 E8: 3+ of a >3-valent vertex's
// edges filleted, one shared radius r, the rest left perfectly sharp — reached only via
// solveCorner's `len(ps) >= 3 && len(ps) < len(faces)` branch, fillet.go). Consulted
// geometry-math-advisor before building this (2026-07-31): the ordinary trihedral sphere
// (solveBlend, 3 planes = the full vertex) does NOT generalize here for free — DRAWEXE 8.0 confirms
// OCCT itself falls through ChFiKPart's own analytic Sphere special case
// (ChFiKPart_ComputeData_Sphere.cxx exists and is checked) to its general Plate/
// PerformMoreThreeCorner builder for this family (measured directly on tolblend_simple/D5's own
// 2-adjacent-edge sub-case: a genuine 16x16-pole BSplineSurface, not a recognized analytic type) —
// proof that no closed form exists in GENERAL. The reason: a sphere tangent to the TOUCHED faces
// only touches the face SHARED between two adjacent filleted edges at a single POINT (a plane
// tangent to a sphere touches at one point, not along a boundary curve), so unless that tangent
// point happens to be exactly where both edges' own rolling-ball SPINES cross, the sphere does not
// actually close the gap and the true patch is free-form.
//
// So this file does NOT attempt the free-form fill (a separate, larger epic — see the doc comment
// on solvePartialCorner). It runs the one test that DOES decide, case by case, whether the cheap
// analytic sphere happens to be exact: solve the sphere tangent to the TOUCHED faces only (ignoring
// any face at the vertex no filleted edge touches), then certify — via the SAME
// fullRoundArmSpinesConcurrent check the full-round K-gon already uses (armSpinePerpDistance is
// that check's measurement, factored out and shared here) — that the sphere's centre lies on every
// filleted edge's own 2-face rolling-ball spine. When it holds, the sphere's tangent point on the
// shared face IS the intersection of both edges' offset tangent lines there (each edge's tangent
// line on that face is the projection of the moving rolling-ball centre as it slides along the
// spine; the spine-concurrence certificate is exactly the statement that the SAME centre point
// services both), and the construction is exact.
//
// A SECOND, independent gap was found empirically while investigating this (traced by disabling the
// panic recover and reading the stack — verbatim in wave-report-R2.md): even when a touched-face
// sphere IS exact and spine-concurrent, the shared sphere-patch assembler (spherePatchFace/
// chainArcs, fillet_sphere_patch.go) still cannot build it, because it hard-assumes the corner's
// registered arcs form ONE CLOSED CYCLE (each touched face shared by exactly two filleted arms, e.g.
// the ordinary trihedral triangle or the full-round K-gon). A genuine partial corner (K<N) always
// leaves the two "outermost" touched faces of the filleted fan shared by only ONE arm each — the
// N-K sharp edges must still reach the ORIGINAL vertex point, so the true boundary is an OPEN arc
// fan closed by straight segments through that surviving vertex, which chainArcs never builds (it
// panics: `spherePatchFlipped` indexes loop.pts[2] on a 1-point loop). partialCornerLoopGap detects
// this structurally at solve time — before ever reaching the crash-prone assembler — so a case that
// fails it declines cleanly instead of panicking. This is the closure piece the future N-sided/
// Gordon-fill epic must also supply (see the spec note there); the touched-face-sphere exactness
// test above will remain useful groundwork for that epic even where this file declines.
//
// tolblend_simple/D5 (2 of a 4-valent vertex's edges) was ALSO tried through this exact mechanism —
// it is the smallest instance of the same family and was the case that surfaced both gaps above.
// But routing every 2-edge shared-face corner with faces>3 through here (instead of only the
// ordinary solveTwoEdgeCorner miter) FALSIFIED on the full -race suite:
// TestClassifyEndCornersExcludesKGreaterThanOne (simple/V3, a genuine 2-edge shared-face miter at
// its OWN 5-valent vertex) proved the ordinary miter's cyl∩cyl seam is LOCAL to the two picked
// edges and their 3 relevant faces — it needs no awareness of the vertex's total valence, and is
// correct in general at valence>3. So D5's specific invalidity (it reaches solveMiter and the
// welded body fails IsSolid) is a genuine, case-specific seam defect on ITS geometry, not a missing
// valence check — solveCorner therefore still routes EVERY len(ps)==2 corner through the ordinary
// solveTwoEdgeCorner, unchanged from before this file existed; only len(ps)>=3 reaches here.

// solvePartialCorner solves the K<N partial planar corner at v (all K filleted edges share ONE
// radius r; the touched-face sphere test decides whether the analytic patch is exact). It is
// reached only when the vertex valence (faces) exceeds 3 and fewer edges are filleted than faces
// meet there — the ordinary trihedral (faces==3) and full-round (ps==faces) dispatch branches
// handle every other configuration untouched. Example: cb, _, err :=
// solvePartialCorner(body, v, faces, ps, r).
func solvePartialCorner(body *topo.Body, v *topo.Vertex, faces []*topo.Face, ps []filletPick, r float64) (*cornerBlend, *cornerMiter, error) {
	if err := partialCornerArmsSupported(ps); err != nil {
		return nil, nil, err
	}
	touched := touchedFacesOfPicks(ps)
	if missing := partialCornerLoopGap(touched, ps); missing != nil {
		return nil, nil, fmt.Errorf("fillet: partial corner where %d of %d edges at the vertex are filleted needs its patch boundary closed through %d gap face(s) bordering the surviving sharp edges at the vertex — that boundary-assembly capability is not yet built (the free-form/N-sided corner filling epic)", len(ps), len(faces), len(missing))
	}
	weld := ResolutionForBody(body).Weld()
	cb, err := solvePartialCornerSphere(body, v, touched, r, weld)
	if err != nil {
		return nil, nil, err
	}
	if dist := partialCornerSpineDistance(v, ps, cb.center, r); dist > weld {
		return nil, nil, fmt.Errorf("fillet: partial corner where %d of %d edges at the vertex are filleted has a touched-face tangent sphere but it is not concurrent with every filleted edge's own rolling-ball spine (radius %g, spine distance %g > weld %g) — the free-form corner filling is not yet supported", len(ps), len(faces), r, dist, weld)
	}
	return cb, nil, nil
}

// partialCornerLoopGap reports the touched faces shared by only ONE filleted pick (nil when none —
// the arcs alone would close into the ordinary trihedral triangle or full-round K-gon). Those are
// exactly the faces where the corner boundary must instead run through a surviving sharp edge back
// to the original vertex point — a boundary chainArcs cannot build (see the file doc comment). This
// is checked BEFORE any geometry is solved, so a structurally-unclosable partial corner declines
// cleanly instead of reaching the panic that motivated it.
func partialCornerLoopGap(touched []*topo.Face, ps []filletPick) []*topo.Face {
	degree := make(map[uint64]int, len(touched))
	for _, p := range ps {
		for _, f := range p.edge.Faces() {
			degree[f.ID()]++
		}
	}
	var gap []*topo.Face
	for _, f := range touched {
		if degree[f.ID()] < 2 {
			gap = append(gap, f)
		}
	}
	return gap
}

// solvePartialCornerSphere solves the sphere tangent to the touched faces only: the exact 3-plane
// solve (solveBlend, byte-identical to the ordinary trihedral path) when the K filleted edges
// happen to touch exactly 3 distinct faces (two adjacent edges sharing one face, D5's case), or the
// least-squares common-tangent-sphere certificate (commonTangentSphereCentre, shared with the
// full-round K-gon) when they touch more (three consecutive edges of a 4-valent vertex touch all 4
// faces, D3's case) — declining with the measured plane residual when no common sphere exists.
func solvePartialCornerSphere(body *topo.Body, v *topo.Vertex, touched []*topo.Face, r, weld float64) (*cornerBlend, error) {
	if len(touched) == 3 {
		return solveBlend(body, v, touched, r)
	}
	s, residual, err := commonTangentSphereCentre(touched, r)
	if err != nil {
		return nil, err
	}
	if residual > weld {
		return nil, fmt.Errorf("fillet: partial corner touching %d faces has no common tangent sphere (radius %g, plane residual %g > weld %g) — the free-form corner filling is not yet supported", len(touched), r, residual, weld)
	}
	return tangentSphereBlend(v, touched, s, r)
}

// touchedFacesOfPicks returns the distinct faces adjacent to the picks' edges — the SUBSET of the
// vertex's full face set that the partial corner's sphere must be tangent to (unlike facesAtVertex,
// which returns every face at the vertex including ones no filleted edge touches).
func touchedFacesOfPicks(ps []filletPick) []*topo.Face {
	seen := map[uint64]*topo.Face{}
	for _, p := range ps {
		for _, f := range p.edge.Faces() {
			seen[f.ID()] = f
		}
	}
	out := make([]*topo.Face, 0, len(seen))
	for _, f := range seen {
		out = append(out, f)
	}
	return out
}

// partialCornerArmsSupported gates the partial-corner solve to the configuration the sphere
// expresses: constant-radius CONVEX arms only (the material-side 3-plane/least-squares solve
// assumes one sense, same precondition as the full-round K-gon).
func partialCornerArmsSupported(ps []filletPick) error {
	if p := varyingPick(ps); p != nil {
		return fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a partial planar corner", p.r0, p.r1)
	}
	for _, p := range ps {
		if ClassifyEdgeConvexity(p.edge) != EdgeConvex {
			return fmt.Errorf("fillet: partial planar corner requires all-convex filleted arms (edge %d is not convex)", p.edge.ID())
		}
	}
	return nil
}

// partialCornerSpineDistance is a diagnostic-only helper (used by the falsification tests) that
// reports the worst perpendicular distance from the solved sphere centre to any filleted edge's own
// rolling-ball spine — the same quantity fullRoundArmSpinesConcurrent gates on, exposed so a decline
// reason can carry the measured number instead of just pass/fail.
func partialCornerSpineDistance(v *topo.Vertex, ps []filletPick, s math.Point3, r float64) float64 {
	worst := 0.0
	for _, p := range ps {
		if d := armSpinePerpDistance(v, p, s, r); d > worst {
			worst = d
		}
	}
	return worst
}

// armSpinePerpDistance is armSpineConcurrent's underlying measurement, factored out so it can be
// reported as a number (not just a weld-tolerance boolean) for decline messages and falsification.
func armSpinePerpDistance(v *topo.Vertex, p filletPick, s math.Point3, r float64) float64 {
	faces := p.edge.Faces()
	if len(faces) != 2 {
		return stdmath.Inf(1)
	}
	pa, okA := faces[0].Geometry().(geom.Plane)
	pb, okB := faces[1].Geometry().(geom.Plane)
	if !okA || !okB {
		return stdmath.Inf(1)
	}
	nA, nB := outwardPlaneNormal(faces[0], pa), outwardPlaneNormal(faces[1], pb)
	axis, err := math.UnitVector3FromVector(p.edge.StartVertex().Point().VectorTo(p.edge.EndVertex().Point()))
	if err != nil {
		return stdmath.Inf(1)
	}
	frame := v.Point().TranslateBy(nA.Add(nB).Scale(-r / (1 + nA.Dot(nB))))
	d := frame.VectorTo(s)
	perp := d.Sub(axis.AsVector().Scale(d.Dot(axis.AsVector())))
	return perp.Length()
}
