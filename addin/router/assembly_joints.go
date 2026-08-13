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

// The assembly joint surface (M12-F02, #359/#364): author the simplified joints that
// establish a degree-of-freedom set between two occurrences, set their limits and flip, and
// the DS-joint (DOF/imposed-motion) view. Joint origins resolve like constraint geometry
// (occurrence id + reference key → definition-space primitive). Joints and constraints solve
// together: every mutating call re-runs the assembly's combined solve.

// registerAssemblyJointHandlers wires the assemblyJoints.* and dsJoints.* methods.
func (r *Router) registerAssemblyJointHandlers() {
	r.readOnly(wire.MethodAssemblyJointsList, assemblyQuery(assemblyJointsList))
	r.mutating(wire.MethodAssemblyJointsAddRigid, "Add Joint", jointAdder((*assembly.JointSet).AddRigid))
	r.mutating(wire.MethodAssemblyJointsAddRotational, "Add Joint", jointAdder((*assembly.JointSet).AddRotational))
	r.mutating(wire.MethodAssemblyJointsAddSlider, "Add Joint", jointAdder((*assembly.JointSet).AddSlider))
	r.mutating(wire.MethodAssemblyJointsAddCylindrical, "Add Joint", jointAdder((*assembly.JointSet).AddCylindrical))
	r.mutating(wire.MethodAssemblyJointsAddPlanar, "Add Joint", jointAdder((*assembly.JointSet).AddPlanar))
	r.mutating(wire.MethodAssemblyJointsAddBall, "Add Joint", jointAdder((*assembly.JointSet).AddBall))
	r.mutating(wire.MethodAssemblyJointsDelete, "Delete Joint", typedAssembly(assemblyJointDelete))
	r.mutating(wire.MethodAssemblyJointsSetLimits, "Edit Joint", typedAssembly(assemblyJointSetLimits))
	r.mutating(wire.MethodAssemblyJointsSetFlip, "Edit Joint", typedAssembly(assemblyJointSetFlip))
	r.mutating(wire.MethodAssemblyJointsSetState, "Edit Joint", typedAssembly(assemblyJointSetState))
	r.mutating(wire.MethodAssemblyJointsSetOrigin, "Edit Joint Origin", typedAssembly(assemblyJointSetOrigin))
	r.readOnly(wire.MethodDSJointsList, assemblyQuery(dsJointsList))
	r.mutating(wire.MethodDSJointsAdd, "Add Joint", dsJointAdd)
	r.mutating(wire.MethodDSJointsSetImposedMotion, "Edit Joint Motion", typedAssembly(dsJointSetImposedMotion))
	r.mutating(wire.MethodDSJointsDelete, "Delete Joint", typedAssembly(dsJointDelete))
}

// assemblyJointsList returns the active assembly's joint set.
func assemblyJointsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.AssemblyJointsResult, error) {
	return jointsListResult(asm), nil
}

// jointsListResult renders the active assembly's joint set.
func jointsListResult(asm *compdef.AssemblyComponentDefinition) wire.AssemblyJointsResult {
	set := asm.Joints()
	out := make([]wire.JointInfo, 0, set.Count())
	for _, j := range set.All() {
		out = append(out, jointInfo(j))
	}
	return wire.AssemblyJointsResult{Joints: out}
}

// jointAdder builds a handler that resolves the two joint origins and adds a joint via add.
func jointAdder(add func(*assembly.JointSet, assembly.Ref, assembly.Ref) assembly.Joint) handlerFunc {
	return func(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
		asm, err := modelaccess.ActiveAssembly(s)
		if err != nil {
			return nil, err
		}
		var in wire.AddJointArgs
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		a, err := resolveConstraintRef(asm, in.A, "assemblyJoints.add")
		if err != nil {
			return nil, err
		}
		b, err := resolveConstraintRef(asm, in.B, "assemblyJoints.add")
		if err != nil {
			return nil, err
		}
		j := add(asm.Joints(), a, b)
		if in.Flip {
			_ = asm.Joints().SetFlip(j.ID(), true)
		}
		if in.Gap != 0 {
			_ = asm.Joints().SetGap(j.ID(), in.Gap)
		}
		asm.SolveConstraints()
		return json.Marshal(wire.AssemblyJointResult{Joint: jointInfo(j)})
	}
}

// assemblyJointDelete removes a joint, re-solves, and returns the remaining set.
func assemblyJointDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.DeleteJointArgs) (wire.AssemblyJointsResult, error) {
	if !asm.Joints().Delete(in.ID) {
		return wire.AssemblyJointsResult{}, fmt.Errorf("%s: no joint with id %d", wire.MethodAssemblyJointsDelete, in.ID)
	}
	asm.SolveConstraints()
	return jointsListResult(asm), nil
}

