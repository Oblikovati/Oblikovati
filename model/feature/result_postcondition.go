// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The feature engine's POST-CONDITION (ADR-0061 stage 6).
//
// "Validate is a post-condition of every public kernel operation. An invalid body is an error, never
// a return value." The kernel holds that line at its own exits — but the engine that stores what the
// kernel returns held no line at all: a body handed back by a feature became fs.result, the browser's
// healthy green tick and the viewport's mesh with nothing checked. Measured with a fake feature
// returning a solid whose single face leaves four boundary edges: health = ok, diagnostics = none,
// one body in the result.
//
// One site, not one per feature. It runs in evaluateBody, between the recompute and the health
// classification, so every feature in the history — the 200-odd kinds and any added later — inherits
// it without opting in.

// ErrInvalidFeatureResult is the post-condition's refusal: the feature rebuilt without error but the
// body it produced is not a valid B-rep. It reaches [PartFeatures.classify] like any other recompute
// error, so the feature goes Sick, its dependents are quarantined, and the invalid body is DROPPED —
// the running state falls back to the prefix rather than carrying a broken body forward.
var ErrInvalidFeatureResult = errors.New("the rebuilt body is not a valid B-rep")

// ErrAdoptedInvalidBody is the post-condition's NON-FATAL verdict: the body is invalid but the
// feature only adopted it (see [AdoptedBodiesFeature]), so the fault belongs to the source document.
// classify maps it to health.Warning with this reason and KEEPS the body — the same shape the engine
// already uses for ErrDeferred and for reference-heal drift. It must reach health, not only the
// diagnostics list: a green tick over a torn imported body is the silence this stage exists to end
// (finding 2 of the stage-6 review).
var ErrAdoptedInvalidBody = errors.New("the adopted body is not a valid B-rep")

// CodeFeatureInvalidResult names the post-condition failure on the diagnostic channel so an add-in
// and the UI see WHICH invariant broke, not only that the feature is sick.
const CodeFeatureInvalidResult diag.Code = "feature.invalid-result"

// AdoptedBodiesFeature is a feature whose bodies come from OUTSIDE the modelling engine rather than
// from a kernel operation: an import, or another document's geometry pulled through a derive. Its
// output can only be as valid as the source it came from, and refusing every imperfect import is a
// product decision rather than a kernel one — so the post-condition reports such a body as a
// health.Warning carrying the Validate reason, and keeps it, instead of sickening the feature that
// adopted it. Measured: two of the OCCT blend-parity corpus's own STEP fixtures (simple/H3 and
// simple/H5) import with three boundary edges each.
type AdoptedBodiesFeature interface {
	Feature
	// AdoptsExternalBodies reports that this feature's output is adopted, not built.
	AdoptsExternalBodies() bool
}

// The COMPLETE adopted set, surveyed rather than sampled (finding 1 of the stage-6 review, which
// caught DerivedAssemblyComponent and ShrinkwrapComponent missing: the same imperfect STEP body was
// reported through a part derive and SICKENED the feature through an assembly derive). It is the
// non-parametric base plus the three derive-family features already grouped by DeriveStatus in
// derived_assembly.go — each either replays bodies it was handed or pulls another document's, and
// each falls back to `frozen` bodies captured at BreakLink, which are adopted twice over:
//
//	NonParametricBaseFeature  wraps bodies a translator produced (STEP/STL import).
//	DerivedPartComponent      pulls a source PART's bodies, placed by a transform.
//	DerivedAssemblyComponent  pulls a source ASSEMBLY's placed bodies and merges the included ones.
//	ShrinkwrapComponent       simplifies a source assembly's bodies; the simplification cannot be
//	                          more valid than what it simplifies.
//
// AssemblyProxyCutFeature is deliberately NOT here: it reads another occurrence's bodies as a TOOL
// and BUILDS a boolean result, so its output is this engine's work and carries the full
// post-condition.
var (
	_ AdoptedBodiesFeature = (*NonParametricBaseFeature)(nil)
	_ AdoptedBodiesFeature = (*DerivedPartComponent)(nil)
	_ AdoptedBodiesFeature = (*DerivedAssemblyComponent)(nil)
	_ AdoptedBodiesFeature = (*ShrinkwrapComponent)(nil)
)

// postconditionError runs [ops.ValidateTopology] on the bodies the feature BUILT and returns the
// named verdict for the first invalid one, recording it as a Defect on rec: ErrInvalidFeatureResult
// for a body this engine built, ErrAdoptedInvalidBody for one it only adopted. nil when every built
// body is valid.
//
// It runs ops.ValidateTopology, not ops.Validate: level 1 is the per-edge and Euler tests over the
// body's edge and loop counts, which is what makes it affordable on every feature of every
// recompute. ops.Validate adds the hole-containment level, which projects every multi-loop planar
// face's loops into the face plane — materially more work for a verdict (HolesContained) that is not
// part of Valid and that this post-condition would discard (finding 5 of the stage-6 review).
func postconditionError(f Feature, before, after []*topo.Body, rec *diag.Recorder) error {
	report, found := firstInvalidBuiltBody(before, after)
	if !found {
		return nil
	}
	detail := invalidBodyDetail(f.Kind(), report)
	rec.Recordf(CodeFeatureInvalidResult, diag.Defect, "%s", detail)
	if adoptsExternalBodies(f) {
		return fmt.Errorf("%w: %s", ErrAdoptedInvalidBody, detail)
	}
	return fmt.Errorf("%w: %s", ErrInvalidFeatureResult, detail)
}

// firstInvalidBuiltBody returns the level-1 validity report of the first body the feature built that
// fails it, in the order the feature returned them.
func firstInvalidBuiltBody(before, after []*topo.Body) (ops.ValidationReport, bool) {
	for _, b := range builtBodies(before, after) {
		if r := ops.ValidateTopology(b); !r.Valid {
			return r, true
		}
	}
	return ops.ValidationReport{}, false
}

// invalidBodyDetail names which invariant broke and on which entity — the CLAUDE.md rule that an
// exception message carries the offending value, so the browser says more than "recompute failed".
func invalidBodyDetail(kind string, r ops.ValidationReport) string {
	return fmt.Sprintf("%s produced an invalid body (closed=%v manifold=%v oriented=%v euler=%d): %v",
		kind, r.Closed, r.Manifold, r.OrientationOK, r.EulerCharacteristic, r.Issues)
}

// adoptsExternalBodies reports whether the feature declares its bodies adopted rather than built.
func adoptsExternalBodies(f Feature) bool {
	a, ok := f.(AdoptedBodiesFeature)
	return ok && a.AdoptsExternalBodies()
}

// builtBodies returns the bodies of after that are not one of before — the ones this feature actually
// BUILT. A feature that leaves a body alone hands back the same pointer (the identity the engine
// already relies on in producerOf), and that body was validated at the exit of the feature that built
// it, so re-validating it here would make the post-condition cost O(features × bodies) for an answer
// that cannot have changed: a built body's geometry is immutable.
func builtBodies(before, after []*topo.Body) []*topo.Body {
	var built []*topo.Body
	for _, b := range after {
		if b != nil && !holdsBody(before, b) {
			built = append(built, b)
		}
	}
	return built
}
