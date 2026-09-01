// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// zzLedgerSeg is a throwaway straight boundary for the ledger's bookkeeping tests (the ledger stores
// endSegs opaquely — it never inspects the curve).
func zzLedgerSeg(x float64) endSeg {
	return endSeg{from: math.P3(x, 0, 0), to: math.P3(x, 1, 0)}
}

// TestLedgerEveryRailTwoIncident is the layer's OWN structural invariant (design invariant #2): a shared
// boundary must be claimed by EXACTLY two faces. This is the test that makes rail identity a certificate
// rather than a habit — a future refactor that drops one face's claim (the crack that has cost this vein a
// case at a time) fails HERE, with the rail named, instead of surfacing downstream as Validate.Closed=false.
func TestLedgerEveryRailTwoIncident(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	shared := led.add("arm/near", zzLedgerSeg(0))
	if why := led.certifyTwoIncident(); !strings.Contains(why, "0-incident") {
		t.Fatalf("an unclaimed rail must fail the certificate, got %q", why)
	}
	led.seg(shared, railForward, blendClaimant(0))
	if why := led.certifyTwoIncident(); !strings.Contains(why, "1-incident") || !strings.Contains(why, "arm/near") {
		t.Fatalf("a once-claimed rail must fail the certificate naming it, got %q", why)
	}
	led.seg(shared, railReversed, blendClaimant(1))
	if why := led.certifyTwoIncident(); why != "" {
		t.Fatalf("a rail claimed by two distinct faces must pass, got %q", why)
	}
	led.seg(shared, railForward, blendClaimant(2))
	if why := led.certifyTwoIncident(); !strings.Contains(why, "3-incident") {
		t.Fatalf("a thrice-claimed rail is non-manifold and must fail, got %q", why)
	}
}

// TestLedgerSelfPairedRailFailsTheCertificate is the regression for the reference-counting hole: ONE face
// claiming the same rail twice used to certify as clean 2-incidence, so a boundary the neighbouring face
// never closed could reach assembleBody. The certificate must count DISTINCT faces and name the self-pair.
func TestLedgerSelfPairedRailFailsTheCertificate(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	shared := led.add("arm/near", zzLedgerSeg(0))
	led.seg(shared, railForward, blendClaimant(0))
	led.seg(shared, railReversed, blendClaimant(0))
	why := led.certifyTwoIncident()
	if !strings.Contains(why, "ONE face") || !strings.Contains(why, "arm/near") {
		t.Fatalf("one face claiming a rail twice must fail the certificate naming it, got %q", why)
	}
}

// TestLedgerReversedClaimIsTheSameCurve pins the identity property the ledger exists for: the second face
// reads the SAME registered curve traversed backwards — never a re-derived congruent one. The endpoints must
// therefore swap exactly, with no re-fitting.
func TestLedgerReversedClaimIsTheSameCurve(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	id := led.add("host/rail", zzLedgerSeg(3))
	fwd, okF := led.seg(id, railForward, blendClaimant(0))
	rev, okR := led.seg(id, railReversed, blendClaimant(1))
	if !okF || !okR {
		t.Fatalf("a registered rail must resolve in both senses, got ok=%v/%v", okF, okR)
	}
	if fwd.from != rev.to || fwd.to != rev.from {
		t.Fatalf("reversed claim endpoints %v→%v do not mirror %v→%v", rev.from, rev.to, fwd.from, fwd.to)
	}
}

// TestLedgerChainClaimsEveryPiece checks a rail CHAIN claims each piece once per read, in order forward and
// end-to-end reversed — the property the two faces of a split boundary rely on.
func TestLedgerChainClaimsEveryPiece(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	ids := []railID{led.add("c0", zzLedgerSeg(0)), led.add("c1", zzLedgerSeg(1))}
	fwd, okF := led.chain(ids, railForward, blendClaimant(0))
	rev, okR := led.chain(ids, railReversed, blendClaimant(1))
	if !okF || !okR || len(fwd) != 2 || len(rev) != 2 {
		t.Fatalf("chain ok=%v/%v lengths %d/%d, want true/true and 2/2", okF, okR, len(fwd), len(rev))
	}
	if rev[0].from != fwd[1].to || rev[1].to != fwd[0].from {
		t.Fatalf("reversed chain %v…%v does not mirror forward %v…%v", rev[0].from, rev[1].to, fwd[0].from, fwd[1].to)
	}
	if why := led.certifyTwoIncident(); why != "" {
		t.Fatalf("two distinct claimants per piece must pass the certificate, got %q", why)
	}
}

// TestLedgerEndsDoesNotClaim pins that reading a chain's feet is NOT a claim — the executor needs the feet
// to build neighbouring geometry, and counting that as a face claim would make the certificate vacuous.
func TestLedgerEndsDoesNotClaim(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	ids := []railID{led.add("c0", zzLedgerSeg(0))}
	if _, _, ok := led.ends(ids); !ok {
		t.Fatal("ends declined a registered chain")
	}
	if why := led.certifyTwoIncident(); !strings.Contains(why, "0-incident") {
		t.Fatalf("ends must not count as a claim, got %q", why)
	}
}

// TestLedgerUnsetHandleDeclines pins the noRail guard. The certificate CANNOT cover an unset handle — it
// iterates the REGISTERED rails, and nothing is registered under noRail — so resolving one must decline
// outright rather than hand back a zero endSeg that would weld a degenerate segment into a face loop.
func TestLedgerUnsetHandleDeclines(t *testing.T) {
	t.Parallel()
	led := newCornerWeldLedger()
	led.add("only", zzLedgerSeg(0))
	if _, ok := led.seg(noRail, railForward, blendClaimant(0)); ok {
		t.Fatal("an unset handle must decline, not resolve")
	}
	if _, ok := led.chain([]railID{noRail}, railForward, blendClaimant(0)); ok {
		t.Fatal("a chain carrying an unset handle must decline")
	}
	if _, ok := led.seg(railID(7), railForward, blendClaimant(0)); ok {
		t.Fatal("an out-of-range handle must decline, not resolve")
	}
}
