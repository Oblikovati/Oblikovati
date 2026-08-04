// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-host retrim splice machinery for the mixed-sense curved-host 2r-torus corner weld
// (fillet_curved_mixed_weld.go). Split out of that file for the CLAUDE.md files-under-500 rule — a pure
// move, no behaviour change. These helpers re-clip a two-arm corner host (the two picked edges meeting at
// the trihedral vertex, replaced by the arm rails + optional top-contact arc, flanks re-terminated) and
// grow a concave far-runout cap around its cross-section arc, generalising the single-arm concave grow
// retrim (fillet_arm_concave_retrim.go: concaveRetrimLoop / concaveCapLoop) from one picked edge to two.

// cornerBite is one arm's contribution to a two-arm corner-host retrim: its host contact rail (oriented
// far→near) and the picked edge whose loop segment the rail replaces.
type cornerBite struct {
	rail endSeg
	edge *topo.Edge
}

// mixedCornerHostRetrim re-clips one two-arm corner host: the two picked edges meeting at vCorner are
// removed, replaced by [firstRail(far→near), mid…, secondRail(near→far)], and the two OUTER flanking edges
// are re-terminated onto the chain's extreme feet (grow OR recede, along each flank's own supporting
// line/circle). mid is the top-contact arc (d) on the shared top plane, nil on the two walls (their rails
// meet at a single triple point).
//
// This is now a thin ADAPTER over the corner-weld layer's chain splice (cornerChainHostRetrim): a bite of
// one rail replacing one loop edge is the single-element case of a rail CHAIN replacing a contiguous RUN,
// so there is one implementation, not two. The layer's splice reduces to the previous code path
// element-for-element when both runs and both chains have length 1 — which is every call here (M8's three
// corner hosts), so M8's loop seg sequences are unchanged.
func mixedCornerHostRetrim(host *topo.Face, biteA, biteB cornerBite, mid []endSeg, vCorner math.Point3, tol float64) (filletFace, bool) {
	return cornerChainHostRetrim(host, singleChainBite(biteA), singleChainBite(biteB), mid, vCorner, tol)
}

// singleChainBite lifts a one-rail / one-edge cornerBite into the layer's chain form.
func singleChainBite(b cornerBite) cornerHostChainBite {
	return cornerHostChainBite{rails: []endSeg{b.rail}, consumed: []*topo.Edge{b.edge}}
}

// reterminateBothEnds re-terminates a single flanking edge on BOTH ends (the n==3 corner-host loop, where
// the two picked edges leave one shared flank): its from→newFrom and to→newTo, both on the flank's own
// supporting line (straight) or circle (arc). Declines when a foot leaves that geometry.
func reterminateBothEnds(s endSeg, newFrom, newTo math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newFrom, tol) || !pointOnLine(s.from, s.to, newTo, tol) {
			return endSeg{}, false
		}
		return endSeg{from: newFrom, to: newTo}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), newFrom, newTo, tol)
}

// growCapArc grows a concave far-runout cap around one cross-section arc: the far-vertex corner is replaced
// by the arc and the two flanking edges meeting there are re-terminated onto the arc's feet (each on its
// own host-shared line/circle) — the segs-level form of concaveCapLoop, so a cap shared by two concave
// arms accumulates both bites. Declines when the far vertex is not a loop corner or a foot is off a flank.
func growCapArc(segs []endSeg, arc endSeg, far math.Point3, tol float64) ([]endSeg, bool) {
	n := len(segs)
	j := indexOfVertex(segs, far, tol)
	if j < 0 || n < 3 {
		return nil, false
	}
	prevIdx := (j - 1 + n) % n
	arcSeg, ok := matchArcFeet(segs[prevIdx], segs[j], arc, tol)
	if !ok {
		return nil, false
	}
	prev, okp := reterminateSegTo(segs[prevIdx], arcSeg.from, tol)
	next, okn := reterminateSegFrom(segs[j], arcSeg.to, tol)
	if !okp || !okn {
		return nil, false
	}
	return spliceCapRing(segs, prevIdx, j, prev, arcSeg, next), true
}
