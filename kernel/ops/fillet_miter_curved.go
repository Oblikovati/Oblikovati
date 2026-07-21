// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-face miter seam (curved-miter-seam-derivation.md, families B & C). Two picked edges that
// SHARE a face meet at a valence-3 vertex; the third edge stays sharp. Each blends with the SAME
// rolling-ball radius r, so the two constant-r arm tubes cannot be joined by a corner sphere — they
// MUTUALLY TRIM along one seam curve, the equal-r bisector arm₁∩arm₂. When a contact face is a
// CYLINDER (not planar) one arm is an exact torus and the other an exact cylinder, and the seam is a
// torus∩cylinder quartic — no analytic primitive, so it is returned as a sampled polyline exactly
// like the planar sampleMiterSeam. sampleCurvedMiterSeam is the sibling sampler; sampleMiterSeam
// (the all-planar mirror-plane case) is left byte-identical.

// miterSeamNewtonIters caps the projector's Levenberg–Marquardt refinement; each seam point is
// projected onto BOTH arm tubes (dist-to-spine = r) from a branch-biased seed. The cap is generous
// because rejected (λ-growing) trial steps count against it near the quartic's tangent waist.
const miterSeamNewtonIters = 200

// curvedMiterArms is one miter corner's two equal-radius rolling-ball arms as the exact analytic
// surfaces whose intersection is the seam: exactly one torus (a circle-edge Cyl∧Plane arm) and one
// cylinder (a line-edge Plane∧Plane / equal-parallel Cyl∧Cyl / axis-∥ Cyl∧Plane arm). ok=false when
// the pair is anything else (sphere/torus/BSpline face, cone/BSpline outer, non-equal cyl∩cyl) — the
// honest-reject boundary the caller floors on (do-no-harm).
type curvedMiterArms struct {
	tor    geom.Torus
	cyl    geom.Cylinder
	torIdx int // index in ps of the torus arm's pick
	cylIdx int // index in ps of the cylinder arm's pick
}

// buildCurvedMiterArms builds both miter edges' exact rolling-ball arm surfaces and returns them as
// the torus∩cylinder pair the seam sampler needs — ok=false unless there is EXACTLY one torus and
// one cylinder arm (the equal-r bisector covers only that pairing; every sphere/torus/BSpline face
// and non-equal cyl∩cyl arm keeps flooring). Each arm is convex-external only.
func buildCurvedMiterArms(ps []filletPick, r float64, res Resolution) (curvedMiterArms, bool) {
	arms := curvedMiterArms{torIdx: -1, cylIdx: -1}
	for i, p := range ps {
		s, ok := miterEdgeArmSurface(p.edge, r, res)
		if !ok {
			return curvedMiterArms{}, false
		}
		switch a := s.(type) {
		case geom.Torus:
			arms.tor, arms.torIdx = a, i
		case geom.Cylinder:
			arms.cyl, arms.cylIdx = a, i
		}
	}
	return arms, arms.torIdx >= 0 && arms.cylIdx >= 0
}

// miterEdgeArmSurface builds one miter edge's exact rolling-ball arm surface (a geom.Torus for a
// circle-edge Cyl∧Plane rim, a geom.Cylinder for a Plane∧Plane line edge or an equal-radius
// parallel-axis Cyl∧Cyl edge). Convex-external only; ok=false for any other host pair or a concave
// rim — the honest-reject boundary.
func miterEdgeArmSurface(e *topo.Edge, r float64, res Resolution) (geom.Surface, bool) {
	if ClassifyEdgeConvexity(e) != EdgeConvex {
		return nil, false
	}
	if cyl, pl, ok := cylinderPlaneEdge(e); ok {
		return cylinderPlaneMiterArm(e, cyl, pl, r, res)
	}
	if a, b, nA, nB, err := edgePlanarFaces(e); err == nil {
		_, _ = a, b
		return planarMiterArmCylinder(e, nA, nB, r)
	}
	return equalParallelCylMiterArm(e, r, res)
}

