// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The boolean's SIZE classification: the one thing about a pair that can be decided before any
// geometry is built, and the one the pipeline had no answer for (ADR-0061 stage 6).
//
// WHAT THE SWEEP FOUND (boolean_drill_sweep_test.go, and the table in ADR-0061). It drives an axial
// drill through the RING corpus body over twelve decades of tool thickness, five points per decade,
// classifying each outcome against an independent quadrature oracle. Three regimes, not one:
//
//	thickness / Weld    behaviour
//	<= 0.0998           the ring comes back UNTOUCHED: err=nil, one face, removed=0, and NOTHING
//	                    recorded. The silent exit.
//	0.158 .. 63 000     refused loudly and by name (no-exact-curved-path)
//	63 000 .. 6.3e6     refused by the Requicha bracket (analytic-volume-reject): a VALID body of
//	                    materially wrong volume, which the bracket alone keeps from shipping
//	>= 1e7              exact: torus + cylinder, 2 loops each, volume within 3.6e-6 of the oracle
//
// Only the FIRST regime is a resolution limit. The second is a capability gap in the torus-cylinder
// section at small radius — real, already refused by name, and pinned by its own corpus row
// (TestASmallBoreIsRefusedNotShippedWrong); calling it "sub-resolution" would relabel a bug as a
// policy and hide it. So the floor is the top of the silent band, rounded up to the model's own
// coincidence scale: geom.Resolution.Weld, which sits ~6x above the highest silent point measured.
//
// A solid operand thinner than that has two opposite boundary faces closer together than the
// distance at which this model calls two points the same point: it has no interior left to build
// with. The ground rule is that an unsupported configuration is refused AT CLASSIFICATION with a
// named decline, before any geometry is built. This is that classification.

// ErrSubResolutionOperand is the boolean's refusal for an operand whose material is thinner than the
// model's coincidence resolution. It is a REFUSAL, not a degradation: no geometry is attempted, so
// there is no partial body to report, and the caller (a feature) quarantines instead of storing a
// result that either removed nothing or failed its own acceptance gate.
var ErrSubResolutionOperand = errors.New("boolean: an operand is thinner than the model's resolution")

// CodeBooleanSubResolutionTool names the refusal on the diagnostic channel so it reaches feature
// health, the API and the UI rather than living only in an error string. The remedy is a modelling
// one — author at a working unit closer to the feature, per geom.SpanCeilingWarning — so the user
// has to be able to SEE which operand was too thin and by how much.
const CodeBooleanSubResolutionTool diag.Code = "boolean.sub-resolution-tool"

// operandSize is one operand's measured material thickness. measured is false where there is nothing
// to measure — a nil, empty or non-solid body — and a thickness that was never measured can never
// refuse anything.
type operandSize struct {
	thickness float64
	measured  bool
}

// operandSizes is the boolean's SIZE classification: the resolution of the extent the pair spans, and
// each operand's material thickness measured against it.
//
// It is decided ONCE per operation, by classifyOperandSize at a public entry, and CARRIED to the
// stages that need it rather than recomputed there. The predicate used to run twice for every curved
// pair — once in BooleanWithDiagnostics and again inside curvedExactGuarded — which is two chances to
// answer differently about one pair, the thing the ground rule on deciding an incidence once forbids
// (#3524).
type operandSizes struct {
	res          Resolution
	target, tool operandSize
}

// classifyOperandSize is the boolean's size classification: it measures both operands against the
// pair's resolution and returns the named refusal when either operand's material is below it. It runs
// before classify() and therefore before any intersection, imprint or stitch. The floor is
// geom.Resolution.Resolves — the SAME predicate the UI's feature-scale warning reads, so the two
// cannot disagree (finding 3 of the stage-6 review).
//
// Example:
//
//	sizes, err := classifyOperandSize(op, target, tool, rec)
//	if err != nil { return nil, err }
func classifyOperandSize(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (operandSizes, error) {
	res := pairExtentResolution(target, tool)
	sizes := operandSizes{res: res, target: measureOperand(target, res), tool: measureOperand(tool, res)}
	if err := subResolutionRefusal(op, "target", sizes.target, res, rec); err != nil {
		return sizes, err
	}
	return sizes, subResolutionRefusal(op, "tool", sizes.tool, res, rec)
}

// measureOperand measures one operand's material thickness against the pair's resolution. Only a
// SOLID is measured: a sheet body's zero thickness is its representation, not material below
// resolution, so the test does not apply to it.
func measureOperand(b *topo.Body, res Resolution) operandSize {
	thickness, ok := solidThickness(b, res.Weld())
	return operandSize{thickness: thickness, measured: ok}
}

// subResolutionRefusal refuses one measured operand by name, recording the Defect that carries the
// refusal to feature health.
func subResolutionRefusal(op PartFeatureOperation, role string, size operandSize, res Resolution, rec *diag.Recorder) error {
	if !size.measured || res.Resolves(size.thickness) {
		return nil
	}
	detail := fmt.Sprintf("%s %s is %g thick, below this model's resolution %g: %s",
		op, role, size.thickness, res.Weld(), geom.ScaleRemedy)
	rec.Recordf(CodeBooleanSubResolutionTool, diag.Defect, "%s", detail)
	return fmt.Errorf("%w: %s", ErrSubResolutionOperand, detail)
}

// pairExtentResolution is the resolution of the EXTENT the two operands span together — the union of
// their range boxes, not ResolutionForBodies' largest single operand.
//
// The difference matters only here, and it is the difference between agreeing with the UI and not.
// ResolutionForBodies answers "what weld suits a multi-body op", for which the bigger operand is the
// right scale. Resolvability asks a different question — "can a model of this EXTENT hold a feature
// this small" — and the extent the user sees is the part's whole range box, which is what
// PartComponentDefinition.FeatureScaleWarning measures. Feeding the one predicate two different
// extents put the boundary in two places: measured on the RING pair, a 2e-8-thick drill sits above
// the largest-operand floor (1.86e-8) and below the union-box floor (2.005e-8), so the UI warned
// while the boolean built. TestTheBooleanFloorAgreesWithTheFeatureScaleWarning holds them together.
func pairExtentResolution(target, tool *topo.Body) Resolution {
	box := target.RangeBox().Union(tool.RangeBox())
	return geom.ResolutionForBox(box)
}