// assemblyJointSetLimits sets a joint's linear/angular bounds and returns its info.
func assemblyJointSetLimits(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetJointLimitsArgs) (wire.AssemblyJointResult, error) {
	l := in.Limits
	lim := assembly.NewJointLimits(l.LinearMin, l.HasLinearMin, l.LinearMax, l.HasLinearMax, l.AngularMin, l.HasAngularMin, l.AngularMax, l.HasAngularMax)
	if err := asm.Joints().SetLimits(in.ID, lim); err != nil {
		return wire.AssemblyJointResult{}, err
	}
	return jointResult(asm, in.ID), nil
}

// assemblyJointSetState applies any subset of a joint's seating and state (#1970/#1974) — gap,
// linear/angular rest position, locked, protected — re-solves, and returns the joint's info.
func assemblyJointSetState(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetJointStateArgs) (wire.AssemblyJointResult, error) {
	joints := asm.Joints()
	if err := applyJointState(joints, in); err != nil {
		return wire.AssemblyJointResult{}, err
	}
	asm.SolveConstraints()
	return jointResult(asm, in.ID), nil
}

// assemblyJointSetOrigin defines one of a joint's two origins — inferred, offset, or the midplane
// between two faces — and re-solves the assembly (#1973).
func assemblyJointSetOrigin(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetJointOriginArgs) (wire.AssemblyJointResult, error) {
	j := asm.Joints().ByID(in.ID)
	if j == nil {
		return wire.AssemblyJointResult{}, fmt.Errorf("%s: no joint with id %d", wire.MethodAssemblyJointsSetOrigin, in.ID)
	}
	if in.Which != 1 && in.Which != 2 {
		return wire.AssemblyJointResult{}, fmt.Errorf("%s: which must be 1 or 2, got %d", wire.MethodAssemblyJointsSetOrigin, in.Which)
	}
	mode, ok := types.ParseAssemblyJointOriginMode(in.Mode)
	if !ok {
		return wire.AssemblyJointResult{}, fmt.Errorf("%s: unknown origin mode %q", wire.MethodAssemblyJointsSetOrigin, in.Mode)
	}
	if err := applyJointOrigin(asm, j, in, mode); err != nil {
		return wire.AssemblyJointResult{}, err
	}
	asm.SolveConstraints()
	return jointResult(asm, in.ID), nil
}

// applyJointOrigin dispatches the origin definition onto joint origin one or two (#1973).
func applyJointOrigin(asm *compdef.AssemblyComponentDefinition, j assembly.Joint, in wire.SetJointOriginArgs, mode types.AssemblyJointOriginMode) error {
	one := in.Which == 1
	switch mode {
	case types.JointOriginInfer:
		pick(one, j.SetOriginOneAsInfer, j.SetOriginTwoAsInfer)()
	case types.JointOriginOffset:
		pick(one, func() { j.SetOriginOneAsOffset(in.XOffset, in.YOffset) }, func() { j.SetOriginTwoAsOffset(in.XOffset, in.YOffset) })()
	case types.JointOriginBetweenTwoFaces:
		fa, fb, err := resolveTwoFaces(asm, in)
		if err != nil {
			return err
		}
		pick(one, func() { j.SetOriginOneAsBetweenTwoFaces(fa, fb) }, func() { j.SetOriginTwoAsBetweenTwoFaces(fa, fb) })()
	}
	return nil
}

// pick returns whichOne when one is true, else whichTwo — a small dispatch helper for origin one/two.
func pick(one bool, whichOne, whichTwo func()) func() {
	if one {
		return whichOne
	}
	return whichTwo
}

// resolveTwoFaces resolves the two face references of a between-two-faces origin.
func resolveTwoFaces(asm *compdef.AssemblyComponentDefinition, in wire.SetJointOriginArgs) (assembly.Ref, assembly.Ref, error) {
	fa, err := resolveConstraintRef(asm, in.FaceA, wire.MethodAssemblyJointsSetOrigin)
	if err != nil {
		return assembly.Ref{}, assembly.Ref{}, err
	}
	fb, err := resolveConstraintRef(asm, in.FaceB, wire.MethodAssemblyJointsSetOrigin)
	if err != nil {
		return assembly.Ref{}, assembly.Ref{}, err
	}
	return fa, fb, nil
}

