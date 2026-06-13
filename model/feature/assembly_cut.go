// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// AssemblyCutFeature is an assembly-machining feature: a tool body authored in the
// assembly's space, booleaned against each running body it is applied to. It is the
// V1 representative of the part feature triangle reused in assembly context (M11-F08,
// #633) — like [CombineFeature] it is a thin wrapper over [ops.Boolean], but its tool
// is fixed in assembly space rather than picked from the running bodies, because an
// assembly feature cuts placed component geometry, not a single part's bodies.
//
// The assembly host applies it once per participating occurrence, threading that
// occurrence's assembly-space bodies through Recompute, so the same tool machines
// every participant in place without touching the shared part definitions.
//
// Example: a slot milled through every bracket in an assembly —
//
//	tool, _ := brep.SolidBlock(min, max, "asmCut")
//	f := feature.NewAssemblyCutFeature(tool, ops.Cut)
type AssemblyCutFeature struct {
	kind string
	tool *topo.Body
	op   ops.PartFeatureOperation
}

// NewAssemblyCutFeature returns an assembly feature that applies op with tool. op is
// typically [ops.Cut] (machining away material); [ops.Join] adds the tool as shared
// stock and [ops.Intersect] keeps the common volume.
func NewAssemblyCutFeature(tool *topo.Body, op ops.PartFeatureOperation) *AssemblyCutFeature {
	return &AssemblyCutFeature{kind: "assemblyCut", tool: tool, op: op}
}

// NewAssemblyHoleFeature returns an assembly hole: a drilled cylinder of the given
// diameter and depth from center along axisInto, cut from each participant — a
// parametric assembly-context feature kind that needs no sketch (M11-F08 kind set,
// #735). The tool is faceted (the exact analytic cylinder is a NURBS-phase refinement),
// fixed in assembly space; it reuses the cut machinery and reports kind "assemblyHole".
func NewAssemblyHoleFeature(center math.Point3, axisInto math.UnitVector3, diameter, depth float64) (*AssemblyCutFeature, error) {
	if diameter <= 0 || depth <= 0 {
		return nil, fmt.Errorf("assemblyHole: diameter %g and depth %g must be positive", diameter, depth)
	}
	cyl := drillTool(center, axisInto, diameter/2, depth, "asmHole")
	return &AssemblyCutFeature{kind: "assemblyHole", tool: cyl, op: ops.Cut}, nil
}

// Kind implements [Feature].
func (f *AssemblyCutFeature) Kind() string { return f.kind }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblyCutFeature) Operation() ops.PartFeatureOperation { return f.op }

// ToolBody returns the assembly-space tool, satisfying [ToolFeature].
func (f *AssemblyCutFeature) ToolBody() *topo.Body { return f.tool }

// Recompute booleans the tool against every running body, replacing each with its
// result (an empty result — the tool consumed the whole body — drops it). A missing
// tool is a lost-input failure the engine turns into feature health, not a panic.
func (f *AssemblyCutFeature) Recompute(in Input) (Output, error) {
	if f.tool == nil {
		return Output{}, fmt.Errorf("assemblyCut: nil tool body")
	}
	out := make([]*topo.Body, 0, len(in.Bodies))
	for i, target := range in.Bodies {
		res, err := ops.Boolean(f.op, target, f.tool)
		if err != nil {
			return Output{}, fmt.Errorf("assemblyCut: boolean on body %d: %w", i, err)
		}
		if res != nil && len(res.Faces()) > 0 {
			out = append(out, res)
		}
	}
	return Output{Bodies: out}, nil
}
