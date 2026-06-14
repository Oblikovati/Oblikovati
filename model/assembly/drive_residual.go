// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/solve"
)

// Driving a joint (M12-F03) reuses the ONE solver (ADR-0011): a drive step adds a single
// extra residual that pins the joint's otherwise-free driven variable to the step value, so
// the assembly re-solves to that pose. The joint residual bundle already leaves exactly the
// driven DOF (F02); the pin removes it. A drivenPin is therefore a relationship that solves
// together with the constraints and joints over the same placement set.

// drivenPin pins a joint's resolved driven variable (angular or linear) to value — the extra
// residual a drive step adds on top of the assembly's constraints and joints.
type drivenPin struct {
	joint    *assemblyJoint
	resolved types.DriveVariable // already angular or linear (never natural)
	value    float64
}

// Suppressed is always false: a pin only exists for the duration of one drive step.
func (p *drivenPin) Suppressed() bool { return false }

// anchors are the driven joint's two anchors, so the pin resolves exactly when the joint does.
func (p *drivenPin) anchors() []anchor { return p.joint.anchors() }

// setHealth is a no-op: the pin is transient and carries no reportable health.
func (p *drivenPin) setHealth(health.Status) {}

// bind emits the single pin residual over the joint's two origin placements.
func (p *drivenPin) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := p.joint.boundPlacements(b)
		r, ok := drivenResidual(p.resolved, p.joint.a.prim, p.joint.b.prim, pa.matrix(), pb.matrix(), p.value)
		if !ok {
			return nil
		}
		return []float64{r}
	})
}

// drivableVariable resolves a requested drive variable against a joint kind: it maps
// DriveNatural to the kind's natural variable and validates an explicit choice, returning the
// concrete (angular|linear) variable and whether the pairing is drivable. Rigid (0 DOF) and
// the multi-DOF planar/ball joints have no single natural driven scalar and are not drivable.
func drivableVariable(kind types.AssemblyJointType, requested types.DriveVariable) (types.DriveVariable, bool) {
	switch kind {
	case types.JointRotational:
		return resolveAgainst(requested, types.DriveAngular, false)
	case types.JointSlider:
		return resolveAgainst(requested, types.DriveLinear, false)
	case types.JointCylindrical:
		return resolveAgainst(requested, types.DriveAngular, true) // both variables allowed
	default:
		return types.DriveNatural, false
	}
}

// resolveAgainst maps natural→preferred and accepts an explicit variable when it is the
// preferred one (or, when bothAllowed, the other concrete one).
func resolveAgainst(requested, preferred types.DriveVariable, bothAllowed bool) (types.DriveVariable, bool) {
	switch requested {
	case types.DriveNatural:
		return preferred, true
	case types.DriveAngular, types.DriveLinear:
		if requested == preferred || bothAllowed {
			return requested, true
		}
	}
	return types.DriveNatural, false
}

// drivenResidual returns the single residual that pins the resolved variable to value: for an
// angular drive the relative roll about the joint axis equals value; for a linear drive the
// axial position equals value. ok is false when the geometry lacks what the pin needs.
func drivenResidual(resolved types.DriveVariable, a, b Primitive, ma, mb math.Matrix4, value float64) (float64, bool) {
	switch resolved {
	case types.DriveAngular:
		return angularPinResidual(a, b, ma, mb, value)
	case types.DriveLinear:
		return gapResidual(worldPoint(ma, a), worldPoint(mb, b), worldDir(mb, b), value), true
	default:
		return 0, false
	}
}

// angularPinResidual pins B's roll about the joint axis to angle radians from A's roll
// reference: rotate A's part-tied reference by angle about the axis, then measure the signed
// roll to B's reference (zero when B sits at angle). ok is false only when the axis degenerates.
func angularPinResidual(a, b Primitive, ma, mb math.Matrix4, angle float64) (float64, bool) {
	axisV := worldDir(mb, b)
	axis, err := math.UnitVector3FromVector(axisV)
	if err != nil {
		return 0, false
	}
	target := math.QuaternionFromAxisAngle(axis, angle).Matrix4().TransformVector(worldRollRef(ma, a))
	return target.Cross(worldRollRef(mb, b)).Dot(axisV), true
}

// worldRollRef returns a part-tied direction perpendicular to the joint axis, used to measure
// the relative roll: the primitive's secondary axis when it carries one (consistent with
// rollLockResidual), else a deterministic perpendicular derived from the axis. Both are mapped
// through the occurrence frame m, so the reference rotates with the part.
func worldRollRef(m math.Matrix4, prim Primitive) math.Vector3 {
	if prim.hasSecondary {
		return worldSecondary(m, prim)
	}
	u, _ := tangentFrame(prim.dir.AsVector())
	return m.TransformVector(u)
}
