// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestPendingListIsPinned keeps the pending list SHRINK-ONLY. Adding an entry must be a deliberate,
// reviewed act — an accidental growth (a regression quietly parked here) fails this gate. The count
// is the one measured at 09a9b2d1: 99 FAIL(faulty) + 5 FAIL(area).
func TestPendingListIsPinned(t *testing.T) {
	t.Parallel()
	if got := len(pendingCapability); got != pendingCapabilityCount {
		t.Fatalf("pendingCapability has %d entries, want %d — a case was added or removed without "+
			"updating pendingCapabilityCount; the list may only shrink by building the capability",
			got, pendingCapabilityCount)
	}
}

// TestPendingEntriesNameRealCases fails on a typo or a stale entry pointing at a case the corpus no
// longer holds. Without it a misspelled key would silently stop matching, and the case it was meant
// to cover would fail the parity gate for reasons nobody could trace back to this list.
func TestPendingEntriesNameRealCases(t *testing.T) {
	t.Parallel()
	inCorpus := map[quarantineKey]bool{}
	for _, r := range Corpus() {
		inCorpus[quarantineKey{grid: r.Grid, name: r.Case}] = true
	}
	for k := range pendingCapability {
		if !inCorpus[k] {
			t.Errorf("pendingCapability names %s/%s, which is not in the corpus", k.grid, k.name)
		}
	}
}

// TestPendingEntriesCarryAReason pins the list's purpose: it is a specification of remaining work, so
// every entry must say what is missing. A blank reason would make it indistinguishable from silencing.
func TestPendingEntriesCarryAReason(t *testing.T) {
	t.Parallel()
	for k, why := range pendingCapability {
		if len(why) < 20 {
			t.Errorf("pendingCapability[%s/%s] reason is too thin to be actionable: %q", k.grid, k.name, why)
		}
	}
}

// TestPendingAndQuarantineAreDisjoint keeps the two mechanisms from blurring. Quarantine means "a
// passing area would be COINCIDENTAL, masking broken geometry"; pending means "the capability is not
// built yet". A case in both would be scored twice and reasoned about wrongly.
func TestPendingAndQuarantineAreDisjoint(t *testing.T) {
	t.Parallel()
	for k := range pendingCapability {
		if _, held := quarantined[k]; held {
			t.Errorf("%s/%s is both pending and quarantined — pick one mechanism", k.grid, k.name)
		}
	}
}

// TestPendingDoesNotTouchTheScoreboard is the anti-laundering gate. The pending list exists ONLY to
// stop the per-case parity gate blocking the pipeline; it must never flatter the scoreboard. So the
// honest rollup still counts these cases as FAIL(faulty)/FAIL(area) — assertHardenedRollup's 132/148
// is unchanged by the list's existence, and ScoreCase must not consult it.
func TestPendingDoesNotTouchTheScoreboard(t *testing.T) {
	t.Parallel()
	dir := CorpusFixtureDir()
	for _, r := range Corpus() {
		if _, pending := pendingCapabilityReason(r); !pending {
			continue
		}
		if o := ScoreCase(r, dir); o.IsPass() {
			t.Fatalf("%s/%s is on the pending list yet the scoreboard scores it %s — the pending list "+
				"must never be readable as green", r.Grid, r.Case, o)
		}
	}
}
