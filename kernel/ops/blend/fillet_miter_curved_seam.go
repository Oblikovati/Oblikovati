// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The torus∩cylinder miter seam sampler (families B/C) — split out of fillet_miter_curved.go (which
// now owns just the arm-pair RECOGNITION/dispatch) to keep both files under the 500-line/one-
// responsibility rule. Everything here is specific to the ONE-torus+ONE-cylinder arm pair; the
// torus∧torus family (D, fillet_miter_curved_torustorus.go) has its own closed-form sampler and does
// not call anything in this file.

// miterSeamNewtonIters caps the projector's Levenberg–Marquardt refinement; each seam point is
// projected onto BOTH arm tubes (dist-to-spine = r) from a branch-biased seed. The cap is generous
// because rejected (λ-growing) trial steps count against it near the quartic's tangent waist.
const miterSeamNewtonIters = 200

// sampleCurvedMiterSeam samples the seam arm₁∩arm₂ (a torus∩cylinder quartic) as a []math.Point3
// polyline from sTop (the shared-face tangent point of the corner ball) to sBot (the sharp-edge
// common tangent point), each point lying on BOTH arm tubes so the two arm faces weld to the same
// chords (watertight). It walks the straight lerp sTop→sBot and projects each station onto the seam
// (Newton on dist-to-spine=r for both arms); the endpoints are the corner ball's contacts with the
// shared face and the sharp edge. Returns ok=false on any projector or endpoint decline.
func sampleCurvedMiterSeam(arms curvedMiterArms, shared, torOuter *topo.Face, v *topo.Vertex, r float64, res opstol.Resolution) ([]math.Point3, math.Point3, bool) {
	center, ok := miterCornerBallCenter(arms, v.Point(), res)
	if !ok {
		return nil, math.Point3{}, false
	}
	sTop, sBot, ok := curvedSeamEndpoints(arms, center, shared, torOuter, v.Point(), r, res)
	if !ok {
		return nil, math.Point3{}, false
	}
	seam, ok := walkCurvedSeam(arms, r, center, sTop, sBot, res)
	return seam, center, ok
}

