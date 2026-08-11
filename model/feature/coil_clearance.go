// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Turn clearance for coils (Oblikovati#2080). A coil whose rail rises less over one revolution
// than its profile is tall re-enters the turn below it, so the swept solid passes through itself.
// Nothing caught that: ops.Validate is topology only, and the overlapping turns barely move the
// volume (84.71 against 84.78 for a clean coil), so neither the topology gate nor a volume sanity
// check could see it. Measured on a 1-tall wire, pitch 1.0 was clean and pitch 0.8 delivered 256
// interpenetrating face pairs — as a valid, solid, plausible-looking body.
//
// The test is geometric rather than arithmetic on the rail. "Rise per turn must clear the profile
// extent" is right only for a plain helix: a TAPERED coil moves each turn to a different radius,
// so it can rise less than the profile is tall and still not touch itself. A rail rule would
// refuse those conical springs wrongly.

// coilClearsItsOwnTurns reports an error when the built coil passes through itself. The diagnostic
// carries the profile's extent along the axis, because a rise that cannot clear it is what causes
// this in practice, and that is the number the author has to change.
//
// Example: err := coilClearsItsOwnTurns(body, sections, def.Axis) // nil when the turns clear
func coilClearsItsOwnTurns(body *topo.Body, sections [][]math.Point3, axis *WorkAxis) error {
	hits := ops.SelfIntersections(body, ops.DefaultQuality())
	if len(hits) == 0 {
		return nil
	}
	return fmt.Errorf("coil: the coil passes through itself at %d place(s), first near %v; "+
		"a turn must clear the one below it, and this profile is %g deep along the axis — "+
		"raise the pitch above that, or taper the coil so consecutive turns sit at different radii",
		len(hits), hits[0].Witness, coilProfileAxialDepth(sections, axis))
}

// coilProfileAxialDepth is how far the placed profile reaches along the coil axis — the rise one
// revolution has to beat. It is measured on the built section rather than the sketch, so a profile
// that is tilted or offset relative to the axis is measured as it actually sits.
func coilProfileAxialDepth(sections [][]math.Point3, axis *WorkAxis) float64 {
	if len(sections) == 0 || axis == nil {
		return 0
	}
	dir := axis.Direction().AsVector()
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range sections[0] {
		d := float64(p.AsVector().Dot(dir))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	return hi - lo
}
