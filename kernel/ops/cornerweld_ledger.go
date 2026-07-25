// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/math"
)

// The corner-weld RAIL LEDGER (corner-weld-layer-design.md ADR-2). Every crack in this vein has traced
// back to the same defect: two faces that share a boundary each built their OWN congruent-but-not-identical
// curve, so assembleBody's point weld found four endpoints where it needed two. The existing welds carry
// that as DISCIPLINE — armRails keeps hostA/hostB as fields so the retrim consumes the identical object,
// cylinderRulingOuterOnHost exists to be "the SINGLE source of truth", n4PatchFace emits single curve-segs
// rather than a sampled polyline. Four hand-maintained couplings for two corner classes.
//
// The ledger makes it STRUCTURAL: a shared boundary is registered ONCE and handed out by handle (railID).
// A face cannot re-derive it, because a face never holds a curve — it holds a handle. certifyTwoIncident
// then turns the residual failure mode (a rail one face forgot to reference) into a NAMED plan-level
// decline BEFORE assembleBody, instead of a downstream Validate.Closed=false with no pointer to the cause.

// railID is a handle into a cornerWeldLedger. Two faces sharing a boundary carry the SAME railID, so they
// cannot diverge — the executor resolves the handle to one endSeg. Axis F of the design becomes structural.
type railID int

// noRail is the zero-value-safe "unset" handle. It is NOT railID(0), which is the first real rail.
const noRail railID = -1

// railSense selects the orientation a face reads a shared rail in. The ledger stores each rail once, in one
// direction; the second face reads it reversed (reverseEndSegs), which is the same curve traversed the
// other way — never a re-derived one.
type railSense int

const (
	railForward  railSense = iota // as registered
	railReversed                  // reverseEndSegs of the registered seg
)

// cornerWeldLedger holds every shared boundary curve of one corner site exactly once, plus a reference
// count per rail so the weld can certify 2-incidence before assembling.
type cornerWeldLedger struct {
	rails []endSeg
	refs  []int
	names []string // per-rail diagnostic label, so a decline names WHICH boundary is unpaired
}

// newCornerWeldLedger returns an empty ledger. Example:
//
//	led := newCornerWeldLedger()
//	near := led.add("torus/near", nearSeg)
//	armLoop := []endSeg{led.seg(near, railForward), …}
//	patchLoop := []endSeg{led.seg(near, railReversed), …}
//	if why := led.certifyTwoIncident(); why != "" { return nil, why }
func newCornerWeldLedger() *cornerWeldLedger {
	return &cornerWeldLedger{}
}

// add registers one shared boundary curve and returns its handle. name is a diagnostic label only.
func (l *cornerWeldLedger) add(name string, s endSeg) railID {
	l.rails = append(l.rails, s)
	l.refs = append(l.refs, 0)
	l.names = append(l.names, name)
	return railID(len(l.rails) - 1)
}

// seg resolves a handle to its endSeg in direction d, counting the reference. Every call is one face
// claiming this boundary, so certifyTwoIncident can check each rail was claimed exactly twice.
func (l *cornerWeldLedger) seg(id railID, d railSense) endSeg {
	if id < 0 || int(id) >= len(l.rails) {
		return endSeg{} // an unset handle: the certificate below reports the unpaired rail
	}
	l.refs[id]++
	if d == railReversed {
		return reverseEndSegs([]endSeg{l.rails[id]})[0]
	}
	return l.rails[id]
}

// chain resolves a rail chain in order (railForward) or reversed end-to-end (railReversed — the whole
// chain traversed backwards, each piece reversed), counting one reference per piece.
func (l *cornerWeldLedger) chain(ids []railID, d railSense) []endSeg {
	out := make([]endSeg, 0, len(ids))
	if d == railForward {
		for _, id := range ids {
			out = append(out, l.seg(id, railForward))
		}
		return out
	}
	for i := len(ids) - 1; i >= 0; i-- {
		out = append(out, l.seg(ids[i], railReversed))
	}
	return out
}

// ends returns a rail chain's outer endpoints (chain[0].from, chain[last].to) WITHOUT counting a
// reference — the executor needs the feet to build neighbouring geometry, which is not a claim.
func (l *cornerWeldLedger) ends(ids []railID) (math.Point3, math.Point3, bool) {
	if len(ids) == 0 || ids[0] < 0 || int(ids[len(ids)-1]) >= len(l.rails) {
		return math.Point3{}, math.Point3{}, false
	}
	return l.rails[ids[0]].from, l.rails[ids[len(ids)-1]].to, true
}

// certifyTwoIncident is invariant #2 of the layer contract: every registered rail must have been claimed
// by EXACTLY two faces. A rail claimed once is a crack (one face left that boundary open); a rail claimed
// three times is a non-manifold weld. Returning the offending rail's name + count turns what used to be a
// downstream "assembled weld did not certify as a valid solid" into a decline that says WHICH boundary and
// HOW it is wrong — the single biggest debugging-cost reduction the design identifies.
func (l *cornerWeldLedger) certifyTwoIncident() string {
	for i, n := range l.refs {
		if n != 2 {
			return fmt.Sprintf("corner weld: rail %q is %d-incident (want exactly 2 faces sharing it; %d rails total)",
				l.names[i], n, len(l.rails))
		}
	}
	return ""
}
