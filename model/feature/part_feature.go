// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/health"
)

// PartFeature wraps a [Feature] with the engine's per-feature state: identity,
// name, dependencies, suppression, health, and the cached body state produced just
// after it (so the clean prefix is reused, never recomputed — ADR-0010).
type PartFeature struct {
	id         ID
	name       string
	feature    Feature
	deps       []ID
	suppress   bool
	condition  *suppressionCondition
	health     health.Health
	dirty      bool
	recomputes int
	cached     []*topo.Body
	diags      []diag.Diagnostic // kernel diagnostics the OPERATIONS reported last evaluation (#1601)
	// resultBodies are the bodies this feature produced that survived into the current result, and
	// resultDiags is what they CARRY (the assembler's build report, the tessellator's) — computed on
	// first ask, because reading it means meshing them, and memoized until the bodies change (#2058).
	resultBodies []*topo.Body
	resultDiags  []diag.Diagnostic
	resultRead   bool
	seq          uint64 // global creation stamp; see model/seq

	// paramReads is the model parameters this feature read DIRECTLY during its last
	// evaluation — a sheet-metal thickness, a suppression condition (NOT the
	// dimensions of a consumed sketch, which are reported via ConsumedSketches). The
	// engine uses it, with the consumed sketches' footprints, to skip the feature on a
	// parameter edit that touches none of them (Oblikovati#1414). Held as depend.Keys so
	// the same attribution serves a future non-parameter source (ADR-0044).
	paramReads []depend.Key
}

// ID returns the feature's stable handle (unchanged by rename).
func (f *PartFeature) ID() ID { return f.id }

// Seq returns the feature's global creation stamp, shared with sketches and work
// features so the browser interleaves all three by creation order (issue #132).
func (f *PartFeature) Seq() uint64 { return f.seq }

// Name/SetName get and set the display name; the id is stable across renames.
func (f *PartFeature) Name() string     { return f.name }
func (f *PartFeature) SetName(n string) { f.name = n }

// Kind returns the wrapped feature's type name.
func (f *PartFeature) Kind() string { return f.feature.Kind() }

// Diagnostics returns the kernel diagnostics for this feature's current state — degradations that did
// not sicken it but that users and add-ins must be able to SEE rather than discover downstream. It is
// what the OPERATIONS reported while it rebuilt (a boolean faceting analytic surfaces, a CSG fallback
// — #1601), followed by what the body it produced CARRIES (the assembler's build report, the
// tessellator's — #2058).
//
// The second half is computed HERE, on the first ask after a recompute, and memoized until the
// feature's surviving bodies change: reading it means meshing them, which can cost far more than
// building them, so the price falls on the caller who wants the answer rather than on every rebuild.
// Call it from the goroutine that drives the model, like the rest of the recompute engine.
func (f *PartFeature) Diagnostics() []diag.Diagnostic {
	if !f.resultRead {
		f.resultDiags, f.resultRead = bodyDegradations(f.resultBodies), true
	}
	if len(f.resultDiags) == 0 {
		return f.diags
	}
	return append(append([]diag.Diagnostic(nil), f.diags...), f.resultDiags...)
}

// setResultBodies records the bodies this feature produced that survived the recompute, invalidating
// the memoized report only when they actually changed — an unchanged body's verdict cannot have,
// since a built body's geometry is immutable.
func (f *PartFeature) setResultBodies(bodies []*topo.Body) {
	if sameBodies(f.resultBodies, bodies) {
		return
	}
	f.resultBodies, f.resultDiags, f.resultRead = bodies, nil, false
}

// Definition returns the wrapped feature recipe.
func (f *PartFeature) Definition() Feature { return f.feature }

// Health returns the feature's current evaluation health.
func (f *PartFeature) Health() health.Health { return f.health }

// Suppressed reports whether the feature is explicitly suppressed.
func (f *PartFeature) Suppressed() bool { return f.suppress }

// SetSuppressed toggles explicit suppression and marks the feature dirty.
func (f *PartFeature) SetSuppressed(s bool) {
	f.suppress = s
	f.dirty = true
}

// Dependencies returns the feature ids whose output this feature consumes.
func (f *PartFeature) Dependencies() []ID { return append([]ID(nil), f.deps...) }

// SetDependencies records which earlier features this one consumes (derived from
// its resolved input refs). Reorder validation and dirty propagation use these.
func (f *PartFeature) SetDependencies(deps []ID) { f.deps = append([]ID(nil), deps...) }

// RecomputeCount is how many times the feature has been evaluated — used by tests
// to assert the clean prefix was reused.
func (f *PartFeature) RecomputeCount() int { return f.recomputes }

// SetSuppressionCondition makes the feature suppressed whenever the named
// parameter's value compares to threshold per cmp (conditional suppression). It
// marks the feature dirty so the next recompute re-evaluates the condition.
func (f *PartFeature) SetSuppressionCondition(paramName string, cmp ComparisonType, threshold float64) {
	f.condition = &suppressionCondition{paramName: paramName, cmp: cmp, threshold: threshold}
	f.dirty = true
}

// ClearSuppressionCondition removes any conditional suppression.
func (f *PartFeature) ClearSuppressionCondition() {
	f.condition = nil
	f.dirty = true
}
