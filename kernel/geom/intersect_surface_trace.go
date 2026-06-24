// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Surface–surface intersection by a predictor–corrector continuation tracer (Oblikovati#1319). The
// old path contoured a signed-distance field on a fixed 96×96 grid (marching squares), which silently
// loses sub-grid loops, never sees a tangential contact (the field does not change sign), and pairs
// saddle branches arbitrarily. This tracer instead marches the joint zero of the two surfaces: from
// each seed it corrects onto the curve (project onto the intersection of the two tangent planes) and
// steps along nb×no, the curve tangent, with adaptive direction — so loops close, branches stay
// connected, and a tangency (parallel normals) is reported as a point rather than dropped.

const (
	ssiCorrectIters     = 40   // max corrector iterations per point
	ssiMaxStepsPerCurve = 6000 // guard against a non-closing march
	ssiMaxCurves        = 256  // guard against pathological seeding
	ssiSeedSteps        = 160  // seed-scan resolution (finer than the old 96 so thinner features seed)
	// ssiTangencyCos is the |nb·no| above which the surfaces count as tangent (normals (anti)parallel).
	// It must be loose enough to cover the FUZZY neighbourhood of a tangency, where the normals are
	// nearly — but not exactly — parallel: too strict a value (e.g. 1−1e-10) lets the corrector treat a
	// point a hair off the contact as a real crossing and emit a spurious tiny curve there. 1−1e-5
	// corresponds to crossings shallower than ~0.18°, which are tangencies for all practical purposes.
	ssiTangencyCos = 1 - 1e-5
	// ssiToleranceFraction and ssiStepFraction scale the on-curve tolerance and the march step to the
	// base patch's 3D extent (so both are model-relative, ADR-0042). 1e-7 is the stated acceptance
	// tolerance reachable by the NURBS Gauss–Newton projection; 4e-3 keeps a full curve to a few
	// thousand points while resolving curvature.
	ssiToleranceFraction = 1e-7
	ssiStepFraction      = 4e-3
	// Step multiples used while marching: a seed within ssiDedupSteps of an existing curve is a
	// duplicate of it; the loop closes when the march returns within ssiLoopCloseSteps of its start;
	// a tangency seed may sit up to ssiTangencyGapSteps from the contact (the strict normal test, not
	// this gate, is the real discriminator).
	ssiDedupSteps       = 1.5
	ssiLoopCloseSteps   = 0.75
	ssiTangencyGapSteps = 5.0
)

// traceIntersectionCurves returns the intersection curve(s) of base and other as polylines whose every
// point lies on BOTH surfaces within tol. tol and the march step are model-relative (derived from the
// base's domain extent in 3D). Tangential contacts yield a single-point "curve" flagged by being a
// degenerate (1–2 point) polyline at a point where the normals are parallel.
func traceIntersectionCurves(base, other Surface, grid SurfaceGrid) [][]math.Point3 {
	g := resolveGrid(base, grid)
	if g.UMax <= g.UMin || g.VMax <= g.VMin {
		return nil
	}
	tol := ssiTolerance(base, g)
	step := ssiStep(base, g)
	var curves [][]math.Point3
	for _, seed := range ssiSeeds(base, other, g) {
		if len(curves) >= ssiMaxCurves {
			break
		}
		// A seed where the surfaces are close AND their normals are (anti)parallel is a tangential
		// contact, not a transversal curve: refine it to the contact point rather than running the
		// (ill-conditioned, singular) curve corrector there, which would otherwise emit a spurious
		// near-tangency point and suppress the true one.
		if nearTangency(base, other, seed, step) {
			if contact, isT := refineTangency(base, other, seed, tol); isT && !nearAnyCurve(curves, contact, step) {
				curves = append(curves, []math.Point3{contact})
			}
			continue
		}
		pc, nb, no, ok := correctToBothSurfaces(base, other, seed, tol)
		if !ok {
			continue
		}
		if nearAnyCurve(curves, pc, step*ssiDedupSteps) {
			continue // this seed lands on an already-traced curve (within a march step of it)
		}
		curves = append(curves, marchCurve(base, other, pc, nb, no, g, step, tol))
	}
	return curves
}

// marchCurve traces one curve through pc by stepping forward then backward along the curve tangent,
// correcting each predicted point back onto both surfaces, until the curve closes into a loop or leaves
// the base's parameter window in both directions.
func marchCurve(base, other Surface, pc math.Point3, nb, no math.Vector3, g SurfaceGrid, step, tol float64) []math.Point3 {
	fwd, closed := marchOneWay(base, other, pc, nb, no, g, step, tol, true)
	if closed {
		return append([]math.Point3{pc}, fwd...)
	}
	bwd, _ := marchOneWay(base, other, pc, nb, no, g, step, tol, false)
	out := make([]math.Point3, 0, len(bwd)+1+len(fwd))
	for i := len(bwd) - 1; i >= 0; i-- {
		out = append(out, bwd[i])
	}
	out = append(out, pc)
	return append(out, fwd...)
}

