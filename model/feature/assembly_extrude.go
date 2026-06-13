// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// AssemblyExtrudeFeature is an assembly-machining feature authored from a sketch profile
// rather than a fixed tool box — the assembly-context extrude (M11-F08 kind set, #735).
// Its sketch is drawn on an assembly work plane, so the profile already lives in
// assembly space; at recompute it extrudes that profile into a prism by the given
// distance and booleans it against each participant body. The same buildPrism path the
// part extrude uses produces the tool, so a profiled pocket/boss machines every
// participant in place without touching the shared part definitions.
//
// Example: cut a slot profiled in sketch sk through every participant —
//
//	f := feature.NewAssemblyExtrudeFeature(sk, 0, ops.Cut, func() float64 { return 10 })
type AssemblyExtrudeFeature struct {
	sketch       *sketch.Sketch
	profileIndex int
	op           ops.PartFeatureOperation
	distance     func() float64
}

// NewAssemblyExtrudeFeature returns an extrude of the sketch's profileIndex-th closed
// region, applying op over distance (a closure, so a parameter edit reflows it). op is
// typically [ops.Cut] (a profiled pocket) or [ops.Join] (a profiled boss).
func NewAssemblyExtrudeFeature(skt *sketch.Sketch, profileIndex int, op ops.PartFeatureOperation, distance func() float64) *AssemblyExtrudeFeature {
	return &AssemblyExtrudeFeature{sketch: skt, profileIndex: profileIndex, op: op, distance: distance}
}

// Kind implements [Feature].
func (f *AssemblyExtrudeFeature) Kind() string { return "assemblyExtrude" }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblyExtrudeFeature) Operation() ops.PartFeatureOperation { return f.op }

// Recompute resolves the sketch's profile, extrudes it into an assembly-space prism by
// the current distance, and booleans the prism against every running body (an emptied
// body is dropped). A missing/open profile or a non-positive distance is a lost input
// the engine turns into feature health, not a panic.
func (f *AssemblyExtrudeFeature) Recompute(in Input) (Output, error) {
	tool, err := f.buildTool()
	if err != nil {
		return Output{}, err
	}
	out, err := applyToolToAll(f.op, in.Bodies, tool)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: out}, nil
}

// buildTool extrudes the resolved profile into the assembly-space prism tool.
func (f *AssemblyExtrudeFeature) buildTool() (*topo.Body, error) {
	profiles := f.sketch.Profiles()
	if f.profileIndex < 0 || f.profileIndex >= profiles.Count() {
		return nil, fmt.Errorf("assemblyExtrude: profile %d out of range (sketch has %d)", f.profileIndex, profiles.Count())
	}
	p := profiles.Item(f.profileIndex)
	if !p.IsClosed() {
		return nil, fmt.Errorf("assemblyExtrude: profile %d is open; an extrude needs a closed region", f.profileIndex)
	}
	d := f.distance()
	if d <= 0 {
		return nil, fmt.Errorf("assemblyExtrude: distance %g must be positive", d)
	}
	return buildPrismWithHoles(p, f.sketch.Plane(), span{near: 0, far: d}, 0, "asmExtrude"), nil
}
