// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The CHAIN-CAPABLE host-loop rebuild — the successor to transformLoop's single-vertex substitution.
//
// WHAT transformLoop CAN SAY. Its `subs`/`ends` branches replace one loop VERTEX with a point (or a
// tIn→tOut corner round) at that vertex's own position in the ring. Every rebuilt boundary therefore
// still passes through the same edges, in the same order: a substitution can only slide a corner ALONG
// one of the two edges meeting at it.
//
// WHAT IT CANNOT SAY, and what this file adds. A run-out that reaches past the face's own boundary
// consumes one or more WHOLE rim edges and stops part-way along a later one. Three independently-derived
// corpus roots are exactly that shape:
//   - the far-end trim landing off the stop FACE (complex/D8: the band's terminal section leaves the
//     radius-24 corner round at its u=0 ruling and the round's whole far rim arc must disappear —
//     selfcross-trim-report.md §5.1);
//   - the setback band overrunning its own host (simple/Y2, Y4: the receded boundary line crosses the
//     host's notch, so the substituted tangent point sits past the adjacent edge's own end —
//     planar-retrim-selfcross-report.md §8.1);
//   - tangent-chain propagation, whose whole premise is that one blend spans a chain of faces.
// Each was blocked on the same missing sentence, which the retrim layer can now say.
//
// THE SEAM. chainRetrimLoop is pure and topo-free: it speaks the retrim layer's own value type (endSeg)
// plus the geom.Surface the ring lives on, and knows nothing about fillets, corners or faces. The
// surface is there for ONE reason — the smaller-span pick is an AREA, and an area on a curved host is
// only meaningful in that host's own metric (fillet_chain_span_area.go). Producing the CONTACT CHAIN is
// the consumer's job — the far-end trim builds it from the band∩wall section, an obstacle-limited
// setback from the band∩obstacle imprint — and absorbing it into the host's boundary is this file's.
// Nothing here is wired into transformLoop: the existing callers keep their own splices verbatim, so a
// consumer adopts the primitive one at a time (strangler) and the 111 corpus greens flowing through
// transformLoop cannot move until one does.

// chainRetrimLoop re-trims a host face's boundary RING against a retrim CHAIN that may OVERRUN it,
// consuming whole edges of the ring where the chain's own span replaces them.
//
// The chain runs head→tail along the material the fillet leaves; either extreme may lie off the ring
// (the overrun). It is first clipped back to the boundary, then spliced in place of the SMALLER-AREA
// span between its two landings — the same criterion, and the same splice, the single-ball and canal
// imprints already use, so a chain that does not overrun rebuilds exactly as it does today.
//
// Example: the Y2 host plane's ring plus the setback line (100,0,85)→(0,0,85), which runs past the
// host's own slot wall, rebuilds as the 6-edge loop of closed-form area 8450.
func chainRetrimLoop(host geom.Surface, ring, chain []endSeg, tol float64) ([]endSeg, bool) {
	clipped, ok := clipChainToRing(ring, chain, tol)
	if !ok {
		return nil, false
	}
	return spliceCornerBiteChain(host, ring, clipped, tol)
}

// clipChainToRing trims a retrim chain's OVERRUN — the leading and trailing runs that lie past the host
// ring — back to the boundary, so both extremes land ON it.
//
// It is the primitive's INVARIANT: whatever it returns, chainLandsOnRing holds for it. A chain that
// already lands on the ring is returned UNCHANGED (so routing an existing splice through here is
// byte-identical), and a chain whose overrun never comes back is DECLINED rather than snapped to the
// nearest boundary point — a guessed landing is a silently wrong solid, a decline is the do-no-harm
// floor the rest of the retrim layer already falls to.
func clipChainToRing(ring, chain []endSeg, tol float64) ([]endSeg, bool) {
	if len(ring) < 3 || len(chain) == 0 {
		return nil, false
	}
	head, ok := clipChainHead(ring, chain, tol)
	if !ok {
		return nil, false
	}
	out, ok := clipChainTail(ring, head, tol)
	if !ok {
		return nil, false
	}
	if !chainLandsOnRing(ring, out, tol) {
		return nil, false // the clip did not put both extremes on the boundary — decline, never guess
	}
	return out, true
}

