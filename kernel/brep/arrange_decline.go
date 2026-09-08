// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/diag"
)

// Where a planar subdivision that will not converge becomes a NAMED refusal instead of a hang
// (ADR-0061 stage 6, review round 2).
//
// The T-junction pass subdivides "until stable", and on geometry whose features sit at the same
// scale as its absolute 1e-7 tolerance it never becomes stable — measured on the RING corpus body
// cut by an axial drill of radius 1.585e-7, which did not return in any budget the suite could give
// it. That is the one outcome the ground rules do not admit: an operation may refuse and it may
// return a valid answer, but it may not fail to answer. splitTJunctions now stops at a provable
// budget (tjSplitBudget) and says so; this is what "says so" means downstream.

// ErrUnconvergedArrangement is the refusal: the planar subdivision hit its split budget, so the cell
// complex is untrustworthy and every face built from it would be a guess. The boolean turns it into
// its own named refusal rather than shipping a body traced from an unstable arrangement.
var ErrUnconvergedArrangement = errors.New("brep: the planar subdivision did not converge")

// CodeArrangementUnconverged marks that refusal on the diagnostic channel so it reaches feature
// health, the API and the UI. A tracked defect, not an expected outcome: an arrangement that will
// not converge is a conditioning failure of the T-junction tolerance against the input's own scale,
// and the fix is in the pass, not in the caller.
const CodeArrangementUnconverged diag.Code = "arrangement.unconverged"

// unconvergedArrangement builds the named refusal, carrying the segment count so a report says how
// big the arrangement that failed was.
func unconvergedArrangement(segs int) error {
	return fmt.Errorf("%w: %d segments exceeded the T-junction split budget; the cells cannot be trusted",
		ErrUnconvergedArrangement, segs)
}

// recordArrangementDecline reports the refusal on the recorder, naming WHICH of the splits declined —
// there are four (the closed-surface trim, the ruled-wall trim, the uv-plane trim and the planar face
// split), they fail for the same reason and a report that did not distinguish them would send the
// reader to the wrong one. Every site calls this; TestEveryArrangingSplitReportsANonConvergentArrangement
// is the guard that keeps it so as new splits are added.
func recordArrangementDecline(rec *diag.Recorder, site string, err error) {
	if !errors.Is(err, ErrUnconvergedArrangement) {
		return
	}
	rec.Recordf(CodeArrangementUnconverged, diag.Defect,
		"the %s subdivision did not converge (%s): refusing the boolean rather than building faces "+
			"from an unstable cell complex", site, err)
}

// The four splits that arrange a face and must report a non-converging one. Named here so the
// diagnostic's wording is one list rather than four string literals scattered across the package.
const (
	siteClosedSurfaceTrim = "closed-surface (u,v)"
	siteWallTrim          = "ruled-wall (u,v)"
	siteUVPlaneTrim       = "uv-plane"
	sitePlanarSplit       = "planar face"
)
