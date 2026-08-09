// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Combine (Inventor's CombineFeature): boolean a base solid against one or more TOOL bodies in a
// single feature. Lifted out of modify.go when it grew the multi-tool and keep-tools options
// (#1894), which are the two things that make it more than a two-body boolean.

// CombineDefinition booleans the running body at TargetIndex against the bodies at ToolIndices,
// under one operation. KeepTools leaves the tools in the part afterwards instead of consuming
// them, so the same tool can go on to cut something else (Inventor's KeepToolBodies).
type CombineDefinition struct {
	TargetIndex int
	ToolIndices []int
	Operation   ops.PartFeatureOperation
	KeepTools   bool
}

// CombineFeature booleans the target and its tools in the running state into one result.
type CombineFeature struct{ def *CombineDefinition }

// Definition returns the combine recipe.
func (c *CombineFeature) Definition() *CombineDefinition { return c.def }

// Kind implements [Feature].
func (c *CombineFeature) Kind() string { return "combine" }

// Recompute folds every tool into the target under the definition's operation, then replaces the
// consumed bodies with the result. An out-of-range, self-referencing or repeated index is an
// error (the feature goes Sick) rather than a silently dropped tool.
func (c *CombineFeature) Recompute(in Input) (Output, error) {
	if err := c.def.validate(in.Bodies); err != nil {
		return Output{}, err
	}
	res := in.Bodies[c.def.TargetIndex]
	for _, ti := range c.def.ToolIndices {
		next, err := ops.BooleanWithDiagnostics(c.def.Operation, res, in.Bodies[ti], in.Diag)
		if err != nil {
			return Output{}, err
		}
		res = next
		if res == nil || len(res.Faces()) == 0 {
			break // a cut or intersect that consumed everything; further tools cannot restore it
		}
	}
	return Output{Bodies: c.def.replaceCombined(in.Bodies, res)}, nil
}

// validate checks every index names a distinct running body, so a bad recipe reports what is
// wrong instead of quietly combining the wrong pair.
func (d *CombineDefinition) validate(bodies []*topo.Body) error {
	if !validIndex(d.TargetIndex, bodies) {
		return fmt.Errorf("combine: target index %d out of range (have %d bodies)", d.TargetIndex, len(bodies))
	}
	if len(d.ToolIndices) == 0 {
		return fmt.Errorf("combine: no tool bodies given for target %d", d.TargetIndex)
	}
	seen := map[int]bool{}
	for _, ti := range d.ToolIndices {
		switch {
		case !validIndex(ti, bodies):
			return fmt.Errorf("combine: tool index %d out of range (have %d bodies)", ti, len(bodies))
		case ti == d.TargetIndex:
			return fmt.Errorf("combine: tool index %d is the target; a body cannot be its own tool", ti)
		case seen[ti]:
			return fmt.Errorf("combine: tool index %d given twice; each tool applies once", ti)
		}
		seen[ti] = true
	}
	return nil
}

// replaceCombined returns the bodies with the target (and, unless KeepTools, the tools) removed
// and the non-empty result appended. Kept tools stay in their original order, so a later feature
// addressing them by index still finds them where the tree shows them.
func (d *CombineDefinition) replaceCombined(bodies []*topo.Body, res *topo.Body) []*topo.Body {
	dropped := map[int]bool{d.TargetIndex: true}
	if !d.KeepTools {
		for _, ti := range d.ToolIndices {
			dropped[ti] = true
		}
	}
	var out []*topo.Body
	for i, b := range bodies {
		if !dropped[i] {
			out = append(out, b)
		}
	}
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out
}

// AddCombine booleans two running bodies (by index) under op — the single-tool combine.
func (c *ModifyFeatures) AddCombine(targetIndex, toolIndex int, op ops.PartFeatureOperation) *PartFeature {
	return c.AddCombineTools(targetIndex, []int{toolIndex}, op, false)
}

// AddCombineTools booleans the target against several tool bodies at once, optionally keeping the
// tools afterwards (#1894).
//
//	mods.AddCombineTools(0, []int{1, 2}, ops.Cut, true) // drill both tools, keep them for reuse
func (c *ModifyFeatures) AddCombineTools(targetIndex int, toolIndices []int,
	op ops.PartFeatureOperation, keepTools bool) *PartFeature {
	return c.engine.Add(&CombineFeature{def: &CombineDefinition{
		TargetIndex: targetIndex, ToolIndices: toolIndices, Operation: op, KeepTools: keepTools,
	}})
}
