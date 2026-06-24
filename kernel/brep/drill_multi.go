// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Multi-hole exact drilling (M2 Phase 3, Oblikovati/Oblikovati#1336). CutCylindricalHole drills an exact
// round hole only into an ALL-planar slab (facesOf rejects any curved face), so the second bore — whose
// target already carries the first hole's cylinder wall — fell back to triangle-soup CSG, and a chain of
// bores accumulated CSG noise into a non-manifold mesh (the chained-drift defect). drillThroughCurved
// lifts the drill to the curvedFace model: it copies EVERY existing face unchanged (planar walls and the
// prior bores' cylinder walls alike), adds the new hole circle to the two pierced caps, adds one new
// cylinder wall, and re-welds through curvedStitch (which welds line AND circle edges, sharing the new
// circle between each cap and the wall). So every bore stays an exact curved B-rep and a drilled plate
// chains without drift — no bore touches CSG.

// curvedDrillCap is a planar cap face the new hole axis pierces, with its pierce centre, its axial
// parameter (to order entry before exit), and its index in the face list (so the other faces copy through).
type curvedDrillCap struct {
	face   curvedFace
	idx    int
	center math.Point3
	param  float64
}

// drillThroughCurved cuts target − (cylinder of axis base→ua, radius) as an exact through-hole, accepting a
// target that already has curved faces (unlike CutCylindricalHole). It returns an error — so the caller
// keeps its fallback — when the cylinder does not pierce exactly two perpendicular planar caps whose
// interiors clear the circle and its existing holes (a partial / clipped / overlapping hole).
func drillThroughCurved(target *topo.Body, base math.Point3, ua math.Vector3, radius float64) (*topo.Body, error) {
	faces := facesOfAny(target)
	caps, err := findDrillCaps(faces, base, ua, radius)
	if err != nil {
		return nil, err
	}
	circLo, err := geom.NewCircle(caps[0].center, ua, radius)
	if err != nil {
		return nil, err
	}
	circHi := geom.Circle{Center: caps[1].center, Normal: circLo.Normal, RefDir: circLo.RefDir, Radius: radius}

	out := copyFacesExcept(faces, caps[0].idx, caps[1].idx)
	out = append(out, capWithHole(caps[0].face, circLo), capWithHole(caps[1].face, circHi))
	out = append(out, drillWallFace(caps[0].center, ua, radius, circLo, circHi))
	body := curvedStitch(out)
	if body == nil {
		return nil, fmt.Errorf("brep: multi-hole drill produced an empty body (r=%g at %+v)", radius, base)
	}
	return body, nil
}

// findDrillCaps returns the exactly-two planar faces the axis pierces cleanly (perpendicular, the circle
// strictly inside and clear of existing holes), ordered entry (lower param) before exit. A perpendicular
// face the axis misses is left for copying; a circle that straddles a face boundary or an existing hole is
// an error (a partial/overlapping hole the general boolean must handle).
func findDrillCaps(faces []curvedFace, base math.Point3, ua math.Vector3, radius float64) ([2]curvedDrillCap, error) {
	var caps []curvedDrillCap
	for i, f := range faces {
		pl, ok := f.surface.(geom.Plane)
		if !ok || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) < 1-1e-7 {
			continue // a wall (planar or curved) the hole runs alongside — copied unchanged
		}
		c := base.TranslateBy(ua.Scale(math.Scalar(pierceParam(base, ua, pl))))
		pierced, clean := circleVsCap(c, radius, f, pl)
		if !pierced {
			continue // a perpendicular face the axis misses — copied unchanged
		}
		if !clean {
			return [2]curvedDrillCap{}, fmt.Errorf("brep: hole circle (r=%g at %+v) straddles a pierced face boundary or an existing hole; partial holes need the general boolean", radius, c)
		}
		caps = append(caps, curvedDrillCap{face: f, idx: i, center: c, param: pierceParam(base, ua, pl)})
	}
	if len(caps) != 2 {
		return [2]curvedDrillCap{}, fmt.Errorf("brep: a through-hole needs exactly 2 perpendicular pierced faces, found %d", len(caps))
	}
	if caps[0].param > caps[1].param {
		caps[0], caps[1] = caps[1], caps[0]
	}
	return [2]curvedDrillCap{caps[0], caps[1]}, nil
}

