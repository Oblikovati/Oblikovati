// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "oblikovati.org/event"

// Assembly feature-program event type id. It continues the assembly event block
// (occurrence events are 0x0B01–0x0B05; the feature program is 0x0B06) and rides the
// same per-assembly bus, so an add-in subscribing to an assembly receives both its
// occurrence and its feature-program changes (M11-F08, #725). Ids are stable across
// versions (they cross the gRPC seam); never renumber.
const tidAssemblyFeaturesRecomputed event.TypeID = 0x0B06

// FeatureHealth is one feature's post-recompute state in an
// [AssemblyFeaturesRecomputed] event: its id, whether it is suppressed, and its health
// reason ("" when healthy). It carries enough for a consumer to react to health/result
// changes without re-reading the whole program.
type FeatureHealth struct {
	ID         uint64
	Suppressed bool
	Health     string
}

// AssemblyFeaturesRecomputed is raised (After) when the assembly feature program is
// re-evaluated — after every add, participation/suppression edit, or rollback-marker
// move, since each triggers a recompute. It carries each feature's resulting health so
// the push relay can forward a curated feature-change event (M11-F08, #725).
type AssemblyFeaturesRecomputed struct {
	Features []FeatureHealth
}

// EventID implements event.Event.
func (AssemblyFeaturesRecomputed) EventID() event.TypeID { return tidAssemblyFeaturesRecomputed }

// SetBus installs the bus the feature program raises [AssemblyFeaturesRecomputed] on.
// The assembly definition wires this to its shared occurrence/feature bus; a program
// with no bus (a bare collection in a unit test) raises nothing.
func (fs *AssemblyFeatures) SetBus(bus *event.Bus) { fs.bus = bus }

// raiseRecomputed emits the post-recompute health snapshot, if a bus is installed.
func (fs *AssemblyFeatures) raiseRecomputed() {
	if fs.bus == nil {
		return
	}
	snap := make([]FeatureHealth, len(fs.items))
	for i, af := range fs.items {
		snap[i] = FeatureHealth{ID: af.id, Suppressed: af.suppress, Health: af.health.Reason}
	}
	event.Emit(fs.bus, event.After, AssemblyFeaturesRecomputed{Features: snap})
}
