// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Torus side split in PARAMETER SPACE (Oblikovati/Oblikovati#1406). The doubly-curved torus is the surface
// most often hit by a real revolve, and its plane cut carves a quartic SPIRIC section whose topology (one
// oval, two ovals, figure-eight, the cap vs its genus-1 complement) used to be a ladder of bespoke
// closed-form builders. torusUV routes every SPIRIC cut through the SAME (u,v)-arrangement trimmer the ruled
// sides use (trimByImprint, curved_halfspace_uv_side.go): project the spiric section into the torus's
// (u = azimuth, v = tube angle) chart, subdivide, classify by the section's sign, and re-emit — the topology
// emerging from the kept cells' boundary, not from a predicate. The PERPENDICULAR cut (two concentric
// circles, an annular lid the arrangement's lid chainer cannot assemble) stays analytic (torusHalfSpace).
//
// Unlike a ruled side the torus is DOUBLY periodic: both u and v wrap, and it has no rim circles — the only
// boundary of a kept region is the cut itself. So torusUV places BOTH an azimuth (u) and a tube (v) seam
// clear of the section, and its frame is the four artificial seam edges (which fold and dissolve), never a
// rim.

// torusUV is a torus expressed as a uvSide: the surface plus the placed azimuth/tube seams that rotate the
// (u,v) parameter origin so the section falls clear of the artificial seams (#1406).
type torusUV struct {
	torus        geom.Torus
	seamU, seamV float64
}

// torusUV satisfies uvSide: a doubly-periodic closed surface with no rims.
var _ uvSide = (*torusUV)(nil)

// paramOf inverts a 3D point on the torus to its seam-relative (u, v) = (azimuth, tube angle), both in
// [0, 2π) — the inverse of point3 (uvSide).
func (c torusUV) paramOf(p math.Point3) math.Point2 {
	u, v := c.torus.ParamAt(p)
	return math.P2(wrapAngle(u-c.seamU), wrapAngle(v-c.seamV))
}

// placeSeams puts BOTH seams in the widest gap of the section's u- and v-crossings (uvSide), so a contractible
// oval lands clear of both artificial seams; a section that wraps a period (the two-oval band wraps v) has no
// gap there and its seam stays at 0, crossed by the frame and folded.
func (c *torusUV) placeSeams(imprint []geom.Curve3) {
	var us, vs []float64
	for _, cv := range imprint {
		for i := 0; i <= imprintSampleCount; i++ {
			lo, hi := cv.Domain()
			u, v := c.torus.ParamAt(cv.PointAt(lo + (hi-lo)*float64(i)/imprintSampleCount))
			us, vs = append(us, u), append(vs, v)
		}
	}
	c.seamU, c.seamV = widestGapMid(us), widestGapMid(vs)
}

// vPeriodic reports that a torus's tube angle v wraps — the welder folds the v-seam and all-seam frame loops
// are dropped (uvSide).
func (c torusUV) vPeriodic() bool { return true }

// wrapsAllU is unused by the torus orientation (it classifies loops by winding, not a wrap flag), so it
// reports false (uvSide).
func (c torusUV) wrapsAllU() bool { return false }

// multiFace: a torus half-space cut leaves one connected face; it is not on the general curved∩curved path (uvSide, #1403).
func (c torusUV) multiFace() bool { return false }

// assembleSegments samples the spiric section, seam-splits it in u, and adds the four artificial seam edges
// that close the (u,v) rectangle (uvSide). There is no v-band clip (v is periodic) and no rim — the torus is
// closed, so the section is the only real boundary; the frame seams fold and dissolve.
func (c torusUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	out := make([]uvSeg, 0, len(imprint)*imprintSampleCount+4)
	for _, cv := range imprint {
		for _, s := range c.sampleSection(cv) {
			// Split on BOTH seams: an oval wraps the tube (v-seam) and a tilted section can wrap the azimuth
			// (u-seam), so neither alone closes the doubly-periodic rectangle for a two-oval band (#1406).
			for _, su := range splitSeamCrossing(s) {
				out = append(out, splitVSeamCrossing(su)...)
			}
		}
	}
	return append(out, c.frameSegments()...)
}

// sampleSection samples one spiric section curve over its whole domain into tagged (u,v) segments, each
// carrying the curve and its endpoint parameters so the boundary re-emission recovers the exact arc.
func (c torusUV) sampleSection(curve geom.Curve3) []uvSeg {
	lo, hi := curve.Domain()
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevT := lo
	prevP := c.sectionUV(curve, lo)
	for i := 1; i <= imprintSampleCount; i++ {
		t := lo + (hi-lo)*float64(i)/imprintSampleCount
		p := c.sectionUV(curve, t)
		segs = append(segs, uvSeg{a: prevP, b: p, curve: curve, tA: prevT, tB: t, kind: segImprint})
		prevT, prevP = t, p
	}
	return segs
}

