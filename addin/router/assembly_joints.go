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
	r.readOnly(wire.MethodAssemblyJointsList, assemblyJointsList)
	r.readOnly(wire.MethodAssemblyJointsAddRigid, jointAdder((*assembly.JointSet).AddRigid))
	r.readOnly(wire.MethodAssemblyJointsAddRotational, jointAdder((*assembly.JointSet).AddRotational))
	r.readOnly(wire.MethodAssemblyJointsAddSlider, jointAdder((*assembly.JointSet).AddSlider))
	r.readOnly(wire.MethodAssemblyJointsAddCylindrical, jointAdder((*assembly.JointSet).AddCylindrical))
	r.readOnly(wire.MethodAssemblyJointsAddPlanar, jointAdder((*assembly.JointSet).AddPlanar))
	r.readOnly(wire.MethodAssemblyJointsAddBall, jointAdder((*assembly.JointSet).AddBall))
	r.readOnly(wire.MethodAssemblyJointsDelete, assemblyJointDelete)
	r.readOnly(wire.MethodAssemblyJointsSetLimits, assemblyJointSetLimits)
	r.readOnly(wire.MethodAssemblyJointsSetFlip, assemblyJointSetFlip)
	r.readOnly(wire.MethodDSJointsList, dsJointsList)
	r.readOnly(wire.MethodDSJointsAdd, dsJointAdd)
	r.readOnly(wire.MethodDSJointsSetImposedMotion, dsJointSetImposedMotion)
	r.readOnly(wire.MethodDSJointsDelete, dsJointDelete)
}

// assemblyJointsList returns the active assembly's joint set.
func assemblyJointsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	set := asm.Joints()
	out := make([]wire.JointInfo, 0, set.Count())
	for _, j := range set.All() {
		out = append(out, jointInfo(j))
	}
	return json.Marshal(wire.AssemblyJointsResult{Joints: out})
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
		asm.SolveConstraints()
		return json.Marshal(wire.AssemblyJointResult{Joint: jointInfo(j)})
	}
}

// assemblyJointDelete removes a joint, re-solves, and returns the remaining set.
func assemblyJointDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteJointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if !asm.Joints().Delete(in.ID) {
		return nil, fmt.Errorf("%s: no joint with id %d", wire.MethodAssemblyJointsDelete, in.ID)
	}
	asm.SolveConstraints()
	return assemblyJointsList(s, nil)
}

// assemblyJointSetLimits sets a joint's linear/angular bounds and returns its info.
func assemblyJointSetLimits(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetJointLimitsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	l := in.Limits
	lim := assembly.NewJointLimits(l.LinearMin, l.HasLinearMin, l.LinearMax, l.HasLinearMax, l.AngularMin, l.HasAngularMin, l.AngularMax, l.HasAngularMax)
	if err := asm.Joints().SetLimits(in.ID, lim); err != nil {
		return nil, err
	}
	return jointResult(asm, in.ID)
}

// assemblyJointSetFlip sets a joint's flip sense and returns its info.
func assemblyJointSetFlip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetJointFlipArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := asm.Joints().SetFlip(in.ID, in.Flip); err != nil {
		return nil, err
	}
	asm.SolveConstraints()
	return jointResult(asm, in.ID)
}

// jointResult marshals the joint with the given id from the active assembly.
func jointResult(asm *compdef.AssemblyComponentDefinition, id uint64) (json.RawMessage, error) {
	return json.Marshal(wire.AssemblyJointResult{Joint: jointInfo(asm.Joints().ByID(id))})
}

// dsJointsList returns the active assembly's DS-joint set.
func dsJointsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	set := asm.DSJoints()
	out := make([]wire.DSJointInfo, 0, set.Count())
	for _, j := range set.All() {
		out = append(out, dsJointInfo(j))
	}
	return json.Marshal(wire.DSJointsResult{Joints: out})
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
func dsJointSetImposedMotion(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetImposedMotionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := asm.DSJoints().SetImposedMotion(in.ID, in.DOFIndex, imposedMotion(in.ImposedMotion), in.Value); err != nil {
		return nil, err
	}
	return json.Marshal(wire.DSJointResult{Joint: dsJointInfo(asm.DSJoints().ByID(in.ID))})
}

// dsJointDelete removes a DS joint and returns the remaining set.
func dsJointDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeleteDSJointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if !asm.DSJoints().Delete(in.ID) {
		return nil, fmt.Errorf("%s: no DS joint with id %d", wire.MethodDSJointsDelete, in.ID)
	}
	return dsJointsList(s, nil)
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
