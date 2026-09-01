// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED Cylinder∧Cylinder SSI-seam canal BAND (OCCT blend simple/K2 K3 K4, bfuseblend/B4 B5
// class): a closed seam loop where two transversally-crossing cylinders meet — a bore exiting a
// boss wall (convex, bcut) or a fused boss valley (concave, bfuse) — rounds into a closed
// variable-section canal whose spine is the intersection of the two ball-centre offset cylinders
// (fillet_cylcyl_seam_station.go carries the derivation). The band is lofted from EXACT stations
// with geom.LoftCanalStations and welded through the SAME host-agnostic closed-rim rebuild the
// analytic torus and elliptic canal bands ship (rebuildRim) — the wrap host recedes along its own
// seam ruling, the other host's rim loop is replaced by the closed contact rail, and the band is
// inserted between the two rails.
//
// Everything here is gated on the `*cylCylSeamBand` arm payload, which ONLY this builder sets, so
// the cylinder/plane arms, the equal-parallel miter arm (P5) and every existing closed-rim band
// keep their dispatch byte-identically (do-no-harm). Parallel-axis pairs are excluded up front
// (line spine — the equal-parallel arm family), so P5's engine is never shadowed.

// cylCylSeamBand is the arm payload: the lofted band (as the armSurface for dispatch/diagnostics)
// plus the fully-solved closed-rim rebuild for THIS pick. group is stashed by the weld dispatch
// (cylCylSeamGroupOf) with every pick of the op, so a multi-seam op (bfuseblend/B4's entry+exit
// loops) can rebuild sequentially — each later seam re-solved against the intermediate body.
type cylCylSeamBand struct {
	geom.Surface                    // the lofted canal band (closed loop) or the exact cylinder arm (line)
	rim          *rimFillet         // the solved closed-rim rebuild payload (nil on a line pick)
	r            float64            // the pick's constant rolling-ball radius
	line         bool               // an equal-parallel valley LINE pick (welds via the single-arm runout)
	group        []cylCylSeamOpPick // all cyl∧cyl seam picks of the op, in fils order (weld-time stash)
}

// cylCylSeamOpPick identifies one seam pick portably across rebuilds: the edge's stable reference
// key (rebuildRim preserves lineage on every untouched edge), its two endpoint POINTS (the
// geometric fallback — the runout weld's assembleBody relineages every edge by face provenance,
// so a later pick's key does not survive that rebuild while its untouched geometry does), its
// radius, and its weld mode.
type cylCylSeamOpPick struct {
	key        []byte
	start, end math.Point3
	r          float64
	line       bool
}

// cylCylSeamArmEdge dispatches a CLOSED Cylinder∧Cylinder seam edge to the canal band builder.
// handled=true ONLY when the band was built, so every decline falls through to the byte-identical
// curvedAdjacentError refusal (do-no-harm) — the same contract as the elliptic closed-rim arm.
func cylCylSeamArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if p.varying() {
		return edgeFillet{}, false // constant radius only
	}
	if e.StartVertex().ID() != e.EndVertex().ID() {
		return cylCylParallelValleyArmEdge(body, e, p) // open seam: the equal-parallel concave LINE family
	}
	wrapF, otherF, ok := cylCylSeamHosts(e)
	if !ok {
		return edgeFillet{}, false
	}
	rim, surf, reason := solveCylCylSeamRim(body, e, wrapF, otherF, p.r0)
	if reason != "" {
		return edgeFillet{}, false // decline → the flat refusal, byte-identical
	}
	faces := e.Faces()
	band := &cylCylSeamBand{Surface: surf, rim: rim, r: p.r0}
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: band}, true
}

