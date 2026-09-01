// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// corner90 builds a unit-radius 90° fillet corner with walls along +X and +Y: ta=(1,0,0),
// tb=(0,1,0), shoulder (1,1,0). The cross-section lies in the z=0 plane.
func corner90() (corner, cornerInputs) {
	in := cornerInputs{nA: math.V3(1, 0, 0), nB: math.V3(0, 1, 0)}
	c := corner{cen: math.P3(0, 0, 0), ta: math.P3(1, 0, 0), tb: math.P3(0, 1, 0)}
	return c, in
}

func curv3(a, b, cc math.Point3) float64 {
	ab := float64(a.DistanceTo(b))
	bc := float64(b.DistanceTo(cc))
	ca := float64(cc.DistanceTo(a))
	area := 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(cc)).Length())
	if ab*bc*ca < 1e-18 {
		return 0
	}
	return 4 * area / (ab * bc * ca)
}

// TestShoulderIsSharpCorner: the shoulder of a 90° unit corner is the sharp corner (1,1,0) where the
// two wall tangent lines meet.
func TestShoulderIsSharpCorner(t *testing.T) {
	t.Parallel()
	c, in := corner90()
	if s := shoulder(c, in); !s.IsEqualTo(math.P3(1, 1, 0), 1e-12) {
		t.Errorf("shoulder = %v, want (1,1,0)", s)
	}
}

// endCurv estimates the cross-section's curvature at its first interior station (near the start
// tangency line) for a given sampling density k.
func endCurv(ch []math.Point3) float64 { return curv3(ch[0], ch[1], ch[2]) }

// TestG2ChordsZeroEndCurvature: the G2 cross-section's curvature vanishes at the tangency line — far
// below the circular arc's 1/r there (the jump G2 removes) and shrinking toward 0 under refinement
// (the boundary curvature is exactly 0 at t=0; the discrete estimate is at the first station).
func TestG2ChordsZeroEndCurvature(t *testing.T) {
	t.Parallel()
	c, in := corner90()
	g2coarse := endCurv(g2Chords(c, in, 24))
	g2fine := endCurv(g2Chords(c, in, 240))
	arc := endCurv(arcChords(c, in, 24))
	if arc < 0.8 {
		t.Errorf("arc start curvature = %g, want ~1/r=1 (the jump G2 removes)", arc)
	}
	if g2coarse > 0.4 {
		t.Errorf("G2 start curvature = %g, want far below the arc's ~1", g2coarse)
	}
	if g2fine > g2coarse*0.3 {
		t.Errorf("G2 start curvature did not vanish under refinement: %g (k=24) → %g (k=240), want →0", g2coarse, g2fine)
	}
}

// TestG2ChordsTangentToWalls: the G2 profile leaves ta along wall A (perpendicular to nA) — G1
// tangency, like the arc (checked on the near-start chord, so a small finite-difference tolerance).
func TestG2ChordsTangentToWalls(t *testing.T) {
	t.Parallel()
	c, in := corner90()
	g2 := g2Chords(c, in, 240)
	start := g2[0].VectorTo(g2[1])
	if d := float64(start.Dot(in.nA)) / float64(start.Length()); stdmath.Abs(d) > 1e-2 {
		t.Errorf("G2 takeoff not tangent to wall A: dir·nA = %g, want ~0", d)
	}
}

// TestConicRhoControlsFullness: a larger rho pulls the conic toward the shoulder (fuller), so the
// profile midpoint sits farther from the ta–tb chord.
func TestConicRhoControlsFullness(t *testing.T) {
	t.Parallel()
	c, in := corner90()
	bulge := func(rho float64) float64 {
		ch := conicChords(c, in, 24, rho)
		mid := ch[len(ch)/2]
		// distance of the midpoint from the straight ta→tb chord
		chord := c.ta.VectorTo(c.tb)
		v := c.ta.VectorTo(mid)
		t := v.Dot(chord) / chord.Dot(chord)
		foot := c.ta.TranslateBy(chord.Scale(t))
		return float64(foot.DistanceTo(mid))
	}
	flat, full := bulge(0.25), bulge(0.75)
	if full <= flat {
		t.Errorf("conic fullness not monotonic in rho: rho=0.25 bulge %g, rho=0.75 bulge %g", flat, full)
	}
	// The parabola (rho=0.5) should bulge between the two.
	if mid := bulge(0.5); mid <= flat || mid >= full {
		t.Errorf("rho=0.5 bulge %g should lie between %g and %g", mid, flat, full)
	}
}

// TestConicArcEndpointsMatch: every cross-section starts at ta and ends at tb, so the band retrims to
// the same tangent lines and the topology is unchanged.
func TestConicArcEndpointsMatch(t *testing.T) {
	t.Parallel()
	c, in := corner90()
	for _, ch := range [][]math.Point3{arcChords(c, in, 8), g2Chords(c, in, 8), conicChords(c, in, 8, 0.5)} {
		if !ch[0].IsEqualTo(c.ta, 1e-12) || !ch[len(ch)-1].IsEqualTo(c.tb, 1e-12) {
			t.Errorf("cross-section endpoints = %v..%v, want ta=%v tb=%v", ch[0], ch[len(ch)-1], c.ta, c.tb)
		}
	}
}
