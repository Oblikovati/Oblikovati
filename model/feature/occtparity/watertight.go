// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// isWatertightSolid is the ONE shared "is this a genuine watertight solid" predicate for both the
// scoreboard (ScoreCase) and the gating test (assertCaseResult) — matching the bar the codebase's own
// watertight tests already assert (fillet_b2_winding_test.go, fillet_hole_containment_test.go,
// d5_e4_watertight_test.go's assertWatertight): Valid && Closed && Manifold && HolesContained &&
// IsSolid && Volume>0. HolesContained is checked EXPLICITLY — ops.Validate deliberately does NOT fold
// it into Valid yet (kernel/ops/validate.go:60-62), so a malformed protruding-hole face passes Valid
// but poisons the tessellator. That gap let a hole-loop defect slip past the old area-only gate as a
// false green (S6/S9/T3/U3/U4 — coincidentally-in-tolerance area over a malformed face; #2007,
// quarantine.go). props/ok are the caller's ALREADY-tessellated caseProperties() result, so this reads
// the SAME tessellation pass rather than re-tessellating (runcase.go's caseProperties doc: "a wasteful
// ~2×" — the very perf regression this must not reintroduce).
//
// Example:
//
//	props, ok := caseProperties(res, filletOK)
//	valid := isWatertightSolid(res, filletOK, props, ok)
func isWatertightSolid(res []*topo.Body, filletOK bool, props ops.GeometryProperties, ok bool) bool {
	if !filletOK || !ok || len(res) != 1 || res[0] == nil || props.Volume <= 0 {
		return false
	}
	rep := ops.Validate(res[0])
	return rep.Valid && rep.Closed && rep.Manifold && rep.HolesContained && res[0].IsSolid()
}
