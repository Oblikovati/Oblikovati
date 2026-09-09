// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// Where a closed-form section's CONDITIONING demotion becomes a reported defect (ADR-0061 stage 5,
// third slice).
//
// The intersector refuses two very different things with the same ok=false. Most refusals are "no
// bucket claims this pair" — a torus against a torus, a B-spline against anything — and they are not a
// degradation: nothing was given up, the marcher was always going to take them. A CONDITIONING demotion
// is the other kind: the closed form applies to this pair and cannot name its own answer at these
// numbers, so the exact pipeline gives up ground it normally holds. That is a fallback, and the ground
// rules say a fallback is a diag.Defect that reaches feature health, the API and the UI — not an ADR
// paragraph. geom owns no I/O, so it returns the reason ([geom.SectionDecline]) and this records it.

// CodeSectionConditioningDemotion marks a surface pair whose closed-form section APPLIED but could not
// be named at this pair's numbers, so the operation fell to the general marcher (or declined). A
// tracked degradation, not an error: unlike "no closed form claims this pair", it says the exact path
// was available and was given up, and it names which certificate refused.
const CodeSectionConditioningDemotion diag.Code = "section.conditioning-demotion"

// recordSectionDecline reports a conditioning demotion on the recorder, naming the reason and the two
// surfaces it refused. An ordinary "no closed form" refusal records NOTHING: it happens on every
// marched boolean in the system, and a diagnostic that fires on the normal case is noise.
func recordSectionDecline(rec *diag.Recorder, why geom.SectionDecline, a, b curvedFace) {
	if !why.IsConditioning() {
		return
	}
	rec.Recordf(CodeSectionConditioningDemotion, diag.Defect,
		"the %T ∩ %T closed-form section applied but could not be named: %s; falling back to the general marcher",
		a.surface, b.surface, why)
}
