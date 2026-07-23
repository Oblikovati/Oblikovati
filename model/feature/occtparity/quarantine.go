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
	// U4: the green-gate hardening audit (#2007, .superpowers/sdd/green-gate-validate-audit-report.md)
	// found its result's ops.Validate().HolesContained == false — "hole loop protrudes outside the outer
	// loop of planar face" (a malformed B-rep face). S6/S9/T3 (Group A, one-boss sphere/torus) were freed
	// by the single-boss setback tiling (fillet_setback_*.go: 2 flanks + one central run-out absorbs the
	// footprint with a watertight fill, boss wall intact; #2007) — each now passes the full watertight bar
	// (Valid && Closed && Manifold && HolesContained && IsSolid) at area S6 +0.074% / S9 +0.267% / T3 +0.114%
	// and is pinned in fingerprint_pins_test.go. U3 (Group B, obstacle-path dipsPast mis-detection on an
	// oblique elliptical footprint) was freed by the dipArcOrder fix (fillet_obstacle_detect.go,
	// u3-dipspast-report.md): dipsPast tested the ascending-index arc unconditionally, which — because rim
	// sample 0 (the curve's t=0 seam point) is an arbitrary reference unrelated to the boundary — happened
	// to be the 53-of-64-sample BULGE arc for U3 instead of the true 11-sample dip arc that wraps through
	// index 0; dipArcOrder now picks whichever of the two crossing-bounded arcs is actually shorter (the
	// genuinely local mid-span excursion) before both the dip test and the downstream rebuild consume it.
	// U3 now passes the full watertight bar at area +0.116%, pinned in fingerprint_pins_test.go. U4 was
	// the residual dual-host multi-boss composition (recon Group C); it is now FREED by the U4-5 dual-host
	// multi-rail weld (kernel/ops/fillet_obstacle_dual*.go: 2 notched hosts + 2 split walls + 2 wings + 4
	// corner-blend panels — 2 coons4 slivers + 2 exact-station canal cores), which builds the full welded
	// body that clears the watertight bar (Valid && Closed && Manifold && HolesContained && IsSolid) at
	// whole-body area within 0.001% of the oracle, every result face WIRE:1, and every fillet panel
	// tessellating fold-free to its per-face oracle in production (#2007 Group C). Pinned in
	// fingerprint_pins_test.go; its per-case gates live in kernel/ops/fillet_obstacle_dual_test.go.
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
