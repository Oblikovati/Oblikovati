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
// geom owns no I/O and takes no recorder, so it returns the REASON and its caller records it. The brep
// boolean turns a CONDITIONING reason into a diag.Defect on the operation's recorder
// (CodeSectionConditioningDemotion), the same way it already turns an unresolved tangent contact into
// one, and the ORDINARY reason into an Info (CodeSectionUnclaimedPair) wherever that refusal declines
// the whole boolean rather than merely routing one pair to the marcher — because there the caller's own
// message is the same sentence for every refusal there is (Oblikovati/Oblikovati#3525).

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
	// DeclineTorusTangentStation is a tube angle at which the other surface TOUCHES the tube circle instead
	// of crossing it: the station polynomial vanishes at one of its own extrema, so two azimuths that would
	// bound a branch are one and no pairing of branches can carry the station. It is a statement about
	// the INPUT, and it is the refusal a grazing pair of surfaces gets (Oblikovati/Oblikovati#3515).
	DeclineTorusTangentStation
	// DeclineTorusLaneUnaccounted is a curve set whose azimuths are not the ones the stations certify —
	// too few, too many, or two curves on one branch — at a station with no tangency to explain it. It is
	// a statement about this REDUCTION rather than about the input: the certificate that no section curve
	// was dropped or doubled. A grazing pair declines [DeclineTorusTangentStation] instead, because a
	// user whose surfaces touch must not be told the kernel's loops do not add up.
	DeclineTorusLaneUnaccounted
	// DeclineTorusLaneSeparation is a window whose two branches never separate by more than the stitch
	// resolution, so the loop is a sliver two faces could not be told apart across.
	DeclineTorusLaneSeparation
	// DeclineTorusSectionOffItsForm is a section the reduction built whose own points do not satisfy the
	// station polynomial they were solved from. Every step before it certifies a part — a root, a fold's
	// extremum, a lane's anchor, a branch count — and their composition can still place a curve off the
	// other surface; this is the post-condition that reads where the curve actually is.
	DeclineTorusSectionOffItsForm
	// DeclineOpenSection is a section the closed form SOLVED but that does not come back to where it
	// started. A pairing whose scope is "every crossing is an island on both charts" splits each side by
	// even-odd containment alone, and an open arc opens a region it never closes, so the imprint would
	// leave the two charts disagreeing about which side is material (Oblikovati/Oblikovati#3525).
	DeclineOpenSection
)

// String names the decline for a diagnostic message.
func (d SectionDecline) String() string {
	if int(d) >= len(sectionDeclineNames) {
		return "SectionDecline(?)"
	}
	return sectionDeclineNames[d]
}

var sectionDeclineNames = [...]string{
	DeclineNone:                   "none",
	DeclineNoClosedForm:           "no closed form claims this surface pair",
	DeclineTorusLaneStation:       "a torus station with no azimuth dependence",
	DeclineTorusLaneTracks:        "the torus section's extremum tracks are not separable",
	DeclineTorusTangentStation:    "the other surface touches the torus's tube circle without crossing it",
	DeclineTorusLaneUnaccounted:   "the torus section's curves are not the azimuths the stations certify",
	DeclineTorusLaneSeparation:    "the torus section's branches never separate past the stitch resolution",
	DeclineTorusSectionOffItsForm: "the torus section's own points do not satisfy the form it was solved from",
	DeclineOpenSection:            "the closed-form section does not close on itself",
}

// IsConditioning reports whether the refusal is a CONDITIONING demotion — a closed form that applies to
// this pair but whose answer is unusable at these numbers, whether because it cannot be named
// ([DeclineTorusLaneTracks] and its siblings) or because it does not close ([DeclineOpenSection]) —
// rather than the ordinary "no bucket claims this pair". Only the first is a degradation worth
// reporting as a defect: it is where the exact pipeline gives up ground it normally holds.
func (d SectionDecline) IsConditioning() bool {
	return d != DeclineNone && d != DeclineNoClosedForm
}
