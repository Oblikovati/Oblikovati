// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CutCounterboreHole drills a counterbore (K1b): a shallow recess of counterRadius × counterDepth
// at the entry face, stepping down via a flat annular shoulder to a bore of boreRadius that
// either goes through (boreThrough) or stops boreLen below the shoulder. The result is one
// assembly built from a planar slab — entry holed, recess wall, annular shoulder, bore wall, and
// either a holed exit face or a flat bottom — so it does NOT chain two curved cuts (which would
// trip the planar-only boolean). Every untouched face is copied and keeps its reference key.
// Unsupported shapes (a circle clipping a face, a blind bore that exits) return an error.
func CutCounterboreHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, boreRadius, boreLen, counterRadius, counterDepth float64, boreThrough bool) (*topo.Body, error) {
	if boreRadius <= 0 || counterRadius <= boreRadius || counterDepth <= 0 || (!boreThrough && boreLen <= 0) {
		return nil, fmt.Errorf("brep: counterbore needs 0<boreR(%g)<counterR(%g), counterDepth(%g)>0, boreLen(%g)>0 unless through", boreRadius, counterRadius, counterDepth, boreLen)
	}
	ua := unit(axisDir)
	if ua.LengthSquared() < 0.5 {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	copied, entry, err := classifyBlindDrill(slab, base, ua, counterRadius)
	if err != nil {
		return nil, err
	}
	shoulder := base.TranslateBy(ua.Scale(math.Scalar(counterDepth)))
	end, exitIdx, copied, err := counterboreEnd(slab, copied, entry, base, shoulder, ua, boreRadius, boreLen, boreThrough)
	if err != nil {
		return nil, err
	}
	return assembleCounterbore(copied, entry, exitIdx, base, shoulder, end, ua, boreRadius, counterRadius, boreThrough)
}

// counterboreEnd resolves where the bore ends: a through bore finds the exit face (returned via
// its index in the copied list); a blind bore stops inside and is checked for containment.
func counterboreEnd(slab *topo.Body, copied []curvedFace, entry curvedFace, base, shoulder math.Point3, ua math.Vector3, boreRadius, boreLen float64, through bool) (end math.Point3, exitIdx int, rest []curvedFace, err error) {
	if !through {
		end = shoulder.TranslateBy(ua.Scale(math.Scalar(boreLen)))
		if err := checkBlindFits(slab, entry, end, boreRadius); err != nil {
			return math.Point3{}, -1, nil, err
		}
		return end, -1, copied, nil
	}
	for i, f := range copied {
		if float64(faceNormal(f).Dot(ua)) <= 1-1e-7 {
			continue
		}
		c := base.TranslateBy(ua.Scale(math.Scalar(pierceParam(base, ua, facePlane(f)))))
		if circleInsideFace(c, f, boreRadius) {
			return c, i, copied, nil
		}
	}
	return math.Point3{}, -1, nil, fmt.Errorf("brep: counterbore bore (r=%g) found no through exit face", boreRadius)
}

// assembleCounterbore welds the planar faces (copied + entry + optional exit), holes the entry
// with the recess and the exit with the bore, and adds the recess wall, annular shoulder, bore
// wall, and (blind) flat bottom.
func assembleCounterbore(copied []curvedFace, entry curvedFace, exitIdx int, base, shoulder, end math.Point3, ua math.Vector3, boreRadius, counterRadius float64, through bool) (*topo.Body, error) {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("brep", "cbore", 0)))
	// Copied faces lead the welded list, so a through exit keeps its copied-slice index here;
	// entry is appended after them.
	planar := append(append([]curvedFace{}, copied...), entry)
	entryIdx := len(planar) - 1

	w := newWelder3(planarStitchGrid)
	rings, edgeUse := weldPlanarFaces(w, planar)
	tv := make([]*topo.Vertex, len(w.points))
	for i, p := range w.points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("brep", "vertex", i)))
	}
	lineEdges := buildEdges(bld, w.points, tv, edgeUse)

	eEntry, eShoulderOut, recessSeam, recessCyl, err := buildHoleEdges(bld, []drillCap{{center: base}, {center: shoulder}}, ua, counterRadius)
	if err != nil {
		return nil, err
	}
	eShoulderIn, eEnd, boreSeam, boreCyl, err := buildHoleEdges(bld, []drillCap{{center: shoulder}, {center: end}}, ua, boreRadius)
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
		bld.AddFace(facePlane(f), f.lineage, specs...) // copied/holed faces keep their key (K1a)
	}

	shPlane, err := geom.NewPlane(shoulder, ua.Scale(-1))
	if err != nil {
		return nil, err
	}
	bld.AddFace(shPlane, topo.NewLineage(topo.Tok("brep", "shoulder", 0)),
		topo.OuterLoop(topo.Fwd(eShoulderOut)), topo.InnerLoop(topo.Fwd(eShoulderIn)))
	bld.AddReversedFace(recessCyl, topo.NewLineage(topo.Tok("brep", "recesswall", 0)),
		topo.OuterLoop(topo.Rev(eEntry), topo.Fwd(recessSeam), topo.Rev(eShoulderOut), topo.Rev(recessSeam)))
	bld.AddReversedFace(boreCyl, topo.NewLineage(topo.Tok("brep", "borewall", 0)),
		topo.OuterLoop(topo.Rev(eShoulderIn), topo.Fwd(boreSeam), topo.Rev(eEnd), topo.Rev(boreSeam)))
	if !through {
		botPlane, err := geom.NewPlane(end, ua.Scale(-1))
		if err != nil {
			return nil, err
		}
		bld.AddFace(botPlane, topo.NewLineage(topo.Tok("brep", "holebottom", 0)), topo.OuterLoop(topo.Fwd(eEnd)))
	}
	return bld.Build(), nil
}
