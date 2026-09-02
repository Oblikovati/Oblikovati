// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The fillet's cross-section: sampling the blend profile into chords (split out of fillet.go for
// #2217).
//
// This is the section half of the ADR-0050 spine-section-marcher shape: given a corner frame and a
// cross-section kind (arc, conic, G2), emit the chord points that the assembly welds into faces. A
// variable-radius edge samples the same functional at each intermediate radius point.

// filletChordsPerTurn matches holeFacets' density: chords are sized as if the full circle
// had this many sides, so a 90° wedge gets 8.
const filletChordsPerTurn = 32

// runoutTol is the radius at or below which a variable fillet is treated as a run-out: the blend
// collapses to a single apex on the edge (no end face), so the fillet fades smoothly into the corner.
const runoutTol = 1e-9

// cornerChordCount is the number of chord segments spanning the corner's rolling-ball wedge — sized
// as if the full circle had filletChordsPerTurn sides (a 90° wedge gets 8), with a floor of 4.
func cornerChordCount(in cornerInputs) int {
	wedge := stdmath.Acos(float64(in.nA.Dot(in.nB)))
	k := max(int(stdmath.Ceil(wedge/(2*stdmath.Pi/filletChordsPerTurn))), 4)
	return k
}

// validateRadiusPoints checks intermediate fillet radius points are strictly inside the edge
// (0 < T < 1), positive, and strictly increasing in T (so the ruled profiles stay in order).
func validateRadiusPoints(mids []FilletRadiusPoint) error {
	prev := 0.0
	for _, m := range mids {
		if m.T <= 0 || m.T >= 1 {
			return fmt.Errorf("fillet: intermediate radius point T=%g must be strictly between 0 and 1", m.T)
		}
		if m.R <= 0 {
			return fmt.Errorf("fillet: intermediate radius %g must be > 0", m.R)
		}
		if m.T <= prev {
			return fmt.Errorf("fillet: intermediate radius points must be strictly increasing in T (got %g after %g)", m.T, prev)
		}
		prev = m.T
	}
	return nil
}

// midProfiles builds one corner cross-section per intermediate radius point: the rolling-ball circle
// at the interpolated edge point and radius, sampled as chords with the same frame as the end corners
// (#695). They have no end face/blend — they are pure ruling profiles between c0 and c1.
func midProfiles(e *topo.Edge, in cornerInputs, mids []FilletRadiusPoint, k int, cross FilletCrossSection, rho float64) []corner {
	if len(mids) == 0 {
		return nil
	}
	p0, p1 := e.StartVertex().Point(), e.EndVertex().Point()
	span := p0.VectorTo(p1)
	out := make([]corner, 0, len(mids))
	for _, m := range mids {
		p := p0.TranslateBy(span.Scale(m.T))
		cen := p.TranslateBy(in.offDir.Scale(m.R))
		c := corner{a: in.a, b: in.b, cen: cen, ta: cen.TranslateBy(in.nA.Scale(m.R)), tb: cen.TranslateBy(in.nB.Scale(m.R)),
			mid: cen.TranslateBy(slerpVec(in.nA, in.nB, 0.5).Scale(m.R))}
		c.chords = crossSectionChords(c, in, k, cross, rho)
		out = append(out, c)
	}
	return out
}

// sampleCornerChords samples both corners' cross-section profiles at the same stations, so chord j of
// one corner pairs with chord j of the other as a straight ruling of the blend band.
func sampleCornerChords(c0, c1 *corner, in cornerInputs, cross FilletCrossSection, rho float64) {
	k := cornerChordCount(in)
	c0.chords = crossSectionChords(*c0, in, k, cross, rho)
	c1.chords = crossSectionChords(*c1, in, k, cross, rho)
}

// crossSectionChords samples a corner's cross-section ta…tb into k+1 points for the requested shape
// (M36-F08): the circular arc (G1), a curvature-continuous G2 quintic, or a rho-controlled conic.
func crossSectionChords(c corner, in cornerInputs, k int, cross FilletCrossSection, rho float64) []math.Point3 {
	switch cross {
	case FilletG2:
		return g2Chords(c, in, k)
	case FilletConic:
		return conicChords(c, in, k, rho)
	default:
		return arcChords(c, in, k)
	}
}

// arcChords samples a corner's arc ta…tb as k+1 points: cen + r·slerp(nA→nB), the exact
// rolling-ball contact directions at evenly spaced stations.
func arcChords(c corner, in cornerInputs, k int) []math.Point3 {
	r := c.cen.DistanceTo(c.ta)
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		dir := slerpVec(in.nA, in.nB, float64(j)/float64(k))
		out[j] = c.cen.TranslateBy(dir.Scale(r))
	}
	return out
}

