// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// CutCylindricalHole drills a clean cylindrical through-hole in a planar-faced slab — the
// first end-to-end plane∩cylinder boolean (K1b slice 3) and the geometry a "through all"
// Hole feature needs. The cylinder axis (line through base along axisDir) must enter and
// exit through two planar faces whose interiors fully contain the hole circle; every other
// face is parallel to the axis and is copied unchanged. The result is a TRUE curved B-rep:
// the two pierced faces gain a circular hole, a single cylinder face forms the hole wall
// (its material side faces the axis, AddReversedFace), and — because copied/pierced faces
// keep their source lineage — every original face's reference key survives the cut (K1a/K1b).
//
// Partial holes (the circle clipping a face boundary, or a blind hole) need the general
// curved arrangement and return an error here — that is a later K1b slice.
func CutCylindricalHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, radius float64) (*topo.Body, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("brep: drill radius must be positive, got %g", radius)
	}
	ua := unit(axisDir)
	if ua.LengthSquared() < 0.5 {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	copied, caps, err := classifyDrillFaces(slab, base, ua, radius)
	if err != nil {
		return nil, err
	}
	return assembleDrilled(copied, caps, ua, radius)
}

// drillCap is a planar face the hole axis pierces (an entry/exit face), with the pierce
// point and its parameter along the axis (used to order entry before exit).
type drillCap struct {
	face   planarFace
	center math.Point3
	param  float64
}

// classifyDrillFaces splits the slab's planar faces into the two the axis drills through
// (perpendicular to the axis, interior fully containing the circle) and the rest (copied).
func classifyDrillFaces(slab *topo.Body, base math.Point3, ua math.Vector3, radius float64) (copied []planarFace, caps []drillCap, err error) {
	faces, ok := facesOf(slab)
	if !ok {
		return nil, nil, ErrNonPlanar
	}
	for _, f := range faces {
		if stdmath.Abs(float64(f.normal.Dot(ua))) < 1-1e-7 {
			copied = append(copied, f) // a wall the hole runs alongside — unchanged
			continue
		}
		t := pierceParam(base, ua, f.plane)
		c := base.TranslateBy(ua.Scale(math.Scalar(t)))
		if !circleInsideFace(c, f, radius) {
			return nil, nil, fmt.Errorf("brep: hole circle (r=%g at %+v) does not fit inside the pierced face; partial holes need the general boolean", radius, c)
		}
		caps = append(caps, drillCap{face: f, center: c, param: t})
	}
	if len(caps) != 2 {
		return nil, nil, fmt.Errorf("brep: a through-hole needs exactly 2 perpendicular pierced faces, found %d", len(caps))
	}
	if caps[0].param > caps[1].param {
		caps[0], caps[1] = caps[1], caps[0]
	}
	return copied, caps, nil
}

// pierceParam returns the parameter t where the axis line (base + t·ua) meets the plane.
func pierceParam(base math.Point3, ua math.Vector3, pl geom.Plane) float64 {
	n := unit(pl.Normal())
	return float64(base.VectorTo(pl.Origin).Dot(n)) / float64(ua.Dot(n))
}

// circleInsideFace reports whether the hole circle (center, radius, in the face plane) lies
// strictly inside the face — its center and a ring of rim samples are all in the face interior.
func circleInsideFace(center math.Point3, f planarFace, radius float64) bool {
	if !pointInFace2D(to2D(f.plane, center), f) {
		return false
	}
	u, v := f.plane.UAxis.AsVector(), f.plane.VAxis.AsVector()
	const samples = 24
	for i := 0; i < samples; i++ {
		a := 2 * stdmath.Pi * float64(i) / samples
		rim := center.TranslateBy(u.Scale(math.Scalar(radius * stdmath.Cos(a)))).TranslateBy(v.Scale(math.Scalar(radius * stdmath.Sin(a))))
		if !pointInFace2D(to2D(f.plane, rim), f) {
			return false
		}
	}
	return true
}

// assembleDrilled welds the copied + pierced planar faces, adds a circular hole to each
// pierced face, and joins them with a single cylinder wall face into a watertight solid.
func assembleDrilled(copied []planarFace, caps []drillCap, ua math.Vector3, radius float64) (*topo.Body, error) {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("brep", "drill", 0)))
	planar := append(append([]planarFace{}, copied...), caps[0].face, caps[1].face)

	w := newWelder3()
	rings, edgeUse := weldPlanarFaces(w, planar)
	tv := make([]*topo.Vertex, len(w.points))
	for i, p := range w.points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("brep", "vertex", i)))
	}
	lineEdges := buildEdges(bld, w.points, tv, edgeUse)
	holeLo, holeHi, seam, cyl, err := buildHoleEdges(bld, caps, ua, radius)
	if err != nil {
		return nil, err
	}

	loEntry, hiEntry := len(planar)-2, len(planar)-1
	for fi, f := range planar {
		specs := planarLoopSpecs(rings[fi], lineEdges)
		switch fi {
		case loEntry:
			specs = append(specs, topo.InnerLoop(topo.Fwd(holeLo)))
		case hiEntry:
			specs = append(specs, topo.InnerLoop(topo.Fwd(holeHi)))
		}
		bld.AddFace(f.plane, f.lineage, specs...) // copied/pierced faces keep their key (K1a)
	}
	// The wall's surface normal is outward-radial, so its material side faces the axis.
	bld.AddReversedFace(cyl, topo.NewLineage(topo.Tok("brep", "wall", 0)),
		topo.OuterLoop(topo.Rev(holeLo), topo.Fwd(seam), topo.Rev(holeHi), topo.Rev(seam)))
	return bld.Build(), nil
}

// weldPlanarFaces welds every face's loops to shared vertex indices and tallies undirected
// edge uses (so faces sharing a box edge resolve to one edge).
func weldPlanarFaces(w *welder3, planar []planarFace) (rings [][][]int, edgeUse map[[2]int]int) {
	rings = make([][][]int, len(planar))
	edgeUse = map[[2]int]int{}
	for fi, f := range planar {
		for _, loop := range f.loops {
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
