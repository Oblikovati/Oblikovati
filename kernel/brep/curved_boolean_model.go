// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved boolean face model (M2 Phase 1, Oblikovati/Oblikovati#1334). The planar boolean
// flattens a body to planarFace{plane, 3D point rings}; the general curved boolean needs the
// same four-stage pipeline (imprint → split → classify → stitch) lifted to ANY analytic
// surface with curved (line/circle/ellipse) boundary edges. curvedFace is that lifted model —
// planarFace is the special case surface == geom.Plane. This file is the model + flatten only;
// imprint/split/classify/stitch land in the following slices of #1334.

// loopEdge is one boundary edge as a loop traverses it: the underlying analytic curve plus the
// parameter interval walked, t0→t1, already oriented to the loop's direction (so PointAt(t0) is
// where the edge begins in this loop). A closed edge — a full seam circle whose start and end
// vertex coincide — spans the curve's whole domain.
type loopEdge struct {
	curve  geom.Curve3
	t0, t1 float64
}

// start and end return the loop-oriented endpoints of the edge.
func (e loopEdge) start() math.Point3 { return e.curve.PointAt(e.t0) }
func (e loopEdge) end() math.Point3   { return e.curve.PointAt(e.t1) }

// curvedLoop is one boundary loop of a curved face: an ordered ring of loopEdges where each
// edge's end meets the next edge's start (the last meets the first).
type curvedLoop struct {
	edges []loopEdge
}

// curvedFace is a face flattened for the curved boolean (the curved generalization of
// planarFace, ADR-0027 §M2): its analytic surface, ordered boundary loops (loops[0] is the
// outer loop, the rest holes), the face sense (reversed when the surface normal points into
// the material — a cut wall, AddReversedFace), and the source face's lineage so a result face
// keeps its reference key (K1a). A boundary-less closed face (full sphere/torus) has no loops.
type curvedFace struct {
	surface  geom.Surface
	reversed bool
	loops    []curvedLoop
	lineage  topo.Lineage
}

// facesOfAny flattens EVERY face of a body into a curvedFace, unlike facesOf which rejects any
// curved face. The model holds any geom.Surface; whether a given face pair can be imprinted in
// closed form is decided later (curvedImprint), not here. Used by the general curved boolean.
//
// Example:
//
//	cyl, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,0,1), 3, 4)
//	faces := brep.facesOfAny(cyl) // 3 faces: two planar caps + one cylindrical side
func facesOfAny(b *topo.Body) []curvedFace {
	out := make([]curvedFace, 0, len(b.Faces()))
	for _, f := range b.Faces() {
		out = append(out, curvedFace{
			surface:  f.Geometry(),
			reversed: f.Reversed(),
			loops:    loopsOf(f),
			lineage:  f.Lineage(),
		})
	}
	return out
}

// loopsOf extracts a face's boundary loops as curvedLoops (outer loop first, matching
// topo.Loop.IsOuter), each edge carried as its analytic curve over the parameter interval the
// loop walks.
func loopsOf(f *topo.Face) []curvedLoop {
	var outer, holes []curvedLoop
	for _, l := range f.Loops() {
		cl := curvedLoop{edges: loopEdgesOf(l)}
		if l.IsOuter() {
			outer = append(outer, cl)
		} else {
			holes = append(holes, cl)
		}
	}
	return append(outer, holes...)
}

// loopEdgesOf turns a loop's edge uses into oriented loopEdges.
func loopEdgesOf(l *topo.Loop) []loopEdge {
	uses := l.EdgeUses()
	out := make([]loopEdge, 0, len(uses))
	for _, u := range uses {
		out = append(out, orientedLoopEdge(u))
	}
	return out
}

// orientedLoopEdge resolves one edge use to a loop-oriented loopEdge. A closed edge (start and
// end vertex coincide — a full circle seam) walks the curve's whole domain; an open edge walks
// from its start vertex's parameter to its end vertex's, swapped when the use is reversed.
func orientedLoopEdge(u *topo.EdgeUse) loopEdge {
	e := u.Edge()
	c := e.Geometry()
	lo, hi := c.Domain()
	if e.StartVertex() == e.EndVertex() {
		if u.Reversed() {
			return loopEdge{curve: c, t0: hi, t1: lo}
		}
		return loopEdge{curve: c, t0: lo, t1: hi}
	}
	t0, _ := geom.CurveParamAtPoint3(c, e.StartVertex().Point())
	t1, _ := geom.CurveParamAtPoint3(c, e.EndVertex().Point())
	if u.Reversed() {
		return loopEdge{curve: c, t0: t1, t1: t0}
	}
	return loopEdge{curve: c, t0: t0, t1: t1}
}
