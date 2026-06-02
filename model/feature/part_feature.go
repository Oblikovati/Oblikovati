// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/health"
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
}

// ID returns the feature's stable handle (unchanged by rename).
func (f *PartFeature) ID() ID { return f.id }

// Name/SetName get and set the display name; the id is stable across renames.
func (f *PartFeature) Name() string     { return f.name }
func (f *PartFeature) SetName(n string) { f.name = n }

// Kind returns the wrapped feature's type name.
func (f *PartFeature) Kind() string { return f.feature.Kind() }

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
