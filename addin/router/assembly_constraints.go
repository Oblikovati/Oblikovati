// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
)

// The assembly constraint surface (M12-F01, #358/#363): add the relationships that
// position occurrences, solve the active assembly, and report its health/DOF. A geometry
// input is an occurrence id plus a reference key on its component, which we resolve to a
// definition-space primitive the engine consumes (the occurrence's placement transforms
// it at solve time). Each add re-solves and returns the new constraint's info.

// registerAssemblyConstraintHandlers wires the assemblyConstraints.* methods.
func (r *Router) registerAssemblyConstraintHandlers() {
	r.handlers[wire.MethodAssemblyConstraintsList] = assemblyConstraintsList
	r.handlers[wire.MethodAssemblyConstraintsAddMate] = assemblyAddMate
	r.handlers[wire.MethodAssemblyConstraintsAddFlush] = assemblyAddFlush
	r.handlers[wire.MethodAssemblyConstraintsAddAngle] = assemblyAddAngle
	r.handlers[wire.MethodAssemblyConstraintsAddTangent] = assemblyAddTangent
	r.handlers[wire.MethodAssemblyConstraintsAddInsert] = assemblyAddInsert
	r.handlers[wire.MethodAssemblyConstraintsAddSymmetry] = assemblyAddSymmetry
	r.handlers[wire.MethodAssemblyConstraintsAddRotateRotate] = assemblyAddRotateRotate
	r.handlers[wire.MethodAssemblyConstraintsAddRotateTranslate] = assemblyAddRotateTranslate
	r.handlers[wire.MethodAssemblyConstraintsAddTranslateTranslate] = assemblyAddTranslateTranslate
	r.handlers[wire.MethodAssemblyConstraintsAddTransitional] = assemblyAddTransitional
	r.handlers[wire.MethodAssemblyConstraintsAddCustom] = assemblyAddCustom
	r.handlers[wire.MethodAssemblyConstraintsDelete] = assemblyConstraintDelete
	r.handlers[wire.MethodAssemblyConstraintsSetLimits] = assemblyConstraintSetLimits
	r.handlers[wire.MethodAssemblyConstraintsSolve] = assemblyConstraintsSolve
	r.handlers[wire.MethodAssemblyConstraintsHealth] = assemblyConstraintsHealth
}

// assemblyConstraintsList returns the active assembly's constraint set.
func assemblyConstraintsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	set := asm.Constraints()
	out := make([]wire.AssemblyConstraintInfo, 0, set.Count())
	for _, c := range set.All() {
		out = append(out, constraintInfo(c))
	}
	return json.Marshal(wire.ConstraintsResult{Constraints: out})
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
	return solvedConstraint(asm, asm.Constraints().AddAngle(a, b, in.Angle, angleSolution(in.Solution)))
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
func assemblyConstraintDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteAssemblyConstraintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if !asm.Constraints().Delete(in.ID) {
		return nil, fmt.Errorf("%s: no constraint with id %d", wire.MethodAssemblyConstraintsDelete, in.ID)
	}
	asm.SolveConstraints()
	return assemblyConstraintsList(s, nil)
}

// assemblyConstraintSetLimits sets a constraint's driven-value bounds and returns its info.
func assemblyConstraintSetLimits(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetConstraintLimitsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	l := in.Limits
	lim := assembly.NewLimits(l.Min, l.HasMin, l.Max, l.HasMax, l.Resting, l.HasResting)
	if err := asm.Constraints().SetLimits(in.ID, lim); err != nil {
		return nil, err
	}
	return json.Marshal(wire.ConstraintResult{Constraint: constraintInfo(asm.Constraints().ByID(in.ID))})
}

// assemblyConstraintsSolve re-solves the assembly and returns its health/DOF report.
func assemblyConstraintsSolve(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(healthResult(asm.SolveConstraints()))
}

// assemblyConstraintsHealth reports the current health/DOF without re-solving.
func assemblyConstraintsHealth(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(healthResult(asm.Constraints().Health()))
}