// curvedSeamEndpoints are the seam's two endpoints (curved-miter-closure-derivation §1b — they are
// ASYMMETRIC, not the two ±r·n̂ mutual-tangency points the first derivation assumed): sTop is the
// tube∩tube∩SHARED-face vertex (the mutual-tangency point on the shared side, where torus and cylinder
// normals coincide), sBot the tube∩tube∩(torus-outer-host) vertex where the seam exits through the
// torus arm's outer plane (P5: sTop=(48.148,0.034,145) on the shared wall, sBot=(53.332,5.124,150) on
// the top plane). ok=false when either endpoint declines.
func curvedSeamEndpoints(arms curvedMiterArms, center math.Point3, shared, torOuter *topo.Face, vp math.Point3, r float64, res opstol.Resolution) (math.Point3, math.Point3, bool) {
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

// miterSeamBottom is sBot: the torus∩cylinder point on the TORUS arm's OUTER host, dispatched on that
// host's surface type (curved-miter-closure §1b — sBot is a tube∩tube∩outer-host vertex, NOT a second
// mutual-tangency point). A PLANE outer host is P5's convex top-rim case; a CYLINDER outer host is
// M3/M9's concave base cove — the MIRROR of P5's roles (box-top shared / cyl-wall outer,
// fillet_miter_curved_cylouter.go). Both take the crossing nearer the corner vertex vp (the physical
// branch). ok=false for any other outer-host type (sphere/cone/torus/BSpline — outside the analytic scope).
func miterSeamBottom(arms curvedMiterArms, torOuter *topo.Face, vp math.Point3, res opstol.Resolution) (math.Point3, bool) {
	switch g := torOuter.Geometry().(type) {
	case geom.Plane:
		return miterSeamBottomPlane(arms, torOuter, g, vp, res)
	case geom.Cylinder:
		return miterSeamBottomCyl(arms, g, vp, res)
	default:
		return math.Point3{}, false
	}
}

// miterSeamBottomPlane is sBot for a PLANE torus-outer host (P5's convex top rim, e.g. z=150): the torus
// is tangent to its outer plane along the contact circle (major circle pushed minor·n̂_out onto the
// plane), and sBot is where that circle meets the cylinder arm, taking the crossing nearer vp. ok=false
// when the cylinder axis is not ⊥ the outer plane, or the contact circle misses the cylinder.
func miterSeamBottomPlane(arms curvedMiterArms, torOuter *topo.Face, pl geom.Plane, vp math.Point3, res opstol.Resolution) (math.Point3, bool) {
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
// interior stations. k follows the planar sampler (the sTop→sBot wedge over the chords-per-turn
// budget, ≥4).
//
// Each interior station's branch bias is the LAST ACCEPTED seam point, not the fixed corner vertex vp
// (A1, simple/W4). seamAxialRoot's "nearest to the bias point" choice between the quartic's two petals
// is only a reliable physical-branch discriminant NEAR vp — both endpoints are literally defined by
// proximity to it. Away from it, along a corner whose second pick spans nearly the model's full extent
// (W4: sBot ~3 units from vp), the WRONG petal can become the nearer one to the fixed vp mid-walk even
// though the right petal is still the physically continuous one; the walk then jumps petals for exactly
// one station, producing a chord wildly off both tubes (measured 0.17 off a torus of major 1.2 / minor
// 0.2 — an X discontinuity between adjacent samples otherwise smooth in Y and Z). Seeding each station
// from the PREVIOUS sample (sTop for the first) is the standard predictor-corrector continuation fix:
// the true seam moves continuously, so at a fine enough angular step the physical petal is always the
// one nearest wherever the walk already is, never a fixed faraway point. Falsified by mutation (bias
// reverted to the corner vertex, which is what this walk used to take as a parameter): W4's station 5
// jumps 0.36 along X from station 4 — invisible to the endpoint-only sTop/sBot checks, caught by
// TestEveryLoopSegmentLiesOnItsFace.
func walkCurvedSeam(arms curvedMiterArms, r float64, center, sTop, sBot math.Point3, res opstol.Resolution) ([]math.Point3, bool) {
	k := curvedSeamChordCount(center, sTop, sBot)
	thetaTop := cylinderAngleOf(arms.cyl, sTop)
	thetaBot := seamUnwrapAngle(cylinderAngleOf(arms.cyl, sBot), thetaTop)
	out := make([]math.Point3, k+1)
	out[0], out[k] = sTop, sBot
	bias := sTop
	for j := 1; j < k; j++ {
		theta := thetaTop + (thetaBot-thetaTop)*float64(j)/float64(k)
		p, ok := seamPointAtCylAngle(arms, r, theta, center, bias, res)
		if !ok {
			return nil, false
		}
		out[j], bias = p, p
	}
	return out, true
}

// seamPointAtCylAngle is the seam point at a given cylinder cross-section angle θ: along the cylinder
// tube's ruling at θ (a line parallel to the axis, ANCHORED at the corner-ball centre's axial station
// so the ±2r search brackets the seam) it finds where the torus-tube membership changes sign, bisects
// to that root, and polishes onto BOTH tubes. The two tubes' spines are tangent at the corner ball
// centre, so the ruling meets the seam on TWO petals (two axial roots); the root nearer the corner
// vertex vp is the physical (material) branch.
func seamPointAtCylAngle(arms curvedMiterArms, r float64, theta float64, center, vp math.Point3, res opstol.Resolution) (math.Point3, bool) {
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
	for range 60 {
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
// over the 2π/filletChordsPerTurn budget, floored at 4 (the planar sampleMiterSeam count). Shared by
// the torus∧torus family's own walker (fillet_miter_curved_torustorus.go) — the wedge measure itself
// is arm-shape-agnostic (it only reads the three points), so it needs no family-specific variant.
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
func projectOntoSeam(arms curvedMiterArms, r float64, start math.Point3, res opstol.Resolution) (math.Point3, bool) {
	tol := res.Weld() * r * r
	p := start
	lambda := r * r * 1e-3
	f1, f2, ok := seamResiduals(arms, r, p)
	if !ok {
		return math.Point3{}, false
	}
	for range miterSeamNewtonIters {
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
