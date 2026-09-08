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

// CodeFeatureInvalidResult names the post-condition failure on the diagnostic channel so an add-in
// and the UI see WHICH invariant broke, not only that the feature is sick.
const CodeFeatureInvalidResult diag.Code = "feature.invalid-result"

// AdoptedBodiesFeature is a feature whose bodies come from OUTSIDE the modelling engine rather than
// from a kernel operation: a non-parametric base wrapping an imported STEP/STL body, a derived
// component pulling another document's. Its output can only be as valid as the file it came from, and
// refusing every imperfect import is a product decision rather than a kernel one — so the
// post-condition REPORTS such a body instead of sickening the feature that adopted it. Measured: two
// of the OCCT blend-parity corpus's own STEP fixtures (simple/H3 and simple/H5) import with three
// boundary edges each.
type AdoptedBodiesFeature interface {
	Feature
	// AdoptsExternalBodies reports that this feature's output is adopted, not built.
	AdoptsExternalBodies() bool
}

// Both features whose bodies come from outside the engine declare it here, so the compiler keeps the
// declaration and the post-condition's exemption in step.
var (
	_ AdoptedBodiesFeature = (*NonParametricBaseFeature)(nil)
	_ AdoptedBodiesFeature = (*DerivedPartComponent)(nil)
)

// postconditionError runs [ops.Validate] on the bodies the feature BUILT and returns the named error
// for the first invalid one, recording it as a Defect on rec. It returns nil when every built body is
// valid, and nil-with-a-Defect for an [AdoptedBodiesFeature].
//
// ops.Validate is the CHEAPEST of the ordered validity levels — topology and Euler only, over the
// body's edge list, reading no geometry and no tessellation — which is what makes it affordable on
// every feature of every recompute. The self-intersection and tolerance-consistency levels are
// separate operations and deliberately not run here.
func postconditionError(f Feature, before, after []*topo.Body, rec *diag.Recorder) error {
	report, found := firstInvalidBuiltBody(before, after)
	if !found {
		return nil
	}
	detail := invalidBodyDetail(f.Kind(), report)
	rec.Recordf(CodeFeatureInvalidResult, diag.Defect, "%s", detail)
	if adoptsExternalBodies(f) {
		return nil // the defect belongs to the imported file, not to the feature that adopted it
	}
	return fmt.Errorf("%w: %s", ErrInvalidFeatureResult, detail)
}

// firstInvalidBuiltBody returns the validity report of the first body the feature built that fails
// ops.Validate, in the order the feature returned them.
func firstInvalidBuiltBody(before, after []*topo.Body) (ops.ValidationReport, bool) {
	for _, b := range builtBodies(before, after) {
		if r := ops.Validate(b); !r.Valid {
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
