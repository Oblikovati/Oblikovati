// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
)

// The assembly constraint surface (M12-F01, #358/#363): add the relationships that
// position occurrences, solve the active assembly, and report its health/DOF. A geometry
// input is an occurrence id plus a reference key on its component, which we resolve to a
// definition-space primitive the engine consumes (the occurrence's placement transforms
// it at solve time). Each add re-solves and returns the new constraint's info.

// registerAssemblyConstraintHandlers wires the assemblyConstraints.* methods.
func (r *Router) registerAssemblyConstraintHandlers() {
	r.readOnly(wire.MethodAssemblyConstraintsList, assemblyQuery(assemblyConstraintsList))
	r.mutating(wire.MethodAssemblyConstraintsAddMate, "Add Constraint", assemblyAddMate)
	r.mutating(wire.MethodAssemblyConstraintsAddFlush, "Add Constraint", assemblyAddFlush)
	r.mutating(wire.MethodAssemblyConstraintsAddAngle, "Add Constraint", assemblyAddAngle)
	r.mutating(wire.MethodAssemblyConstraintsAddTangent, "Add Constraint", assemblyAddTangent)
	r.mutating(wire.MethodAssemblyConstraintsAddInsert, "Add Constraint", assemblyAddInsert)
	r.mutating(wire.MethodAssemblyConstraintsSnap, "Add Constraint", assemblySnapConstrain)
	r.mutating(wire.MethodAssemblyConstraintsAddSymmetry, "Add Constraint", assemblyAddSymmetry)
	r.mutating(wire.MethodAssemblyConstraintsAddRotateRotate, "Add Constraint", assemblyAddRotateRotate)
	r.mutating(wire.MethodAssemblyConstraintsAddRotateTranslate, "Add Constraint", assemblyAddRotateTranslate)
	r.mutating(wire.MethodAssemblyConstraintsAddTranslateTranslate, "Add Constraint", assemblyAddTranslateTranslate)
	r.mutating(wire.MethodAssemblyConstraintsAddTransitional, "Add Constraint", assemblyAddTransitional)
	r.mutating(wire.MethodAssemblyConstraintsAddCustom, "Add Constraint", assemblyAddCustom)
	r.mutating(wire.MethodAssemblyConstraintsDelete, "Delete Constraint", typedAssembly(assemblyConstraintDelete))
	r.mutating(wire.MethodAssemblyConstraintsSetLimits, "Edit Constraint", typedAssembly(assemblyConstraintSetLimits))
	r.mutating(wire.MethodAssemblyConstraintsSolve, "", assemblyQuery(assemblyConstraintsSolve))
	r.readOnly(wire.MethodAssemblyConstraintsHealth, assemblyQuery(assemblyConstraintsHealth))
}

// assemblyConstraintsList returns the active assembly's constraint set.
func assemblyConstraintsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.ConstraintsResult, error) {
	return constraintsListResult(asm), nil
}

// constraintsListResult renders the active assembly's constraint set.
func constraintsListResult(asm *compdef.AssemblyComponentDefinition) wire.ConstraintsResult {
	set := asm.Constraints()
	out := make([]wire.AssemblyConstraintInfo, 0, set.Count())
	for _, c := range set.All() {
		out = append(out, constraintInfo(c))
	}
	return wire.ConstraintsResult{Constraints: out}
}

func assemblyAddMate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddMateArgs](s, raw, wire.MethodAssemblyConstraintsAddMate, func(v wire.AddMateArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddMate(a, b, in.Offset, mateSolution(in.Solution)))
}

func assemblyAddFlush(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddFlushArgs](s, raw, wire.MethodAssemblyConstraintsAddFlush, func(v wire.AddFlushArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddFlush(a, b, in.Offset))
}

func assemblyAddAngle(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddAngleArgs](s, raw, wire.MethodAssemblyConstraintsAddAngle, func(v wire.AddAngleArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	if angleSolution(in.Solution) == types.AngleSolutionReferenceVector {
		return assemblyAddAngleReferenceVector(asm, in, a, b)
	}
	return solvedConstraint(asm, asm.Constraints().AddAngle(a, b, in.Angle, angleSolution(in.Solution)))
}

// assemblyAddAngleReferenceVector adds the reference-vector angle solution, which needs the
// explicit third axis; a missing ReferenceVector is rejected with a clear error (#1972).
func assemblyAddAngleReferenceVector(asm *compdef.AssemblyComponentDefinition, in wire.AddAngleArgs, a, b assembly.Ref) (json.RawMessage, error) {
	if in.ReferenceVector.Occurrence == 0 && in.ReferenceVector.Entity == "" {
		return nil, fmt.Errorf("%s: the reference-vector angle solution needs a referenceVector entity", wire.MethodAssemblyConstraintsAddAngle)
	}
	refVec, err := resolveConstraintRef(asm, in.ReferenceVector, wire.MethodAssemblyConstraintsAddAngle)
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddAngleAbout(a, b, refVec, in.Angle))
}

