// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
)

// The assembly drive surface (M12-F03, #366): sweep a joint's driven variable through a range
// and return the per-step occurrence placements — the frames of a kinematic motion study.
// Each step re-solves the active assembly with the variable pinned; with collision detection
// the sweep halts at the first interfering frame. The drive is non-destructive (the engine
// restores the assembly), so preview leaves the model unchanged.

// registerAssemblyDriveHandlers wires the assemblyDrive.* methods.
func (r *Router) registerAssemblyDriveHandlers() {
	r.readOnly(wire.MethodAssemblyDrivePreview, typedAssembly(assemblyDrivePreview))
}

// assemblyDrivePreview drives the requested joint through the given settings and returns the
// resulting frames.
func assemblyDrivePreview(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.DriveJointArgs) (wire.DriveResult, error) {
	res, err := asm.DriveJoint(in.Joint, driveSettingsFromWire(in.Settings))
	if err != nil {
		return wire.DriveResult{}, err
	}
	return driveResultToWire(res), nil
}

// driveSettingsFromWire builds the engine's drive settings from the wire DTO.
func driveSettingsFromWire(d wire.DriveSettingsDTO) assembly.DriveSettings {
	return assembly.NewDriveSettings(driveVariable(d.Variable), d.Start, d.End, d.Step,
		d.RepetitionCount, d.RepetitionStartEndStart, d.CollisionDetection)
}

// driveVariable maps a wire variable string to the enum (empty/unknown ⇒ the joint's natural
// variable).
func driveVariable(v string) types.DriveVariable {
	switch v {
	case "angular":
		return types.DriveAngular
	case "linear":
		return types.DriveLinear
	default:
		return types.DriveNatural
	}
}

// driveResultToWire encodes the engine's drive frames into the wire result.
func driveResultToWire(res assembly.DriveResult) wire.DriveResult {
	frames := make([]wire.DriveFrame, 0, len(res.Frames))
	for _, f := range res.Frames {
		frames = append(frames, wire.DriveFrame{Value: f.Value, Collided: f.Collided, Placements: drivePlacements(f.Placements)})
	}
	return wire.DriveResult{Frames: frames, StoppedByCollision: res.StoppedByCollision, StoppedAtStep: res.StoppedAtStep}
}

// drivePlacements encodes one frame's occurrence transforms into the wire form.
func drivePlacements(ps []assembly.OccurrencePlacement) []wire.DrivePlacement {
	out := make([]wire.DrivePlacement, 0, len(ps))
	for _, p := range ps {
		out = append(out, wire.DrivePlacement{Occurrence: p.Occurrence, Transform: types.Matrix{Cells: p.Transform.Cells()}})
	}
	return out
}
