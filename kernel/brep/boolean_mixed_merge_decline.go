// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
)

// Why a pair that IS one face did not become one (ADR-0061 stage 5).
//
// Two faces on one surface that share a boundary are one face. Every exit from the merge that leaves
// them as two is therefore a body with an edge the model does not have, and the ground rules do not
// let that ship silently: "a fallback, approximation, or dropped element is a diag.Defect that reaches
// the result". The one exit that is NOT a degradation is the ordinary case — two faces that share no
// boundary are simply two faces — and it is the only reason here that records nothing.

// CodeCocylindricalMergeUndecided marks two faces on ONE surface whose shared boundary the merge could
// not dissolve exactly. The pair ships as two faces where one belongs, and this says which of the
// reasons refused rather than leaving the extra edge to be found downstream.
const CodeCocylindricalMergeUndecided diag.Code = "boolean.cocylindrical-merge-undecided"

// mergeDecline is the reason a merge exit gives. The zero value is a completed merge.
type mergeDecline string

const (
	// mergeJoined is the success value: the two faces became one.
	mergeJoined mergeDecline = ""
	// declineUnshared is the ORDINARY exit, and the only silent one: the two lie on one surface but
	// share no boundary, so they are two faces and nothing was given up.
	declineUnshared mergeDecline = "they share no boundary"
	// declineAmbiguousPairing: an edge of one runs with two edges of the other, so their common
	// boundary is subdivided differently on the two sides and re-chaining it would have to choose.
	declineAmbiguousPairing mergeDecline = "an edge of one runs with two of the other, so their common " +
		"boundary is subdivided differently on the two sides"
	// declineOpenWalk: the fused boundary did not close, so the dissolved set was not a boundary.
	declineOpenWalk mergeDecline = "the fused boundary did not close: the dissolved edges are not a " +
		"boundary between the two faces"
	// declineUndecidedChart: ADR-0063 refuses to guess which side of the fused rings the face is.
	declineUndecidedChart mergeDecline = "the fused loops do not determine a trim in (u,v)"
	// declineMixedComplement: one is a closed-surface complement (all its loops are holes) and the
	// other is not, so which loop of the merged face is its outer one is undetermined.
	declineMixedComplement mergeDecline = "one is a closed-surface complement and the other is not, so " +
		"the merged face's outer loop is undetermined"
)

// reportable is true for every decline that leaves a face pair the model should not have — everything
// but the ordinary "they share no boundary".
func (d mergeDecline) reportable() bool { return d != mergeJoined && d != declineUnshared }

// recordMergeDecline reports a pair that shares a boundary and could not be made one face.
func recordMergeDecline(rec *diag.Recorder, a curvedFace, why mergeDecline) {
	if !why.reportable() {
		return
	}
	rec.Recordf(CodeCocylindricalMergeUndecided, diag.Defect,
		"two %T faces on one surface share a boundary that bounds nothing, and the body keeps both: %s",
		a.surface, string(why))
}