// shoulder is the sharp-corner point where the two walls' tangent lines (at ta along wall A, at tb
// along wall B) meet — cen + r·(nA+nB)/(1+nA·nB) — the apex a conic/G2 cross-section pulls toward.
func shoulder(c corner, in cornerInputs) math.Point3 {
	r := c.cen.DistanceTo(c.ta)
	cdot := in.nA.Dot(in.nB)
	return c.cen.TranslateBy(in.nA.Add(in.nB).Scale(r / (1 + cdot)))
}

// exactSectionWeight returns the shoulder weight of the pick's cross-section when it is a
// rational quadratic — an ARC (w = cos of the half wedge, since cos(wedge) = nA·nB) or a rho
// CONIC (w = rho/(1−rho)) — and ok=false for the G2 quintic, which is not (#1606).
func exactSectionWeight(p filletPick, in cornerInputs) (float64, bool) {
	switch {
	case p.cross.IsArc():
		return stdmath.Sqrt((1 + in.nA.Dot(in.nB)) / 2), true // tol-free: dihedral wedge < π keeps this > 0
	case p.cross == FilletConic:
		rho := p.rho
		if rho <= 0 || rho >= 1 {
			rho = 0.5
		}
		return rho / (1 - rho), true
	default:
		return 0, false
	}
}

// plainEnds reports whether both corners terminate against plain end faces or run-outs — the
// configurations the exact ruled blend covers; miter seams and corner-sphere blends keep the
// chord strips (their shared boundaries are chord polylines).
func plainEnds(c0, c1 corner) bool {
	return !c0.miter && !c0.blend && !c1.miter && !c1.blend
}

// setBlendShoulders stamps every profile's shoulder control point (and, for a conic
// cross-section, the weight its end trim must carry) on the exact blend's corners.
func setBlendShoulders(ef *edgeFillet, in cornerInputs, p filletPick) {
	conicW := 0.0
	if p.cross == FilletConic {
		conicW = ef.secW
	}
	stamp := func(c *corner) {
		c.sh = shoulder(*c, in)
		c.crossW = conicW
	}
	stamp(&ef.c0)
	stamp(&ef.c1)
	for i := range ef.mids {
		stamp(&ef.mids[i])
	}
}

// conicChords samples a rho-controlled conic cross-section (rational quadratic Bézier ta–S–tb) into
// k+1 points. The shoulder weight w follows the projective discriminant rho = w/(1+w): rho=0.5 ⇒ w=1
// (parabola), rho<0.5 flatter, rho>0.5 fuller. rho≤0 or ≥1 falls back to the parabola.
func conicChords(c corner, in cornerInputs, k int, rho float64) []math.Point3 {
	s := shoulder(c, in)
	if rho <= 0 || rho >= 1 {
		rho = 0.5
	}
	w := rho / (1 - rho) // shoulder weight
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		out[j] = rationalQuad(c.ta, s, c.tb, w, float64(j)/float64(k))
	}
	return out
}

// rationalQuad evaluates the rational quadratic Bézier with end weights 1 and shoulder weight w at t.
func rationalQuad(p0, p1, p2 math.Point3, w, t float64) math.Point3 {
	b0 := (1 - t) * (1 - t)
	b1 := 2 * (1 - t) * t * w
	b2 := t * t
	den := b0 + b1 + b2
	x := (b0*float64(p0.X) + b1*float64(p1.X) + b2*float64(p2.X)) / den
	y := (b0*float64(p0.Y) + b1*float64(p1.Y) + b2*float64(p2.Y)) / den
	z := (b0*float64(p0.Z) + b1*float64(p1.Z) + b2*float64(p2.Z)) / den
	return math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z))
}

// g2Chords samples a curvature-continuous (G2) cross-section into k+1 points. It is a quintic Bézier
// whose first three control points are collinear along wall A's tangent (ta→shoulder) and last three
// along wall B's (shoulder→tb), so the profile's curvature is ZERO at both tangency lines — matching
// the flat walls' zero curvature, i.e. no curvature jump where the blend meets them.
func g2Chords(c corner, in cornerInputs, k int) []math.Point3 {
	s := shoulder(c, in)
	ctrl := [6]math.Point3{
		c.ta, c.ta.Lerp(s, 1.0/3), c.ta.Lerp(s, 2.0/3),
		s.Lerp(c.tb, 1.0/3), s.Lerp(c.tb, 2.0/3), c.tb,
	}
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		out[j] = bezier5(ctrl, float64(j)/float64(k))
	}
	return out
}

// bezier5 evaluates a quintic Bézier via de Casteljau.
func bezier5(ctrl [6]math.Point3, t float64) math.Point3 {
	p := ctrl
	pts := p[:]
	for n := 5; n > 0; n-- {
		for i := 0; i < n; i++ {
			pts[i] = pts[i].Lerp(pts[i+1], t)
		}
	}
	return pts[0]
}
