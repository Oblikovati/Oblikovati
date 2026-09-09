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
// DECLINES the exact boolean rather than merely routing the pair to the marcher. Info, not Defect,
// because nothing degrades AT THIS GATE: the boolean hard-errors on the refusal (ADR-0061 retired the
// CSG fallback, so a decline is a refusal and not a demotion) and the degradation is already reported
// as a Defect by the caller, ops.CodeBooleanNoExactCurvedPath. This is that Defect's EXPLANATION — the
// two faces and the gate the generic message cannot name (#3525).
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

// recordSectionDecline reports on the recorder that a face pair's section refused, naming the two
// FACES — surface kind and lineage — which gate refused and what it measured. A CONDITIONING demotion
// is a Defect: the exact path applied and was given up. Anything else is an Info that explains the
// caller's own Defect. DeclineNone records nothing: there was no refusal to report.
//
// It takes faces, not surfaces: "failure is local — an operation returns a partial result naming the
// faulty entity", and on a 40-face part "geom.Torus ∩ geom.Torus" names no entity a user can find.
func recordSectionDecline(rec *diag.Recorder, r sectionRefusal, a, b curvedFace) {
	if r.why == geom.DeclineNone {
		return
	}
	first, second := orderedFaceLabels(a, b)
	if r.why.IsConditioning() {
		recordSectionDeclineOnce(rec, CodeSectionConditioningDemotion, diag.Defect,
			"the %s ∩ %s closed-form section applied but its answer is unusable: %s; the exact path is declined here",
			first, second, r)
		return
	}
	recordSectionDeclineOnce(rec, CodeSectionUnclaimedPair, diag.Info,
		"the %s ∩ %s section was refused before any geometry was built: %s; the exact path is declined here",
		first, second, r)
}

// orderedFaceLabels names the two faces in one explicit total order (lexicographic), so the SAME pair
// reads the same however the pairing reached it. A refusal is a property of the unordered pair, and
// each pairing runs in both operand orders: without the order the same refusal reached a user twice,
// once as "A ∩ B" and once as "B ∩ A", which no dedupe on the message could collapse. It also makes
// the message byte-identical across runs, which the ground rules require of every output.
func orderedFaceLabels(a, b curvedFace) (first, second string) {
	la, lb := faceLabel(a), faceLabel(b)
	if la <= lb {
		return la, lb
	}
	return lb, la
}

// faceLabel names one operand of a refusal: its surface kind and the lineage that identifies the face
// it came from. A face built without a lineage (a bare primitive surface in a unit test) reads as its
// kind alone rather than as an empty key.
func faceLabel(f curvedFace) string {
	if key := f.lineage.KeyString(); key != "" {
		return fmt.Sprintf("%T %s", f.surface, key)
	}
	return fmt.Sprintf("%T", f.surface)
}

// recordSectionDeclineOnce records the diagnostic unless the recorder already carries that exact
// (code, detail) pair.
//
// One boolean asks the SAME pair twice by construction — the pairings run in both operand orders, and
// ops.booleanGeneralExact enters brep.BooleanDiag twice — so a single refusal reached a user four
// times over. Four identical lines in a report whose whole subject is what a user reads is a defect of
// its own (#3525, review round 1). Repeats are dropped rather than counted: the message says which
// gate refused which pair, and saying it twice adds nothing to that.
func recordSectionDeclineOnce(rec *diag.Recorder, code diag.Code, sev diag.Severity, format string, args ...any) {
	detail := fmt.Sprintf(format, args...)
	for _, d := range rec.Records() {
		if d.Code == code && d.Detail == detail {
			return
		}
	}
	rec.Record(diag.Diagnostic{Code: code, Severity: sev, Detail: detail})
}
