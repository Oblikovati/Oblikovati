// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/solve"
)

// Joints (M12-F02) are reduced-DOF parameterizations on the SAME solver as constraints
// (ADR-0011): each joint type emits a bundle of the F01 residual helpers that leaves exactly
// the joint's degrees of freedom, so the DOF + over/under reporting fall out of the existing
// rank analysis. A Joint is therefore just a typed relationship whose bind() dispatches on
// its kind; constraints and joints solve together (relationships.go).

// Joint is the internal interface every assembly joint implements: the public read surface
// (contract.AssemblyJoint) plus the solver-binding and mutation hooks the JointSet drives.
type Joint interface {
	contract.AssemblyJoint
	SetSuppressed(bool)
	AnchorRefs() (AnchorRef, AnchorRef)
	anchors() []anchor
	bind(b binder) []solve.Residual
	setHealth(s health.Status)
	setLimits(l *jointLimits)
	setFlip(flip bool)
	setGap(g float64)
	setPositions(linear, angular float64)
	setLocked(v bool)
	setProtected(v bool)
}

// jointBase adds the joint kind, flip sense, linear/angular limits, and the joint's seating and
// state (#1970/#1974) to the shared relationship base.
type jointBase struct {
	relationshipBase
	kind types.AssemblyJointType
	flip bool
	lim  *jointLimits
	// gap seats the two joint origins this far apart along the joint Z-axis (#1970); linearPosition
	// and angularPosition are where the joint currently rests along its free DOF.
	gap             float64
	linearPosition  float64
	angularPosition float64
	// locked freezes the joint's remaining free DOF (removed from the DOF report without
	// suppressing); protected marks the DOF as protected from being consumed by other
	// relationships (#1974).
	locked    bool
	protected bool
}

// Type returns the joint kind.
func (j *jointBase) Type() types.AssemblyJointType { return j.kind }

// Flip reports the reversed primary-axis sense.
func (j *jointBase) Flip() bool { return j.flip }

// DegreesOfFreedom is the joint kind's nominal free DOF (rigid 0 … ball 3), or 0 when the joint is
// locked — its remaining freedom is frozen out of the report without suppressing it (#1974).
func (j *jointBase) DegreesOfFreedom() int {
	if j.locked {
		return 0
	}
	return j.kind.DegreesOfFreedom()
}

// Gap is the axial seating between the two joint origins (#1970).
func (j *jointBase) Gap() float64 { return j.gap }

// setGap sets the joint's axial seating.
func (j *jointBase) setGap(g float64) { j.gap = g }

// LinearPosition / AngularPosition are where the joint currently rests along its free DOF (#1970).
func (j *jointBase) LinearPosition() float64  { return j.linearPosition }
func (j *jointBase) AngularPosition() float64 { return j.angularPosition }

// setPositions sets the joint's current linear and angular rest positions.
func (j *jointBase) setPositions(linear, angular float64) {
	j.linearPosition, j.angularPosition = linear, angular
}

// Locked reports whether the joint's free DOF is frozen (#1974).
func (j *jointBase) Locked() bool { return j.locked }

// setLocked freezes or releases the joint's free DOF.
func (j *jointBase) setLocked(v bool) { j.locked = v }

// Protected reports whether the joint's DOF is protected from other relationships (#1974).
func (j *jointBase) Protected() bool { return j.protected }

// setProtected marks the joint's DOF protected or not.
func (j *jointBase) setProtected(v bool) { j.protected = v }

// Limits returns the joint's driven-value bounds, or a nil interface when unbounded.
func (j *jointBase) Limits() contract.JointLimits {
	if j.lim == nil {
		return nil
	}
	return j.lim
}

// setLimits sets or clears the joint's bounds.
func (j *jointBase) setLimits(l *jointLimits) { j.lim = l }

// setFlip sets the joint's flip sense.
func (j *jointBase) setFlip(flip bool) { j.flip = flip }

// jointDefinition is the read surface of a joint's definition — its kind and the
// origin-definition kinds of its two joint origins.
type jointDefinition struct {
	kind             types.AssemblyJointType
	aOrigin, bOrigin types.AssemblyJointOriginDefinitionType
}

// JointType is the kind of joint this definition builds.
func (d jointDefinition) JointType() types.AssemblyJointType { return d.kind }

