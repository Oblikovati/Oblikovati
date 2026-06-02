// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// The remaining sketched features carry their full Definition (the triangle) and
// extent/operation surface, but their B-rep generation is phased: revolve is kernel
// phase A (analytic surfaces of revolution), sweep/loft/coil are phase B (NURBS).
// Until generation lands, Recompute reports NotYetImplemented, so the feature goes
// health-sick honestly rather than producing nothing silently.

// RevolveDefinition is the recipe for a revolve: a profile spun about an axis.
type RevolveDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Angle        func() float64 // 0 ⇒ full revolution
	Operation    ops.PartFeatureOperation
}

// RevolveFeature spins a profile about an axis.
type RevolveFeature struct{ def *RevolveDefinition }

func (r *RevolveFeature) Definition() *RevolveDefinition { return r.def }
func (r *RevolveFeature) Kind() string                   { return "revolve" }
func (r *RevolveFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-093-revolve-generation")
}

// RevolveFeatures adds revolves into the engine.
type RevolveFeatures struct{ engine *PartFeatures }

// NewRevolveFeatures binds the collection to an engine.
func NewRevolveFeatures(engine *PartFeatures) *RevolveFeatures { return &RevolveFeatures{engine} }

// Add adds a revolve of the profile about axis through angle (nil ⇒ full).
func (c *RevolveFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{Sketch: skt, ProfileIndex: profileIndex, Axis: axis, Angle: angle, Operation: op}
	return c.engine.Add(&RevolveFeature{def: def})
}

// SweepDefinition is the recipe for a sweep: a profile along a path.
type SweepDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Path         *sketch.Path
	Twist        func() float64
	Operation    ops.PartFeatureOperation
}

// SweepFeature sweeps a profile along a path.
type SweepFeature struct{ def *SweepDefinition }

func (s *SweepFeature) Definition() *SweepDefinition { return s.def }
func (s *SweepFeature) Kind() string                 { return "sweep" }
func (s *SweepFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-094-sweep-generation")
}

// LoftDefinition is the recipe for a loft: a blend through sections.
type LoftDefinition struct {
	Sections  []*sketch.Profile
	Closed    bool
	Operation ops.PartFeatureOperation
}

// LoftFeature blends through sections.
type LoftFeature struct{ def *LoftDefinition }

func (l *LoftFeature) Definition() *LoftDefinition { return l.def }
func (l *LoftFeature) Kind() string                { return "loft" }
func (l *LoftFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-095-loft-generation")
}

// CoilDefinition is the recipe for a coil (helical sweep).
type CoilDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Pitch        func() float64
	Revolutions  func() float64
	Taper        float64
	Operation    ops.PartFeatureOperation
}

// CoilFeature sweeps a profile along a helix.
type CoilFeature struct{ def *CoilDefinition }

func (c *CoilFeature) Definition() *CoilDefinition { return c.def }
func (c *CoilFeature) Kind() string                { return "coil" }
func (c *CoilFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-096-coil-generation")
}

// RibDefinition is the recipe for a rib: a thin support from an open profile.
type RibDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Thickness    func() float64
	Operation    ops.PartFeatureOperation
}

// RibFeature thickens an open profile into a support.
type RibFeature struct{ def *RibDefinition }

func (r *RibFeature) Definition() *RibDefinition { return r.def }
func (r *RibFeature) Kind() string               { return "rib" }
func (r *RibFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-096-rib-generation")
}
