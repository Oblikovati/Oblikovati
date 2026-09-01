// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Open-run terminal caps (ADR-0050 P6). Filleting a CONTIGUOUS OPEN sub-run of a tangent chain (e.g.
// three of the eight top-rim edges of a box whose vertical edges are already rounded) rounds the run as
// a stripe exactly like the closed loop, but each end must be closed off. The tube ends in a flat
// quarter-disk cap lying in the section plane ⊥ the spine, bridging the end section arc back to the
// terminal vertex. This mirrors OCCT's free-end ChFi3d_CoupeParPlan (a planar cut of the fillet
// surface) and reuses the same idiom as our arc-fillet setback end-caps.
//
// The cap is only correct where the run stops PART-WAY along a rim: the rim continues unfilleted past
// the terminal, so material really does remain over the cap's own area and the terminal vertex really
// does survive. This header used to justify the cap differently — "the corner tip survives, it is at
// distance r·√2 > r from the ball centre, outside the bite" — which has the sign backwards. Rounding a
// CONVEX edge keeps what is inside the rolling ball and takes away the corner outside it: measured on
// the 4 cm box, a point at the tip is not in the result. Where the run stops at a sharp corner nothing
// remains over the cap and the section plane is an existing face's, so that case is recognised and
// rebuilt instead — see fillet_stripe_endface.go (#2083).

// addCapFaces adds the two flat setback cap faces of an open run — the quarter-disk pocket between each
// end section arc (topFoot→apex→wallFoot) and the surviving corner vertex. Like addBlendFaces, the loop
// DIRECTION is topological — the cap walks its section arc opposite to the blend band's use of it, which
// also pairs both corner connectors against their host-face retrims — and only the face's Reversed flag
// comes from the winding-vs-normal test.
func (g *stripeBuild) addCapFaces() {
	lin := func(t int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", "cap", t)) }
	for t := range 2 {
		if g.ends[t].active() {
			continue // the run-out lands on an existing face, which carries the section arc itself (#2083)
		}
		tm := g.st.term[t]
		out := g.capOutward(t)
		plane, err := geom.NewPlane(tm.vertex.Point(), out)
		if err != nil {
			continue // a degenerate section plane; Validate rejects the incomplete solid downstream
		}
		loop, ring := g.capLoop(t, g.termFeet(t))
		if capFaceFlipped(ring, out) {
			g.bld.AddReversedFace(plane, lin(t), topo.OuterLoop(loop...))
			continue
		}
		g.bld.AddFace(plane, lin(t), topo.OuterLoop(loop...))
	}
}

// capLoop is terminal t's triangular cap boundary, walked so its section arc opposes the blend band's
// use of that arc (band 0 enters cap 0, band n−1 exits cap 1 — see blendLoop's sharedFwd branches).
func (g *stripeBuild) capLoop(t int, feet [2]*topo.Vertex) ([]topo.Use, []math.Point3) {
	vtx := g.verts[g.st.term[t].vertex]
	startS1 := !g.sharedFwd[0]
	if t == 1 {
		startS1 = g.sharedFwd[len(g.st.segs)-1]
	}
	if startS1 {
		loop := []topo.Use{
			dirUse(g.connTop[t], vtx),      // corner → shared-face foot
			dirUse(g.cap[t], feet[0]),      // shared-face foot → wall foot (the section arc)
			dirUse(g.connWall[t], feet[1]), // wall foot → corner
		}
		return loop, []math.Point3{vtx.Point(), feet[0].Point(), feet[1].Point()}
	}
	loop := []topo.Use{
		dirUse(g.connWall[t], vtx),    // corner → wall foot
		dirUse(g.cap[t], feet[1]),     // wall foot → shared-face foot
		dirUse(g.connTop[t], feet[0]), // shared-face foot → corner
	}
	return loop, []math.Point3{vtx.Point(), feet[1].Point(), feet[0].Point()}
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
	return probe.CentroidPts([]math.Point3{a, b})
}
