// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CutCountersinkHole drills a countersink (K1b): a conical recess that widens to sinkRadius at
// the entry face and narrows (at the cone half-angle) down to the bore, which continues through
// or blind. The result is one assembly from the planar slab — entry holed, a true CONE frustum
// wall, the shared transition circle, a cylinder bore wall, and a holed exit or flat bottom.
// Like the counterbore it is built directly (not by chaining curved cuts). The cone widens
// toward the surface, so its axis points back out of the part (apex deep inside).
func CutCountersinkHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, boreRadius, boreLen, sinkRadius, sinkHalfAngle float64, boreThrough bool) (*topo.Body, error) {
	if boreRadius <= 0 || sinkRadius <= boreRadius || sinkHalfAngle <= 0 || sinkHalfAngle >= stdmath.Pi/2 {
		return nil, fmt.Errorf("brep: countersink needs 0<boreR(%g)<sinkR(%g) and 0<halfAngle(%g)<π/2", boreRadius, sinkRadius, sinkHalfAngle)
	}
	ua := unit(axisDir)
	uaUnit, err := math.UnitVector3FromVector(ua)
	if err != nil {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	copied, entry, err := classifyBlindDrill(slab, base, ua, sinkRadius)
	if err != nil {
		return nil, err
	}
	tan := stdmath.Tan(sinkHalfAngle)
	depthCS := (sinkRadius - boreRadius) / tan
	trans := base.TranslateBy(ua.Scale(math.Scalar(depthCS)))
	apex := base.TranslateBy(ua.Scale(math.Scalar(sinkRadius / tan))) // r=0 deep on the axis
	cone, err := geom.NewCone(apex, ua.Scale(-1), sinkHalfAngle)      // widens back toward the surface
	if err != nil {
		return nil, err
	}
	end, exitIdx, copied, err := counterboreEnd(slab, copied, entry, base, trans, ua, boreRadius, boreLen, boreThrough)
	if err != nil {
		return nil, err
	}
	return assembleCountersink(copied, entry, exitIdx, base, trans, end, uaUnit, cone, boreRadius, sinkRadius, boreThrough)
}

// assembleCountersink welds the planar faces, holes the entry with the sink opening, and closes
// the recess with a cone frustum wall, a cylinder bore wall (sharing the transition circle), and
// a holed exit or flat bottom.
func assembleCountersink(copied []planarFace, entry planarFace, exitIdx int, base, trans, end math.Point3, ua math.UnitVector3, cone geom.Cone, boreRadius, sinkRadius float64, through bool) (*topo.Body, error) {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("brep", "csink", 0)))
	planar := append(append([]planarFace{}, copied...), entry)
	entryIdx := len(planar) - 1

	w := newWelder3()
	rings, edgeUse := weldPlanarFaces(w, planar)
	tv := make([]*topo.Vertex, len(w.points))
	for i, p := range w.points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("brep", "vertex", i)))
	}
	lineEdges := buildEdges(bld, w.points, tv, edgeUse)

	// All circles share the cone's Ref so the cone seam is a straight ruling (angle 0). Seam
	// vertices are shared between each circle and the seams (and the transition between both
	// walls), so the body is watertight.
	entryC := geom.Circle{Center: base, Normal: ua, RefDir: cone.Ref, Radius: sinkRadius}
	transC := geom.Circle{Center: trans, Normal: ua, RefDir: cone.Ref, Radius: boreRadius}
	endC := geom.Circle{Center: end, Normal: ua, RefDir: cone.Ref, Radius: boreRadius}
	pE, pT, pB := entryC.PointAt(0), transC.PointAt(0), endC.PointAt(0)
	vE := bld.AddVertex(pE, csinkLin("seam", 0))
	vT := bld.AddVertex(pT, csinkLin("seam", 1))
	vB := bld.AddVertex(pB, csinkLin("seam", 2))
	eEntry := bld.AddEdge(entryC, vE, vE, csinkLin("sink", 0))
	eTrans := bld.AddEdge(transC, vT, vT, csinkLin("trans", 0))
	eEnd := bld.AddEdge(endC, vB, vB, csinkLin("bore", 0))
	coneSeam := bld.AddEdge(geom.NewLineSegment(pE, pT), vE, vT, csinkLin("coneseam", 0))
	boreSeam := bld.AddEdge(geom.NewLineSegment(pT, pB), vT, vB, csinkLin("boreseam", 0))
	boreCyl, err := geom.NewCylinder(trans, ua.AsVector(), boreRadius)
	if err != nil {
		return nil, err
	}

	for fi, f := range planar {
		specs := planarLoopSpecs(rings[fi], lineEdges)
		switch fi {
		case entryIdx:
			specs = append(specs, topo.InnerLoop(topo.Fwd(eEntry)))
		case exitIdx:
			specs = append(specs, topo.InnerLoop(topo.Fwd(eEnd)))
		}
		bld.AddFace(f.plane, f.lineage, specs...) // keeps reference keys (K1a)
	}
	bld.AddReversedFace(cone, topo.NewLineage(topo.Tok("brep", "sinkwall", 0)),
		topo.OuterLoop(topo.Rev(eEntry), topo.Fwd(coneSeam), topo.Rev(eTrans), topo.Rev(coneSeam)))
	bld.AddReversedFace(boreCyl, topo.NewLineage(topo.Tok("brep", "borewall", 0)),
		topo.OuterLoop(topo.Fwd(eTrans), topo.Fwd(boreSeam), topo.Rev(eEnd), topo.Rev(boreSeam)))
	if !through {
		botPlane, err := geom.NewPlane(end, ua.AsVector().Scale(-1))
		if err != nil {
			return nil, err
		}
		bld.AddFace(botPlane, topo.NewLineage(topo.Tok("brep", "holebottom", 0)), topo.OuterLoop(topo.Fwd(eEnd)))
	}
	return bld.Build(), nil
}

// csinkLin is the lineage for a generated countersink edge/vertex.
func csinkLin(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("brep", role, i)) }
