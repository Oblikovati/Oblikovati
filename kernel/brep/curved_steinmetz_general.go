// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz boolean through the GENERAL curved∩curved (u,v)-arrangement pipeline (#1403,
// approach A). The bicylinder is the lone ruled crossing whose SSI imprint SELF-INTERSECTS: two equal-radius
// perpendicular cylinders meet in TWO ellipses that cross at the pinch points P± = O ± R·n, so a (u,v)
// arrangement fed the two closed loops would discover the crossing from sampled polylines and mis-trace the
// degree-4 pinch vertices. Instead the recogniser SPLITS the imprint at the analytic pinches up front: it
// feeds the FOUR open elliptical arcs (each P−→P+) the bespoke constructor already builds (steinmetzArcs),
// so the arrangement sees two shared pinch vertices, never a self-crossing closed loop. The general angular-
// next-edge tracer (walkLoop) then separates the two lobes per side, the pinched flag tells wrapsAllU the
// region is lobes not a band, and curvedStitch welds the four arcs at the exact shared P± (every arc passes
// through O±R·n to machine precision, so the 3D weld coincides them). No bespoke loop→body assembler.

// steinmetzImprintArcs returns the four open elliptical arcs of the bicylinder imprint as analytic curves:
// the E+ ellipse split at the pinches into a front and back arc, and the E− ellipse likewise. Each arc runs
// pinch-to-pinch, so the arrangement's only shared vertices are the two pinch points (no self-crossing).
func steinmetzImprintArcs(o math.Point3, dirA, dirB math.Vector3, r float64) []geom.Curve3 {
	ePF, ePB, eMF, eMB := steinmetzArcs(o, dirA, dirB, r)
	return []geom.Curve3{ePF, ePB, eMF, eMB}
}

// steinmetzSideSplit trims one cylinder side by the four-arc imprint, keeping the cells `inside` selects
// under op. It frames the side as a (u,v) solid with pinched set (so wrapsAllU reports the lobes as a
// non-wrapping multi-face region) and pins the arrangement seam ONTO the pinch azimuth (pinchSeam), so every
// arc stays within one azimuth half and no arc — nor a degree-4 pinch vertex — is shattered by the seam. The
// kept lobe walls need no re-seaming or re-orientation: once each shared elliptical arc is stored anchored to
// its vertices (edgeCurveFor), the lobe's (u,v) trim loop is a simple polygon that tessellates outward (#1403).
func steinmetzSideSplit(f curvedFace, cyl geom.Cylinder, band coneSideBand_, arcs []geom.Curve3, pinch math.Point3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, bool) {
	c := newCylinderUVSolid(cyl, band, op, isB, inside)
	c.pinched = true
	c.seamHint, c.hasSeamHint = pinchSeam(c.paramOf(pinch).X), true
	faces, _, err := trimByImprint(&c, f, cyl, arcs, ruledSolidMaterial(&c))
	if err != nil || len(faces) == 0 {
		return nil, false
	}
	return faces, true
}

// unwrapArcSegs makes one open imprint arc's (u,v) sampling CONTINUOUS in u — removing the 2π jump at a
// pinch endpoint that paramOf's [0,2π) branch introduces — and shifts the whole arc by whole turns so its
// mean azimuth lands in [0,2π). An arc running pinch-to-pinch then stays within one azimuth half (its
// endpoint sits exactly on the seam, 0 or 2π, rather than wrapping across it), so splitSeamCrossing leaves
// it whole and the lobe it bounds seals (#1403). Used only for the pinched Steinmetz arcs, whose endpoints
// touch the seam; the ordinary closed-loop imprints keep the raw branch for splitSeamCrossing to resolve.
func unwrapArcSegs(segs []uvSeg) []uvSeg {
	if len(segs) == 0 {
		return segs
	}
	us := make([]float64, len(segs)+1)
	us[0] = float64(segs[0].a.X)
	for i, s := range segs {
		us[i+1] = unwrapAzimuthNear(us[i], float64(s.b.X))
	}
	mean := 0.0
	for _, u := range us {
		mean += u
	}
	shift := turnsToCanonical(mean / float64(len(us)))
	out := make([]uvSeg, len(segs))
	for i, s := range segs {
		s.a = math.P2(math.Scalar(us[i]+shift), s.a.Y)
		s.b = math.P2(math.Scalar(us[i+1]+shift), s.b.Y)
		out[i] = s
	}
	return out
}