func assemblyAddTangent(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddTangentArgs](s, raw, wire.MethodAssemblyConstraintsAddTangent, func(v wire.AddTangentArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddTangent(a, b, in.Inside))
}

func assemblyAddInsert(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddInsertArgs](s, raw, wire.MethodAssemblyConstraintsAddInsert, func(v wire.AddInsertArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddInsert(a, b, in.Offset, in.Aligned))
}

// assemblySnapConstrain infers the grip-snap constraint between the two geometry inputs (or honours
// the prefer override), creates it, and re-solves so the part jumps into place.
func assemblySnapConstrain(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.SnapConstraintArgs](s, raw, wire.MethodAssemblyConstraintsSnap, func(v wire.SnapConstraintArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) {
		return v.A, v.B
	})
	if err != nil {
		return nil, err
	}
	prefer, err := snapPrefer(in.Prefer)
	if err != nil {
		return nil, err
	}
	c, _, err := asm.Constraints().InferGripConstraint(a, b, prefer)
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, c)
}

func assemblyAddSymmetry(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSymmetryArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, b, plane, err := resolveThree(asm, in.A, in.B, in.Plane, wire.MethodAssemblyConstraintsAddSymmetry)
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddSymmetry(a, b, plane))
}

func assemblyAddRotateRotate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddRotateRotateArgs](s, raw, wire.MethodAssemblyConstraintsAddRotateRotate, func(v wire.AddRotateRotateArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddRotateRotate(a, b, in.Ratio))
}

func assemblyAddRotateTranslate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddRotateTranslateArgs](s, raw, wire.MethodAssemblyConstraintsAddRotateTranslate, func(v wire.AddRotateTranslateArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddRotateTranslate(a, b, in.Distance))
}

func assemblyAddTranslateTranslate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddTranslateTranslateArgs](s, raw, wire.MethodAssemblyConstraintsAddTranslateTranslate, func(v wire.AddTranslateTranslateArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) {
		return v.A, v.B
	})
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddTranslateTranslate(a, b, in.Ratio))
}

func assemblyAddTransitional(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, _, a, b, err := twoRefArgs[wire.AddTransitionalArgs](s, raw, wire.MethodAssemblyConstraintsAddTransitional, func(v wire.AddTransitionalArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddTransitional(a, b))
}

func assemblyAddCustom(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, a, b, err := twoRefArgs[wire.AddCustomArgs](s, raw, wire.MethodAssemblyConstraintsAddCustom, func(v wire.AddCustomArgs) (wire.ConstraintGeomRef, wire.ConstraintGeomRef) { return v.A, v.B })
	if err != nil {
		return nil, err
	}
	return solvedConstraint(asm, asm.Constraints().AddCustom(a, b, in.Kind, in.Params))
}

// assemblyConstraintDelete removes a constraint, re-solves, and returns the remaining set.
func assemblyConstraintDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.DeleteAssemblyConstraintArgs) (wire.ConstraintsResult, error) {
	if !asm.Constraints().Delete(in.ID) {
		return wire.ConstraintsResult{}, fmt.Errorf("%s: no constraint with id %d", wire.MethodAssemblyConstraintsDelete, in.ID)
	}
	asm.SolveConstraints()
	return constraintsListResult(asm), nil
}

// assemblyConstraintSetLimits sets a constraint's driven-value bounds and returns its info.
func assemblyConstraintSetLimits(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetConstraintLimitsArgs) (wire.ConstraintResult, error) {
	l := in.Limits
	lim := assembly.NewLimits(l.Min, l.HasMin, l.Max, l.HasMax, l.Resting, l.HasResting)
	if err := asm.Constraints().SetLimits(in.ID, lim); err != nil {
		return wire.ConstraintResult{}, err
	}
	return wire.ConstraintResult{Constraint: constraintInfo(asm.Constraints().ByID(in.ID))}, nil
}

// assemblyConstraintsSolve re-solves the assembly and returns its health/DOF report.
func assemblyConstraintsSolve(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.AssemblyHealthResult, error) {
	return healthResult(asm.SolveConstraints()), nil
}

// assemblyConstraintsHealth reports the current health/DOF without re-solving.
func assemblyConstraintsHealth(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.AssemblyHealthResult, error) {
	return healthResult(asm.Constraints().Health()), nil
}
