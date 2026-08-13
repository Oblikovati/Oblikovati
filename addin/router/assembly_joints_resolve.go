// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/assembly"
)

// Joint-router plumbing: render a joint / DS joint into its wire shape and build the engine's
// joint limits from the wire bounds. Kept apart from the handlers so each stays a few lines.

// jointInfo renders a joint into its wire shape.
func jointInfo(j assembly.Joint) wire.JointInfo {
	if j == nil {
		return wire.JointInfo{}
	}
	a, b := j.AnchorRefs()
	aMode, bMode := j.OriginModes()
	ax, ay, bx, by := j.OriginOffsets()
	return wire.JointInfo{
		ID:               j.ID(),
		Type:             j.Type().String(),
		Name:             j.Name(),
		A:                wire.ConstraintGeomRef{Occurrence: a.Occurrence, Entity: a.Entity},
		B:                wire.ConstraintGeomRef{Occurrence: b.Occurrence, Entity: b.Entity},
		Flip:             j.Flip(),
		DegreesOfFreedom: j.DegreesOfFreedom(),
		Limits:           jointLimitsInfo(j.Limits()),
		Health:           j.Health().String(),
		Suppressed:       j.Suppressed(),
		Gap:              j.Gap(),
		LinearPosition:   j.LinearPosition(),
		AngularPosition:  j.AngularPosition(),
		Locked:           j.Locked(),
		Protected:        j.Protected(),
		OriginOneMode:    originModeSpelling(aMode),
		OriginTwoMode:    originModeSpelling(bMode),
		OriginOneXOffset: ax, OriginOneYOffset: ay,
		OriginTwoXOffset: bx, OriginTwoYOffset: by,
	}
}

// originModeSpelling is a joint origin mode's wire spelling, or "" for the default (infer) so the
// field is omitted on the common case (#1973).
func originModeSpelling(m types.AssemblyJointOriginMode) string {
	if m == types.JointOriginInfer {
		return ""
	}
	return m.String()
}

// jointLimitsInfo renders a joint's limits, or nil when unbounded.
func jointLimitsInfo(l contract.JointLimits) *wire.JointLimits {
	if l == nil {
		return nil
	}
	out := &wire.JointLimits{}
	if v, ok := l.LinearMinimum(); ok {
		out.HasLinearMin, out.LinearMin = true, v
	}
	if v, ok := l.LinearMaximum(); ok {
		out.HasLinearMax, out.LinearMax = true, v
	}
	if v, ok := l.AngularMinimum(); ok {
		out.HasAngularMin, out.AngularMin = true, v
	}
	if v, ok := l.AngularMaximum(); ok {
		out.HasAngularMax, out.AngularMax = true, v
	}
	return out
}

// dsJointInfo renders a DS joint into its wire shape.
func dsJointInfo(j contract.DSJoint) wire.DSJointInfo {
	if j == nil {
		return wire.DSJointInfo{}
	}
	dofs := make([]wire.DSDOFInfo, 0, j.DOFCount())
	for i := 0; i < j.DOFCount(); i++ {
		d := j.DOF(i)
		dofs = append(dofs, wire.DSDOFInfo{
			Rotational:    d.Rotational(),
			ImposedMotion: d.ImposedMotion().String(),
			Value:         d.Value(),
		})
	}
	return wire.DSJointInfo{ID: j.ID(), Type: j.Type().String(), Name: j.Name(), DegreesOfFreedom: dofs}
}
