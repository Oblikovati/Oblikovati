// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Loft skinning — the TANGENCY / CONTINUITY conditions (M48 #2236 split of loft_skin.go). Derives the
// per-section tangent (and G2/G3 second/third) vectors that the hermite blend interpolates against, and
// applies the end conditions (apex, angle takeoff, face-tangent) that pin the loft's start/end slope to
// a neighbour or an adjacent face. The blend that consumes these lives in loft_skin_fit.go.

// continuityOrder maps a face-continuity condition to the derivative order it matches across the
// section edge: Tangent = 1 (G1), Smooth = 2 (G2, curvature), G3 = 3 (curvature-rate). Non-face
// conditions return 0. The canonical mapping lives in api/types.LoftCondition.ContinuityOrder().
func continuityOrder(c LoftCondition) int { return c.ContinuityOrder() }

// faceContinuity overrides an end section's tangents with the adjacent face's real longitudinal
// derivative (true G1, replacing the normal-only approximation) and returns the per-point second
// (G2) and third (G3) derivatives the quintic/septic end-segment blend uses to match the face's
// curvature and curvature-rate. The returned slices are nil below the requested order; both are nil
// when there is no adjacent surface or the condition is not face continuity. The takeoff speed
// follows the existing impact·chord convention so G1 magnitudes are unchanged.
func faceContinuity(tangents []math.Vector3, sec, neighbor []math.Point3, surf geom.Surface, end LoftEnd, isStart bool) (second, third []math.Vector3) {
	order := continuityOrder(end.Condition)
	if surf == nil || order == 0 {
		return nil, nil
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	if order >= 2 {
		second = make([]math.Vector3, len(sec))
	}
	if order >= 3 {
		third = make([]math.Vector3, len(sec))
	}
	c, sign := centroidOf(sec), takeoffSign(isStart)
	for j := range sec {
		outward := c.VectorTo(sec[j]) // away from the section centroid (the flare side)
		edge := edgeDirAt(sec, j)     // boundary tangent at this point
		t1, g2, g3, ok := faceEndDeriv(surf, sec[j], edge, outward)
		if !ok {
			continue // degenerate surface here — keep the approximate tangent
		}
		speed := impact * float64(sec[j].DistanceTo(neighbor[j])) // c: matches applyFaceTangent's scale
		tangents[j] = t1.Scale(math.Scalar(sign * speed))         // m0 = ±c · unit cross-boundary tangent
		// P(t)=γ(s(t)) with s=±c·t and |t1|=1 ⇒ P^(k)(0)=(±c)^k·γ^(k) — matching geometric curvature
		// (G2) and curvature-rate (G3) at the seam, reparam-invariant. The ODD orders carry the
		// mirror at a LAST section, where the face is at t=1 and the loft runs back from it (#2082).
		if second != nil {
			second[j] = g2.Scale(math.Scalar(speed * speed))
		}
		if third != nil {
			third[j] = g3.Scale(math.Scalar(sign * speed * speed * speed))
		}
	}
	return second, third
}

// faceEndDeriv returns, at the boundary point nearest p, the adjacent face surface's unit
// cross-boundary tangent direction (perpendicular to the boundary edge, in the surface tangent
// plane, pointing outward) and the surface's SECOND and THIRD derivatives along that same direction.
// The loft leaves the face along this tangent (G1) and, for Smooth/G3 ends, with this curvature (G2)
// and curvature-rate (G3) — continuing the real surface. The directional 2nd derivative is analytic
// (includes the mixed term); the 3rd is a central difference of it along the direction, so it is
// exact-to-tolerance without the mixed third partials the kernel does not expose. ok is false at a
// degenerate point.
func faceEndDeriv(surf geom.Surface, p math.Point3, edge, outwardRef math.Vector3) (t1, g2, g3 math.Vector3, ok bool) {
	u, v := surf.ParamAt(p)
	su, sv := surf.DerivativesAt(u, v)
	n := surf.NormalAt(u, v)
	cross := n.Cross(edge) // in-surface direction perpendicular to the boundary edge
	if cross.Dot(outwardRef) < 0 {
		cross = cross.Scale(-1) // orient outward (away from the face interior)
	}
	if cross.Length() < 1e-9 {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, false
	}
	t1 = cross.Scale(1 / cross.Length())
	// Solve du·Su + dv·Sv = t1 (t1 lies in the tangent plane) via the first fundamental form, so the
	// directional 2nd derivative is exact for this direction.
	e, f, g := su.Dot(su), su.Dot(sv), sv.Dot(sv)
	det := e*g - f*f
	if stdmath.Abs(float64(det)) < 1e-18 {
		return t1, math.Vector3{}, math.Vector3{}, true // degenerate metric ⇒ straight (G2/G3 → 0)
	}
	b1, b2 := t1.Dot(su), t1.Dot(sv)
	du := float64((g*b1 - f*b2) / det)
	dv := float64((e*b2 - f*b1) / det)
	g2 = dirSecond(surf, u, v, du, dv)
	const h = 1e-4 // central-difference step in the param direction for the directional 3rd derivative
	ahead := dirSecond(surf, u+h*du, v+h*dv, du, dv)
	behind := dirSecond(surf, u-h*du, v-h*dv, du, dv)
	g3 = ahead.Sub(behind).Scale(math.Scalar(1 / (2 * h)))
	return t1, g2, g3, true
}

// dirSecond is the surface's second derivative at (u,v) along the param direction (du,dv):
// du²·Suu + 2·du·dv·Suv + dv²·Svv (the analytic directional 2nd derivative, mixed term included).
func dirSecond(surf geom.Surface, u, v, du, dv float64) math.Vector3 {
	puu, puv, pvv := geom.SurfaceSecondPartials(surf, u, v)
	return puu.Scale(math.Scalar(du * du)).Add(puv.Scale(math.Scalar(2 * du * dv))).Add(pvv.Scale(math.Scalar(dv * dv)))
}

// sectionTangents is the longitudinal tangent at every section point: a Catmull-Rom tangent
// (half the chord from the previous to the next section) at interior sections, overridden at
// the first/last section by an angled end condition. Feeding these to a Hermite blend with the
// Catmull-Rom tangents reproduces the Catmull-Rom curve exactly, so Free ends are unchanged.
func sectionTangents(sections [][]math.Point3, closed bool, ends loftEnds, wrapShift int) [][]math.Vector3 {
	m, n := len(sections), len(sections[0])
	idx := func(i int) int {
		if closed {
			return ((i % m) + m) % m
		}
		return math.Clamp(i, 0, m-1)
	}
	// Across the closed seam (between section m-1 and 0) the correspondence is offset by the
	// monodromy wrapShift, so a periodic neighbour's matching point is reindexed by it. Without this
	// the Catmull-Rom tangent at the seam sections points across the cross-section (a kink/crease at
	// the seam) instead of along the loop.
	corr := func(j, shift int) int { return ((j+shift)%n + n) % n }
	tan := make([][]math.Vector3, m)
	for i := range m {
		tan[i] = make([]math.Vector3, n)
		predShift, succShift := 0, 0
		if closed && i == 0 {
			predShift = -wrapShift // stepping back to section m-1 crosses the seam
		}
		if closed && i == m-1 {
			succShift = wrapShift // stepping forward to section 0 crosses the seam
		}
		for j := range n {
			p := sections[idx(i-1)][corr(j, predShift)]
			q := sections[idx(i+1)][corr(j, succShift)]
			tan[i][j] = p.VectorTo(q).Scale(0.5)
		}
	}
	if !closed {
		applyEnd(tan[0], sections, 0, 1, ends.first, ends.firstN)
		applyEnd(tan[m-1], sections, m-1, m-2, ends.last, ends.lastN)
	}
	return tan
}

// applyEnd overrides one end section's tangents from its condition, dispatching on whether the
// section is a point (apex) or a profile. endIdx is the end section, neighIdx its neighbour.
func applyEnd(tangents []math.Vector3, sections [][]math.Point3, endIdx, neighIdx int, end LoftEnd, normal math.UnitVector3) {
	sec, neighbor := sections[endIdx], sections[neighIdx]
	isStart := endIdx < neighIdx
	if collapsedLoop(sec) {
		applyApexCondition(tangents, sec, neighbor, end, normal, isStart)
		return
	}
	applyEndCondition(tangents, sec, neighbor, end, normal, isStart)
}

// applyApexCondition shapes how the loft meets a point (apex) section. Sharp/Free keep the
// natural Catmull-Rom tangent — a straight taper (a cone). TangentToPlane makes the surface leave
// the apex tangent to its plane (normal), so each meridian's apex tangent lies in that plane,
// pointing along the neighbour point's radial direction — a rounded dome. Impact scales the dome
// reach; Reversed flips it to a concave (dished) apex.
func applyApexCondition(tangents []math.Vector3, apexSec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, isStart bool) {
	if !end.Condition.IsTangentToPlane() {
		return
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	sign := 1.0 // a start apex leaves outward (+u toward the neighbour); an end apex arrives inward
	if !isStart {
		sign = -1
	}
	if end.Reversed {
		sign = -sign
	}
	apex, nc := apexSec[0], centroidOf(neighbor)
	for j := range apexSec {
		r := radialDir(neighbor[j], nc, normal)
		chord := float64(apex.DistanceTo(neighbor[j]))
		tangents[j] = unitOrFallback(r.Scale(sign), normal.AsVector()).Scale(impact * chord)
	}
}

// applyEndCondition overrides a profile end section's tangents from its condition: an
// angle/direction takeoff (a chosen angle to the section plane) or a face-continuity takeoff
// (Tangent/Smooth — leave the source face tangent to its surface). Free keeps the natural
// Catmull-Rom tangent. neighbor is the adjacent section, normal the section's plane/face normal,
// isStart says whether this is the FIRST section (+u leaves it) or the last (+u arrives at it).
func applyEndCondition(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, isStart bool) {
	switch {
	case end.Condition.CurvesViaAngle():
		applyAngleTakeoff(tangents, sec, neighbor, end, normal, isStart)
	case end.Condition.IsFaceContinuity():
		applyFaceTangent(tangents, sec, neighbor, end, normal, isStart)
	}
}

// takeoffSign turns a LEAVING direction (the way the surface departs the section into the body)
// into the +u tangent the Hermite blend wants. +u leaves the first section but ARRIVES at the last
// one, so the last section's takeoff is the mirror of its leaving direction.
//
// Missing this mirror is what made a 45° takeoff at both ends produce a pear rather than a barrel
// — bulging to 2.27 at the bottom and pinching to 1.75 at the top of an r=2 loft — and, Reversed,
// drove the side wall back down through the end cap at radius 1.74 (Oblikovati#2082).
// applyApexCondition has always mirrored; the profile branches did not.
func takeoffSign(isStart bool) float64 {
	if isStart {
		return 1
	}
	return -1
}

// applyAngleTakeoff aims each tangent at end.Angle from the section plane (sin·normal + cos·
// radial), scaled by impact·chord; Reversed flips the through-plane component (an undercut). The
// direction built here is how the surface LEAVES the section — takeoffSign turns it into a +u
// tangent, which mirrors it at a last section.
func applyAngleTakeoff(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, isStart bool) {
	inward := centroidOf(sec).VectorTo(centroidOf(neighbor)) // the way the body lies from here
	nf := alignToward(normal, inward)
	if end.Reversed {
		nf = nf.Negate()
	}
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	sa, ca := stdmath.Sin(end.Angle), stdmath.Cos(end.Angle)
	sign, c := takeoffSign(isStart), centroidOf(sec)
	for j := range sec {
		r := radialDir(sec[j], c, normal)
		base := float64(sec[j].DistanceTo(neighbor[j]))
		leave := nf.AsVector().Scale(sa).Add(r.Scale(ca))
		tangents[j] = unitOrFallback(leave, inward).Scale(sign * impact * base)
	}
}

// applyFaceTangent makes the loft leave the source face tangent to its surface: each tangent is
// the in-surface direction perpendicular to the boundary edge (normal × edgeDir), oriented
// outward, scaled by impact·chord. For a planar source face the loft's tangent plane along the
// shared edge equals the face plane — exact G1 continuity. Smooth (G2) reuses this as a faceted-
// kernel approximation. Reversed flips the takeoff inward.
func applyFaceTangent(tangents []math.Vector3, sec, neighbor []math.Point3, end LoftEnd, normal math.UnitVector3, isStart bool) {
	impact := end.Impact
	if impact <= 0 {
		impact = 1
	}
	sign, n := takeoffSign(isStart), normal.AsVector()
	c := centroidOf(sec)
	for j := range sec {
		t := n.Cross(edgeDirAt(sec, j))
		if t.Dot(c.VectorTo(sec[j])) < 0 { // orient outward (away from the section centroid)
			t = t.Negate()
		}
		if end.Reversed {
			t = t.Negate()
		}
		base := float64(sec[j].DistanceTo(neighbor[j]))
		tangents[j] = unitOrFallback(t, c.VectorTo(sec[j])).Scale(sign * impact * base)
	}
}

// edgeDirAt is the boundary direction at loop point j (the chord from the previous to the next).
func edgeDirAt(loop []math.Point3, j int) math.Vector3 {
	n := len(loop)
	return loop[(j-1+n)%n].VectorTo(loop[(j+1)%n])
}

// alignToward returns n flipped, if needed, to point into the same half-space as ref.
func alignToward(n math.UnitVector3, ref math.Vector3) math.UnitVector3 {
	if n.AsVector().Dot(ref) < 0 {
		return n.Negate()
	}
	return n
}

// radialDir is the outward in-plane unit direction from the section centroid c to point p
// (zero when p sits on the section axis, e.g. a point section).
func radialDir(p, c math.Point3, normal math.UnitVector3) math.Vector3 {
	nrm := normal.AsVector()
	v := c.VectorTo(p)
	v = v.Sub(nrm.Scale(v.Dot(nrm)))
	l := v.Length()
	if l < 1e-12 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / l)
}

// unitOrFallback returns the unit of v, or the unit of fallback when v is ~zero.
func unitOrFallback(v, fallback math.Vector3) math.Vector3 {
	if v.Length() < 1e-12 {
		v = fallback
	}
	l := v.Length()
	if l == 0 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / l)
}
