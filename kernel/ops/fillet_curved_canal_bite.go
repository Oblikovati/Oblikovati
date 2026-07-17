// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The unified canal HOST bite (M6' C4 W3b, architecture: canal-armweld-architecture.md §"W3 addendum —
// per-host bite-loop composition"). ONE generalized canalHostBite re-clips every canal corner roll host,
// superseding the foot-locus-only retrimCanalHost (W3), which could bridge only the wall and declined
// because the foot-locus ENDPOINTS (the arm-rail junctions W0/W1) are ~37u interior to the wall band. The
// W3b insight: it is the arm rails' OUTER ends that anchor on the host loop, not the foot-locus endpoints.
//
// Each host's inner bite is COMPOSED from the arms' already-built host rails (collected from the W2
// bundles — shared-edge identity, never a shared-w.center rebuild) plus the host's canal foot-locus when
// it has one, chained by shared endpoints. The four N7 hosts fall out of the SAME assembler by
// (rail-count, bridge?): the wall (2, feet[0]), each plane (2, none — rails meet at a point), the s_10
// boss (0, feet[1]). The surviving far span reuses farPathSegs VERBATIM. Any non-meeting chain junction
// (risk #2) or non-anchoring outer end (risk #1) → honest-decline with the exact gap, never a widened tol.

// canalHostBite composes ONE canal corner host's retrimmed face: collect its arm rails, bridge them with
// the host foot-locus (if any), chain the inner bite, then close the surviving far span. Honest-declines
// (non-empty reason, carrying the host + the junction/anchor gap) on any non-meeting chain junction or a
// rail outer end that does not anchor on the host's bitten loop — NEVER a widened tolerance (a mis-closed
// host loop corrupts the mesh). Example:
//
//	ff, reason := canalHostBite(wallFace, bundles, boundaries, rolls, w, res)
//	if reason != "" { /* decline the weld — do-no-harm (the reason carries the measured gap) */ }
func canalHostBite(host *topo.Face, bundles []canalArmBundle, b canalBoundaries, rolls []geom.Surface, w cornerWeld, res Resolution) (filletFace, string) {
	tol := res.Weld() * w.radius
	inner, reason := canalInnerBite(host, bundles, b, rolls, tol)
	if reason != "" {
		return filletFace{}, reason
	}
	star, ok := bittenLoop(host, innerBiteKey(inner), tol)
	if !ok {
		return filletFace{}, fmt.Sprintf("canal host bite (%T): no unambiguous bitten loop (tol %.3e)", host.Geometry(), tol)
	}
	loop, reason := canalCloseFar(host, star, inner, bundles, w, tol)
	if reason != "" {
		return filletFace{}, reason
	}
	return filletFace{surface: host.Geometry(), loops: hostLoopsWithBiteReplaced(host, star, loop), parent: host.Lineage()}, ""
}

// hostLoopsWithBiteReplaced rebuilds host's loop list with the bitten loop star SUBSTITUTED by its
// retrimmed form `bite`, PRESERVING every loop's original position — and thus its outer/inner role, since
// the assembler marks loop 0 outer and the rest inner (addCurvedFaces). This is the fix for an INNER
// bitten loop: N7's wall notch window is a HOLE, not the outer rim, so the pre-F3 "[bite, ...others]"
// order forced the window to loop 0 (outer) and demoted the wall's rim-seam boundary to a hole — which
// mis-trims the cylinder (a self-overlapping mesh, ~9% area error). For an OUTER bitten loop (every B3
// host, where star is already Face.Loops()[0]) the order is unchanged. Face.Loops() is outer-first by
// construction, so the outer rim keeps position 0 whenever star is inner.
func hostLoopsWithBiteReplaced(host *topo.Face, star *topo.Loop, bite filletLoop) []filletLoop {
	loops := make([]filletLoop, 0, len(host.Loops()))
	for _, l := range host.Loops() {
		if l == star {
			loops = append(loops, bite)
			continue
		}
		loops = append(loops, unchangedLoop(l))
	}
	return loops
}

