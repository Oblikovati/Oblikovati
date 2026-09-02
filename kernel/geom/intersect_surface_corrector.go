// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Pulling a point back ONTO both surfaces, and the tolerances the sweep measures against (split
// out of intersect_surface_trace.go for #2214).
//
// The corrector is Newton in the plane spanned by the two surface normals: solve the 2x2 tangent
// system for the step that zeroes both signed distances, and iterate. Near tangency that system is
// ill-conditioned, so tangency is DETECTED here and refined separately rather than marched
// through.

// correctToBothSurfaces pulls p onto the intersection of base and other by repeatedly stepping it toward
// both surfaces (the standard SSI corrector). Returns the corrected point, the unit normals there, and
// whether it converged below tol. Where the normals are well separated each step is the tangent-plane
// intersection; through a tangency/pinch neighbourhood — where that step is ill-conditioned — it falls
// back to a damped descent step so the curve is traced THROUGH the singularity rather than dropped at it
// (correctorStep, #1404). ok is false only when no zero is reached within ssiCorrectIters (a genuine
// mid-air stall: the predicted point has no nearby crossing, e.g. a march walking off a NURBS patch).
func correctToBothSurfaces(base, other Surface, p math.Point3, tol float64) (math.Point3, math.Vector3, math.Vector3, bool) {
	var nb, no math.Vector3
	for range ssiCorrectIters {
		ub, vb, db := ProjectPointToSurface(base, p)
		uo, vo, do := ProjectPointToSurface(other, p)
		pb, po := base.PointAt(ub, vb), other.PointAt(uo, vo)
		nb, no = base.NormalAt(ub, vb), other.NormalAt(uo, vo)
		sb := float64(nb.Dot(pb.VectorTo(p))) // signed gap to base's tangent plane
		so := float64(no.Dot(po.VectorTo(p)))
		// Converge on the actual residual distances, NOT the tangent-plane gaps: where a (clamped)
		// projection lands on a NURBS patch boundary the gap reads ~0 while the point is still off the
		// surface, so a march that walks off the patch must fail here rather than accept a false zero.
		if db < tol && do < tol {
			return p, nb, no, true
		}
		p = p.TranslateBy(correctorStep(nb, no, sb, so))
	}
	return p, nb, no, false
}

// correctorStep returns one corrector displacement that drives the two signed tangent-plane gaps (sb, so)
// toward zero. Where the normals are well separated it is the exact tangent-plane intersection step (lands
// p on BOTH tangent planes). Where they are near-parallel — a tangency or pinch, the configuration #1404
// must survive — that 2×2 system is ill-conditioned and blows up, so it falls back to a damped
// steepest-descent step on ½(sb²+so²), whose gradient is sb·nb+so·no: unconditionally stable, still
// monotone, and so converges onto a shallow transversal crossing (a near-pinch) and onto a true contact
// (a pinch) instead of giving up and dropping the curve there.
func correctorStep(nb, no math.Vector3, sb, so float64) math.Vector3 {
	if a, b, ok := tangentPlaneSolve(nb, no, sb, so); ok {
		return nb.Scale(math.Scalar(a)).Add(no.Scale(math.Scalar(b)))
	}
	gradient := nb.Scale(math.Scalar(sb)).Add(no.Scale(math.Scalar(so)))
	return gradient.Scale(-ssiDescentGain)
}

// tangentPlaneSolve finds the step δ = a·nb + b·no that lands p on BOTH tangent planes:
// nb·δ = −sb, no·δ = −so. With c = nb·no this is a 2×2 system; ok is false when |1−c²| ≈ 0
// (parallel normals — the planes are coincident/parallel and the intersection line is undefined).
func tangentPlaneSolve(nb, no math.Vector3, sb, so float64) (a, b float64, ok bool) {
	c := float64(nb.Dot(no))
	den := 1 - c*c
	if stdmath.Abs(den) < 1-ssiTangencyCos {
		return 0, 0, false
	}
	b = (sb*c - so) / den
	a = -sb - b*c
	return a, b, true
}

// nearTangency reports whether the surfaces are close at p (within a march step) AND their normals are
// (anti)parallel there — the signature of a tangential contact, distinct from a transversal crossing
// (where the surfaces cross at a non-zero angle, so their normals are not parallel).
func nearTangency(base, other Surface, p math.Point3, step float64) bool {
	ub, vb, _ := ProjectPointToSurface(base, p)
	uo, vo, do := ProjectPointToSurface(other, p)
	// Gate generously (a seed can sit several steps from the contact) — the strict normal-parallelism
	// test (ssiTangencyCos = 1 − 1e-10) is the real discriminator: only a genuine tangency passes it,
	// a transversal crossing (even a shallow one) does not.
	if do > ssiTangencyGapSteps*step {
		return false
	}
	c := stdmath.Abs(float64(base.NormalAt(ub, vb).Dot(other.NormalAt(uo, vo))))
	return c > ssiTangencyCos
}