// applyJointState sets each present state field onto the joint with id in.ID.
func applyJointState(joints *assembly.JointSet, in wire.SetJointStateArgs) error {
	if in.Gap != nil {
		if err := joints.SetGap(in.ID, *in.Gap); err != nil {
			return err
		}
	}
	if in.LinearPosition != nil || in.AngularPosition != nil {
		j := joints.ByID(in.ID)
		if j == nil {
			return fmt.Errorf("%s: no joint with id %d", wire.MethodAssemblyJointsSetState, in.ID)
		}
		lin, ang := valueOr(in.LinearPosition, j.LinearPosition()), valueOr(in.AngularPosition, j.AngularPosition())
		if err := joints.SetPositions(in.ID, lin, ang); err != nil {
			return err
		}
	}
	if in.Locked != nil {
		if err := joints.SetLocked(in.ID, *in.Locked); err != nil {
			return err
		}
	}
	if in.Protected != nil {
		return joints.SetProtected(in.ID, *in.Protected)
	}
	return nil
}

// valueOr returns *p when set, else fallback.
func valueOr(p *float64, fallback float64) float64 {
	if p != nil {
		return *p
	}
	return fallback
}

// assemblyJointSetFlip sets a joint's flip sense and returns its info.
func assemblyJointSetFlip(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetJointFlipArgs) (wire.AssemblyJointResult, error) {
	if err := asm.Joints().SetFlip(in.ID, in.Flip); err != nil {
		return wire.AssemblyJointResult{}, err
	}
	asm.SolveConstraints()
	return jointResult(asm, in.ID), nil
}

// jointResult renders the joint with the given id from the active assembly.
func jointResult(asm *compdef.AssemblyComponentDefinition, id uint64) wire.AssemblyJointResult {
	return wire.AssemblyJointResult{Joint: jointInfo(asm.Joints().ByID(id))}
}

// dsJointsList returns the active assembly's DS-joint set.
func dsJointsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.DSJointsResult, error) {
	return dsJointsListResult(asm), nil
}

// dsJointsListResult renders the active assembly's DS-joint set.
func dsJointsListResult(asm *compdef.AssemblyComponentDefinition) wire.DSJointsResult {
	set := asm.DSJoints()
	out := make([]wire.DSJointInfo, 0, set.Count())
	for _, j := range set.All() {
		out = append(out, dsJointInfo(j))
	}
	return wire.DSJointsResult{Joints: out}
}

// dsJointAdd adds a DS joint of the requested kind between two origins.
func dsJointAdd(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddDSJointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	a, err := resolveConstraintRef(asm, in.A, wire.MethodDSJointsAdd)
	if err != nil {
		return nil, err
	}
	b, err := resolveConstraintRef(asm, in.B, wire.MethodDSJointsAdd)
	if err != nil {
		return nil, err
	}
	j := asm.DSJoints().Add(dsJointType(in.Type), a, b)
	return json.Marshal(wire.DSJointResult{Joint: dsJointInfo(j)})
}

// dsJointSetImposedMotion sets a DS joint DOF's imposed motion and value.
func dsJointSetImposedMotion(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetImposedMotionArgs) (wire.DSJointResult, error) {
	if err := asm.DSJoints().SetImposedMotion(in.ID, in.DOFIndex, imposedMotion(in.ImposedMotion), in.Value); err != nil {
		return wire.DSJointResult{}, err
	}
	return wire.DSJointResult{Joint: dsJointInfo(asm.DSJoints().ByID(in.ID))}, nil
}

// dsJointDelete removes a DS joint and returns the remaining set.
func dsJointDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.DeleteDSJointArgs) (wire.DSJointsResult, error) {
	if !asm.DSJoints().Delete(in.ID) {
		return wire.DSJointsResult{}, fmt.Errorf("%s: no DS joint with id %d", wire.MethodDSJointsDelete, in.ID)
	}
	return dsJointsListResult(asm), nil
}

// dsJointType maps a wire kind string to the DS joint enum.
func dsJointType(s string) types.DSJointType {
	switch s {
	case "rotational":
		return types.DSJointRotational
	case "prismatic":
		return types.DSJointPrismatic
	case "cylindrical":
		return types.DSJointCylindrical
	case "planar":
		return types.DSJointPlanar
	case "spherical":
		return types.DSJointSpherical
	default:
		return types.DSJointRigid
	}
}

// imposedMotion maps a wire imposed-motion string to the enum.
func imposedMotion(s string) types.DSDOFImposedMotionType {
	switch s {
	case "driven":
		return types.DSDOFDriven
	case "locked":
		return types.DSDOFLocked
	default:
		return types.DSDOFFree
	}
}