// cylCylParallelValleyArmEdge builds the exact CYLINDER arm of a CONCAVE equal-radius
// parallel-axis Cylinder∧Cylinder LINE seam (OCCT blend simple/P1 P6, complex/E8 class: two fused
// parallel bosses whose valley seam is a straight ruling). The offset cylinders of two parallel
// axes intersect in RULINGS, so the ball-centre spine is a LINE and the band an EXACT right
// circular cylinder — equalParallelCylMiterArm's concave branch (R+r ∩ R+r, the P2/P3 miter-arm
// derivation) reused verbatim, dispatched here for the corner-free seam pick. The arm then rides
// the EXISTING single-arm runout weld (both seam ends terminate on capping planes), exactly as a
// convex Plane∧Cylinder line arm does. The CONVEX equal-parallel edge stays with
// cylCylMiterArmEdge (P5) byte-identically: this branch is concave-only.
func cylCylParallelValleyArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	if !isCylCylValleyLineSeam(e) {
		return edgeFillet{}, false
	}
	if !cylCylValleyEndsAreTerminal(e) {
		return edgeFillet{}, false // an INTERIOR cap end (P6 class) — the runout weld would chop the continuing wall
	}
	arm, ok := parallelValleyArmCylinder(e, p.r0, opstol.ForBody(body))
	if !ok {
		return edgeFillet{}, false // non-parallel axes / offset circles miss — keep the flat refusal
	}
	if !cylCylValleyFeetOnHosts(e, arm.(geom.Cylinder)) {
		return edgeFillet{}, false // a foot ruling lands past a host's split ruling (E8 class) — neighbour recede missing
	}
	ef, _ := curvedArmEdgeFillet(e, arm, true)
	ef.armConcave = true // the ball rolls in the reentrant VOID: the runout weld winds the band accordingly
	if !cylCylValleyEndsAreCapped(ef) {
		return edgeFillet{}, false // no clean single-plane cap at an end — keep the flat refusal
	}
	ef.armSurface = &cylCylSeamBand{Surface: arm, r: p.r0, line: true}
	return ef, true
}

// isCylCylValleyLineSeam is the shape gate of the valley-line family: a straight CONCAVE seam
// bounded by exactly two cylinder walls.
func isCylCylValleyLineSeam(e *topo.Edge) bool {
	faces := e.Faces()
	if len(faces) != 2 {
		return false
	}
	_, okA := faces[0].Geometry().(geom.Cylinder)
	_, okB := faces[1].Geometry().(geom.Cylinder)
	_, isLine := e.Geometry().(geom.LineSegment)
	return okA && okB && isLine && ClassifyEdgeConvexity(e) == EdgeConcave
}

// parallelValleyArmCylinder is the exact rolling-ball cylinder arm of a CONCAVE parallel-axis
// Cylinder∧Cylinder valley line, radii equal OR NOT: each host's ball-centre locus is its own
// ρ = R + ε·r coaxial offset cylinder (cylinderOffsetRadius's concave branch — tangent-external),
// the two offsets meet in a pair of rulings (intersectCoplanarCircles handles r1≠r2, the closure-§2
// general branch), and the arm axis is the ruling nearer the picked edge. It generalizes
// equalParallelCylMiterArm's concave case past its equal-radius guard, which E8's R22.5∧R10 valley
// fails; the general solver lives HERE because the miter-arm file is owned by another wave cluster.
// ok=false for non-parallel axes, an unreadable wall sign, or offset circles that miss.
func parallelValleyArmCylinder(e *topo.Edge, r float64, res opstol.Resolution) (geom.Surface, bool) {
	faces := e.Faces()
	cA := faces[0].Geometry().(geom.Cylinder)
	cB := faces[1].Geometry().(geom.Cylinder)
	d := cA.AxisDir.AsVector()
	if stdmath.Abs(stdmath.Abs(float64(cA.AxisDir.Dot(cB.AxisDir)))-1) > res.Weld() {
		return nil, false // non-parallel axes — the line-spine premise does not hold
	}
	rhoA, okA := cylinderOffsetRadius(e, cA, r, EdgeConcave)
	rhoB, okB := cylinderOffsetRadius(e, cB, r, EdgeConcave)
	if !okA || !okB {
		return nil, false
	}
	plus, minus, ok := intersectCoplanarCircles(cA.Origin, rhoA, cB.Origin, rhoB, d, res)
	if !ok {
		return nil, false
	}
	base := nearerRuling(e, plus, minus)
	arm, err := geom.NewCylinderWithRef(base, d, base.VectorTo(edgeMidpoint(e)), r)
	return arm, err == nil
}