// cylinderPlaneMiterArm builds the arm of a Cylinder∧Plane miter edge: an exact torus (axis ⊥ plane,
// circle edge) or an exact cylinder (axis ∥ plane, line edge), reusing torusArmSurface /
// cylinderArmSurface with the plane host's material-outward normal.
func cylinderPlaneMiterArm(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) (geom.Surface, bool) {
	outwardN, ok := planeHostNormal(e, pl)
	if !ok {
		return nil, false
	}
	switch classifyCurvedArm(cyl, pl, res) {
	case armTorus:
		cylFace, planeFace := cylinderPlaneHostFaces(e, cyl, pl)
		if t, ok := torusArmSurface(cyl, pl, cylFace, planeFace, edgeMidpoint(e), outwardN, r, res); ok {
			return t, true
		}
	case armCylinder:
		if c, ok := cylinderArmSurface(e, cyl, pl, outwardN, r); ok {
			return c, true
		}
	}
	return nil, false
}

// planarMiterArmCylinder builds the rolling-ball cylinder arm of a Plane∧Plane miter edge: the
// constant-r cylinder whose axis is the edge line offset into the material by offDir·r (the same
// centre line the planar edge fillet's cyl uses), so the arm face and seam agree.
func planarMiterArmCylinder(e *topo.Edge, nA, nB math.Vector3, r float64) (geom.Surface, bool) {
	offDir := nA.Add(nB).Scale(math.Scalar(-1 / (1 + float64(nA.Dot(nB)))))
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return nil, false
	}
	base := e.StartVertex().Point().TranslateBy(offDir.Scale(math.Scalar(r)))
	arm, err := geom.NewCylinderWithRef(base, axis.AsVector(), nA.Negate(), r)
	return arm, err == nil
}

// miterHasCurvedContact reports whether a miter corner has a CYLINDER (non-planar) contact face — the
// shared face itself, or either edge's outer face — routing it to the curved seam path (families B/C).
// An all-planar miter returns false and keeps the byte-identical planar mirror-plane path.
func miterHasCurvedContact(ps []filletPick, shared *topo.Face) bool {
	if _, planar := shared.Geometry().(geom.Plane); !planar {
		return true
	}
	for _, p := range ps {
		if outer := otherFace(p.edge, shared); outer != nil {
			if _, planar := outer.Geometry().(geom.Plane); !planar {
				return true
			}
		}
	}
	return false
}

// solveCurvedMiter builds a curved-contact miter corner (families B/C): it constructs the two exact
// arm surfaces (a torus + a cylinder), the torus∩cylinder seam, and the corner-ball centre, and packs
// them into the cornerMiter's curved slot for curvedMiterBody to weld. It floors (honest reject) when
// the arm pair is not the supported torus+cylinder or the seam cannot close — do-no-harm, never a
// partial body.
func solveCurvedMiter(v *topo.Vertex, ps []filletPick, shared *topo.Face, r float64) (*cornerMiter, error) {
	res := miterResolution(v, ps)
	arms, ok := buildCurvedMiterArms(ps, r, res)
	if !ok {
		return nil, fmt.Errorf("fillet: curved miter arms unsupported at vertex %d (need one torus + one cylinder equal-r arm; radius %g)", v.ID(), r)
	}
	torOuter := otherFace(ps[arms.torIdx].edge, shared)
	if torOuter == nil {
		return nil, fmt.Errorf("fillet: curved miter torus edge %d has no outer face opposite the shared face", ps[arms.torIdx].edge.ID())
	}
	seam, center, ok := sampleCurvedMiterSeam(arms, shared, torOuter, v, r, res)
	if !ok {
		return nil, fmt.Errorf("fillet: curved miter seam did not close at vertex %d (radius %g)", v.ID(), r)
	}
	curved := &curvedMiterCorner{arms: arms, torEdge: ps[arms.torIdx].edge, cylEdge: ps[arms.cylIdx].edge, shared: shared, center: center}
	return &cornerMiter{vertex: v, shared: shared, sBot: seam[len(seam)-1], seam: seam, curved: curved}, nil
}