// circleVsCap classifies the hole circle against a perpendicular planar face: pierced when its centre is
// inside the face (existing holes excluded), clean when the whole circle is strictly inside and clear of
// every existing hole. It samples the curved loops into a planarFace so it can reuse the planar
// containment tests (pointInFace2D / circleInsideFace), which run even-odd over outer loop plus holes.
func circleVsCap(c math.Point3, radius float64, f curvedFace, pl geom.Plane) (pierced, clean bool) {
	pf := curvedCapAsPlanar(f, pl)
	pierced = pointInFace2D(to2D(pl, c), pf)
	clean = pierced && circleInsideFace(c, pf, radius)
	return pierced, clean
}

// curvedCapAsPlanar samples a planar face's curved loops into 3D point rings (outer first) so the planar
// containment helpers can run on it — a circle hole becomes a 32-gon, a straight edge its corner.
func curvedCapAsPlanar(f curvedFace, pl geom.Plane) planarFace {
	rings := make([][]math.Point3, len(f.loops))
	for i, lp := range f.loops {
		rings[i] = sampleCurvedLoop(lp)
	}
	return planarFace{plane: pl, normal: unit(pl.Normal()), loops: rings, lineage: f.lineage}
}

// sampleCurvedLoop flattens a loop into an ordered 3D point ring.
func sampleCurvedLoop(lp curvedLoop) []math.Point3 {
	var ring []math.Point3
	for _, e := range lp.edges {
		ring = append(ring, sampleLoopEdge(e)...)
	}
	return ring
}

// sampleLoopEdge samples one edge from t0 to t1 (exclusive of t1, the next edge's start): a circle/arc
// into 32 points, a straight edge into its start corner.
func sampleLoopEdge(e loopEdge) []math.Point3 {
	n := 1
	switch e.curve.(type) {
	case geom.Circle, geom.Arc3d:
		n = 32
	}
	pts := make([]math.Point3, 0, n)
	for j := 0; j < n; j++ {
		t := e.t0 + (e.t1-e.t0)*float64(j)/float64(n)
		pts = append(pts, e.curve.PointAt(t))
	}
	return pts
}

// capWithHole returns the cap face with the new hole circle appended as an inner loop (a single closed
// circle edge). The loops slice is copied so the source face is untouched.
func capWithHole(f curvedFace, circ geom.Circle) curvedFace {
	hole := curvedLoop{edges: []loopEdge{{curve: circ, t0: 0, t1: 1}}}
	f.loops = append(append([]curvedLoop{}, f.loops...), hole)
	return f
}

// drillWallFace builds the hole wall as a reversed cylinder face (material faces the axis) whose loop runs
// the bottom circle reversed, up the seam, the top circle reversed, down the seam — the same seamed loop
// SolidCylinder uses, so the wall welds to both caps along the shared circle edges.
func drillWallFace(baseCenter math.Point3, ua math.Vector3, radius float64, circLo, circHi geom.Circle) curvedFace {
	cyl, _ := geom.NewCylinder(baseCenter, ua, radius)
	seam := geom.NewLineSegment(circLo.PointAt(0), circHi.PointAt(0))
	loop := curvedLoop{edges: []loopEdge{
		{curve: circLo, t0: 1, t1: 0},
		{curve: seam, t0: 0, t1: 1},
		{curve: circHi, t0: 1, t1: 0},
		{curve: seam, t0: 1, t1: 0},
	}}
	return curvedFace{surface: cyl, reversed: true, loops: []curvedLoop{loop}, lineage: topo.NewLineage(topo.Tok("brep", "drillwall", 0))}
}

// copyFacesExcept returns every face except the two cap indices, unchanged (the prior bores' walls and the
// slab's side walls pass straight through).
func copyFacesExcept(faces []curvedFace, a, b int) []curvedFace {
	out := make([]curvedFace, 0, len(faces))
	for i, f := range faces {
		if i == a || i == b {
			continue
		}
		out = append(out, f)
	}
	return out
}
