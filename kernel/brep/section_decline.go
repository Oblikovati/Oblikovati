// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// Where a closed-form section's refusal becomes a reported diagnostic (ADR-0061 stage 5, third slice;
// Oblikovati/Oblikovati#3525).
//
// The intersector refuses two very different things with the same ok=false. Most refusals are "no
// bucket claims this pair" — a torus against a torus, a B-spline against anything — and they are not a
// degradation: nothing was given up, the marcher was always going to take them. A CONDITIONING demotion
// is the other kind: the closed form applies to the pair and its answer is unusable at these numbers,
// so the exact pipeline gives up ground it normally holds. That is a fallback, and the ground rules say
// a fallback is a diag.Defect that reaches feature health, the API and the UI — not an ADR paragraph.
// geom owns no I/O, so it returns the reason ([geom.SectionDecline]) and this records it.
//
// BOTH kinds are recorded, at different severities. The pairings here do not fall back per pair: an
// ok=false DECLINES the whole mixed boolean, and the caller then reports one generic
// "no exact analytic path claims this configuration". That message alone cannot tell a torus pair from
// an ill-conditioned lane from a section that does not close, which is the silence #3525 names — so the
// ordinary refusal rides along as an Info that says WHICH gate refused, while the degradation stays the
// Defect it always was.

// CodeSectionConditioningDemotion marks a surface pair whose closed-form section APPLIED but whose
// answer could not be used at this pair's numbers, so the operation fell to the general marcher (or
// declined). A tracked degradation, not an error: unlike "no closed form claims this pair", it says the
// exact path was available and was given up, and it names which certificate refused.
const CodeSectionConditioningDemotion diag.Code = "section.conditioning-demotion"

// CodeSectionUnclaimedPair marks a surface pair no closed form claims, recorded where that refusal
// DECLINES the exact boolean rather than merely routing the pair to the marcher. Info, not Defect:
// nothing degraded here — the caller's own decline is the degradation — but without it the user cannot
// tell which of the pairings refused, or on which two surfaces (#3525).
const CodeSectionUnclaimedPair diag.Code = "section.unclaimed-pair"

// sectionRefusal is one named refusal plus the evidence behind it: the [geom.SectionDecline] that says
// WHICH gate refused, and, for a gate that measured something, the offending value in the shape the
// ground rules require of an exception message. detail is empty for a gate with nothing to measure.
type sectionRefusal struct {
	why    geom.SectionDecline
	detail string
}

// refusal names a gate that has no measurement to report.
func refusal(why geom.SectionDecline) sectionRefusal {
	return sectionRefusal{why: why}
}

// refusalf names a gate together with the offending value it measured.
//
//	return nil, refusalf(geom.DeclineOpenSection, "endpoint gap %g > sew %g", gap, res.Sew()), false
func refusalf(why geom.SectionDecline, format string, args ...any) sectionRefusal {
	return sectionRefusal{why: why, detail: fmt.Sprintf(format, args...)}
}

// solved is the "no refusal" refusal: the pair was answered, possibly to an empty section.
func solved() sectionRefusal { return sectionRefusal{why: geom.DeclineNone} }

// String renders the refusal for a diagnostic: the gate's own name, plus the value it measured.
func (r sectionRefusal) String() string {
	if r.detail == "" {
		return r.why.String()
	}
	return r.why.String() + " (" + r.detail + ")"
}

// recordSectionDecline reports on the recorder that a surface pair's section refused, naming the two
// surfaces, which gate refused and what it measured. A CONDITIONING demotion is a Defect — the exact
// path applied and was given up; anything else is an Info that explains the caller's own decline.
// DeclineNone records nothing: there was no refusal to report.
func recordSectionDecline(rec *diag.Recorder, r sectionRefusal, a, b geom.Surface) {
	if r.why == geom.DeclineNone {
		return
	}
	if r.why.IsConditioning() {
		rec.Recordf(CodeSectionConditioningDemotion, diag.Defect,
			"the %T ∩ %T closed-form section applied but its answer is unusable: %s; the exact path is declined here", a, b, r)
		return
	}
	rec.Recordf(CodeSectionUnclaimedPair, diag.Info,
		"the %T ∩ %T section was refused before any geometry was built: %s; the exact path is declined here", a, b, r)
}