// cylCylValleyEndsAreTerminal reports whether BOTH host walls STOP at both seam ends: no vertex
// of either wall face extends past the end plane along the seam direction. A seam ending on an
// INTERIOR cap — a host continues past the cap plane (P6's overhung boss: s1 spans z∈[0,200]
// around a seam ending at 150) — fails this, and the single-arm runout weld this arm rides would
// trim the CONTINUING wall at the cap plane (measured on P6: a watertight solid 29.0% under
// OCCT's area — a wrong solid, so the claim is declined; interior-cap termination is the
// runout-retrim family's missing capability, not this arm's). The tolerance is a fraction of the
// seam length — the walls' own cap vertices sit exactly at the end planes.
func cylCylValleyEndsAreTerminal(e *topo.Edge) bool {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	along, err := math.UnitVector3FromVector(a.VectorTo(b))
	if err != nil {
		return false
	}
	lo, hi := float64(a.AsVector().Dot(along.AsVector())), float64(b.AsVector().Dot(along.AsVector()))
	tol := 1e-6 * (hi - lo)
	for _, f := range e.Faces() {
		for _, v := range faceVertexSpan(f, along) {
			if v < lo-tol || v > hi+tol {
				return false // this wall continues past a seam end — interior cap
			}
		}
	}
	return true
}

// faceVertexSpan projects every vertex of the face's boundary onto dir (the seam direction).
func faceVertexSpan(f *topo.Face, dir math.UnitVector3) []float64 {
	seen := map[uint64]bool{}
	var out []float64
	for _, fe := range f.Edges() {
		for _, v := range fe.Vertices() {
			if !seen[v.ID()] {
				seen[v.ID()] = true
				out = append(out, float64(v.Point().AsVector().Dot(dir.AsVector())))
			}
		}
	}
	return out
}

// cylCylValleyFeetOnHosts requires each host's rolling-ball foot RULING to lie inside that host
// face's OWN trimmed region, probed at the seam's mid height. A foot past the host's split ruling
// — complex/E8: the R10 wall's foot sits 0.03125 rad beyond its +x split, on the same-surface
// NEIGHBOUR face — would need that neighbour receded too, which the single-arm runout weld does
// not do: it clamps at the face boundary and ships strips totalling +19.98 area vs OCCT 8.0.0's
// per-face receipts (a wrong solid the per-face oracle caught). Declined honestly instead; the
// neighbour-recede (or same-surface sew) is the missing capability, reported for the retrim family.
func cylCylValleyFeetOnHosts(e *topo.Edge, arm geom.Cylinder) bool {
	mid := edgeMidpoint(e)
	for _, f := range e.Faces() {
		cyl := f.Geometry().(geom.Cylinder)
		if !cylinderFaceCoversAzimuthOf(f, cyl, valleyFootAtHeight(cyl, arm, mid)) {
			return false
		}
	}
	return true
}

// valleyFootAtHeight is the rolling-ball contact ruling's point on one host at the axial height of
// p: the radial projection of the arm axis onto the host wall.
func valleyFootAtHeight(host, arm geom.Cylinder, p math.Point3) math.Point3 {
	d := host.AxisDir.AsVector()
	q := host.Origin.TranslateBy(d.Scale(host.Origin.VectorTo(p).Dot(d)))
	u := q.VectorTo(arm.Origin)
	radial := u.Sub(d.Scale(u.Dot(d)))
	dir, err := math.UnitVector3FromVector(radial)
	if err != nil {
		return q
	}
	return q.TranslateBy(dir.AsVector().Scale(math.Scalar(host.Radius)))
}