// miterResolution builds the model-relative tolerance for a miter corner from its two edges' vertices
// and the corner vertex (ADR-0042) — the sampler/projector tolerances scale with the corner, never a
// bare epsilon.
func miterResolution(v *topo.Vertex, ps []filletPick) Resolution {
	pts := []math.Point3{v.Point()}
	for _, p := range ps {
		pts = append(pts, p.edge.StartVertex().Point(), p.edge.EndVertex().Point())
	}
	return ResolutionForPoints(pts)
}

// sampleCurvedMiterSeam samples the seam arm₁∩arm₂ (a torus∩cylinder quartic) as a []math.Point3
// polyline from sTop (the shared-face tangent point of the corner ball) to sBot (the sharp-edge
// common tangent point), each point lying on BOTH arm tubes so the two arm faces weld to the same
// chords (watertight). It walks the straight lerp sTop→sBot and projects each station onto the seam
// (Newton on dist-to-spine=r for both arms); the endpoints are the corner ball's contacts with the
// shared face and the sharp edge. Returns ok=false on any projector or endpoint decline.
func sampleCurvedMiterSeam(arms curvedMiterArms, shared, torOuter *topo.Face, v *topo.Vertex, r float64, res Resolution) ([]math.Point3, math.Point3, bool) {
	center, ok := miterCornerBallCenter(arms, v.Point(), res)
	if !ok {
		return nil, math.Point3{}, false
	}
	sTop, sBot, ok := curvedSeamEndpoints(arms, center, shared, torOuter, v.Point(), r, res)
	if !ok {
		return nil, math.Point3{}, false
	}
	seam, ok := walkCurvedSeam(arms, r, center, sTop, sBot, v.Point(), res)
	return seam, center, ok
}

// curvedSeamEndpoints are the seam's two endpoints (curved-miter-closure-derivation §1b — they are
// ASYMMETRIC, not the two ±r·n̂ mutual-tangency points the first derivation assumed): sTop is the
// tube∩tube∩SHARED-face vertex (the mutual-tangency point on the shared side, where torus and cylinder
// normals coincide), sBot the tube∩tube∩(torus-outer-host) vertex where the seam exits through the
// torus arm's outer plane (P5: sTop=(48.148,0.034,145) on the shared wall, sBot=(53.332,5.124,150) on
// the top plane). ok=false when either endpoint declines.
func curvedSeamEndpoints(arms curvedMiterArms, center math.Point3, shared, torOuter *topo.Face, vp math.Point3, r float64, res Resolution) (math.Point3, math.Point3, bool) {
	sTop, ok := miterSeamTop(arms, center, shared, r)
	if !ok {
		return math.Point3{}, math.Point3{}, false
	}
	sBot, ok := miterSeamBottom(arms, torOuter, vp, res)
	if !ok {
		return math.Point3{}, math.Point3{}, false
	}
	return sTop, sBot, true
}

// miterSeamTop is sTop: the seam's shared-face endpoint, the mutual-tangency point m*±r·n̂ (n̂ ⟂ both
// arm spine tangents at the corner-ball centre) that lands ON the shared face — the tube∩tube∩shared
// vertex. Of the two tangency candidates it keeps the one closest to the shared surface. ok=false when
// the two spine tangents are parallel (no distinct endpoints — a degenerate/collinear miter).
func miterSeamTop(arms curvedMiterArms, center math.Point3, shared *topo.Face, r float64) (math.Point3, bool) {
	n, ok := seamEndpointNormal(arms, center)
	if !ok {
		return math.Point3{}, false
	}
	e1 := center.TranslateBy(n.AsVector().Scale(math.Scalar(r)))
	e2 := center.TranslateBy(n.AsVector().Scale(math.Scalar(-r)))
	if distanceToSurface(shared, e1) <= distanceToSurface(shared, e2) {
		return e1, true
	}
	return e2, true
}

