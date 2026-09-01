// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corner-weld RAIL LEDGER (corner-weld-layer-design.md ADR-2). Every crack in this vein has traced
// back to the same defect: two faces that share a boundary each built their OWN congruent-but-not-identical
// curve, so assembleBody's point weld found four endpoints where it needed two. The existing welds carry
// that as DISCIPLINE — armRails keeps hostA/hostB as fields so the retrim consumes the identical object,
// cylinderRulingOuterOnHost exists to be "the SINGLE source of truth", n4PatchFace emits single curve-segs
// rather than a sampled polyline. Four hand-maintained couplings for two corner classes.
//
// What the ledger actually buys, stated honestly: SINGLE STORAGE PLUS A CERTIFICATE. A shared boundary is
// registered ONCE and handed out by handle (railID); a face never holds a curve, only a handle, so two
// faces reading one handle cannot diverge. It does NOT make re-derivation "impossible by construction" —
// `add` does not dedupe, so a builder that registers two congruent curves under two handles still gets two
// curve objects. That residue is caught, not prevented: certifyTwoIncident refuses to assemble unless every
// registered rail was claimed by exactly two DISTINCT faces, turning a rail one face forgot to reference
// (and a congruent duplicate, which shows up as two 1-incident rails) into a NAMED plan-level decline
// BEFORE assembleBody, instead of a downstream Validate.Closed=false with no pointer to the cause.

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

// railClaimant identifies WHICH face is claiming a rail. 2-incidence is a statement about distinct FACES,
// so the ledger must be able to tell one face's two references to a rail from two faces' one reference
// each — counting raw calls certifies a self-paired boundary as clean.
type railClaimant struct {
	host  *topo.Face // the existing body face being retrimmed; nil for a newly emitted blend face
	blend int        // ordinal of the emitted blend face when host == nil
}

// blendClaimant identifies the i-th blend face the plan emits (arm faces, then the corner patch).
func blendClaimant(i int) railClaimant { return railClaimant{blend: i} }

// hostClaimant identifies an existing body face — a retrimmed arm host or a far cap.
func hostClaimant(f *topo.Face) railClaimant { return railClaimant{host: f, blend: -1} }

// cornerWeldLedger holds every shared boundary curve of one corner site exactly once, plus the multiset of
// FACES that claimed each rail so the weld can certify 2-incidence before assembling.
type cornerWeldLedger struct {
	claims []map[railClaimant]int // per rail: claimant → how many times that one face claimed it
	rails  []endSeg
	names  []string // per-rail diagnostic label, so a decline names WHICH boundary is unpaired
}

// newCornerWeldLedger returns an empty ledger. Example:
//
//	led := newCornerWeldLedger()
//	near := led.add("torus/near", nearSeg)
//	armSeg, ok := led.seg(near, railForward, blendClaimant(0))
//	patchSeg, ok := led.seg(near, railReversed, blendClaimant(1))
//	if why := led.certifyTwoIncident(); why != "" { return nil, why }
func newCornerWeldLedger() *cornerWeldLedger {
	return &cornerWeldLedger{}
}

// add registers one shared boundary curve and returns its handle. name is a diagnostic label only. It does
// NOT dedupe — see the file header for the honest scope of the invariant.
func (l *cornerWeldLedger) add(name string, s endSeg) railID {
	l.rails = append(l.rails, s)
	l.claims = append(l.claims, map[railClaimant]int{})
	l.names = append(l.names, name)
	return railID(len(l.rails) - 1)
}

// seg resolves a handle to its endSeg in direction d, recording by's claim on it. Every call is one face
// claiming this boundary, so certifyTwoIncident can check each rail was claimed by exactly two distinct
// faces.
//
// ok=false for an UNSET or out-of-range handle (noRail). Nothing is registered under such a handle, so
// there is no curve to hand out AND no certificate can report it — certifyTwoIncident iterates the
// REGISTERED rails, and an unregistered one is invisible to it. Handing back a zero endSeg would weld a
// degenerate segment into a face loop, so the caller must decline instead.
func (l *cornerWeldLedger) seg(id railID, d railSense, by railClaimant) (endSeg, bool) {
	if id < 0 || int(id) >= len(l.rails) {
		return endSeg{}, false
	}
	l.claims[id][by]++
	if d == railReversed {
		return reverseEndSegs([]endSeg{l.rails[id]})[0], true
	}
	return l.rails[id], true
}

// chain resolves a rail chain in order (railForward) or reversed end-to-end (railReversed — the whole
// chain traversed backwards, each piece reversed), recording one claim per piece. ok=false if any handle
// is unset, on the same grounds as seg.
func (l *cornerWeldLedger) chain(ids []railID, d railSense, by railClaimant) ([]endSeg, bool) {
	out := make([]endSeg, 0, len(ids))
	for i := range ids {
		id, sense := ids[i], railForward
		if d == railReversed {
			id, sense = ids[len(ids)-1-i], railReversed
		}
		s, ok := l.seg(id, sense, by)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// ends returns a rail chain's outer endpoints (chain[0].from, chain[last].to) WITHOUT recording a claim —
// the executor needs the feet to build neighbouring geometry, which is not a claim.
func (l *cornerWeldLedger) ends(ids []railID) (math.Point3, math.Point3, bool) {
	if len(ids) == 0 || ids[0] < 0 || int(ids[len(ids)-1]) >= len(l.rails) {
		return math.Point3{}, math.Point3{}, false
	}
	return l.rails[ids[0]].from, l.rails[ids[len(ids)-1]].to, true
}

// certifyTwoIncident is invariant #2 of the layer contract: every registered rail must have been claimed
// by EXACTLY two DISTINCT faces. A rail claimed by one face is a crack (the other face left that boundary
// open); by three is a non-manifold weld; TWICE BY ONE FACE is a self-pairing that would slip past a
// reference count while still leaving the neighbouring face's boundary open. Returning the offending rail's
// name + count turns what used to be a downstream "assembled weld did not certify as a valid solid" into a
// decline that says WHICH boundary and HOW it is wrong — the single biggest debugging-cost reduction the
// design identifies.
func (l *cornerWeldLedger) certifyTwoIncident() string {
	for i, claims := range l.claims {
		if face, n := repeatedClaimant(claims); n > 1 {
			return fmt.Sprintf("corner weld: rail %q was claimed %d times by ONE face (%+v) — a face cannot"+
				" pair a boundary with itself (%d rails total)", l.names[i], n, face, len(l.rails))
		}
		if n := len(claims); n != 2 {
			return fmt.Sprintf("corner weld: rail %q is %d-incident (want exactly 2 distinct faces sharing it; %d rails total)",
				l.names[i], n, len(l.rails))
		}
	}
	return ""
}

// repeatedClaimant returns a claimant that claimed the rail more than once, and its count (0 when none did).
func repeatedClaimant(claims map[railClaimant]int) (railClaimant, int) {
	for c, n := range claims {
		if n > 1 {
			return c, n
		}
	}
	return railClaimant{}, 0
}
