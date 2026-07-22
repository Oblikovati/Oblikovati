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

// quarantined maps each held case to the reason it is held. H6: ROOT 1 (the cone-sector tessellation
// over-area on BOTH congruent host cones) is now fixed, which UNMASKED a second, independent defect —
// ROOT 2, resolveArcFillet builds the concave open-torus fillet on the wrong side (R−r not R+r). H6 is
// held so this red never later reads as parity by coincidence if the geometry shifts.
var quarantined = map[quarantineKey]string{
	{grid: "simple", name: "H6"}: "ROOT 1 cone-sector tessellation fixed (both congruent host cones now " +
		"133286, was ×1.26 inflated); that UNMASKED ROOT 2 — resolveArcFillet builds the concave open-torus " +
		"fillet inverted (R−r not R+r), so H6 now honestly FAILS the area gate at −1.09%. Held to document " +
		"the defect and guard against a future coincidental pass. Do NOT un-gate until resolveArcFillet is " +
		"rebuilt for concave open-torus rims.",
	// S6/S9/T3/U3/U4: the green-gate hardening audit (#2007, .superpowers/sdd/green-gate-validate-audit-
	// report.md) found each result's ops.Validate().HolesContained == false — "hole loop protrudes outside
	// the outer loop of planar face" (a malformed B-rep face the mid-span obstacle rebuild does not yet
	// cover for these dual-host/torus configurations; T6's sibling defect, fillet_hole_containment_test.go).
	// Each area was coincidentally inside the 1% Deps tolerance over the malformed face (S6 +0.275%, S9
	// +0.788%, T3 +0.350%, U3 +0.502%, U4 +0.928%), so the OLD area-only gate counted them PASS. Held so a
	// coincidental area match never again masks a real hole-containment defect. Do NOT un-gate until
	// ops.Validate(result).HolesContained == true for each.
	{grid: "simple", name: "S6"}: "malformed hole-loop (HolesContained=false); area +0.275% coincidentally " +
		"inside 1% Deps; #2007.",
	{grid: "simple", name: "S9"}: "malformed hole-loop (HolesContained=false); area +0.788% coincidentally " +
		"inside 1% Deps; #2007.",
	{grid: "simple", name: "T3"}: "malformed hole-loop (HolesContained=false); area +0.350% coincidentally " +
		"inside 1% Deps; #2007.",
	{grid: "simple", name: "U3"}: "malformed hole-loop (HolesContained=false); area +0.502% coincidentally " +
		"inside 1% Deps; #2007.",
	{grid: "simple", name: "U4"}: "malformed hole-loop (HolesContained=false); area +0.928% coincidentally " +
		"inside 1% Deps; #2007.",
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