// miterSeamBottom is sBot: the torus∩cylinder point on the TORUS arm's outer host (a plane, e.g. P5's
// top plane z=150). Per curved-miter-closure §1b sBot is a tube∩tube∩outer-host vertex, NOT a second
// mutual-tangency point: the torus is tangent to its outer plane along the contact circle (major circle
// pushed minor·n̂_out onto the plane), and sBot is where that circle meets the cylinder arm, taking the
// crossing nearer the corner vertex vp (the physical branch). ok=false when the outer host is not a
// plane ⊥ the cylinder axis, or the contact circle misses the cylinder.
func miterSeamBottom(arms curvedMiterArms, torOuter *topo.Face, vp math.Point3, res Resolution) (math.Point3, bool) {
	pl, ok := torOuter.Geometry().(geom.Plane)
	if !ok {
		return math.Point3{}, false
	}
	nOut := outwardPlaneNormal(torOuter, pl)
	if stdmath.Abs(float64(arms.cyl.AxisDir.AsVector().Dot(nOut)))-1 < -res.Weld() {
		return math.Point3{}, false // cylinder axis not ⊥ the outer plane — outside the analytic scope
	}
	contactC := arms.tor.Center.TranslateBy(nOut.Scale(math.Scalar(arms.tor.MinorRadius)))
	cylC := cylinderBallCenter(arms.cyl, contactC) // cyl axis ∩ contact plane (axis ⊥ plane through contactC)
	p1, p2, ok := intersectCoplanarCircles(contactC, arms.tor.MajorRadius, cylC, arms.cyl.Radius, nOut, res)
	if !ok {
		return math.Point3{}, false
	}
	if p1.DistanceTo(vp) <= p2.DistanceTo(vp) {
		return p1, true
	}
	return p2, true
}

// seamEndpointNormal is n̂ = unit(t̂₁ × t̂₂), t̂ᵢ the two arm spines' tangents at the corner-ball centre
// (torus major-circle tangent AxisDir×radial; cylinder axis). n̂ is ⟂ both spine tangents, the
// direction along which the seam's two tangential endpoints sit at ±r from m*. Declines when a spine
// tangent is undefined or the two are parallel.
func seamEndpointNormal(arms curvedMiterArms, center math.Point3) (math.UnitVector3, bool) {
	t1, ok := torusSpineTangent(arms.tor, center)
	if !ok {
		return math.UnitVector3{}, false
	}
	t2 := arms.cyl.AxisDir
	n, err := math.UnitVector3FromVector(t1.AsVector().Cross(t2.AsVector()))
	return n, err == nil
}

// distanceToSurface is the distance from p to its closest point on s (the shared-face-contact test).
func distanceToSurface(f *topo.Face, p math.Point3) float64 {
	_, _, foot := geom.ClosestPointOnSurface(f.Geometry(), p)
	return float64(foot.DistanceTo(p))
}

// walkCurvedSeam emits k+1 seam points from sTop to sBot: the two exact tangential endpoints plus
// interior stations. Each interior seed is the straight lerp sTop→sBot nudged toward the corner
// vertex vp (biasing onto the PHYSICAL branch — the torus∩cylinder quartic has two arcs joining the
// endpoints, and the material seam is the one wrapping the removed corner), then projected onto the
// seam. k follows the planar sampler (the sTop→sBot wedge over the chords-per-turn budget, ≥4).
func walkCurvedSeam(arms curvedMiterArms, r float64, center, sTop, sBot, vp math.Point3, res Resolution) ([]math.Point3, bool) {
	k := curvedSeamChordCount(center, sTop, sBot)
	thetaTop := cylinderAngleOf(arms.cyl, sTop)
	thetaBot := seamUnwrapAngle(cylinderAngleOf(arms.cyl, sBot), thetaTop)
	out := make([]math.Point3, k+1)
	out[0], out[k] = sTop, sBot
	for j := 1; j < k; j++ {
		theta := thetaTop + (thetaBot-thetaTop)*float64(j)/float64(k)
		p, ok := seamPointAtCylAngle(arms, r, theta, center, vp, res)
		if !ok {
			return nil, false
		}
		out[j] = p
	}
	return out, true
}