// sectionUV returns the seam-relative (u,v) of a section curve at parameter t. A SpiricArc carries its (u,v)
// ANALYTICALLY (v = V0+t·(V1−V0), u = UAt(v)), so its two branches land on an IDENTICAL pinch vertex (their
// u-values differ by exactly 2π, which wrapAngle collapses) — where inverting the 3D point through ParamAt
// instead gives u-values a few 1e-8 apart that can straddle the welder grid and leave the oval unclosed
// (#1406). Any other imprint curve (a future curved∩curved section) falls back to the 3D inversion paramOf.
func (c torusUV) sectionUV(curve geom.Curve3, t float64) math.Point2 {
	if sa, ok := curve.(geom.SpiricArc); ok {
		v := sa.V0 + t*(sa.V1-sa.V0)
		return math.P2(wrapAngle(spiricU(sa, v)-c.seamU), wrapAngle(v-c.seamV))
	}
	return c.paramOf(curve.PointAt(t))
}

// spiricU returns the azimuth u on a spiric branch at tube angle v, but at a PINCH (|w|≈1, where the two
// branches meet at the oval's v-extreme) it snaps u to the exact shared vertex (Phi when w=+1, Phi+π when
// w=−1) independent of the branch sign. UAt alone gives the two branches u-values a few 1e-8 apart there
// (w is −1+ε in floating point, so arccos is π−√(2ε), not exactly π), which can straddle the arrangement
// welder grid and leave the oval unclosed (#1406). Away from a pinch the branches are genuinely distinct,
// so UAt is used directly.
func spiricU(sa geom.SpiricArc, v float64) float64 {
	r, bigR := sa.Torus.MinorRadius, sa.Torus.MajorRadius
	w := (sa.K - sa.C*r*stdmath.Sin(v)) / (sa.M * (bigR + r*stdmath.Cos(v)))
	if stdmath.Abs(w) >= 1-1e-9 {
		if w < 0 {
			return sa.Phi + stdmath.Pi
		}
		return sa.Phi
	}
	return sa.UAt(v)
}

// frameSegments returns the four artificial seam edges bounding the (u,v) rectangle: the two azimuth seams
// (u=0, u=2π over the full tube) and the two tube seams (v=0, v=2π over the full azimuth). All are segSeam:
// the torus is closed across each, so they bound nothing real and either fold to reverse twins (a region
// wrapping the seam) or form an all-seam loop dropped by dropArtificialLoops (the complement's frame).
func (c torusUV) frameSegments() []uvSeg {
	twoPi := 2 * stdmath.Pi
	return []uvSeg{
		{a: math.P2(0, 0), b: math.P2(twoPi, 0), kind: segSeam},
		{a: math.P2(0, twoPi), b: math.P2(twoPi, twoPi), kind: segSeam},
		{a: math.P2(0, 0), b: math.P2(0, twoPi), kind: segSeam},
		{a: math.P2(twoPi, 0), b: math.P2(twoPi, twoPi), kind: segSeam},
	}
}

// emitRun re-emits one boundary run (uvSide). A torus kept region's real boundary is the spiric section
// alone (the frame seams fold or drop), so only an imprint run is expected; a surviving seam run would be a
// topology the trim does not yet handle and defers (ok=false → ErrUnsupportedHalfSpace → CSG fallback).
func (c torusUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	if run[0].kind == segImprint {
		return emitImprintRun(run)
	}
	return loopEdge{}, false
}

// orientLoops orients the torus's kept boundary loops and reports whether the kept face is OUTERLESS (uvSide).
// On a closed surface winding cannot tell the small cap (kept inside the oval) from its genus-1 complement
// (kept outside) by geometry alone, so the (u,v) signed area decides: keptBoundaryEdges orients kept-material-
// on-the-left, so a single CCW oval (area>0) encloses kept material and is the cap's outer loop, while a CW
// oval (area<0) bounds a dropped island and is the complement's hole (the face then has no outer loop). The
// face traverses each loop forward; the lid takes the section arcs reversed, so each shared spiric edge is
// used once each way and the cap/lid weld is watertight.
func (c torusUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	outerless := len(loops) == 1 && loops[0].area < 0
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
		lid = append(lid, reverseEdgeChain(e.section)...)
	}
	return faceLoops, lid, outerless
}

