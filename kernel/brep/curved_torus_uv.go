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
// sides use (trimByImprint, curved_uv_side.go): project the spiric section into the torus's
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

// uPeriodic reports that a torus's azimuth u wraps (u=0≡2π), so the welder folds the u-seam (uvSide, #1591).
func (c torusUV) uPeriodic() bool { return true }

// wrapsAllU is unused by the torus orientation (it classifies loops by winding, not a wrap flag), so it
// reports false (uvSide).
func (c torusUV) wrapsAllU() bool { return false }

// multiFace: a torus half-space cut leaves one connected face; it is not on the general curved∩curved path (uvSide, #1403).
func (c torusUV) multiFace() bool { return false }

// wrappingSolidFaces: a torus is not on the general ruled solid-membership wrapping path, so it always defers
// to the ordinary (u,v) emission (Oblikovati#1476).
func (c torusUV) wrappingSolidFaces(_ []Face2D, _ []uvSeg, _ geom.Surface, _ curvedFace) ([]curvedFace, []loopEdge, bool) {
	return nil, nil, false
}

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

// wrapAngle folds an angle into [0, 2π).
func wrapAngle(x float64) float64 {
	twoPi := 2 * stdmath.Pi
	x = stdmath.Mod(x, twoPi)
	if x < 0 {
		x += twoPi
	}
	return x
}

// seamOrigin is the surface parameter of the chart's (0,0): a torus places BOTH seams, so both
// parameters are offset (uvSide, ADR-0063).
func (c torusUV) seamOrigin() math.Point2 { return math.P2(c.seamU, c.seamV) }
