// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CutCylindricalHole drills a clean cylindrical through-hole in a slab — the geometry a
// "through all" Hole feature needs. The cylinder axis (line through base along axisDir) must
// enter and exit through two planar faces whose interiors fully contain the hole circle;
// every other face is parallel to the axis and is copied unchanged. The result is a TRUE
// curved B-rep: the two pierced faces gain a circular hole, a single cylinder face forms the
// hole wall (its material side faces the axis), and — because copied/pierced faces keep their
// source lineage — every original face's reference key survives the cut (K1a/K1b).
//
// This is the "curved-on-planar" boolean KIND (ADR-0045): the tool cylinder crosses the slab's
// PLANAR faces in a circle that lies STRICTLY INSIDE one face, so the contact is a single closed
// conic added as an inner loop — no (u,v) SSI arrangement is involved. It delegates to the shared
// curvedStitch drill assembly (drillThroughCurved), the same machinery the multi-hole, blind and
// counterbore variants use — one drill assembly, not a second bespoke planar welder (#1403).
//
// It is a modelling PRIMITIVE, not a boolean recognizer: the caller has already decided it wants a
// clean through-hole. Partial holes (the circle clipping a face boundary, or a blind hole) return an
// error here — ask [Boolean] for those, which trims each face in its own chart and handles both.
func CutCylindricalHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, radius float64) (*topo.Body, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("brep: drill radius must be positive, got %g", radius)
	}
	ua := unit(axisDir)
	if ua.LengthSquared() < 0.5 {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	// A through-all drill: the axis is conceptually unbounded, so no tool-span gate applies.
	return drillThroughCurved(slab, base, ua, radius, stdmath.Inf(-1), stdmath.Inf(1))
}

// drillCap is a point where the hole axis pierces an entry/exit face — the centre of the hole circle
// on that face. The blind/counterbore/countersink assemblers pass their cap centres in entry→exit order
// to buildHoleEdges to build the hole circles and wall.
type drillCap struct {
	center math.Point3
}

// pierceParam returns the parameter t where the axis line (base + t·ua) meets the plane.
func pierceParam(base math.Point3, ua math.Vector3, pl geom.Plane) float64 {
	n := unit(pl.Normal())
	return float64(base.VectorTo(pl.Origin).Dot(n)) / float64(ua.Dot(n))
}

// circleInsideFace reports whether the hole circle (center, radius, in the face plane) lies
// strictly inside the face — its center and a ring of rim samples are all in the face interior.
func circleInsideFace(center math.Point3, f curvedFace, radius float64) bool {
	if !pointInFace2D(to2D(facePlane(f), center), f) {
		return false
	}
	u, v := facePlane(f).UAxis.AsVector(), facePlane(f).VAxis.AsVector()
	const samples = 24
	for i := range samples {
		a := 2 * stdmath.Pi * float64(i) / samples
		rim := center.TranslateBy(u.Scale(math.Scalar(radius * stdmath.Cos(a)))).TranslateBy(v.Scale(math.Scalar(radius * stdmath.Sin(a))))
		if !pointInFace2D(to2D(facePlane(f), rim), f) {
			return false
		}
	}
	return true
}

// weldPlanarFaces welds every face's loops to shared vertex indices and tallies undirected
// edge uses (so faces sharing a box edge resolve to one edge).
func weldPlanarFaces(w *welder3, planar []curvedFace) (rings [][][]int, edgeUse map[[2]int]int) {
	rings = make([][][]int, len(planar))
	edgeUse = map[[2]int]int{}
	for fi, f := range planar {
		for _, loop := range planarRings(f) {
			r := w.ring(loop)
			rings[fi] = append(rings[fi], r)
			for i := range r {
				edgeUse[canonEdge(r[i], r[(i+1)%len(r)])]++
			}
		}
	}
	return rings, edgeUse
}

// planarLoopSpecs turns a face's welded rings (outer first) into loop specs on shared edges.
func planarLoopSpecs(faceRings [][]int, edges map[[2]int]*topo.Edge) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, len(faceRings))
	for ri, r := range faceRings {
		specs[ri] = loopSpec(ri == 0, r, edges)
	}
	return specs
}

// buildHoleEdges builds the two closed hole circles (sharing a frame so the seam is axial),
// the seam edge joining them, and the wall cylinder. The circles are reused by both a pierced
// face (as a hole) and the wall, so they stitch the wall to the slab watertight.
func buildHoleEdges(bld *topo.Builder, caps []drillCap, ua math.Vector3, radius float64) (holeLo, holeHi, seam *topo.Edge, cyl geom.Cylinder, err error) {
	circLo, err := geom.NewCircle(caps[0].center, ua, radius)
	if err != nil {
		return nil, nil, nil, geom.Cylinder{}, err
	}
	circHi := geom.Circle{Center: caps[1].center, Normal: circLo.Normal, RefDir: circLo.RefDir, Radius: radius}
	pLo, pHi := circLo.PointAt(0), circHi.PointAt(0)
	vLo := bld.AddVertex(pLo, topo.NewLineage(topo.Tok("brep", "seam", 0)))
	vHi := bld.AddVertex(pHi, topo.NewLineage(topo.Tok("brep", "seam", 1)))
	holeLo = bld.AddEdge(circLo, vLo, vLo, topo.NewLineage(topo.Tok("brep", "hole", 0)))
	holeHi = bld.AddEdge(circHi, vHi, vHi, topo.NewLineage(topo.Tok("brep", "hole", 1)))
	seam = bld.AddEdge(geom.NewLineSegment(pLo, pHi), vLo, vHi, topo.NewLineage(topo.Tok("brep", "seam", 2)))
	cyl, err = geom.NewCylinder(caps[0].center, ua, radius)
	return holeLo, holeHi, seam, cyl, err
}