// finalizeLoops is a no-op for the torus: it has no apex pole to drop (uvSide).
func (c torusUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return loops }

// spiricMaterial is the torus plane-cut predicate: a (u,v) point is kept where the section's signed value
// g(u,v) = (R + r·cos v)·M·cos(u − Phi) + C·r·sin v − K is negative — exactly the plane's negative side
// (g equals the signed plane distance n·(P − o)), the side HalfSpaceCut keeps. It reads the seams live so it
// binds the shifted frame placeSeams set; u,v are seam-relative, so the absolute angles add the seams back.
func (c *torusUV) spiricMaterial(plane geom.Plane) materialPredicate {
	phi, m, k, cc := geom.TorusSectionCoeffs(c.torus, plane)
	r, bigR := c.torus.MinorRadius, c.torus.MajorRadius
	return func(uv math.Point2) bool {
		uAbs, vAbs := float64(uv.X)+c.seamU, float64(uv.Y)+c.seamV
		g := (bigR+r*stdmath.Cos(vAbs))*m*stdmath.Cos(uAbs-phi) + cc*r*stdmath.Sin(vAbs) - k
		return g < 0
	}
}

// wrapAngle folds an angle into [0, 2π).
func wrapAngle(x float64) float64 {
	twoPi := 2 * stdmath.Pi
	x = stdmath.Mod(x, twoPi)
	if x < 0 {
		x += twoPi
	}
	return x
}

// torusSideSplit trims a bare torus face by a plane along its SPIRIC section, through the general
// (u,v)-arrangement trimmer (#1406). It builds the section itself (the analytic intersection defers the
// spiric quartic), then trims with the section's sign as the material predicate. The perpendicular cut has
// no spiric section (torusSpiricSection ok=false) and is handled analytically upstream, not here.
func torusSideSplit(f curvedFace, tor geom.Torus, plane geom.Plane) ([]curvedFace, []loopEdge, error) {
	section, ok := torusSpiricSection(tor, plane)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	c := torusUV{torus: tor}
	return trimByImprint(&c, f, tor, section, func() materialPredicate { return c.spiricMaterial(plane) })
}

// torusSpiricSection returns the torus∩plane spiric section as SpiricArc branches — the "tracer" the unified
// path feeds (#1406). The coefficients and v-range come from the existing closed forms; the topology is read
// from how much of the tube the section covers: a near-full-wrap section (absent arc ≤ figureEightWrapTol) is
// the two-oval band / figure-eight, two full-tube branches over v∈[0,2π]; otherwise a single oval, two
// branches over its [v0,v1] pinch range. A perpendicular cut (M≈0, no spiric) or a clearing plane returns
// ok=false.
func torusSpiricSection(t geom.Torus, plane geom.Plane) ([]geom.Curve3, bool) {
	phi, m, k, c := geom.TorusSectionCoeffs(t, plane)
	if m <= cylinderAxisCosTol {
		return nil, false // plane perpendicular to the axis: the analytic two-circle cut, not spiric
	}
	if torusSectionAbsentArc(t, m, k, c) <= figureEightWrapTol {
		return spiricBranches(t, phi, m, k, c, 0, 2*stdmath.Pi), true // two ovals / figure-eight (full tube)
	}
	v0, v1, _, ok := torusObliqueOvalRange(t, m, k, c)
	if !ok {
		return nil, false // a clearing plane, or a topology not yet supported
	}
	// Centre the oval's tube-angle range on [−π, π] (torusObliqueOvalRange may report it 2π-shifted, e.g.
	// [5π/3, 7π/3], when the valid stretch wraps the seam). The branch edges carry V0/V1, and the downstream
	// spiric mesher charts the oval from them — the analytic builders emit the centred [−vc, vc], so matching
	// that keeps the mesh on the oval rather than its 2π-shifted twin (#1406).
	for v0 > stdmath.Pi {
		v0, v1 = v0-2*stdmath.Pi, v1-2*stdmath.Pi
	}
	return spiricBranches(t, phi, m, k, c, v0, v1), true
}

// spiricBranches builds the +1 and −1 spiric branches over the tube-angle range [v0, v1] — the two arcs that
// bound one oval (meeting at its v-extremes), or the two full-tube ovals when [v0,v1] is the whole period.
func spiricBranches(t geom.Torus, phi, m, k, c, v0, v1 float64) []geom.Curve3 {
	return []geom.Curve3{
		geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: v0, V1: v1},
		geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: v0, V1: v1},
	}
}