// canalInnerBite collects host's arm rails + optional foot-locus bridge and chains them into the inner
// bite (outer→…→outer) by shared endpoints. Declines (naming the host + the non-meeting junction gap) if
// the rails + bridge do not chain — a REAL defect (a rail at the wrong reflected centre / a mis-tagged
// foot-locus), never widened (risk #2). A host with neither rails nor a foot-locus is a routing error.
func canalInnerBite(host *topo.Face, bundles []canalArmBundle, b canalBoundaries, rolls []geom.Surface, tol float64) ([]endSeg, string) {
	pieces := armRailsOnHost(host, bundles)
	if chords, ok := footLocusChordsForHost(host, b, rolls, tol); ok {
		pieces = append(pieces, chords...)
	}
	if len(pieces) == 0 {
		return nil, fmt.Sprintf("canal inner bite (%T): host has neither arm rails nor a foot-locus", host.Geometry())
	}
	inner, reason := chainBiteSegs(pieces, tol)
	if reason != "" {
		return nil, fmt.Sprintf("canal inner bite (%T): %s", host.Geometry(), reason)
	}
	return inner, ""
}

// armRailsOnHost COLLECTS the already-built host contact rails (W2, shared-edge identity) of every arm
// that rolls on host — 0..2 for N7 (2 on the wall / each plane, 0 on the s_10 boss). It reads each arm's
// rail from its canal bundle keyed on the host face the rail lies on (bundle.hosts), NEVER rebuilding a
// rail at a shared w.center: the per-arm reflected centres differ, so a shared-centre rebuild (the
// single-ball retrimCornerHost) would misplace the feet (architecture §"Assembly decision").
func armRailsOnHost(host *topo.Face, bundles []canalArmBundle) []endSeg {
	var rails []endSeg
	for _, bundle := range bundles {
		for k := 0; k < 2; k++ {
			if bundle.hosts[k] == host {
				rails = append(rails, bundle.rails[k])
			}
		}
	}
	return rails
}

// footLocusIndexForHost returns which canal foot-locus (0 = wall/feet[0], 1 = s_10 boss/feet[1]) bites
// host, mapped by the ROLL-HOST identity in CanalCorner.Rolls (rolls[0]=wall, rolls[1]=s_10 boss surface)
// — NOT guessed from the host's geometry (the "recognize by payload, not shape" rule, canalProvider.Fits).
// A host on neither roll surface (the two planes) gets no foot-locus; then the arm rails alone compose the
// bite. The single source both footLocusForHost (routing predicate) and footLocusChordsForHost (the sampled
// bite) share, so the roll-host match is written once.
func footLocusIndexForHost(host *topo.Face, rolls []geom.Surface, tol float64) (int, bool) {
	if len(rolls) < 2 {
		return 0, false
	}
	if sameRollSurface(host.Geometry(), rolls[0], tol) {
		return 0, true
	}
	if sameRollSurface(host.Geometry(), rolls[1], tol) {
		return 1, true
	}
	return 0, false
}

// footLocusForHost returns the canal foot-locus that bites host as a single whole-curve endSeg — the
// ROUTING/identity view (canalHostFace's has-bridge predicate). The retrimmed host loop is built from the
// point-sampled footLocusChordsForHost, not this one segment.
func footLocusForHost(host *topo.Face, b canalBoundaries, rolls []geom.Surface, tol float64) (endSeg, bool) {
	i, ok := footLocusIndexForHost(host, rolls, tol)
	if !ok {
		return endSeg{}, false
	}
	return footLocusBite(b.feet[i]), true
}

// footLocusChordsForHost samples host's canal foot-locus into ringSegSamples straight sub-chords matching
// the corner patch's OWN sampling (sampleCurve3Open with the patch's rev direction, b.feetRev[i]), so the
// retrimmed host welds point-for-point to the corner patch along the shared foot-locus (F3 watertightness,
// derivation §7.1). Empty (false) when host carries no foot-locus (the two planes / a non-roll host).
func footLocusChordsForHost(host *topo.Face, b canalBoundaries, rolls []geom.Surface, tol float64) ([]endSeg, bool) {
	i, ok := footLocusIndexForHost(host, rolls, tol)
	if !ok {
		return nil, false
	}
	return footLocusChords(b.feet[i], b.feetRev[i]), true
}

