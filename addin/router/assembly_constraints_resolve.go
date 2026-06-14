// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// The constraint-router plumbing: resolve a wire geometry ref to an engine input, build a
// constraint's wire info, and map solution/limits/health between the two. Kept apart from
// the handlers so each stays a few lines.

// twoRefArgs is the shared preamble for the two-input add handlers: it resolves the active
// assembly, decodes the typed args, and resolves the args' two geometry refs to engine
// inputs. The refs accessor pulls the A/B geometry refs out of the decoded args.
func twoRefArgs[T any](s *app.Session, raw json.RawMessage, method string, refs func(T) (wire.ConstraintGeomRef, wire.ConstraintGeomRef)) (*compdef.AssemblyComponentDefinition, T, assembly.Ref, assembly.Ref, error) {
	var in T
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, in, assembly.Ref{}, assembly.Ref{}, err
	}
	if err := decode(raw, &in); err != nil {
		return nil, in, assembly.Ref{}, assembly.Ref{}, err
	}
	refA, refB := refs(in)
	a, err := resolveConstraintRef(asm, refA, method)
	if err != nil {
		return nil, in, assembly.Ref{}, assembly.Ref{}, err
	}
	b, err := resolveConstraintRef(asm, refB, method)
	if err != nil {
		return nil, in, assembly.Ref{}, assembly.Ref{}, err
	}
	return asm, in, a, b, nil
}

// resolveThree resolves three geometry refs (the symmetry inputs: two geometries + plane).
func resolveThree(asm *compdef.AssemblyComponentDefinition, a, b, c wire.ConstraintGeomRef, method string) (assembly.Ref, assembly.Ref, assembly.Ref, error) {
	ra, err := resolveConstraintRef(asm, a, method)
	if err != nil {
		return assembly.Ref{}, assembly.Ref{}, assembly.Ref{}, err
	}
	rb, err := resolveConstraintRef(asm, b, method)
	if err != nil {
		return assembly.Ref{}, assembly.Ref{}, assembly.Ref{}, err
	}
	rc, err := resolveConstraintRef(asm, c, method)
	if err != nil {
		return assembly.Ref{}, assembly.Ref{}, assembly.Ref{}, err
	}
	return ra, rb, rc, nil
}

// resolveConstraintRef turns a wire geometry ref into an engine input: the occurrence plus
// the definition-space primitive its reference key names on the component.
func resolveConstraintRef(asm *compdef.AssemblyComponentDefinition, ref wire.ConstraintGeomRef, method string) (assembly.Ref, error) {
	occ, err := occurrenceByID(asm, ref.Occurrence, method)
	if err != nil {
		return assembly.Ref{}, err
	}
	prim, err := primitiveOnComponent(occ, ref.Entity, method)
	if err != nil {
		return assembly.Ref{}, err
	}
	return assembly.Ref{Occurrence: occ, Primitive: prim, Entity: ref.Entity}, nil
}

// primitiveOnComponent finds the face, edge, or vertex the reference key names on the
// occurrence's component body and extracts its constraint primitive (definition space).
func primitiveOnComponent(occ *occurrence.Occurrence, key, method string) (assembly.Primitive, error) {
	kb := []byte(key)
	for _, b := range componentBodies(occ) {
		if f, ok := b.FindFaceByKey(kb); ok {
			return assembly.PrimitiveFromFace(f)
		}
		if e, ok := b.FindEdgeByKey(kb); ok {
			return assembly.PrimitiveFromEdge(e)
		}
		if v, ok := b.FindVertexByKey(kb); ok {
			return assembly.PrimitiveFromVertex(v), nil
		}
	}
	return assembly.Primitive{}, fmt.Errorf("%s: reference key %q not found on occurrence %d's component", method, key, occ.ID())
}

// solvedConstraint re-solves the assembly after an add and returns the new constraint info.
func solvedConstraint(asm *compdef.AssemblyComponentDefinition, c assembly.Constraint) (json.RawMessage, error) {
	asm.SolveConstraints()
	return json.Marshal(wire.ConstraintResult{Constraint: constraintInfo(c)})
}

// constraintInfo renders a constraint into its wire shape.
func constraintInfo(c assembly.Constraint) wire.AssemblyConstraintInfo {
	a, b := c.AnchorRefs()
	return wire.AssemblyConstraintInfo{
		ID:         c.ID(),
		Type:       c.Type().String(),
		Name:       c.Name(),
		A:          wire.ConstraintGeomRef{Occurrence: a.Occurrence, Entity: a.Entity},
		B:          wire.ConstraintGeomRef{Occurrence: b.Occurrence, Entity: b.Entity},
		Value:      c.Value(),
		Solution:   solutionLabel(c),
		Limits:     limitsInfo(c.Limits()),
		Health:     c.Health().String(),
		Suppressed: c.Suppressed(),
	}
}

// solutionLabel reports the solution-type discriminator for the kinds that carry one
// (mate/angle), or "" for the rest.
func solutionLabel(c assembly.Constraint) string {
	switch t := c.(type) {
	case interface {
		SolutionType() types.MateConstraintSolutionType
	}:
		return t.SolutionType().String()
	case interface {
		SolutionType() types.AngleConstraintSolutionType
	}:
		return t.SolutionType().String()
	default:
		return ""
	}
}

// limitsInfo renders a constraint's limits, or nil when unbounded.
func limitsInfo(l contract.ConstraintLimits) *wire.ConstraintLimits {
	if l == nil {
		return nil
	}
	out := &wire.ConstraintLimits{}
	if v, ok := l.Minimum(); ok {
		out.HasMin, out.Min = true, v
	}
	if v, ok := l.Maximum(); ok {
		out.HasMax, out.Max = true, v
	}
	if v, ok := l.Resting(); ok {
		out.HasResting, out.Resting = true, v
	}
	return out
}

// healthResult renders an assembly solve/health report into its wire shape, deriving the
// well/under/over-constrained status from the DOF and redundancy counts.
func healthResult(rep assembly.SolveReport) wire.AssemblyHealthResult {
	occs := make([]wire.OccurrenceDOFInfo, len(rep.Occurrences))
	for i, o := range rep.Occurrences {
		occs[i] = wire.OccurrenceDOFInfo{Occurrence: o.Occurrence, DegreesOfFreedom: o.DegreesOfFreedom}
	}
	return wire.AssemblyHealthResult{
		Status:           statusLabel(rep),
		Constraints:      rep.Constraints,
		Redundant:        rep.Redundant,
		DegreesOfFreedom: rep.DegreesOfFreedom,
		Converged:        rep.Converged,
		Occurrences:      occs,
	}
}

// statusLabel maps a report's DOF/redundancy to the constraint-state vocabulary.
func statusLabel(rep assembly.SolveReport) string {
	switch {
	case rep.Redundant > 0:
		return "over-constrained"
	case rep.DegreesOfFreedom > 0:
		return "under-constrained"
	default:
		return "well-constrained"
	}
}

// mateSolution maps the wire solution string to the mate solution enum ("" ⇒ opposed).
func mateSolution(s string) types.MateConstraintSolutionType {
	if s == "aligned" {
		return types.MateSolutionAligned
	}
	return types.MateSolutionOpposed
}

// angleSolution maps the wire solution string to the angle solution enum ("" ⇒ undirected).
func angleSolution(s string) types.AngleConstraintSolutionType {
	switch s {
	case "directed":
		return types.AngleSolutionDirected
	case "reference-vector":
		return types.AngleSolutionReferenceVector
	default:
		return types.AngleSolutionUndirected
	}
}