// marchOneWay steps from start in one direction (forward=along +nb×no, else −) and returns the points
// reached plus whether it closed back onto start (a loop).
func marchOneWay(base, other Surface, start math.Point3, nb, no math.Vector3, g SurfaceGrid, step, tol float64, forward bool) ([]math.Point3, bool) {
	var pts []math.Point3
	p := start
	dir, ok := curveTangent(nb, no, forward)
	if !ok {
		return pts, false
	}
	for i := 0; i < ssiMaxStepsPerCurve; i++ {
		pred := p.TranslateBy(dir.Scale(math.Scalar(step)))
		pc, nbc, noc, ok := correctToBothSurfaces(base, other, pred, tol)
		if !ok {
			return pts, false
		}
		if !inWindow(base, pc, g) {
			return append(pts, pc), false // exited the base window: keep the boundary point
		}
		if i > 2 && start.DistanceTo(pc) < step*ssiLoopCloseSteps {
			return append(pts, start), true // closed the loop
		}
		if moved, err := math.UnitVector3FromVector(p.VectorTo(pc)); err == nil {
			dir = orient(nbc.Cross(noc), moved.AsVector()) // keep marching in the same heading
		}
		pts = append(pts, pc)
		p = pc
	}
	return pts, false
}

// curveTangent returns the unit curve tangent nb×no, reversed for the backward sweep; ok is false at a
// tangency (parallel normals, zero-length cross product).
func curveTangent(nb, no math.Vector3, forward bool) (math.Vector3, bool) {
	t, err := math.UnitVector3FromVector(nb.Cross(no))
	if err != nil {
		return math.Vector3{}, false
	}
	d := t.AsVector()
	if !forward {
		d = d.Scale(-1)
	}
	return d, true
}

// orient flips t to agree with the heading h (so the march does not reverse onto itself).
func orient(t, h math.Vector3) math.Vector3 {
	u, err := math.UnitVector3FromVector(t)
	if err != nil {
		return h
	}
	d := u.AsVector()
	if d.Dot(h) < 0 {
		d = d.Scale(-1)
	}
	return d
}

// correctToBothSurfaces pulls p onto the intersection of base and other by repeatedly projecting it
// onto the intersection LINE of the two surfaces' tangent planes (the standard SSI corrector). Returns
// the corrected point, the unit normals there, and whether it converged below tol. ok is false when
// the normals are (near) parallel — the tangent-plane intersection is undefined, signalling a tangency.
func correctToBothSurfaces(base, other Surface, p math.Point3, tol float64) (math.Point3, math.Vector3, math.Vector3, bool) {
	var nb, no math.Vector3
	for i := 0; i < ssiCorrectIters; i++ {
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
		a, b, ok := tangentPlaneSolve(nb, no, sb, so)
		if !ok {
			return p, nb, no, false
		}
		p = p.TranslateBy(nb.Scale(math.Scalar(a)).Add(no.Scale(math.Scalar(b))))
	}
	return p, nb, no, false
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
	for i := 0; i < ssiCorrectIters; i++ {
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
	mu := (g.UMax - g.UMin) * 1e-9
	mv := (g.VMax - g.VMin) * 1e-9
	return u >= g.UMin-mu && u <= g.UMax+mu && v >= g.VMin-mv && v <= g.VMax+mv
}

// nearAnyCurve reports whether p is within tol of any point of an already-traced curve.
func nearAnyCurve(curves [][]math.Point3, p math.Point3, tol float64) bool {
	for _, c := range curves {
		for _, q := range c {
			if float64(p.DistanceTo(q)) < tol {
				return true
			}
		}
	}
	return false
}

// ssiSeeds returns candidate 3D points near the intersection: the bisected grid sign-changes of the
// signed-distance field (every curve crossing a grid line) plus the grid points where the field is
// smallest in magnitude (catching tangencies and small loops that do not cross a grid line).
func ssiSeeds(base, other Surface, g SurfaceGrid) []math.Point3 {
	du := (g.UMax - g.UMin) / float64(ssiSeedSteps)
	dv := (g.VMax - g.VMin) / float64(ssiSeedSteps)
	field := func(u, v float64) float64 { return SignedDistanceToSurface(other, base.PointAt(u, v)) }
	var seeds []math.Point3
	for i := 0; i <= ssiSeedSteps; i++ {
		u := g.UMin + float64(i)*du
		for j := 0; j <= ssiSeedSteps; j++ {
			v := g.VMin + float64(j)*dv
			f := field(u, v)
			// Sign change to the +u / +v neighbour → a crossing seed (bisected for accuracy).
			if i < ssiSeedSteps && straddlesZero(f, field(u+du, v)) {
				seeds = append(seeds, bisectEdge(base, field, u, v, f, u+du, v))
			}
			if j < ssiSeedSteps && straddlesZero(f, field(u, v+dv)) {
				seeds = append(seeds, bisectEdge(base, field, u, v, f, u, v+dv))
			}
			// Near-zero interior sample → seed for a tangency / sub-grid loop the contour misses.
			if stdmath.Abs(f) < 2*stdmath.Max(du, dv) {
				seeds = append(seeds, base.PointAt(u, v))
			}
		}
	}
	return seeds
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

// ssiExtent estimates the base patch's 3D size over the grid window (the diagonal of its corner box),
// used to scale the tolerance and step.
func ssiExtent(base Surface, g SurfaceGrid) float64 {
	c00 := base.PointAt(g.UMin, g.VMin)
	c11 := base.PointAt(g.UMax, g.VMax)
	d := float64(c00.DistanceTo(c11))
	if d <= 0 {
		return 1
	}
	return d
}