// footLocusBite turns a foot-locus curve into a single whole-curve endSeg: from/to are its domain
// endpoints and curve is the foot-locus ITSELF (the shared object, ADR-C4-2). Used only by the routing
// predicate and the direct unit tests; the assembled bite uses footLocusChords (point-identical to the
// corner patch). arc=false — a foot-locus is a canal BSpline isocurve, not a circular Arc3d.
func footLocusBite(c geom.Curve3) endSeg {
	lo, hi := c.Domain()
	return endSeg{from: c.PointAt(lo), to: c.PointAt(hi), curve: c}
}

// footLocusChords sub-samples a canal foot-locus into ringSegSamples sub-edges across the SAME
// ringSegSamples+1 vertices the corner patch tiles it into: sampleCurve3OpenTrimmed(c, rev) reproduces
// the patch's ringSegSamples interior samples BIT-IDENTICALLY (same points as sampleCurve3Open, same rev),
// and the one excluded domain endpoint (the arm-rail junction the corner patch gets from its neighbouring
// side) is appended — so the retrimmed host and the corner patch share EVERY vertex along the foot-locus
// and weld watertight (F3 crack fix; the pre-F3 single whole-curve edge shared only the two endpoints,
// cracking the 5 interior vertices). Each sub-edge carries the foot-locus curve TRIMMED to its own
// sub-span (curveSpan==vGap, the same convention canalPatchLoops and the arm face now use), NOT the whole
// curve — carrying the whole curve made the shared tessellator sweep the entire foot-locus per sub-edge
// and self-overlap the host loop (N7 cylinder-arm half-cover; n7-tessellation-diagnosis.md §2).
func footLocusChords(c geom.Curve3, rev bool) []endSeg {
	pts, curves := sampleCurve3OpenTrimmed(c, rev) // patch-identical samples + per-sub-edge trimmed curves
	lo, hi := c.Domain()
	far := c.PointAt(hi) // sampleCurve3Open* excludes the HIGH end when forward…
	if rev {
		far = c.PointAt(lo) // …and the LOW end when reversed
	}
	pts = append(pts, far)
	segs := make([]endSeg, len(curves))
	for i := range segs {
		segs[i] = endSeg{from: pts[i], to: pts[i+1], curve: curves[i]}
	}
	return segs
}

// sameRollSurface reports whether host is the SAME roll surface as a CanalCorner.Rolls entry — matched by
// GEOMETRY (type + radius + coaxial axis for a cylinder), so a copied-value roll host matches the body
// face's stored surface without relying on interface bit-identity. Both N7 roll hosts are cylinders (wall
// R=50 / s_10 R=5), distinguished by radius + axis; a plane host matches neither (→ no foot-locus).
func sameRollSurface(host, roll geom.Surface, tol float64) bool {
	hc, ok0 := host.(geom.Cylinder)
	rc, ok1 := roll.(geom.Cylinder)
	if !ok0 || !ok1 {
		return false
	}
	if stdmath.Abs(hc.Radius-rc.Radius) > tol || !hc.AxisDir.IsParallelTo(rc.AxisDir, retrimAxisParallelTol) {
		return false
	}
	d := rc.Origin.VectorTo(hc.Origin)
	axis := rc.AxisDir.AsVector()
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	return float64(perp.Length()) <= tol
}

// innerBiteKey is a point ON the host near the bite — the middle vertex of the inner chain (the foot-locus
// / corner region) — so bittenLoop keys on the wire the canal opens (an inner NOTCH on a boolean-cut wall,
// not the outer rim). Mirrors the single-ball retrim's w.center key, but robust to a foot-locus-only bite.
func innerBiteKey(inner []endSeg) math.Point3 {
	return inner[len(inner)/2].from
}

