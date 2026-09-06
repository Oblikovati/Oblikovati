// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Emission for the loop-framed ruled chart (ADR-0060). The kept cells are grouped into connected
// components; a component whose boundary has two azimuth-wrapping loops is a BAND (a tube), emitted
// as those two loops plus any holes — the two-rim holed band the tessellator meshes (twoRimHoledBandMesh)
// and the Euler count admits, with NO bridging ruling: a bridge would need a vertex on each end loop at
// one common azimuth, which end loops the neighbouring faces already carry as whole closed edges do not
// have. A component that wraps nowhere is a set of contractible patches, grouped by containment.

// wrappingSolidFaces emits every kept component (uvSide). ok=false only when a boundary cannot be
// re-emitted, which the caller reports as an unsupported configuration.
func (c *ruledFaceUV) wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool) {
	var faces []curvedFace
	var lid []loopEdge
	comps := keptComponents(kept, true, false)
	for _, comp := range comps {
		emitted, ok := emitKeptLoops(c, chainLoops(keptBoundaryEdges(comp, true, false)), segs)
		if !ok {
			return nil, nil, false
		}
		compFaces, compLid, ok := c.componentFaces(comp, segs, surface, f, emitted)
		if !ok {
			return nil, nil, false
		}
		faces = append(faces, compFaces...)
		lid = append(lid, compLid...)
	}
	return faces, lid, len(faces) > 0
}

// componentFaces emits ONE connected component: a set of contractible patches when no boundary loop
// wraps the azimuth, a single band when exactly two do, and a decline for anything else — a wrapping
// band has exactly two full-wrap ends.
func (c *ruledFaceUV) componentFaces(comp []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace, emitted []emittedLoop) ([]curvedFace, []loopEdge, bool) {
	ends, holes := c.splitEndsAndHoles(emitted)
	switch len(ends) {
	case 0:
		return c.contractibleFaces(comp, segs, surface, f)
	case 2:
		c.wrapping = true
		var lid []loopEdge
		for _, e := range emitted {
			lid = append(lid, e.section...)
		}
		return []curvedFace{c.faceOf(surface, f, append(ends, holes...), comp)}, lid, true
	}
	return nil, nil, false
}

// splitEndsAndHoles partitions a component's boundary loops into the azimuth-wrapping ends and the rest.
func (c *ruledFaceUV) splitEndsAndHoles(emitted []emittedLoop) (ends, holes []emittedLoop) {
	for _, e := range emitted {
		if c.loopTurnsTheAzimuth(e) {
			ends = append(ends, e)
		} else {
			holes = append(holes, e)
		}
	}
	return ends, holes
}

// contractibleFaces emits a non-wrapping component as one face per outer loop, holes attached by containment.
func (c *ruledFaceUV) contractibleFaces(comp []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool) {
	var faces []curvedFace
	var lid []loopEdge
	charts := chartsOfComponents(c, comp)
	for _, group := range groupLoopFaces(true, false, chainLoops(keptBoundaryEdges(comp, true, false))) {
		emitted, ok := emitKeptLoops(c, group, segs)
		if !ok {
			return nil, nil, false
		}
		for _, e := range emitted {
			lid = append(lid, e.section...)
		}
		faces = append(faces, curvedFace{
			surface: surface, reversed: f.reversed, lineage: f.lineage,
			loops: c.splitAtFrameCrossings(outerFirst(emitted)),
			chart: chartForGroup(charts, group),
		})
	}
	return faces, lid, true
}

// faceOf assembles a face on the wall's surface from emitted loops, keeping the source face's identity.
func (c *ruledFaceUV) faceOf(surface geom.Surface, f curvedFace, loops []emittedLoop, comp []Face2D) curvedFace {
	out := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage,
		chart: chartContours(chartOfKept(comp), c.seamOrigin())}
	for _, e := range loops {
		out.loops = append(out.loops, curvedLoop{edges: e.face})
	}
	out.loops = c.splitAtFrameCrossings(out.loops)
	return out
}

// splitAtFrameCrossings cuts every re-emitted edge at the frame×imprint incidences lying on it. The
// arrangement re-emits a boundary run as ONE edge across an incidence whose imprint dissolved (both cells
// kept), but the incidence is a vertex on the NEIGHBOUR — the tool's side face meets the rim there — and a
// shared edge must subdivide identically on both faces. A seam incidence is not a vertex: the seam is
// artificial. Nor is an incidence on the face's OWN seam ruling — the edge a primitive's wall carries
// twice — whose neighbour across it is this very face: where an imprint circle crosses that ruling
// inside the kept band, the ruling dissolves and nothing on either side keeps the point, and splitting
// the rim there left the box's drilled rim as two arcs on the tool's seam azimuth while the cap across
// it held one circle (ADR-0061 stage 4).
func (c *ruledFaceUV) splitAtFrameCrossings(loops []curvedLoop) []curvedLoop {
	pts := make([]math.Point3, 0, len(c.crossings))
	for _, cr := range c.crossings {
		if c.frameEdgeIsSeam(cr.loop, cr.edge) {
			continue
		}
		fe := c.face.loops[cr.loop].edges[cr.edge]
		pts = append(pts, fe.curve.PointAt(cr.tEdge))
	}
	return splitLoopsAtPoints(loops, pts, c.res)
}

// frameEdgeIsSeam reports whether a frame edge is the face's own seam: the same curve walked twice, once
// each way, by the face's loops.
func (c *ruledFaceUV) frameEdgeIsSeam(loop, edge int) bool {
	curve := c.face.loops[loop].edges[edge].curve
	seen := 0
	for _, l := range c.face.loops {
		for _, e := range l.edges {
			if e.curve == curve {
				seen++
			}
		}
	}
	return seen > 1
}
