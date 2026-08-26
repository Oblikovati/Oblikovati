// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// Drive (M12-F03, Oblikovati/Oblikovati#366) sweeps a joint's driven variable through a value
// range, re-solving the assembly (constraints and joints together, ADR-0011) with the
// variable pinned at each step, and returns the resulting per-step occurrence placements —
// the frames of a kinematic motion study. With collision detection the sweep halts at the
// first frame where a moved component interferes with another. The drive is non-destructive:
// the assembly is restored to its pre-drive placements before returning.

// OccurrencePlacement is one occurrence's transform at a drive frame.
type OccurrencePlacement struct {
	Occurrence uint64
	Transform  math.Matrix4
}

// DriveFrame is one step of a drive: the driven value, the resulting placements of the moved
// occurrences, and whether interference was detected at this frame.
type DriveFrame struct {
	Value      float64
	Placements []OccurrencePlacement
	Collided   bool
}

// DriveResult is a drive's frames in play order, plus — when collision detection halted the
// sweep — the flag and the index of the last (collided) frame.
type DriveResult struct {
	Frames             []DriveFrame
	StoppedByCollision bool
	StoppedAtStep      int
}

// DriveJoint sweeps the joint with jointID through s and returns the frames. It errors when
// the joint is unknown or its kind/variable pairing is not drivable.
func DriveJoint(occs *occurrence.Occurrences, cs *ConstraintSet, js *JointSet, jointID uint64, s DriveSettings) (DriveResult, error) {
	joint, ok := js.ByID(jointID).(*assemblyJoint)
	if !ok {
		return DriveResult{}, fmt.Errorf("assembly: no joint with id %d to drive", jointID)
	}
	resolved, ok := drivableVariable(joint.kind, s.variable)
	if !ok {
		return DriveResult{}, fmt.Errorf("assembly: joint %d (%s) is not drivable for variable %s", jointID, joint.kind, s.variable)
	}
	return sweep(occs, cs, js, joint, resolved, s), nil
}

// sweep runs the value sequence, restoring the assembly afterwards. Each step pins the driven
// joint — and any gears it is geared to (couplingPins, #883) — onto a fresh copy of the
// assembly's constraints+joints, so the base set is never mutated.
func sweep(occs *occurrence.Occurrences, cs *ConstraintSet, js *JointSet, joint *assemblyJoint, resolved types.DriveVariable, s DriveSettings) DriveResult {
	base := combinedRelationships(cs, js)
	saved := snapshotPlacements(occs)
	defer restorePlacements(occs, saved)

	var res DriveResult
	for i, v := range driveValues(s) {
		pins := couplingPins(joint, resolved, v, cs, js)
		solveOver(occs, append(append([]relationship{}, base...), pins...), true)
		frame := DriveFrame{Value: v, Placements: capturePlacements(occs)}
		if s.collision && occurrencesInterfere(occs) {
			frame.Collided = true
			res.Frames = append(res.Frames, frame)
			res.StoppedByCollision, res.StoppedAtStep = true, i
			return res
		}
		res.Frames = append(res.Frames, frame)
	}
	return res
}

// driveValues expands the settings into the ordered value sequence: a start→end ramp by step,
// repeated RepetitionCount times, each odd pass reversed when ping-pong is set.
func driveValues(s DriveSettings) []float64 {
	forward := rampValues(s.start, s.end, s.step)
	reps := max(s.reps, 1)
	var out []float64
	for r := range reps {
		if s.pingPong && r%2 == 1 {
			out = append(out, reversed(forward)...)
		} else {
			out = append(out, forward...)
		}
	}
	return out
}

// rampValues returns start, start±step, … up to end (inclusive), stepping toward end. A
// non-positive step degenerates to the two endpoints so a drive always has frames.
func rampValues(start, end, step float64) []float64 {
	if step <= 0 {
		return []float64{start, end}
	}
	dir := 1.0
	if end < start {
		dir = -1.0
	}
	out := []float64{}
	for v := start; (v-end)*dir < step*0.5; v += step * dir {
		out = append(out, v)
	}
	if len(out) == 0 || out[len(out)-1] != end {
		out = append(out, end)
	}
	return out
}

// reversed returns a new slice with vs in reverse order.
func reversed(vs []float64) []float64 {
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[len(vs)-1-i] = v
	}
	return out
}

// snapshotPlacements records every non-suppressed occurrence's transform so the drive can be
// undone.
func snapshotPlacements(occs *occurrence.Occurrences) map[uint64]math.Matrix4 {
	saved := map[uint64]math.Matrix4{}
	for _, o := range occs.All() {
		if !o.Suppressed() {
			saved[o.ID()] = o.Transform()
		}
	}
	return saved
}

// restorePlacements writes the saved transforms back in one notification batch.
func restorePlacements(occs *occurrence.Occurrences, saved map[uint64]math.Matrix4) {
	occs.SuspendNotifications()
	defer occs.ResumeNotifications()
	for _, o := range occs.All() {
		if m, ok := saved[o.ID()]; ok {
			o.SetTransform(m)
		}
	}
}

// capturePlacements records the transforms of the occurrences a drive can move (non-grounded,
// non-suppressed) — the per-frame delta a consumer animates over the static grounded set.
func capturePlacements(occs *occurrence.Occurrences) []OccurrencePlacement {
	var out []OccurrencePlacement
	for _, o := range occs.All() {
		if o.Suppressed() || o.Grounded() {
			continue
		}
		out = append(out, OccurrencePlacement{Occurrence: o.ID(), Transform: o.Transform()})
	}
	return out
}