// canalCloseFar closes the inner bite into a full host loop with the surviving far span of the host's
// bitten loop: the far path runs from the inner bite's last outer end back to its first, avoiding the
// bitten trihedral vertex. On a FAR-END corner host (the wall) the outer ends are the arm rails' off-loop
// runout feet, so the far path is anchored on the window loop AUGMENTED by the arms' through-vertex
// extension edges (canalFarSpan, F2 derivation §5-6); with none it is farPathSegs verbatim. Honest-declines
// (carrying the measured outer-end→loop anchor gaps) when an outer end neither anchors nor extends onto the
// bitten loop (risk #1) — never snapped.
func canalCloseFar(host *topo.Face, star *topo.Loop, inner []endSeg, bundles []canalArmBundle, w cornerWeld, tol float64) (filletLoop, string) {
	segs := segsFromLoop(star)
	if len(segs) < 3 {
		return filletLoop{}, fmt.Sprintf("canal host bite (%T): bitten loop has %d edges, need ≥3", host.Geometry(), len(segs))
	}
	outerA, outerB := inner[0].from, inner[len(inner)-1].to
	far, ok := canalFarSpan(segs, outerA, outerB, extensionsOnHost(host, bundles), bittenVertex(segs, w.center), tol)
	if !ok {
		return filletLoop{}, canalAnchorDeclineReason(host, segs, outerA, outerB, tol)
	}
	return loopFromSegs(append(inner, far...)), ""
}

// canalAnchorDeclineReason names WHY the far span could not close, carrying the measured distance from each
// inner-bite OUTER end to the host's bitten loop edges (the risk #1 anchor evidence, ANALYTIC — point-to-
// arc / point-to-segment, not the sampled nearestOnLoopEdges). A gap ≫ tol is a real host-extent / rail-
// runout defect to escalate, never a tolerance to widen.
func canalAnchorDeclineReason(host *topo.Face, segs []endSeg, outerA, outerB math.Point3, tol float64) string {
	gA, gB := distPointToLoopEdges(segs, outerA), distPointToLoopEdges(segs, outerB)
	return fmt.Sprintf("canal host bite (%T): far span will not close — outer ends off the bitten loop (A=%.4e B=%.4e, tol %.3e)", host.Geometry(), gA, gB, tol)
}

// chainBiteSegs orders segs into one head-to-tail chain by shared endpoints, reversing segments and
// extending at BOTH ends as needed (so segs[0] need not be a path endpoint). It DECLINES (naming the
// unattached segment + its nearest gap) when a segment cannot join either free end within tol — a
// non-meeting junction is a REAL defect (a rail at the wrong reflected centre / a mis-tagged foot-locus),
// never a tolerance to widen (risk #2). The result runs outer→…→outer for a canal host bite. Example:
//
//	inner, reason := chainBiteSegs([]endSeg{railA, railB, footBridge}, tol)
func chainBiteSegs(segs []endSeg, tol float64) ([]endSeg, string) {
	if len(segs) == 0 {
		return nil, "chainBiteSegs: no segments to chain"
	}
	chain := []endSeg{segs[0]}
	used := make([]bool, len(segs))
	used[0] = true
	for placed := 1; placed < len(segs); placed++ {
		next, ok := attachChainSeg(chain, segs, used, tol)
		if !ok {
			return nil, chainBiteDeclineReason(chain, segs, used, tol)
		}
		chain = next
	}
	return chain, ""
}

// attachChainSeg attaches one unused segment to whichever free end of the chain it meets within tol
// (appending at the tail or prepending at the head, oriented to stay continuous), marking it used. A
// reversed segment keeps its curve (reverseChainSeg), so a foot-locus bridge never loses its BSpline.
func attachChainSeg(chain, segs []endSeg, used []bool, tol float64) ([]endSeg, bool) {
	tail, head := chain[len(chain)-1].to, chain[0].from
	for i, s := range segs {
		if used[i] {
			continue
		}
		if oriented, ok := orientChainSeg(s, tail, tol); ok {
			used[i] = true
			return append(chain, oriented), true
		}
		if oriented, ok := orientChainSeg(s, head, tol); ok {
			used[i] = true
			return append([]endSeg{reverseChainSeg(oriented)}, chain...), true
		}
	}
	return chain, false
}

// orientChainSeg returns s (or its curve-preserving reverse) oriented to START at p, or ok=false when
// neither end of s is within tol of p.
func orientChainSeg(s endSeg, p math.Point3, tol float64) (endSeg, bool) {
	if float64(s.from.DistanceTo(p)) <= tol {
		return s, true
	}
	if float64(s.to.DistanceTo(p)) <= tol {
		return reverseChainSeg(s), true
	}
	return endSeg{}, false
}