// turnsToCanonical returns the whole-turn shift (a multiple of 2π) that brings u into [0, 2π).
func turnsToCanonical(u float64) float64 {
	shift := 0.0
	for u+shift < 0 {
		shift += 2 * stdmath.Pi
	}
	for u+shift >= 2*stdmath.Pi {
		shift -= 2 * stdmath.Pi
	}
	return shift
}

// pinchSeam returns the arrangement seam azimuth for the pinched case: the pinch azimuth ITSELF. Each of the
// four arcs runs pinch-to-pinch, so a seam on a pinch leaves every arc within one azimuth half (no arc
// crosses the seam, so no arc is fragmented and both cylinders re-emit the SAME shared arc — the weld holds).
// The two lobes then sit on opposite sides of the seam, neither straddling it; the pinch point is split
// across the seam but welds back (u=2π≡0), and the angular tracer sorts the four arc directions there
// translation-invariantly (#1403).
func pinchSeam(pinchU math.Scalar) float64 {
	u := float64(pinchU)
	for u < 0 {
		u += 2 * stdmath.Pi
	}
	for u >= 2*stdmath.Pi {
		u -= 2 * stdmath.Pi
	}
	return u
}

// steinmetzGeneral is the shared body of the equal-radius Steinmetz intersect/cut/join through the general
// pipeline: resolve the Steinmetz frame, build the four-arc imprint, trim BOTH cylinder sides keeping the
// lobes the op selects, and hand the kept walls to assemble for cap/orientation assembly. ok=false outside
// the equal-radius perpendicular crossing (steinmetzFrame declines), so kernel/ops keeps its CSG fallback.
func steinmetzGeneral(a, b *topo.Body, op Op,
	assemble func(a *topo.Body, wallA []curvedFace, b *topo.Body, wallB []curvedFace, insideA, insideB func(math.Point3) bool) []curvedFace) (*topo.Body, bool) {
	o, dirA, dirB, r, ok := steinmetzFrame(a, b)
	if !ok {
		return nil, false
	}
	fA, cylA, bandA, okA := cylinderSideFace(a)
	fB, cylB, bandB, okB := cylinderSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, false
	}
	arcs := steinmetzImprintArcs(o, dirA, dirB, r)
	pinch := o.TranslateBy(dirA.Cross(dirB).Scale(math.Scalar(r))) // P+ = O + R·n, a shared pinch point
	wallA, okWA := steinmetzSideSplit(fA, cylA, bandA, arcs, pinch, op, false, insideB)
	wallB, okWB := steinmetzSideSplit(fB, cylB, bandB, arcs, pinch, op, true, insideA)
	if !okWA || !okWB {
		return nil, false
	}
	return curvedStitch(assemble(a, wallA, b, wallB, insideA, insideB)), true
}

// SteinmetzIntersectGeneral builds the bicylinder a ∩ b through the general pipeline (#1403): the four lobe
// walls (two per cylinder) welded into the closed four-face lens-solid, no caps. ok=false outside the
// equal-radius perpendicular crossing.
func SteinmetzIntersectGeneral(a, b *topo.Body, _ *diag.Recorder) (*topo.Body, bool) {
	return steinmetzGeneral(a, b, Intersection, steinmetzIntersectFaces)
}

// steinmetzIntersectFaces assembles the bicylinder boundary: just the four kept lobe walls (the intersection
// of two solid cylinders is bounded only by their walls — no planar caps reach the lens).
func steinmetzIntersectFaces(_ *topo.Body, wallA []curvedFace, _ *topo.Body, wallB []curvedFace, _, _ func(math.Point3) bool) []curvedFace {
	return append(append([]curvedFace{}, wallA...), wallB...)
}