// cylinderFaceCoversAzimuthOf reports whether the face's angular extent about its own axis covers
// the probe point's azimuth: boundary edges are densely sampled, their azimuths unwrapped about
// the probe's (so the periodic branch cut cannot split the span), and the probe (at offset 0) must
// sit inside the sampled [min,max] — trivially true for a full-circumference wall (span ≈ 2π).
// The axial extent needs no check here: the probe rides the seam's own mid height, which the host
// face covers by construction (the seam is one of its boundary edges).
func cylinderFaceCoversAzimuthOf(f *topo.Face, cyl geom.Cylinder, p math.Point3) bool {
	e2 := cyl.AxisDir.AsVector().Cross(cyl.Ref.AsVector())
	azimuth := func(q math.Point3) float64 {
		u := cyl.Origin.VectorTo(q)
		return stdmath.Atan2(float64(u.Dot(e2)), float64(u.Dot(cyl.Ref.AsVector())))
	}
	phi0 := azimuth(p)
	lo, hi := 0.0, 0.0
	for _, be := range f.Edges() {
		pl, err := geom.PolylineFromCurve3(be.Geometry(), cylCylSeamWindingSamples)
		if err != nil {
			continue // a degenerate boundary contributes no azimuth extent
		}
		for _, q := range pl.Vertices {
			off := probe.WrapPi(azimuth(q) - phi0)
			lo, hi = stdmath.Min(lo, off), stdmath.Max(hi, off)
		}
	}
	if hi-lo >= 2*stdmath.Pi-1e-6 {
		return true // full-circumference wall — every azimuth is covered
	}
	return lo < 0 && hi > 0 // the probe azimuth (offset 0) sits strictly inside the sampled span
}

// cylCylValleyEndsAreCapped re-applies the runout classifier's own admission — a clean trihedral
// single-plane cap at BOTH ends (cappingFaceAtFarVertex) — at ARM time, because the wrapped
// payload bypasses isSingleArmRunout's plain-arm checks on the router side.
func cylCylValleyEndsAreCapped(ef edgeFillet) bool {
	filleted := map[uint64]bool{ef.edge.ID(): true}
	for _, v := range ef.edge.Vertices() {
		if _, ok, _ := cappingFaceAtFarVertex(v, ef, filleted); !ok {
			return false
		}
	}
	return true
}

// cylCylSeamHosts splits a closed seam edge's two faces into the WRAP host (the cylinder whose
// axis the loop encircles — the receded side carrying a seam ruling through the rim vertex) and
// the other host (whose loop keeps the seam as a closed single-edge boundary, replaced by the
// contact rail). ok=false unless both faces are cylinders and the loop encircles EXACTLY one axis.
func cylCylSeamHosts(e *topo.Edge) (wrapF, otherF *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, false
	}
	cA, okA := faces[0].Geometry().(geom.Cylinder)
	cB, okB := faces[1].Geometry().(geom.Cylinder)
	if !okA || !okB {
		return nil, nil, false
	}
	wrapsA := seamLoopWrapsAxis(e, cA)
	wrapsB := seamLoopWrapsAxis(e, cB)
	if wrapsA == wrapsB {
		return nil, nil, false // neither or both encircled: not the transversal-crossing wrap class
	}
	if wrapsA {
		return faces[0], faces[1], true
	}
	return faces[1], faces[0], true
}

// cylCylSeamWindingSamples densely polygonizes the seam for the winding-number test; the seam is a
// low-degree imported BSpline, so 256 chords resolve its azimuth walk far below π per step.
const cylCylSeamWindingSamples = 256

