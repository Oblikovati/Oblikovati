// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Terminating a revolve on model geometry instead of a literal angle (Oblikovati#1860) — the
// revolve half of PartFeatureExtentEnum, matching what extrude has carried since #1859.
//
// The whole family reduces to ONE question: through what angle about the axis does the profile
// sweep before it reaches the terminator? A revolve is a rotation, so the answer is an angle, and
// the rest of the feature (revolveSectionsFrom, buildRevolveSolid) is unchanged — the extents only
// decide the (total, start) pair that revolveSpan otherwise reads off the definition's angles.
//
// Measuring that angle needs a fixed "angle zero": the direction, perpendicular to the axis, from
// the axis to the profile's centroid. A to-face / from-to terminator is then the half-plane the
// sweep stops on, which is why such a target must CONTAIN the axis. A target that merely crosses it
// is refused rather than approximated: the swept solid would meet it at a DIFFERENT angle for every
// profile point, so no single stop angle exists and the result would be a silently wrong wedge.
// (Extrude refuses a non-parallel to-face target for the same reason.)

// revolveAngleTol is the angular resolution of an extent stop: below it two half-planes are the
// same half-plane. It is scale-free — an angle carries no length — so it needs no model-size term.
const revolveAngleTol = 1e-9

// revolveExtentSpan resolves the swept (total, start) of a revolve that terminates on geometry.
// base is the profile's model-space outline, which supplies both the angle zero and, for to-next,
// the points whose circular paths are marched against the running bodies.
func revolveExtentSpan(def *RevolveDefinition, base []math.Point3, axis *WorkAxis,
	bodies []*topo.Body) (total, start float64, err error) {
	if def.Extent == ToNextExtent {
		return revolveToNextSpan(def, base, axis, bodies)
	}
	zero, radius, err := revolveProfileDir(base, axis)
	if err != nil {
		return 0, 0, err
	}
	switch def.Extent {
	case ToFaceExtent:
		return revolveToFaceSpan(def, zero, radius, axis)
	case FromToExtent:
		return revolveFromToSpan(def, zero, radius, axis)
	default:
		return 0, 0, fmt.Errorf("revolve: unsupported extent type %d (want angle, to-face, from-to or to-next)", def.Extent)
	}
}

// revolveProfileDir is a revolve's angle zero: the unit direction, perpendicular to the axis, from
// the axis to the profile's centroid. It also returns that centroid's distance from the axis — the
// sweep radius, which is this feature's model scale and so sets the tolerances (ADR-0042). A
// profile centred ON the axis has no such direction, and a revolve of it is degenerate anyway.
func revolveProfileDir(base []math.Point3, axis *WorkAxis) (math.Vector3, float64, error) {
	if len(base) == 0 {
		return math.Vector3{}, 0, errors.New("revolve: the profile has no outline to measure an extent from")
	}
	radial := radialComponent(axis.Origin().VectorTo(centroidOfPoints(base)), axis.Direction().AsVector())
	if radial.Length() <= math.DefaultTolerance {
		return math.Vector3{}, 0, errors.New("revolve: the profile is centred on the axis, so an extent has no sweep angle to measure")
	}
	return radial.AsUnit().AsVector(), float64(radial.Length()), nil
}

// centroidOfPoints averages an outline's points — enough to name the side of the axis the profile
// lies on, which is all the angle zero needs.
func centroidOfPoints(pts []math.Point3) math.Point3 {
	var sum math.Vector3
	for _, p := range pts {
		sum = sum.Add(p.AsVector())
	}
	return sum.Scale(1 / math.Scalar(len(pts))).AsPoint()
}

// radialComponent drops v's component along the (unit) axis, leaving the purely radial part.
func radialComponent(v, axis math.Vector3) math.Vector3 {
	return v.Sub(axis.Scale(v.Dot(axis)))
}

