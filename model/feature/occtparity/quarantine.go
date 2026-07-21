// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// Quarantine is the corpus-level hold list: cases that MUST NOT count as green regardless of their
// area, because a passing (or near-passing) area is known to be COINCIDENTAL — masking a still-broken
// geometry — rather than genuine parity. It is a keyed code override (not a corpus.json field) so it
// survives corpus regeneration and carries a forensic reason at the call site. RunCase, ScoreCase and
// classify all consult it, so a quarantined case reports SkipQuarantine (a distinct SKIP, never Pass)
// and the scoreboard rollup never tallies it as green.
//
// A case leaves quarantine only when its underlying geometry defect is fixed and the entry is deleted
// — never by widening a tolerance.

// quarantineKey identifies a case across grids (grid+case), the same pairing the scoreboard uses.
type quarantineKey struct {
	grid string
	name string
}

// quarantined maps each held case to the VERBATIM reason it is held. H6: ROOT 1 (the cone-sector
// tessellation over-area) is fixed, but H6's whole-body area only looks acceptable because a SECOND,
// independent defect (ROOT 2 — resolveArcFillet builds the concave open-torus fillet on the wrong
// side) offsets it; the area gate must not read that cancellation as parity.
var quarantined = map[quarantineKey]string{
	{grid: "simple", name: "H6"}: "ROOT 1 cone-sector tessellation fixed, but the underlying concave " +
		"open-torus fillet geometry remains inverted (resolveArcFillet builds R−r not R+r, +9.73% before " +
		"the tessellation masked it). H6 passes the area gate at −0.80% COINCIDENTALLY. Do NOT un-gate " +
		"until resolveArcFillet is rebuilt for concave open-torus rims.",
}

// quarantineReason returns the hold reason for a case and whether it is quarantined.
//
// Example:
//
//	if reason, held := quarantineReason(r); held { t.Skipf("quarantined: %s", reason) }
func quarantineReason(r Record) (string, bool) {
	reason, held := quarantined[quarantineKey{grid: r.Grid, name: r.Case}]
	return reason, held
}