// reverseChainSeg reverses one segment for chaining while PRESERVING its curve: an arc is re-derived
// through its midpoint (reverseEndSegs), a general curve (a foot-locus BSpline) keeps its curve object
// with endpoints swapped, and a straight seg just swaps ends — so no chained edge loses its geometry
// (reverseEndSegs alone drops a non-arc curve, which would strip a reversed foot-locus).
func reverseChainSeg(s endSeg) endSeg {
	if s.arc {
		return reverseEndSegs([]endSeg{s})[0]
	}
	return endSeg{from: s.to, to: s.from, curve: s.curve}
}

// chainBiteDeclineReason names the first unattached segment and its nearest gap to either free chain end — the
// precise diagnostic for a non-meeting junction (risk #2), so a decline points at the offending rail /
// foot-locus and its distance, not a bare "did not chain".
func chainBiteDeclineReason(chain, segs []endSeg, used []bool, tol float64) string {
	tail, head := chain[len(chain)-1].to, chain[0].from
	for i, s := range segs {
		if used[i] {
			continue
		}
		g := stdmath.Min(minEndGap(s, tail), minEndGap(s, head))
		return fmt.Sprintf("segment %d does not chain onto either free end (nearest gap %.4e > tol %.3e)", i, g, tol)
	}
	return fmt.Sprintf("chain incomplete (%d/%d segments placed)", len(chain), len(segs))
}

// minEndGap is the smaller distance from either end of s to p.
func minEndGap(s endSeg, p math.Point3) float64 {
	return stdmath.Min(float64(s.from.DistanceTo(p)), float64(s.to.DistanceTo(p)))
}

// distPointToLoopEdges is the analytic nearest distance from p to the loop's edges (exact point-to-arc on
// a rounded edge, point-to-segment on a straight one) — the risk #1 anchor metric, replacing the sampled
// nearestOnLoopEdges' O(chord) error with the true distance the splice tolerance is judged against.
func distPointToLoopEdges(segs []endSeg, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, s := range segs {
		if d := distPointToEdge(s, p); d < best {
			best = d
		}
	}
	return best
}

// distPointToEdge is the analytic distance from p to one loop edge (its arc when rounded, else its chord).
func distPointToEdge(s endSeg, p math.Point3) float64 {
	if arc, ok := s.curve.(geom.Arc3d); ok && s.arc {
		return distPointToArc(arc, p)
	}
	return distPointToSeg(s.from, s.to, p)
}

// distPointToArc is the exact distance from p to a circular arc: to the nearest circle point when p's
// in-plane azimuth falls within the arc's sweep, else to the nearer arc endpoint.
func distPointToArc(arc geom.Arc3d, p math.Point3) float64 {
	w := arc.Center.VectorTo(p)
	nv := arc.Normal.AsVector()
	axial := w.Dot(nv)
	inplane := w.Sub(nv.Scale(axial))
	if arcAzimuthInSweep(arc, inplane) {
		return stdmath.Hypot(float64(inplane.Length())-arc.Radius, float64(axial))
	}
	return stdmath.Min(float64(p.DistanceTo(arc.PointAt(0))), float64(p.DistanceTo(arc.PointAt(1))))
}

// arcAzimuthInSweep reports whether inplane's azimuth (about the arc centre) lies within the arc's sweep.
func arcAzimuthInSweep(arc geom.Arc3d, inplane math.Vector3) bool {
	bin := arc.Normal.Cross(arc.RefDir)
	raw := stdmath.Atan2(float64(inplane.Dot(bin)), float64(inplane.Dot(arc.RefDir.AsVector())))
	frac := wrapToSweep(raw-arc.StartAngle, arc.SweepAngle) / arc.SweepAngle
	return frac >= 0 && frac <= 1
}

// distPointToSeg is the distance from p to segment a→b (clamped orthogonal projection).
func distPointToSeg(a, b, p math.Point3) float64 {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return float64(p.DistanceTo(a))
	}
	t := stdmath.Max(0, stdmath.Min(1, float64(a.VectorTo(p).Dot(d))/l2))
	return float64(p.DistanceTo(a.TranslateBy(d.Scale(math.Scalar(t)))))
}
