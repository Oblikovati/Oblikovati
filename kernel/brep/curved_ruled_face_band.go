// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

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
func (c *ruledFaceUV) wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, bool) {
	var faces []curvedFace
	comps := keptComponents(kept, true, false)
	for _, comp := range comps {
		emitted, ok := emitKeptLoops(c, chainLoops(keptBoundaryEdges(comp, true, false)), segs)
		if !ok {
			return nil, false
		}
		ends, holes := c.splitEndsAndHoles(emitted)
		switch {
		case len(ends) == 0:
			patches, ok := c.contractibleFaces(comp, segs, surface, f)
			if !ok {
				return nil, false
			}
			faces = append(faces, patches...)
		case len(ends) == 2:
			c.wrapping = true
			faces = append(faces, c.faceOf(surface, f, append(ends, holes...)))
		default:
			return nil, false
		}
	}
	return faces, len(faces) > 0
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
func (c *ruledFaceUV) contractibleFaces(comp []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, bool) {
	var faces []curvedFace
	for _, group := range groupLoopFaces(true, false, chainLoops(keptBoundaryEdges(comp, true, false))) {
		emitted, ok := emitKeptLoops(c, group, segs)
		if !ok {
			return nil, false
		}
		faces = append(faces, curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: c.splitAtFrameCrossings(outerFirst(emitted))})
	}
	return faces, true
}

// faceOf assembles a face on the wall's surface from emitted loops, keeping the source face's identity.
func (c *ruledFaceUV) faceOf(surface geom.Surface, f curvedFace, loops []emittedLoop) curvedFace {
	out := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage}
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
// artificial.
func (c *ruledFaceUV) splitAtFrameCrossings(loops []curvedLoop) []curvedLoop {
	pts := make([]math.Point3, 0, len(c.crossings))
	for _, cr := range c.crossings {
		fe := c.face.loops[cr.loop].edges[cr.edge]
		pts = append(pts, fe.curve.PointAt(cr.tEdge))
	}
	return splitLoopsAtPoints(loops, pts, c.res)
}

// loopTurnsTheAzimuth reports whether an emitted loop circles the axis a whole turn — an END of a band —
// by its NET azimuth turn, summed over consecutive samples each unwrapped to its predecessor. The band
// chart's loopWrapsU reads the raw azimuth SPAN instead, and a contractible hole that straddles the
// seam spans the whole range without turning at all; it was filed as a third end, the band fell to the
// generic grouping, and its hole came out wound against its rims (ADR-0060).
func (c *ruledFaceUV) loopTurnsTheAzimuth(e emittedLoop) bool {
	turn, prev := 0.0, 0.0
	first := true
	for _, le := range e.face {
		for k := 0; k <= 16; k++ {
			u := float64(c.paramOf(le.curve.PointAt(le.t0 + (le.t1-le.t0)*float64(k)/16)).X)
			if !first {
				u = unwrapAzimuthNear(prev, u)
				turn += u - prev
			}
			prev, first = u, false
		}
	}
	return stdmath.Abs(turn) > stdmath.Pi
}