// refineTangency pinpoints a tangential contact near p by iterating toward the midpoint of the two
// surfaces' closest points: where the surfaces touch, both feet converge to the contact. It returns
// the contact point and whether it is a GENUINE tangency — the surfaces meet within tol there and
// their normals are (anti)parallel (so the corrector's stall was a real touch, not a mid-air stall).
func refineTangency(base, other Surface, p math.Point3, tol float64) (math.Point3, bool) {
	for range ssiCorrectIters {
		ub, vb, _ := ProjectPointToSurface(base, p)
		uo, vo, _ := ProjectPointToSurface(other, p)
		pb, po := base.PointAt(ub, vb), other.PointAt(uo, vo)
		gap := float64(pb.DistanceTo(po))
		p = pb.TranslateBy(pb.VectorTo(po).Scale(0.5)) // midpoint of the closest approach
		if gap < tol {
			c := stdmath.Abs(float64(base.NormalAt(ub, vb).Dot(other.NormalAt(uo, vo))))
			return p, c > ssiTangencyCos
		}
	}
	return p, false
}

// inWindow reports whether pc's base parameters lie inside the grid window (with a one-step margin so a
// point exactly on the boundary still counts).
func inWindow(base Surface, pc math.Point3, g SurfaceGrid) bool {
	u, v, _ := ProjectPointToSurface(base, pc)
	mu := (g.UMax - g.UMin) * 1e-9 // tol:parametric — fraction of the u-window
	mv := (g.VMax - g.VMin) * 1e-9 // tol:parametric — fraction of the v-window
	return u >= g.UMin-mu && u <= g.UMax+mu && v >= g.VMin-mv && v <= g.VMax+mv
}

// nearAnyCurve reports whether p is within tol of an already-traced curve, measured to the polyline
// SEGMENTS (a single-point tangency marker degenerates to its point). Vertex distance conflated a
// duplicate seed (on the same curve, within chord sag of its polyline) with a distinct neighbouring
// loop as soon as the loops sat closer together than the dedup radius — e.g. the two near-tangent
// circles where a sphere barely clears a cylinder (#1597).
func nearAnyCurve(curves [][]math.Point3, p math.Point3, tol float64) bool {
	for _, c := range curves {
		if len(c) == 1 && float64(p.DistanceTo(c[0])) < tol {
			return true
		}
		for i := 0; i+1 < len(c); i++ {
			if DistancePointToSegment(LineSegment{StartPoint: c[i], EndPoint: c[i+1]}, p) < tol {
				return true
			}
		}
	}
	return false
}

// ssiTolerance is the model-relative on-curve tolerance: 1e-7 of the base's 3D extent (the stated
// acceptance tolerance, reachable by the NURBS Gauss–Newton projection — a 1e-9 target is not).
func ssiTolerance(base Surface, g SurfaceGrid) float64 {
	return ssiToleranceFraction * ssiExtent(base, g)
}

// ssiStep is the nominal march step: a small fraction of the base's 3D extent, so a full curve is a
// few thousand points at most while fine enough to resolve curvature.
func ssiStep(base Surface, g SurfaceGrid) float64 {
	return ssiStepFraction * ssiExtent(base, g)
}

// ssiExtentSamples is the per-axis lattice for the extent estimate. Five samples put nodes at the
// quarter points of a full-period window, so a closed direction contributes the four cardinal points
// of every circular section — its true girth — to the bounding box (#1597).
const ssiExtentSamples = 5

// ssiExtent estimates the base patch's 3D size over the grid window — the diagonal of the bounding
// box of a sampled parameter lattice — used to scale the tolerance and step. The previous
// corner-to-corner estimate collapsed wherever the window is periodic: on a torus both corners map to
// the SAME 3D point (measured 1.5e-14 for R=50/r=10, true ≈171), and on a cylinder they differ by the
// axial height only, missing the girth entirely — which drove step and tolerance toward 0 and pushed
// every trace onto the marching-squares fallback silently (#1597). OCCT sizes its walking step from
// surface bounding boxes the same way (IntWalk_PWalking via Bnd_Box/adaptor resolution).
func ssiExtent(base Surface, g SurfaceGrid) float64 {
	d := float64(sampledPatchBox(base, g).Diagonal().Length())
	if d <= 0 {
		return 1
	}
	return d
}

// sampledPatchBox is the axis-aligned box of base evaluated on an ssiExtentSamples² lattice over the
// grid window (corners included).
func sampledPatchBox(base Surface, g SurfaceGrid) math.Box {
	n := ssiExtentSamples - 1
	pts := make([]math.Point3, 0, ssiExtentSamples*ssiExtentSamples)
	for i := 0; i <= n; i++ {
		u := g.UMin + (g.UMax-g.UMin)*float64(i)/float64(n)
		for j := 0; j <= n; j++ {
			v := g.VMin + (g.VMax-g.VMin)*float64(j)/float64(n)
			pts = append(pts, base.PointAt(u, v))
		}
	}
	return math.BoxFromPoints(pts...)
}