// seamPointAtCylAngle is the seam point at a given cylinder cross-section angle θ: along the cylinder
// tube's ruling at θ (a line parallel to the axis, ANCHORED at the corner-ball centre's axial station
// so the ±2r search brackets the seam) it finds where the torus-tube membership changes sign, bisects
// to that root, and polishes onto BOTH tubes. The two tubes' spines are tangent at the corner ball
// centre, so the ruling meets the seam on TWO petals (two axial roots); the root nearer the corner
// vertex vp is the physical (material) branch.
func seamPointAtCylAngle(arms curvedMiterArms, r float64, theta float64, center, vp math.Point3, res Resolution) (math.Point3, bool) {
	ref := arms.cyl.Ref.AsVector()
	bin := arms.cyl.AxisDir.AsVector().Cross(ref)
	radial := ref.Scale(math.Scalar(stdmath.Cos(theta))).Add(bin.Scale(math.Scalar(stdmath.Sin(theta))))
	anchor := cylinderBallCenter(arms.cyl, center) // axial station of the corner ball on the cylinder axis
	base := anchor.TranslateBy(radial.Scale(math.Scalar(r)))
	root, ok := seamAxialRoot(arms.tor, r, base, arms.cyl.AxisDir.AsVector(), vp)
	if !ok {
		return math.Point3{}, false
	}
	return projectOntoSeam(arms, r, root, res)
}

// seamAxialRoot walks the cylinder ruling base + t·dir over t∈[−2r, 2r] and returns the crossing of
// the torus-tube membership g(t) = dist(P, torus major circle) − r nearest the corner vertex vp (the
// physical petal). It samples the interval and bisects each sign change; ok=false when the ruling
// misses the torus tube entirely (no crossing — a degenerate/non-intersecting arm pair).
func seamAxialRoot(tor geom.Torus, r float64, base math.Point3, dir math.Vector3, vp math.Point3) (math.Point3, bool) {
	g := func(t float64) float64 {
		p := base.TranslateBy(dir.Scale(math.Scalar(t)))
		return torusTubeMembership(tor, r, p)
	}
	const steps = 48
	best, found := math.Point3{}, false
	prev := g(-2 * r)
	for i := 1; i <= steps; i++ {
		t0 := -2*r + 4*r*float64(i-1)/float64(steps)
		t1 := -2*r + 4*r*float64(i)/float64(steps)
		cur := g(t1)
		if prev == 0 || prev*cur < 0 {
			tr := bisectAxial(g, t0, t1)
			p := base.TranslateBy(dir.Scale(math.Scalar(tr)))
			if !found || p.DistanceTo(vp) < best.DistanceTo(vp) {
				best, found = p, true
			}
		}
		prev = cur
	}
	return best, found
}

// torusTubeMembership is dist(p, torus major circle) − minor: zero on the tube, signed inside/out.
func torusTubeMembership(tor geom.Torus, r float64, p math.Point3) float64 {
	axis := tor.AxisDir.AsVector()
	d := tor.Center.VectorTo(p)
	axial := float64(d.Dot(axis))
	rho := float64(d.Sub(axis.Scale(math.Scalar(axial))).Length())
	return stdmath.Hypot(rho-tor.MajorRadius, axial) - r
}

// bisectAxial refines a sign-change bracket [t0,t1] of g to a root by bisection.
func bisectAxial(g func(float64) float64, t0, t1 float64) float64 {
	f0 := g(t0)
	for i := 0; i < 60; i++ {
		mid := 0.5 * (t0 + t1)
		fm := g(mid)
		if f0*fm <= 0 {
			t1 = mid
		} else {
			t0, f0 = mid, fm
		}
	}
	return 0.5 * (t0 + t1)
}

// cylinderAngleOf is p's cross-section angle θ about the cylinder axis (atan2 in the Ref/binormal
// frame) — the seam's driving parameter.
func cylinderAngleOf(cyl geom.Cylinder, p math.Point3) float64 {
	ref := cyl.Ref.AsVector()
	bin := cyl.AxisDir.AsVector().Cross(ref)
	rel := cylinderBallCenter(cyl, p).VectorTo(p)
	return stdmath.Atan2(float64(rel.Dot(bin)), float64(rel.Dot(ref)))
}

