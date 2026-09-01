// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The to-next revolve extent (Oblikovati#1860): stop the sweep at the first material it meets.
//
// Unlike to-face there is no named terminator to measure against, so the stop angle has to be
// FOUND. Each profile point travels a circle about the axis, and the feature stops at the smallest
// angle any of them crosses a face. The circle is marched in short chords and each chord is
// ray-cast at the running bodies — the same tessellated cast extrude's to-next uses (toNextSpan),
// so both features answer "next face" with the same fidelity and the same caveats.
//
// The angle is then read off the HIT POINT rather than the step index, so the result is not
// quantised to the march: a chord's hit sits a sagitta inside its arc, which at the step below is
// ~4e-5 of the sweep radius — far under any model tolerance (ADR-0042).

// revolveToNextStep is the chord length of the search march, in radians. One degree keeps the
// chord's sagitta at r·(1−cos½°) ≈ 3.8e-5·r while bounding a full-turn search at 360 casts per
// profile point.
const revolveToNextStep = stdmath.Pi / 180

// revolveToNextSpan sweeps until the profile first meets existing material, in the sense Direction
// names. Symmetric splits that sweep half each way, like the to-face extent.
func revolveToNextSpan(def *RevolveDefinition, base []math.Point3, axis *WorkAxis,
	bodies []*topo.Body) (float64, float64, error) {
	if len(bodies) == 0 {
		return 0, 0, errors.New("revolve: to-next needs existing material to stop on")
	}
	turn, err := revolveToNextTurn(base, axis, bodies, senseOf(def.Direction))
	if err != nil {
		return 0, 0, err
	}
	switch def.Direction {
	case NegativeDir:
		return turn, -turn, nil
	case SymmetricDir:
		return turn, -turn / 2, nil
	default:
		return turn, 0, nil
	}
}

// revolveToNextTurn is the smallest sweep angle at which ANY profile point reaches a face — the
// first point to arrive stops the whole feature, exactly as the nearest ray hit stops an extrude.
func revolveToNextTurn(base []math.Point3, axis *WorkAxis, bodies []*topo.Body, sense float64) (float64, error) {
	best := stdmath.Inf(1)
	for _, p := range base {
		if turn, ok := firstMaterialTurn(p, axis, bodies, sense); ok && turn < best {
			best = turn
		}
	}
	if stdmath.IsInf(best, 1) {
		return 0, errors.New("revolve: to-next found no material ahead of the profile")
	}
	return best, nil
}

// firstMaterialTurn marches one profile point's circular path and returns the sweep angle of the
// first face it crosses.
func firstMaterialTurn(p math.Point3, axis *WorkAxis, bodies []*topo.Body, sense float64) (float64, bool) {
	for k := range int(2 * stdmath.Pi / revolveToNextStep) {
		from := rotateAboutAxis(p, axis, sense*revolveToNextStep*float64(k))
		to := rotateAboutAxis(p, axis, sense*revolveToNextStep*float64(k+1))
		hit, ok := chordHit(from, to, bodies)
		if !ok {
			continue
		}
		return sweepAngleOf(p, hit, axis, sense), true
	}
	return 0, false
}

// chordHit casts one march chord at the running bodies and returns where it first crosses a face.
// A hit AT the chord's start is skipped: a profile sketched on a body face starts touching it, and
// stopping there would collapse the revolve to nothing.
func chordHit(from, to math.Point3, bodies []*topo.Body) (math.Point3, bool) {
	step := from.VectorTo(to)
	length := step.Length()
	if length <= math.DefaultTolerance {
		return math.Point3{}, false
	}
	dir := step.AsUnit().AsVector()
	for _, b := range bodies {
		_, t, ok := query.RayCastFaces(b, from, dir, ops.DefaultQuality())
		if ok && t > float64(math.DefaultTolerance) && t <= float64(length) {
			return from.TranslateBy(dir.Scale(math.Scalar(t))), true
		}
	}
	return math.Point3{}, false
}

// sweepAngleOf reads the sweep a point has travelled from where it landed, by comparing the radial
// direction of the hit with the radial direction the point started at.
func sweepAngleOf(start, hit math.Point3, axis *WorkAxis, sense float64) float64 {
	a := axis.Direction().AsVector()
	from := radialComponent(axis.Origin().VectorTo(start), a)
	to := radialComponent(axis.Origin().VectorTo(hit), a)
	return turnMagnitude(sense*signedTurnAbout(from, to, a), 0)
}

// rotateAboutAxis spins a point about the work axis by the given angle.
func rotateAboutAxis(p math.Point3, axis *WorkAxis, angle float64) math.Point3 {
	return math.Rotation4(angle, axis.Direction(), axis.Origin()).TransformPoint(p)
}
