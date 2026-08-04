// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/math"
)

// The OUTWARD-growing half of the retrim loop machinery (split from fillet_curved_retrim_loop.go,
// which keeps the ring read/split/far-path core): a BORE-corner (R+r) fillet's far cross-section
// reaches r PAST the host rim, so a rail landing or bite endpoint can land on the EXTENSION of a
// collinear straight survivor edge rather than on the loop itself (corner-blend-weld Piece 3, L9).
// Everything here grows the loop to meet such a point; a convex green, whose landings sit ON the
// loop, never diverts through any of it and stays byte-identical.

// extendStraightSegToLanding handles a BORE-corner runout where an arm's outer rail landing lands on the
// EXTENSION of a straight survivor edge, r beyond its endpoint (corner-blend-weld Piece 3, L9): the torus
// cap contact of a notch-wall fillet sits at radius R+r, so where the rim arc runs out onto a collinear
// flat radial notch face the fillet reaches r past the old rim corner (top plane: (50,20)→(50,25)). When
// p is OFF the loop yet collinear with a straight seg and just beyond one endpoint, that seg is grown to
// p so the far path can close through it. It is a NO-OP when p already lies on the loop (segParam) or on
// no seg's line — so every convex green, whose rail landings sit on the loop, is byte-identical.
func extendStraightSegToLanding(segs []endSeg, p math.Point3, tol float64) []endSeg {
	for _, s := range segs {
		if _, ok := segParam(s, p, tol); ok {
			return segs // already interior to the loop — nothing to extend
		}
	}
	out := make([]endSeg, len(segs))
	extended := false
	for i, s := range segs {
		if grown, ok := grownStraightSeg(s, p, tol); ok && !extended {
			out[i], extended = grown, true
			continue
		}
		out[i] = s
	}
	return out
}

// boreExtendBite splices a bore far-cap bite that GROWS the host loop OUTWARD (corner-blend-weld Piece 3,
// L9): a notch-wall (R+r) fillet's far cross-section reaches r PAST the rim, so ONE bite endpoint (the
// tip) lands on the EXTENSION of a collinear straight survivor edge while the other (the foot) lies on
// the loop — the opposite of spliceCornerBite's inward corner removal. The rim corner between them is
// replaced by: the survivor edge grown to the tip, the bite (tip→foot), then the loop resumed at the foot
// on the FOLLOWING edge. ok=false unless exactly one endpoint is the off-loop collinear tip and the grown
// edge's far end is that tip (a single orientation; the mirror is a later case) — so a convex bite (both
// endpoints on the loop) never diverts here and stays byte-identical.
func boreExtendBite(segs []endSeg, bite endSeg, tol float64) ([]endSeg, bool) {
	fromOn, toOn := pointOnRing(segs, bite.from, tol), pointOnRing(segs, bite.to, tol)
	if fromOn == toOn {
		return nil, false // both on the loop (spliceCornerBite) or both off (not this case)
	}
	tip, tipToFoot := bite.from, bite // from is the off-loop tip; bite already runs tip(from)→foot(to)
	if fromOn {
		tip, tipToFoot = bite.to, reverseEndSegs([]endSeg{bite})[0] // to is the off-loop tip; orient tip→foot
	}
	gi := collinearGrowEdge(segs, tip, tol)
	if gi < 0 {
		return nil, false // no straight survivor edge whose TO grows to the tip
	}
	return growWalk(segs, gi, tip, tipToFoot, tol)
}

// pointOnRing reports whether p is a vertex or interior point of loop segs.
func pointOnRing(segs []endSeg, p math.Point3, tol float64) bool {
	for _, s := range segs {
		if float64(s.from.DistanceTo(p)) <= tol || float64(s.to.DistanceTo(p)) <= tol {
			return true
		}
		if _, ok := segParam(s, p, tol); ok {
			return true
		}
	}
	return false
}

// collinearGrowEdge returns the index of the straight seg whose TO endpoint grows out to tip (tip on its
// line, beyond its to), or −1. This is the edge the bore far cap extends past the rim (L9's flat radial
// top edge (50,0,100)→(50,20,100), grown to (50,25,100)).
func collinearGrowEdge(segs []endSeg, tip math.Point3, tol float64) int {
	for i, s := range segs {
		if grown, ok := grownStraightSeg(s, tip, tol); ok && float64(grown.to.DistanceTo(tip)) <= tol {
			return i
		}
	}
	return -1
}

// growWalk rebuilds the loop from the grown edge gi: grown(gi.from→tip), the bite (tip→foot), the FOLLOWING
// edge split at the foot (foot→its to, dropping the rim corner), then every other survivor edge around the
// ring back to gi. Declines when the foot is not on the following edge (the geometry is not the expected
// grow-then-bite corner).
func growWalk(segs []endSeg, gi int, tip math.Point3, bite endSeg, tol float64) ([]endSeg, bool) {
	n := len(segs)
	grown, _ := grownStraightSeg(segs[gi], tip, tol)
	next := segs[(gi+1)%n]
	if _, ok := segParam(next, bite.to, tol); !ok {
		return nil, false // the foot is not interior to the edge after the grown one — not this corner
	}
	out := []endSeg{grown, bite, subSeg(next, bite.to, next.to)}
	for k := (gi + 2) % n; k != gi; k = (k + 1) % n {
		out = append(out, segs[k])
	}
	return out, true
}

// grownStraightSeg extends straight seg s to endpoint p when p is collinear with s (perp distance ≤ tol)
// and lies just BEYOND one of its endpoints (the extension, not the interior). The endpoint p replaces
// is the near one; the old rim corner it grows past is absorbed by the fillet. ok=false for an arc, a
// non-collinear p, or a p between the endpoints (already handled by insertSplits).
func grownStraightSeg(s endSeg, p math.Point3, tol float64) (endSeg, bool) {
	if s.arc || s.curve != nil {
		return endSeg{}, false // never grow a circular or elliptic survivor edge — only straight lines extend
	}
	d := s.from.VectorTo(s.to)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return endSeg{}, false
	}
	t := float64(s.from.VectorTo(p).Dot(d)) / l2
	if float64(s.from.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p)) > tol {
		return endSeg{}, false // not on this seg's line
	}
	if t < 0 {
		return endSeg{from: p, to: s.to}, true // beyond the from endpoint
	}
	if t > 1 {
		return endSeg{from: s.from, to: p}, true // beyond the to endpoint
	}
	return endSeg{}, false // interior — insertSplits handles it
}
