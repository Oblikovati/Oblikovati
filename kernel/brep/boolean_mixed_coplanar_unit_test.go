// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// holedDisc is a planar face: a unit circle at z=1 with a square hole, the exact-frame face a
// flush contact meets (ADR-0060).
func holedDisc(t *testing.T) curvedFace {
	t.Helper()
	pl, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	circle, _ := geom.NewCircle(math.P3(0, 0, 1), math.V3(0, 0, 1), 1)
	sq := []math.Point3{math.P3(-0.3, -0.3, 1), math.P3(0.3, -0.3, 1), math.P3(0.3, 0.3, 1), math.P3(-0.3, 0.3, 1)}
	var hole []loopEdge
	for i := range sq {
		hole = append(hole, loopEdge{curve: geom.NewLineSegment(sq[i], sq[(i+1)%4]), t0: 0, t1: 1, v0: sq[i], v1: sq[(i+1)%4]})
	}
	return curvedFace{surface: pl, loops: []curvedLoop{{edges: []loopEdge{{curve: circle, t0: 0, t1: 1}}}, {edges: hole}}}
}

func TestFaceContainsExactHonoursACircleRimAndASquareHole(t *testing.T) {
	t.Parallel()
	f := holedDisc(t)
	cases := []struct {
		p    math.Point3
		want bool
	}{
		{math.P3(0.6, 0, 1), true}, {math.P3(0, 0.9, 1), true}, {math.P3(0, 0, 1), false}, {math.P3(0.2, 0.2, 1), false}, {math.P3(1.2, 0, 1), false},
	}
	for _, c := range cases {
		if got := faceContainsExact(f, c.p); got != c.want {
			t.Errorf("contains %v = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestCoplanarStraightImprintsClipToTheExactMaterial(t *testing.T) {
	t.Parallel()
	disc := holedDisc(t)
	pl, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, -1))
	corners := []math.Point3{math.P3(0, -1.5, 1), math.P3(1.5, -1.5, 1), math.P3(1.5, 1.5, 1), math.P3(0, 1.5, 1)}
	var edges []loopEdge
	for i := range corners {
		edges = append(edges, loopEdge{curve: geom.NewLineSegment(corners[i], corners[(i+1)%4]), t0: 0, t1: 1, v0: corners[i], v1: corners[(i+1)%4]})
	}
	block := curvedFace{surface: pl, loops: []curvedLoop{{edges: edges}}}
	segs, ok := coplanarStraightImprints(disc, block)
	if !ok || len(segs) != 2 {
		t.Fatalf("imprints = %v ok=%v, want the two pieces of x=0 between the rim and the hole", segs, ok)
	}
	for _, s := range segs {
		for _, p := range s {
			r := float64(p.X)*float64(p.X) + float64(p.Y)*float64(p.Y)
			onRim, onHole := stdmath.Abs(r-1) < 1e-12, stdmath.Abs(stdmath.Abs(float64(p.Y))-0.3) < 1e-12
			if float64(p.X) != 0 || (!onRim && !onHole) {
				t.Errorf("imprint end %v is not on x=0 at the rim or the hole", p)
			}
		}
	}
}
