// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/api/types"

// Driving a joint must also turn the gears it is geared to: a rotate-rotate constraint is a
// pure ratio that contributes no static residual (motion.go), so the static re-solve alone
// leaves the coupled gear behind. couplingPins fixes that (#883) by adding, for each
// rotate-rotate relationship reachable from the driven gear through the gear graph, a pin on the
// coupled gear's own rotational joint at the ratioed angle — so a drive of one gear propagates
// through the whole train.

// rateCoupling is a rotate-rotate gear relationship reduced to its two occurrences and ratio
// (revolutions of B per revolution of A).
type rateCoupling struct {
	a, b  uint64
	ratio float64
}

// partner returns the occurrence coupled to occ and the angle it turns through (occ turns val;
// its partner turns val·ratio when occ is A, val/ratio when occ is B), or ok=false when the
// coupling does not touch occ.
func (c rateCoupling) partner(occ uint64, val float64) (uint64, float64, bool) {
	switch occ {
	case c.a:
		return c.b, val * c.ratio, true
	case c.b:
		if c.ratio == 0 {
			return 0, 0, false
		}
		return c.a, val / c.ratio, true
	}
	return 0, 0, false
}

// couplingPins returns the drive pins for one step: the driven joint pinned to v, plus — for
// every gear reachable from it through rotate-rotate constraints — a pin on that gear's own
// rotational joint at the ratioed angle, so a geared train is driven as a whole, not just the
// one joint (#883). Without any rotate-rotate constraint it is exactly the single driven pin.
func couplingPins(driver *assemblyJoint, resolved types.DriveVariable, v float64, cs *ConstraintSet, js *JointSet) []relationship {
	pins := []relationship{&drivenPin{joint: driver, resolved: resolved, value: v}}
	couplings := rateCouplings(cs)
	if len(couplings) == 0 {
		return pins
	}
	return append(pins, gearTrainPins(movingOccID(driver), v, couplings, rotationalJointByMovingOccurrence(js))...)
}

// gearPin is one occurrence reached through the gear graph and the angle it turns.
type gearPin struct {
	occ uint64
	val float64
}

// gearTrainPins walks the gear graph breadth-first from the driven occurrence and returns a pin
// for each coupled gear's rotational joint at its ratioed angle (each gear visited once).
func gearTrainPins(start uint64, v float64, couplings []rateCoupling, byOcc map[uint64]*assemblyJoint) []relationship {
	var pins []relationship
	visited := map[uint64]bool{start: true}
	for queue := []gearPin{{start, v}}; len(queue) > 0; {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range couplings {
			next, val, ok := c.partner(cur.occ, cur.val)
			if !ok || visited[next] {
				continue
			}
			j, ok := byOcc[next]
			if !ok {
				continue
			}
			rv, drivable := drivableVariable(j.kind, types.DriveAngular)
			if !drivable {
				continue
			}
			visited[next] = true
			pins = append(pins, &drivenPin{joint: j, resolved: rv, value: val})
			queue = append(queue, gearPin{next, val})
		}
	}
	return pins
}

// rateCouplings extracts the active rotate-rotate gear relationships from the constraint set.
func rateCouplings(cs *ConstraintSet) []rateCoupling {
	var out []rateCoupling
	for _, c := range cs.All() {
		if rr, ok := c.(*RotateRotateConstraint); ok && !rr.Suppressed() {
			out = append(out, rateCoupling{a: rr.a.occ.ID(), b: rr.b.occ.ID(), ratio: rr.Ratio()})
		}
	}
	return out
}

// rotationalJointByMovingOccurrence maps each rotational joint's moving (non-grounded)
// occurrence to the joint, so a coupled gear's spin can be pinned through its own joint.
func rotationalJointByMovingOccurrence(js *JointSet) map[uint64]*assemblyJoint {
	out := map[uint64]*assemblyJoint{}
	for _, j := range js.All() {
		aj, ok := j.(*assemblyJoint)
		if !ok || aj.kind != types.JointRotational {
			continue
		}
		out[movingOccID(aj)] = aj
	}
	return out
}

// movingOccID is the id of a joint's moving occurrence — the non-grounded anchor, the gear that
// spins (falling back to the first anchor when neither is grounded).
func movingOccID(j *assemblyJoint) uint64 {
	if j.a.occ.Grounded() {
		return j.b.occ.ID()
	}
	return j.a.occ.ID()
}
