// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// AssemblyRevolveFeature is the assembly-machining revolve (M11-F08 kind set, #735): a
// sketch profile, authored on an assembly work plane so it already lives in assembly
// space, spun about an axis into a solid of revolution and booleaned against each
// participant body. It reuses [buildRevolveSolid] — the same revolution the part revolve
// runs — so a profiled groove (Cut) or a turned boss (Join) machines every participant in
// place without touching the shared part definitions.
//
// Example: turn a 90° groove profiled in sketch sk about its single centerline —
//
//	f := feature.NewAssemblyRevolveFeature(sk, 0, nil, ops.Cut, func() float64 { return math.Pi / 2 })
type AssemblyRevolveFeature struct {
	sketch       *sketch.Sketch
	profileIndex int
	axis         *WorkAxis // explicit assembly-space axis; nil ⇒ the sketch's single centerline
	op           ops.PartFeatureOperation
	angle        func() float64
}

// NewAssemblyRevolveFeature returns a revolve of the sketch's profileIndex-th closed region
// about axis (nil ⇒ resolve the sketch's single centerline, like the part revolve), applying
// op over angle radians (a closure, so a parameter edit reflows it). op is typically [ops.Cut]
// (a turned groove) or [ops.Join] (a turned boss).
func NewAssemblyRevolveFeature(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, op ops.PartFeatureOperation, angle func() float64) *AssemblyRevolveFeature {
	return &AssemblyRevolveFeature{sketch: skt, profileIndex: profileIndex, axis: axis, op: op, angle: angle}
}

// Kind implements [Feature].
func (f *AssemblyRevolveFeature) Kind() string { return "assemblyRevolve" }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblyRevolveFeature) Operation() ops.PartFeatureOperation { return f.op }

// Recompute resolves the sketch's profile and axis, spins the profile into an assembly-space
// solid of revolution by the current angle, and booleans it against every running body (an
// emptied body is dropped). A missing/open profile, an unresolvable axis, or an out-of-range
// angle is a lost input the engine turns into feature health, not a panic.
func (f *AssemblyRevolveFeature) Recompute(in Input) (Output, error) {
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

// buildTool spins the resolved profile about the resolved axis into the assembly-space tool.
func (f *AssemblyRevolveFeature) buildTool() (*topo.Body, error) {
	prof, err := resolveSingleProfile(f.sketch, f.profileIndex, "assemblyRevolve")
	if err != nil {
		return nil, err
	}
	axis, err := f.resolveAxis()
	if err != nil {
		return nil, err
	}
	a := f.angle()
	if a <= 0 || a > 2*stdmath.Pi+1e-9 {
		return nil, fmt.Errorf("assemblyRevolve: angle %g must be in (0, 2π]", a)
	}
	return buildRevolveSolid(prof, f.sketch.Plane(), axis, a, 0, "asmRevolve")
}

// resolveAxis returns the explicit axis when set, otherwise the sketch's single centerline.
func (f *AssemblyRevolveFeature) resolveAxis() (*WorkAxis, error) {
	if f.axis != nil {
		return f.axis, nil
	}
	return sketchCenterlineAxis(f.sketch, "assemblyRevolve")
}
