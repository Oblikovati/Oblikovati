// SPDX-License-Identifier: GPL-2.0-only

package geom

// Why a closed-form section refused a pair (ADR-0061 stage 5, third slice).
//
// "Never degrade silently: a fallback, approximation or dropped element is a diag.Defect that reaches
// the result." A bare ok=false does not meet that. It says "no closed form claimed this pair", and a
// CONDITIONING demotion is a different thing entirely: the closed form applies, but at this pair's
// numbers it cannot name its own answer, so the pipeline walks the section instead of solving it. The
// two look identical to a caller, and the second is the one a user needs told about — it is where the
// exact pipeline gives up ground it normally holds.
//
// geom owns no I/O and takes no recorder, so it returns the REASON and its caller records it. The
// brep boolean turns a non-none reason into a diag.Defect on the operation's recorder
// (CodeSectionConditioningDemotion), the same way it already turns an unresolved tangent contact into
// one.

// SectionDecline names why [IntersectSurfacesAnalyticDeclining] refused a pair. DeclineNone means it
// did not refuse.
type SectionDecline uint8

const (
	// DeclineNone is "no refusal": the pair was solved, possibly to an empty section.
	DeclineNone SectionDecline = iota
	// DeclineNoClosedForm is "no bucket claims this pair" — the ordinary, expected refusal that sends
	// the pair to the general marcher. It is not a degradation: nothing was given up.
	DeclineNoClosedForm
	// DeclineTorusLaneStation is a tube angle whose station polynomial has no azimuth dependence, so
	// there is no branch pair to read there.
	DeclineTorusLaneStation
	// DeclineTorusLaneTracks is extremum tracks that cross, merge or change in number over the turn,
	// which makes "which branch pair" a guess.
	DeclineTorusLaneTracks
	// DeclineTorusLaneFullTurn is a branch pair that exists at EVERY tube angle: four independent
	// full-period branches rather than a folded pair, a topology this reduction does not carry.
	DeclineTorusLaneFullTurn
	// DeclineTorusLaneUnaccounted is a loop set that does not account for every azimuth the stations
	// carry — the certificate that no section loop was dropped.
	DeclineTorusLaneUnaccounted
	// DeclineTorusLaneSeparation is a window whose two branches never separate by more than the stitch
	// resolution, so the loop is a sliver two faces could not be told apart across.
	DeclineTorusLaneSeparation
)

// String names the decline for a diagnostic message.
func (d SectionDecline) String() string {
	if int(d) >= len(sectionDeclineNames) {
		return "SectionDecline(?)"
	}
	return sectionDeclineNames[d]
}

var sectionDeclineNames = [...]string{
	DeclineNone:                 "none",
	DeclineNoClosedForm:         "no closed form claims this surface pair",
	DeclineTorusLaneStation:     "a torus station with no azimuth dependence",
	DeclineTorusLaneTracks:      "the torus section's extremum tracks are not separable",
	DeclineTorusLaneFullTurn:    "the torus section's branch pair never folds",
	DeclineTorusLaneUnaccounted: "the torus section's loops do not account for every azimuth",
	DeclineTorusLaneSeparation:  "the torus section's branches never separate past the stitch resolution",
}

// IsConditioning reports whether the refusal is a CONDITIONING demotion — a closed form that applies to
// this pair but cannot name its answer at these numbers — rather than the ordinary "no bucket claims
// this pair". Only the first is a degradation worth reporting.
func (d SectionDecline) IsConditioning() bool {
	return d != DeclineNone && d != DeclineNoClosedForm
}
