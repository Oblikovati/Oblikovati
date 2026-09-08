// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
)

// The boolean's SIZE classification: the one thing about a pair that can be decided before any
// geometry is built, and the one the pipeline had no answer for (ADR-0061 stage 6).
//
// A solid operand thinner than the model's seam weld has two opposite boundary faces closer
// together than the tolerance at which the boolean merges seam points. Its own material is below
// the resolution the result is built at, so the seam cannot separate the two sides — measured on
// the RING corpus row: an axial drill of radius 1e-6 through the ring left a body the acceptance
// gate refused (valid=false, closed=false) after the whole intersect→imprint→classify→stitch run,
// and at radius 1e-10 the pipeline returned the ring UNCHANGED, err=nil, with nothing recorded —
// a Cut that removed nothing, silently. Both are the same configuration and neither was named.
//
// The ground rule is that an unsupported configuration is refused AT CLASSIFICATION with a named
// decline, before any geometry is built. This is that classification.

// ErrSubResolutionOperand is the boolean's refusal for an operand whose material is thinner than the
// model's seam-stitch resolution. It is a REFUSAL, not a degradation: no geometry is attempted, so
// there is no partial body to report, and the caller (a feature) quarantines instead of storing a
// result that either removed nothing or failed its own acceptance gate.
var ErrSubResolutionOperand = errors.New("boolean: an operand is thinner than the model's seam resolution")

// CodeBooleanSubResolutionTool names the refusal on the diagnostic channel so it reaches feature
// health, the API and the UI rather than living only in an error string. The remedy is a modelling
// one — author at a working unit closer to the feature, per geom.SpanCeilingWarning — so the user
// has to be able to SEE which operand was too thin and by how much.
const CodeBooleanSubResolutionTool diag.Code = "boolean.sub-resolution-tool"

// declineSubResolutionOperand is the boolean's size classification: it returns the named refusal when
// either operand's material is below the seam-stitch resolution of the pair, and nil otherwise. It
// runs before classify() and therefore before any intersection, imprint or stitch.
func declineSubResolutionOperand(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) error {
	floor := ResolutionForBodies(target, tool).Stitch()
	if err := subResolutionRefusal(op, "target", target, floor, rec); err != nil {
		return err
	}
	return subResolutionRefusal(op, "tool", tool, floor, rec)
}

// subResolutionRefusal refuses one operand by name, recording the Defect that carries the refusal to
// feature health. Only a SOLID is measured: a sheet body's zero thickness is its representation, not
// material below resolution, so the test does not apply to it.
func subResolutionRefusal(op PartFeatureOperation, role string, b *topo.Body, floor float64, rec *diag.Recorder) error {
	thickness, ok := solidThickness(b)
	if !ok || thickness >= floor {
		return nil
	}
	detail := fmt.Sprintf("%s %s is %g thick, below this model's seam resolution %g: %s",
		op, role, thickness, floor, subResolutionRemedy)
	rec.Recordf(CodeBooleanSubResolutionTool, diag.Defect, "%s", detail)
	return fmt.Errorf("%w: %s", ErrSubResolutionOperand, detail)
}

// subResolutionRemedy is the modelling advice the refusal carries, matching geom.SpanCeilingWarning's:
// the fault is the RATIO between the feature and the model, so shrinking the tolerance would not help.
const subResolutionRemedy = "model at a working unit closer to the feature scale, or split the design across documents"

// solidThickness is a solid body's smallest bounding-box extent — how thin its material gets in the
// direction it is thinnest. ok is false for a nil, non-solid or empty body: a sheet has no thickness
// to measure and an empty body no material.
func solidThickness(b *topo.Body) (float64, bool) {
	if b == nil || !b.IsSolid() {
		return 0, false
	}
	box := b.RangeBox()
	if box.IsEmpty() {
		return 0, false
	}
	d := box.Diagonal()
	return float64(min(min(d.X, d.Y), d.Z)), true
}
