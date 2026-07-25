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
	led := newCornerWeldLedger()
	shared := led.add("arm/near", zzLedgerSeg(0))
	if why := led.certifyTwoIncident(); !strings.Contains(why, "0-incident") {
		t.Fatalf("an unclaimed rail must fail the certificate, got %q", why)
	}
	led.seg(shared, railForward)
	if why := led.certifyTwoIncident(); !strings.Contains(why, "1-incident") || !strings.Contains(why, "arm/near") {
		t.Fatalf("a once-claimed rail must fail the certificate naming it, got %q", why)
	}
	led.seg(shared, railReversed)
	if why := led.certifyTwoIncident(); why != "" {
		t.Fatalf("a twice-claimed rail must pass, got %q", why)
	}
	led.seg(shared, railForward)
	if why := led.certifyTwoIncident(); !strings.Contains(why, "3-incident") {
		t.Fatalf("a thrice-claimed rail is non-manifold and must fail, got %q", why)
	}
}

// TestLedgerReversedClaimIsTheSameCurve pins the identity property the ledger exists for: the second face
// reads the SAME registered curve traversed backwards — never a re-derived congruent one. The endpoints must
// therefore swap exactly, with no re-fitting.
func TestLedgerReversedClaimIsTheSameCurve(t *testing.T) {
	led := newCornerWeldLedger()
	id := led.add("host/rail", zzLedgerSeg(3))
	fwd, rev := led.seg(id, railForward), led.seg(id, railReversed)
	if fwd.from != rev.to || fwd.to != rev.from {
		t.Fatalf("reversed claim endpoints %v→%v do not mirror %v→%v", rev.from, rev.to, fwd.from, fwd.to)
	}
}

// TestLedgerChainClaimsEveryPiece checks a rail CHAIN claims each piece once per read, in order forward and
// end-to-end reversed — the property the two faces of a split boundary rely on.
func TestLedgerChainClaimsEveryPiece(t *testing.T) {
	led := newCornerWeldLedger()
	ids := []railID{led.add("c0", zzLedgerSeg(0)), led.add("c1", zzLedgerSeg(1))}
	fwd := led.chain(ids, railForward)
	rev := led.chain(ids, railReversed)
	if len(fwd) != 2 || len(rev) != 2 {
		t.Fatalf("chain lengths %d/%d, want 2/2", len(fwd), len(rev))
	}
	if rev[0].from != fwd[1].to || rev[1].to != fwd[0].from {
		t.Fatalf("reversed chain %v…%v does not mirror forward %v…%v", rev[0].from, rev[1].to, fwd[0].from, fwd[1].to)
	}
	if why := led.certifyTwoIncident(); why != "" {
		t.Fatalf("two claims per piece must pass the certificate, got %q", why)
	}
}

// TestLedgerEndsDoesNotClaim pins that reading a chain's feet is NOT a claim — the executor needs the feet
// to build neighbouring geometry, and counting that as a face claim would make the certificate vacuous.
func TestLedgerEndsDoesNotClaim(t *testing.T) {
	led := newCornerWeldLedger()
	ids := []railID{led.add("c0", zzLedgerSeg(0))}
	if _, _, ok := led.ends(ids); !ok {
		t.Fatal("ends declined a registered chain")
	}
	if why := led.certifyTwoIncident(); !strings.Contains(why, "0-incident") {
		t.Fatalf("ends must not count as a claim, got %q", why)
	}
}

// TestLedgerUnsetHandleFailsTheCertificate pins the noRail guard: an unset handle resolves to the zero seg
// rather than panicking, and the certificate still reports the unpaired rail.
func TestLedgerUnsetHandleFailsTheCertificate(t *testing.T) {
	led := newCornerWeldLedger()
	led.add("only", zzLedgerSeg(0))
	if s := led.seg(noRail, railForward); s.curve != nil || s.from != (math.Point3{}) {
		t.Fatalf("an unset handle must resolve to the zero seg, got %+v", s)
	}
	if why := led.certifyTwoIncident(); why == "" {
		t.Fatal("the registered-but-unclaimed rail must still fail the certificate")
	}
}