// revolveToFaceSpan sweeps from the profile until it first reaches the terminator half-plane, in
// the sense Direction names. Symmetric splits that same sweep half each way, so the total stays the
// measured angle — the revolve counterpart of a symmetric distance extrude.
func revolveToFaceSpan(def *RevolveDefinition, zero math.Vector3, radius float64,
	axis *WorkAxis) (float64, float64, error) {
	stop, err := radialHalfPlaneDir(def.ToPlane, axis, radius, "to-face")
	if err != nil {
		return 0, 0, err
	}
	turn := firstTurnTo(zero, stop, axis.Direction().AsVector(), senseOf(def.Direction))
	switch def.Direction {
	case NegativeDir:
		return turn, -turn, nil
	case SymmetricDir:
		return turn, -turn / 2, nil
	default:
		return turn, 0, nil
	}
}

// revolveFromToSpan bounds the wedge by two radial terminators: it sweeps BACKWARDS from the
// profile to the "from" half-plane and FORWARDS to the "to" half-plane, so the wedge always
// contains the profile that generated it. A pair that meets round the back is a full revolution.
func revolveFromToSpan(def *RevolveDefinition, zero math.Vector3, radius float64,
	axis *WorkAxis) (float64, float64, error) {
	from, err := radialHalfPlaneDir(def.FromPlane, axis, radius, "from-to start")
	if err != nil {
		return 0, 0, err
	}
	to, err := radialHalfPlaneDir(def.ToPlane, axis, radius, "from-to end")
	if err != nil {
		return 0, 0, err
	}
	a := axis.Direction().AsVector()
	back, fwd := firstTurnTo(zero, from, a, -1), firstTurnTo(zero, to, a, 1)
	if total := back + fwd; total < 2*stdmath.Pi-revolveAngleTol {
		return total, -back, nil
	}
	return 0, 0, nil // the two terminators meet round the back: the wedge closes into a full revolution
}

// radialHalfPlaneDir is the terminator's stop direction: the direction, perpendicular to the axis,
// that lies IN the target plane. The target must contain the axis (see the file comment); a nil
// target is an unresolved geometric selector, reported as feature health rather than a panic.
func radialHalfPlaneDir(target *WorkPlane, axis *WorkAxis, radius float64, what string) (math.Vector3, error) {
	if target == nil {
		return math.Vector3{}, fmt.Errorf("revolve: %s target face was not found on the current body", what)
	}
	a, n := axis.Direction().AsVector(), target.Plane().Normal().AsVector()
	off := float64(target.Plane().Origin().VectorTo(axis.Origin()).Dot(n))
	tol := geom.ResolutionForSize(radius).Plane()
	if !n.IsPerpendicularTo(a, math.DefaultTolerance) || off > tol || off < -tol {
		return math.Vector3{}, fmt.Errorf("revolve: %s target must CONTAIN the revolve axis (a radial face); "+
			"a target that merely crosses the axis meets the sweep at a different angle for every profile point", what)
	}
	return a.Cross(n).AsUnit().AsVector(), nil
}

// firstTurnTo is the smallest rotation, in the given sense, carrying the angle zero onto EITHER
// half-plane of the terminator — a plane has two, and the sweep stops at whichever it reaches
// first. A terminator the profile already lies in is a full turn, not a zero-volume revolve.
func firstTurnTo(zero, stop, axisDir math.Vector3, sense float64) float64 {
	raw := signedTurnAbout(zero, stop, axisDir)
	turn := turnMagnitude(sense*raw, 0)
	if other := turnMagnitude(sense*(raw+stdmath.Pi), 0); other < turn {
		turn = other
	}
	if turn <= revolveAngleTol {
		return 2 * stdmath.Pi
	}
	return turn
}

// signedTurnAbout returns the rotation about axisDir that carries from onto to, in (-π, π].
func signedTurnAbout(from, to, axisDir math.Vector3) float64 {
	return stdmath.Atan2(float64(from.Cross(to).Dot(axisDir)), float64(from.Dot(to)))
}

// turnMagnitude wraps a signed rotation into [floor, floor+2π) — the sweep needed to reach it.
func turnMagnitude(a, floor float64) float64 {
	t := stdmath.Mod(a-floor, 2*stdmath.Pi)
	if t < 0 {
		t += 2 * stdmath.Pi
	}
	return t + floor
}

// senseOf maps an extent direction to the rotational sense the extent searches in. Symmetric
// searches forwards and then splits the result, so it shares the positive sense.
func senseOf(dir ExtentDirection) float64 {
	if dir == NegativeDir {
		return -1
	}
	return 1
}
