// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// The thread table wire surface (M09-F01 PBI-101, #325): threads.tableQuery
// enumerates the thread tables progressively and threads.resolve turns a
// designation into the thread data the thread feature, hole tapping (#326),
// and drawings (M14) consume — one source of truth, served from the same
// tables [feature.ParseThreadDesignation] resolves against.

// threadsTableQuery lists thread types always; a type's nominal sizes, a
// size's designations, and a designation's classes as each filter is given.
func threadsTableQuery(_ *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ThreadTableQueryArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, err
		}
	}
	out := wire.ThreadTableQueryResult{ThreadTypes: feature.ThreadTypes()}
	if err := fillThreadLevels(&out, in); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// fillThreadLevels resolves the size/designation/class levels the filters unlock. The
// class level derives its table from the designation itself, so it works without
// threadType.
func fillThreadLevels(out *wire.ThreadTableQueryResult, in wire.ThreadTableQueryArgs) error {
	var err error
	if in.ThreadType != "" {
		if out.NominalSizes, err = feature.ThreadNominalSizes(in.ThreadType); err != nil {
			return err
		}
		if in.NominalSize != "" {
			if out.Designations, err = feature.ThreadDesignationsOf(in.ThreadType, in.NominalSize); err != nil {
				return err
			}
		}
	}
	if in.Designation == "" {
		return nil
	}
	tt, err := feature.ThreadTypeOfDesignation(in.Designation)
	if err != nil {
		return err
	}
	out.Classes, err = feature.ThreadClasses(tt, in.Internal)
	return err
}

// threadsResolve resolves a designation (+class/handedness/tapered) to its thread data.
func threadsResolve(_ *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ResolveThreadArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if in.Designation == "" {
		return nil, errors.New("threads.resolve: designation is required")
	}
	spec, err := feature.ParseThreadDesignation(in.Designation)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ThreadInfoResult{
		Designation: spec.Designation, ThreadType: spec.ThreadType, NominalSize: spec.NominalSize,
		Class: in.Class, Metric: spec.Metric, Internal: in.Internal,
		RightHanded: spec.RightHanded && !in.LeftHanded, Tapered: in.Tapered,
		Pitch: spec.Pitch, MajorDiameter: spec.MajorDiameter, MinorDiameter: spec.MinorDiameter,
		PitchDiameter: spec.PitchDiameter, TapDrillDiameter: spec.TapDrillDiameter,
	})
}