// seamLoopWrapsAxis reports whether the closed seam loop encircles the cylinder's axis: the summed
// azimuth increments of a dense polygonization total ±2π for a wrapping loop and 0 for a
// non-wrapping one (winding number, exact to the ±π-per-chord branch cut).
func seamLoopWrapsAxis(e *topo.Edge, cyl geom.Cylinder) bool {
	pl, err := geom.PolylineFromCurve3(e.Geometry(), cylCylSeamWindingSamples)
	if err != nil {
		return false
	}
	d := cyl.AxisDir.AsVector()
	e1 := cyl.Ref.AsVector()
	e2 := d.Cross(e1)
	total, prev := 0.0, 0.0
	for i, p := range pl.Vertices {
		u := cyl.Origin.VectorTo(p)
		phi := stdmath.Atan2(float64(u.Dot(e2)), float64(u.Dot(e1)))
		if i > 0 {
			total += probe.WrapPi(phi - prev)
		}
		prev = phi
	}
	return stdmath.Abs(total) > stdmath.Pi // a wrapping loop sums to ±2π, a non-wrapping one to 0
}

// solveCylCylSeamRim solves one closed seam into its rimFillet rebuild payload: the exact spine,
// the walk direction (band normal outward), the adaptive station density, the loft, its two rail
// iso-curves, and the receded wrap-host seam. A non-empty reason names the obstruction with the
// offending values; the band is then not built (do-no-harm).
func solveCylCylSeamRim(body *topo.Body, e *topo.Edge, wrapF, otherF *topo.Face, r float64) (*rimFillet, geom.BSplineSurface, string) {
	sp, ok := newCylCylSeamSpine(e, wrapF, otherF, r)
	if !ok {
		return nil, geom.BSplineSurface{}, "cyl∧cyl seam: spine parameters unresolved (tangent seam, parallel axes, or consumed host)"
	}
	res := opstol.ForBody(body)
	st, surf, reason := resolveCylCylSeamStations(sp, e.StartVertex().Point(), res)
	if reason != "" {
		return nil, geom.BSplineSurface{}, reason
	}
	return assembleCylCylSeamRim(e, wrapF, otherF, sp, st, surf)
}

// resolveCylCylSeamStations fixes the walk direction once (band normal outward, like the elliptic
// rim), then doubles the station count until the measured between-station envelope error is within
// the model-relative bound. Still over bound at the cap means the seam is genuinely unresolvable
// at this radius — honest-reject, never loft a band with a known gap.
func resolveCylCylSeamStations(sp cylCylSeamSpine, rimP math.Point3, res opstol.Resolution) (cylCylSeamStations, geom.BSplineSurface, string) {
	phi0 := sp.wrapAzimuthAt(rimP)
	t0 := sp.wrapAxialAt(rimP)
	dir, ok := cylCylSeamWalkDirection(sp, phi0, t0, res)
	if !ok {
		return cylCylSeamStations{}, geom.BSplineSurface{}, "cyl∧cyl seam: walk-direction trial loft declined (open or pinched offset spine)"
	}
	bound := cylCylSeamEnvelopeCoef * res.Weld()
	for n := cylCylSeamStationsMin; n <= cylCylSeamStationsMax; n *= 2 {
		st, ok := sp.closedStationsAt(phi0, dir, t0, n, res.Weld())
		if !ok {
			return cylCylSeamStations{}, geom.BSplineSurface{}, "cyl∧cyl seam: offset spine loses the crossing mid-walk (radius too large for the wrap)"
		}
		surf, err := geom.LoftCanalStations(st.centers, st.wrapFeet, st.otherFeet, sp.r, res.Weld())
		if err != nil {
			return cylCylSeamStations{}, geom.BSplineSurface{}, "cyl∧cyl seam loft: " + err.Error()
		}
		if sp.envelopeError(st, surf) <= bound {
			return st, surf, ""
		}
	}
	return cylCylSeamStations{}, geom.BSplineSurface{}, "cyl∧cyl seam: envelope error above bound at the station cap"
}