// seamUnwrapAngle shifts a by ±2π so it is the value nearest ref within (ref−π, ref+π] — so the θ walk
// runs the short way across the seam span rather than the wrong way round the circle.
func seamUnwrapAngle(a, ref float64) float64 {
	for a-ref > stdmath.Pi {
		a -= 2 * stdmath.Pi
	}
	for a-ref <= -stdmath.Pi {
		a += 2 * stdmath.Pi
	}
	return a
}

// curvedSeamChordCount is the seam's chord count: the sTop→sBot wedge about the corner-ball centre
// over the 2π/filletChordsPerTurn budget, floored at 4 (the planar sampleMiterSeam count).
func curvedSeamChordCount(center, sTop, sBot math.Point3) int {
	a := center.VectorTo(sTop)
	b := center.VectorTo(sBot)
	cosw := math.Clamp(a.Dot(b)/(float64(a.Length())*float64(b.Length())), -1, 1)
	k := int(stdmath.Ceil(stdmath.Acos(cosw) / (2 * stdmath.Pi / filletChordsPerTurn)))
	if k < 4 {
		return 4
	}
	return k
}

// projectOntoSeam projects start onto the seam (an INTERIOR station where the two arm tubes cross)
// by Levenberg–Marquardt on the two equidistance residuals fᵢ = |P−mᵢ|²−r² (mᵢ the nearest spine
// point). The step δ = y₁·(P−m₁)+y₂·(P−m₂) solves the damped (JJᵀ+λI)y = −f; λ is grown on a
// rejected (cost-increasing) step and shrunk on an accepted one, so the near-tangent waist of the
// torus∩cylinder quartic — where plain Gauss–Newton oscillates — is damped and the iterate descends
// to the seam (both fᵢ ≈ 0) from a branch-biased seed. The seam ENDPOINTS are NOT projected here (the
// two tubes are TANGENT there); curvedSeamEndpoints supplies them directly. ok=false when a torus
// spine is undefined (P on the axis) or the iteration does not reach the seam.
func projectOntoSeam(arms curvedMiterArms, r float64, start math.Point3, res Resolution) (math.Point3, bool) {
	tol := res.Weld() * r * r
	p := start
	lambda := r * r * 1e-3
	f1, f2, ok := seamResiduals(arms, r, p)
	if !ok {
		return math.Point3{}, false
	}
	for it := 0; it < miterSeamNewtonIters; it++ {
		if stdmath.Abs(f1) <= tol && stdmath.Abs(f2) <= tol {
			return p, true
		}
		p, f1, f2, lambda, ok = seamLMIterate(arms, r, p, f1, f2, lambda)
		if !ok {
			return math.Point3{}, false
		}
	}
	if stdmath.Abs(f1) <= tol && stdmath.Abs(f2) <= tol {
		return p, true
	}
	return math.Point3{}, false
}

// seamResiduals returns the two equidistance residuals fᵢ = |P−mᵢ|²−r² and the half-gradients' base
// points at P (torus spine nearest point, cylinder axis foot). ok=false when the torus spine is
// undefined at P (P on the axis).
func seamResiduals(arms curvedMiterArms, r float64, p math.Point3) (float64, float64, bool) {
	m1, ok := armBallCenter(arms.tor, p)
	if !ok {
		return 0, 0, false
	}
	m2 := cylinderBallCenter(arms.cyl, p)
	return float64(m1.VectorTo(p).Dot(m1.VectorTo(p))) - r*r, float64(m2.VectorTo(p).Dot(m2.VectorTo(p))) - r*r, true
}

