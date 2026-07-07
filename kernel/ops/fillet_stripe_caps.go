// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Open-run terminal caps (ADR-0050 P6). Filleting a CONTIGUOUS OPEN sub-run of a tangent chain (e.g.
// three of the eight top-rim edges of a box whose vertical edges are already rounded) rounds the run as
// a stripe exactly like the closed loop, but each end must be closed off. The tube ends in a flat
// quarter-disk cap lying in the section plane ⊥ the spine: the corner tip survives (it is at distance
// r·√2 > r from the ball centre, outside the bite, so material remains there), and the cap is the real
// face bridging the end section arc back to that tip. This mirrors OCCT's free-end ChFi3d_CoupeParPlan
// (a planar cut of the fillet surface) and reuses the same idiom as our arc-fillet setback end-caps.

// addCapFaces adds the two flat setback cap faces of an open run — the quarter-disk pocket between each
// end section arc (topFoot→apex→wallFoot) and the surviving corner vertex. It is wound to face OUTWARD
// along the tube axis (away from the tube interior), the same geometric-winding discipline addBlendFaces
// uses, so the mass-properties integral reads material on the correct side.
func (g *stripeBuild) addCapFaces() {
	feet := [2][2]*topo.Vertex{{g.vS1[0], g.vW[0]}, {g.vEndS1, g.vEndW}}
	lin := func(t int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", "cap", t)) }
	for t := 0; t < 2; t++ {
		tm := g.st.term[t]
		vtx := g.verts[tm.vertex]
		out := g.capOutward(t)
		plane, err := geom.NewPlane(tm.vertex.Point(), out)
		if err != nil {
			continue // a degenerate section plane; Validate rejects the incomplete solid downstream
		}
		loop := []topo.Use{
			dirUse(g.connTop[t], vtx),         // corner → shared-face foot
			dirUse(g.cap[t], feet[t][0]),      // shared-face foot → wall foot (the section arc)
			dirUse(g.connWall[t], feet[t][1]), // wall foot → corner
		}
		if capFaceFlipped([]math.Point3{tm.vertex.Point(), tm.topA, tm.wallA}, out) {
			loop = reverseLoop(loop)
		}
		g.bld.AddFace(plane, lin(t), topo.OuterLoop(loop...))
	}
}

// capOutward is terminal t's cap-face outward normal — the spine-axis direction that makes the flat cap
// bound the solid correctly (it opposes the tube on their shared section arc, the manifold requirement
// two outward faces satisfy). It points from the run-out end back toward the tube body: the little
// material pocket the ball leaves at the corner tip sits ON the cap, so the solid is on the tube side.
// Terminal 0 caps segment 0's entry (its interior is that segment's exit); terminal 1 caps the last
// segment's exit (its interior is that segment's entry).
func (g *stripeBuild) capOutward(t int) math.Vector3 {
	tm := g.st.term[t]
	termMid := midpointOf(tm.topA, tm.wallA)
	interiorMid := midpointOf(g.exitFootS1(0).Point(), g.exitFootW(0).Point())
	if t == 1 {
		s := g.st.segs[tm.seg]
		interiorMid = midpointOf(s.topA, s.wallA)
	}
	return termMid.VectorTo(interiorMid)
}

// capFaceFlipped reports whether the triangular cap ring (corner, shared foot, wall foot) winds against
// the desired outward normal — matching blendRingFlipped's convention so both faces orient consistently.
func capFaceFlipped(ring []math.Point3, out math.Vector3) bool {
	nrm := ring[0].VectorTo(ring[1]).Cross(ring[0].VectorTo(ring[2]))
	return float64(nrm.Dot(out)) < 0
}

// midpointOf is the midpoint of two points.
func midpointOf(a, b math.Point3) math.Point3 {
	return centroidPts([]math.Point3{a, b})
}