// cylCylSeamWalkDirection fixes the azimuth walk sense ONCE off a coarse trial loft: +1 when
// walking φ forward already gives the band a surface normal pointing OUT of the solid (the
// material-outward direction at mid-band is −σ·(Q−C)/r), −1 otherwise. Mirrors
// ellipticRimWalkDirection so the rebuild can add the band face un-reversed.
func cylCylSeamWalkDirection(sp cylCylSeamSpine, phi0, t0 float64, res opstol.Resolution) (float64, bool) {
	st, ok := sp.closedStationsAt(phi0, 1, t0, cylCylSeamStationsMin, res.Weld())
	if !ok {
		return 0, false
	}
	surf, err := geom.LoftCanalStations(st.centers, st.wrapFeet, st.otherFeet, sp.r, res.Weld())
	if err != nil {
		return 0, false
	}
	flip, ok := cylCylSeamBandOutward(sp, st, surf)
	if !ok {
		return 0, false
	}
	if flip {
		return -1, true
	}
	return 1, true
}

// cylCylSeamBandOutward reports whether the trial band's surface normal already points to the
// solid's outside at the mid station, and if not that the walk must be flipped. The band's
// material-outward direction is −σ·(Q−C)/r: a convex band (σ=−1, ball inside the material) faces
// away from the centre, a concave one (σ=+1, ball in the void) toward it. The decision dots two
// UNIT vectors, so its degeneracy floor is dimensionless (ADR-0042).
func cylCylSeamBandOutward(sp cylCylSeamSpine, st cylCylSeamStations, surf geom.BSplineSurface) (flip, ok bool) {
	j := len(st.centers) / 2
	vp := spineChordParams(st.centers)
	q := surf.PointAt(0.5, vp[j])
	want, err := math.UnitVector3FromVector(st.centers[j].VectorTo(q).Scale(math.Scalar(-sp.sigma)))
	if err != nil {
		return false, false // probe point coincides with the ball centre — no outward direction
	}
	n := surf.NormalAt(0.5, vp[j])
	dot := float64(n.Dot(want.AsVector()))
	if stdmath.Abs(dot) < cylCylSeamParallelSinTol {
		return false, false // degenerate normal at the probe station — decline
	}
	return dot < 0, true
}

// assembleCylCylSeamRim packages the band into the closed-rim rebuild payload: the two rails read
// back as the loft's OWN u-isocurves (face boundary and face surface agree exactly), the receded
// wrap-host seam ruling (wallSeam), and the seam cross-section's on-arc midpoint. It asserts the
// wrap rail's seam vertex lands on the host's own seam ruling — the invariant that keeps the
// re-aimed wall seam ON the host (rebuildRim's wallSeamCurve is the ruling's retained sub-span).
func assembleCylCylSeamRim(e *topo.Edge, wrapF, otherF *topo.Face, sp cylCylSeamSpine, st cylCylSeamStations, surf geom.BSplineSurface) (*rimFillet, geom.BSplineSurface, string) {
	rimV := e.StartVertex()
	seamEdge, bottomV := wallSeam(wrapF, e, rimV)
	if seamEdge == nil {
		return nil, geom.BSplineSurface{}, "cyl∧cyl seam: wrap host has no seam ruling at the rim vertex to recede"
	}
	wrapRail, err1 := geom.SurfaceIsoCurve(surf, true, 0)
	otherRail, err2 := geom.SurfaceIsoCurve(surf, true, 1)
	if err1 != nil || err2 != nil {
		return nil, geom.BSplineSurface{}, "cyl∧cyl seam: band rail extraction declined"
	}
	c0 := st.centers[0]
	bis, err := math.UnitVector3FromVector(c0.VectorTo(st.wrapFeet[0]).Add(c0.VectorTo(st.otherFeet[0])))
	if err != nil {
		return nil, geom.BSplineSurface{}, "cyl∧cyl seam: station-0 feet are antipodal — no seam arc midpoint"
	}
	rim := &rimFillet{
		cyl: wrapF, cap: otherF, rimEdge: e, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: wrapRail, capTan: otherRail, band: surf, r: sp.r,
		seamMid: c0.TranslateBy(bis.AsVector().Scale(math.Scalar(sp.r))),
		concave: sp.sigma > 0,
	}
	return rim, surf, ""
}
