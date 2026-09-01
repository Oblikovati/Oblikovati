// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/math"
)

// R3 — the single-arm runout's OBLIQUE host-rail re-termination (curved-runout-r3-d4e3-brief.md,
// curved-runout-forensic §R3). D4 (both caps OBLIQUE) and E3 (one perpendicular pole + one OBLIQUE cap)
// close the single-arm both-ends loop only when each host contact rail's OUTER end coincides with the
// oblique cross-section trim's foot. singleRunoutFeet/armRunoutRail build the rails at the PERPENDICULAR
// rolling-ball feet; the oblique far-runout engine (armFarRunout→obliqueRunout) then fixes the true
// analytic OBLIQUE feet closed-form (armRunoutFeet: armSprings ∩ capping) and carries them on run.feet.
// This wiring re-terminates the perpendicular-built rails onto those oblique feet, PER END, so trim
// endpoint == rail outer end == host-loop boundary point and the geom.Sphere host retrim can close.
// It fires ONLY on an oblique end (run.regime == runoutOblique): a perpendicular end's foot already IS
// the rail terminus, so B6/C9/C1/M7/C5/D8 and E3's pole end stay byte-identical (do-no-harm).

// obliqueRetermRails re-terminates the two host contact rails at any OBLIQUE end so each rail's outer end
// lands on that end's oblique runout foot (run.feet — the shared trim endpoint). run0 owns the START
// (.from) ends of railA/railB, run1 the END (.to) ends; each end is re-terminated only when its regime is
// oblique, keeping every perpendicular end bit-identical. Declines (do-no-harm) with the offending ok flags
// when a foot does not lie on a rail's own line/circle — the shared-edge identity must hold, never be snapped.
func obliqueRetermRails(railA, railB endSeg, run0, run1 armRunout, tol float64) (endSeg, endSeg, string) {
	startOblique := run0.regime == runoutOblique
	endOblique := run1.regime == runoutOblique
	rA, okA := reterminateRailEnds(railA, run0.feet[0], run1.feet[0], startOblique, endOblique, tol)
	rB, okB := reterminateRailEnds(railB, run0.feet[1], run1.feet[1], startOblique, endOblique, tol)
	if !okA || !okB {
		return endSeg{}, endSeg{}, fmt.Sprintf("single-arm runout: oblique rail re-termination declined — a foot "+
			"does not lie on its rail's line/circle (ef.a rail ok=%v, ef.b rail ok=%v; start-oblique=%v end-oblique=%v; "+
			"start feet a=%v b=%v, end feet a=%v b=%v)", okA, okB, startOblique, endOblique,
			run0.feet[0], run0.feet[1], run1.feet[0], run1.feet[1])
	}
	return rA, rB, ""
}

// reterminateRailEnds moves a rail's START and/or END outer terminus onto the given oblique feet. It applies
// reterminateRail directly at the .from end, and at the .to end via a reverse/re-terminate/reverse sandwich
// (reterminateRail only re-terminates the .from end). doStart/doEnd gate each end so a perpendicular end is
// left untouched (byte-identical). Declines when a requested foot is off the rail's own line/circle within tol.
func reterminateRailEnds(rail endSeg, startFoot, endFoot math.Point3, doStart, doEnd bool, tol float64) (endSeg, bool) {
	out := rail
	if doStart {
		reterm, ok := reterminateRail(out, startFoot, tol)
		if !ok {
			return endSeg{}, false
		}
		out = reterm
	}
	if !doEnd {
		return out, true
	}
	reterm, ok := reterminateRail(reverseEndSegs([]endSeg{out})[0], endFoot, tol)
	if !ok {
		return endSeg{}, false
	}
	return reverseEndSegs([]endSeg{reterm})[0], true
}
