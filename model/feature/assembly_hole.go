// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// AssemblyHoleFeature is a parametric assembly-machining hole: a drilled cylinder of the
// given diameter and depth from center along axis, cut from each participating occurrence —
// a parametric assembly-context feature kind that needs no sketch (M11-F08 kind set, #735).
// Unlike the box [AssemblyCutFeature], it RETAINS its inputs and rebuilds the drill tool on
// every recompute, so its diameter and depth are editable after placement via
// assemblyFeatures.edit (#752). The tool is faceted (the exact analytic cylinder is a
// NURBS-phase refinement).
//
// Example: drill a 0.5-wide, 1.5-deep hole down +z through every participant —
//
//	h, _ := feature.NewAssemblyHoleFeature(math.P3(0.5, 0.5, 0), zAxis, 0.5, 1.5)
type AssemblyHoleFeature struct {
	center   math.Point3
	axis     math.UnitVector3
	diameter float64
	depth    float64
}

// NewAssemblyHoleFeature returns a parametric assembly hole. diameter and depth must be
// positive; they can be re-dimensioned later through [AssemblyHoleFeature.EditableParams].
func NewAssemblyHoleFeature(center math.Point3, axisInto math.UnitVector3, diameter, depth float64) (*AssemblyHoleFeature, error) {
	if diameter <= 0 || depth <= 0 {
		return nil, fmt.Errorf("assemblyHole: diameter %g and depth %g must be positive", diameter, depth)
	}
	return &AssemblyHoleFeature{center: center, axis: axisInto, diameter: diameter, depth: depth}, nil
}

// Kind implements [Feature].
func (f *AssemblyHoleFeature) Kind() string { return "assemblyHole" }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblyHoleFeature) Operation() ops.PartFeatureOperation { return ops.Cut }

// ToolBody returns the assembly-space drill tool, satisfying [ToolFeature].
func (f *AssemblyHoleFeature) ToolBody() *topo.Body { return f.tool() }

// tool rebuilds the faceted drill cylinder from the current diameter/depth, so an edit
// re-dimensions the bore on the next recompute.
func (f *AssemblyHoleFeature) tool() *topo.Body {
	return drillTool(f.center, f.axis, f.diameter/2, f.depth, "asmHole")
}

// Recompute drills the hole out of every running body. A non-positive diameter or depth —
// possible after an edit — is a lost input the engine turns into feature health, not a
// panic.
func (f *AssemblyHoleFeature) Recompute(in Input) (Output, error) {
	if f.diameter <= 0 || f.depth <= 0 {
		return Output{}, fmt.Errorf("assemblyHole: diameter %g and depth %g must be positive", f.diameter, f.depth)
	}
	out, err := applyToolToAll(ops.Cut, in.Bodies, f.tool())
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: out}, nil
}

// EditableParams exposes the hole's diameter and depth, so assemblyFeatures.edit can
// re-dimension a placed assembly hole (#752).
func (f *AssemblyHoleFeature) EditableParams() []EditableParam {
	return []EditableParam{
		floatParam("Diameter", param.Length, &f.diameter),
		floatParam("Depth", param.Length, &f.depth),
	}
}