// chainLandsOnRing reports whether BOTH extremes of the chain lie on the ring — the precondition
// spliceCornerBiteChain needs (it splits the ring at them) and the postcondition clipChainToRing owes
// it. Stated once, here, so the two cannot drift apart.
func chainLandsOnRing(ring, chain []endSeg, tol float64) bool {
	if len(chain) == 0 {
		return false
	}
	return pointOnRing(ring, chain[0].from, tol) && pointOnRing(ring, chain[len(chain)-1].to, tol)
}

// clipChainHead cuts the chain's LEADING overrun off: every segment before the chain first reaches the
// ring is dropped, and the segment that reaches it is trimmed to start at the meeting.
func clipChainHead(ring, chain []endSeg, tol float64) ([]endSeg, bool) {
	if pointOnRing(ring, chain[0].from, tol) {
		return chain, true // already lands on the boundary — the existing splices' behaviour, verbatim
	}
	for i, s := range chain {
		t, p, ok := ringMeetOnSeg(ring, s, tol, true)
		if !ok {
			continue // this whole segment runs outside the face: it is all overrun
		}
		return append([]endSeg{segFromParam(s, t, p)}, chain[i+1:]...), true
	}
	return nil, false // the overrun never returns to the boundary — nothing to splice onto
}

// clipChainTail is clipChainHead's mirror, walking the chain from its last segment backwards. It is a
// separate walk rather than a reversal because a chain segment's carried curve is not reversible in
// general — reverseChainSeg keeps a non-arc curve's concrete object and only swaps the endpoints
// (fillet_curved_canal_bite.go, so a SpiricArc survives the type switch) — so a reversed segment's
// parameter still runs the ORIGINAL way and trimming it would keep the overrun instead of dropping it.
func clipChainTail(ring, chain []endSeg, tol float64) ([]endSeg, bool) {
	if pointOnRing(ring, chain[len(chain)-1].to, tol) {
		return chain, true
	}
	for i := len(chain) - 1; i >= 0; i-- {
		t, p, ok := ringMeetOnSeg(ring, chain[i], tol, false)
		if !ok {
			continue
		}
		return append(append([]endSeg{}, chain[:i]...), segToParam(chain[i], t, p)), true
	}
	return nil, false
}

// segFromParam is the portion of segment s from parameter t to its end, starting at the already-computed
// point p. A circular or elliptic segment is re-derived through the retrim layer's own exact sub-span
// algebra (subSeg, which keeps a major span major); any other carried curve is restricted with a
// TrimmedCurve3 so the kept piece stays ON the producer's curve instead of degrading to a chord.
func segFromParam(s endSeg, t float64, p math.Point3) endSeg {
	if t <= 0 {
		return s
	}
	if sub, ok := exactSubSegTo(s, p, true); ok {
		return sub
	}
	lo, hi := s.curve.Domain()
	trimmed := geom.TrimmedCurve3{Base: s.curve, Lo: lo + t*(hi-lo), Hi: hi}
	return endSeg{from: p, to: s.to, curve: trimmed, mid: trimmed.PointAt(0.5)}
}

// segToParam is the portion of segment s from its start up to parameter t, ending at p — segFromParam's
// mirror for the trailing clip.
func segToParam(s endSeg, t float64, p math.Point3) endSeg {
	if t >= 1 {
		return s
	}
	if sub, ok := exactSubSegTo(s, p, false); ok {
		return sub
	}
	lo, hi := s.curve.Domain()
	trimmed := geom.TrimmedCurve3{Base: s.curve, Lo: lo, Hi: lo + t*(hi-lo)}
	return endSeg{from: s.from, to: p, curve: trimmed, mid: trimmed.PointAt(0.5)}
}

// exactSubSegTo handles the segment kinds the retrim layer can re-derive from the parent's OWN
// parameters — a straight edge and the circular/elliptic families subSeg already owns — so the clipped
// piece keeps its analytic type instead of becoming a restriction wrapper. ok=false hands any other
// carried curve back to the caller's TrimmedCurve3 path.
func exactSubSegTo(s endSeg, p math.Point3, keepTail bool) (endSeg, bool) {
	from, to := s.from, p
	if keepTail {
		from, to = p, s.to
	}
	if s.curve == nil {
		return endSeg{from: from, to: to}, true
	}
	if _, elliptic := s.curve.(geom.EllipticalArc); s.arc || elliptic {
		return subSeg(s, from, to), true
	}
	return endSeg{}, false
}