// seamLMIterate takes one Levenberg–Marquardt trial step: it damps JJᵀ by λ, and either accepts the
// step (cost fell — shrink λ) or rejects it and retries with a larger λ (holding P), returning the
// updated P, residuals, and λ. ok=false only when the torus spine goes undefined at the trial point.
func seamLMIterate(arms curvedMiterArms, r float64, p math.Point3, f1, f2, lambda float64) (math.Point3, float64, float64, float64, bool) {
	m1, ok := armBallCenter(arms.tor, p)
	if !ok {
		return p, f1, f2, lambda, false
	}
	g1, g2 := m1.VectorTo(p), cylinderBallCenter(arms.cyl, p).VectorTo(p)
	delta := seamLMStep(g1, g2, f1, f2, lambda)
	trial := p.TranslateBy(delta)
	nf1, nf2, ok := seamResiduals(arms, r, trial)
	if !ok {
		return p, f1, f2, lambda, false
	}
	if nf1*nf1+nf2*nf2 < f1*f1+f2*f2 {
		return trial, nf1, nf2, lambda * 0.5, true
	}
	return p, f1, f2, lambda * 4, true
}

// seamLMStep solves the damped normal equations (JJᵀ+λI)y = −f (J's rows the half-gradients g₁,g₂)
// and returns δ = y₁·g₁ + y₂·g₂. The +λI keeps the 2×2 invertible even where g₁∥g₂ (near tangency).
func seamLMStep(g1, g2 math.Vector3, f1, f2, lambda float64) math.Vector3 {
	a := float64(g1.Dot(g1)) + lambda
	b := float64(g1.Dot(g2))
	c := float64(g2.Dot(g2)) + lambda
	det := a*c - b*b
	y1 := (-c*f1 + b*f2) / det
	y2 := (b*f1 - a*f2) / det
	return g1.Scale(math.Scalar(y1)).Add(g2.Scale(math.Scalar(y2)))
}

// miterCornerBallCenter is the corner ball centre m* — the crossing of the two arm spines (the torus
// major circle ∩ the cylinder axis line), where the one ball tangent to both hosts sits. It solves
// the line∩circle in closed form (the line pierces the torus plane, or lies in it — a quadratic) and
// keeps the root nearest the corner vertex. ok=false when they do not cross (no equal-r bisector).
func miterCornerBallCenter(arms curvedMiterArms, vp math.Point3, res Resolution) (math.Point3, bool) {
	roots, ok := lineTorusSpineCrossings(arms.cyl, arms.tor)
	if !ok {
		return math.Point3{}, false
	}
	best, found := math.Point3{}, false
	tol := res.Weld() * (arms.tor.MajorRadius + arms.tor.MinorRadius)
	for _, p := range roots {
		m, mok := armBallCenter(arms.tor, p)
		if !mok || float64(m.DistanceTo(p)) > tol {
			continue // pierce point is off the torus major circle: not a spine crossing
		}
		if !found || p.DistanceTo(vp) < best.DistanceTo(vp) {
			best, found = p, true
		}
	}
	return best, found
}

// lineTorusSpineCrossings returns the point(s) where the cylinder axis line meets the torus major
// circle's plane: the single pierce point when the line crosses the plane, or the two |·−C|=R roots
// when the line lies in the plane (a quadratic in the axial parameter). ok=false when the line is
// parallel to but off the plane (no crossing).
func lineTorusSpineCrossings(cyl geom.Cylinder, tor geom.Torus) ([]math.Point3, bool) {
	o2, d2 := cyl.Origin, cyl.AxisDir.AsVector()
	n := tor.AxisDir.AsVector()
	q := tor.Center.VectorTo(o2)
	denom := float64(d2.Dot(n))
	if stdmath.Abs(denom) > sinFloor {
		t := -float64(q.Dot(n)) / denom
		return []math.Point3{o2.TranslateBy(d2.Scale(math.Scalar(t)))}, true
	}
	if stdmath.Abs(float64(q.Dot(n))) > sinFloor*(tor.MajorRadius+1) {
		return nil, false // line parallel to the torus plane but offset from it — no crossing
	}
	b := float64(q.Dot(d2))
	disc := b*b - (float64(q.Dot(q)) - tor.MajorRadius*tor.MajorRadius)
	if disc < 0 {
		return nil, false
	}
	s := stdmath.Sqrt(disc)
	return []math.Point3{o2.TranslateBy(d2.Scale(math.Scalar(-b + s))), o2.TranslateBy(d2.Scale(math.Scalar(-b - s)))}, true
}