// OriginTypes returns how each of the two joint origins is defined.
func (d jointDefinition) OriginTypes() (a, b types.AssemblyJointOriginDefinitionType) {
	return d.aOrigin, d.bOrigin
}

// jointProxy views a joint in the context of a specific occurrence path (the reference API's
// per-instance joint proxy). For F02 it is a thin wrapper carrying the native joint.
type jointProxy struct{ Joint }

// NativeJoint returns the underlying joint this proxy views.
func (p jointProxy) NativeJoint() contract.AssemblyJoint { return p.Joint }

// assemblyJoint is the one joint implementation; its residual bundle is selected by kind.
type assemblyJoint struct {
	*jointBase
}

// bind returns the joint's residual bundle over the two joint-origin placements.
func (j *assemblyJoint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := j.boundPlacements(b)
		return jointResiduals(j.kind, j.a.prim, j.b.prim, pa.matrix(), pb.matrix(), j.flip, j.gap)
	})
}

// jointResiduals selects the residual bundle that leaves a joint of the given kind its
// degrees of freedom (table in the F02 plan).
func jointResiduals(kind types.AssemblyJointType, a, b Primitive, ma, mb math.Matrix4, flip bool, gap float64) []float64 {
	switch kind {
	case types.JointRigid:
		return rigidJointResiduals(a, b, ma, mb)
	case types.JointRotational:
		return collinearAxisResiduals(a, b, ma, mb, flip, true, gap) // + axial gap
	case types.JointSlider:
		return sliderJointResiduals(a, b, ma, mb, flip)
	case types.JointCylindrical:
		return collinearAxisResiduals(a, b, ma, mb, flip, false, gap) // axes only
	case types.JointPlanar:
		return planarJointResiduals(a, b, ma, mb, flip, gap)
	case types.JointBall:
		return pointMateResiduals(a, b, ma, mb)
	default:
		return nil
	}
}

// jointAxisTarget returns B's primary axis, reversed when the joint is flipped.
func jointAxisTarget(mb math.Matrix4, b Primitive, flip bool) math.Vector3 {
	d := worldDir(mb, b)
	if flip {
		return d.Scale(-1)
	}
	return d
}

// collinearAxisResiduals makes the two joint axes collinear (align 2 + perpendicular 2);
// with axialGap it also fixes the axial position (rotational, 1 DOF), without it leaves the
// slide free (cylindrical, 2 DOF).
func collinearAxisResiduals(a, b Primitive, ma, mb math.Matrix4, flip, axialGap bool, gap float64) []float64 {
	res := alignResiduals(worldDir(ma, a), jointAxisTarget(mb, b, flip))
	res = append(res, perpDistanceResiduals(worldPoint(ma, a), worldPoint(mb, b), worldDir(mb, b))...)
	if axialGap {
		res = append(res, gapResidual(worldPoint(ma, a), worldPoint(mb, b), worldDir(mb, b), gap))
	}
	return res
}

// sliderJointResiduals makes the axes collinear and locks the roll, leaving one translation.
func sliderJointResiduals(a, b Primitive, ma, mb math.Matrix4, flip bool) []float64 {
	res := collinearAxisResiduals(a, b, ma, mb, flip, false, 0)
	if roll, ok := rollLockResidual(a, b, ma, mb); ok {
		res = append(res, roll)
	}
	return res
}

// rigidJointResiduals fully fix the two frames: point coincidence + axis alignment + roll
// lock (0 DOF).
func rigidJointResiduals(a, b Primitive, ma, mb math.Matrix4) []float64 {
	res := pointMateResiduals(a, b, ma, mb)
	res = append(res, alignResiduals(worldDir(ma, a), worldDir(mb, b))...)
	if roll, ok := rollLockResidual(a, b, ma, mb); ok {
		res = append(res, roll)
	}
	return res
}

// planarJointResiduals make the two planes coincident (align 2 + gap 1), leaving two
// in-plane translations and a rotation about the normal.
func planarJointResiduals(a, b Primitive, ma, mb math.Matrix4, flip bool, gap float64) []float64 {
	res := alignResiduals(worldDir(ma, a), jointAxisTarget(mb, b, flip))
	return append(res, gapResidual(worldPoint(ma, a), worldPoint(mb, b), worldDir(mb, b), gap))
}
