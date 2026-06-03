// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// filletResultFaces builds the faces of the filleted body: every original face transformed
// for the fillets touching it (its A/B corners pulled back to the tangent points, its end
// corners replaced by an arc), plus one cylinder face per filleted edge.
func filletResultFaces(body *topo.Body, fils []edgeFillet) []filletFace {
	abSubst, endCorner := filletMaps(fils)
	var out []filletFace
	for _, f := range body.Faces() {
		out = append(out, transformFace(f, abSubst[f], endCorner[f]))
	}
	for _, ef := range fils {
		out = append(out, cylinderFace(ef))
	}
	return out
}

// filletMaps indexes, per face: the corner-vertex → tangent-point pullbacks (where the face
// is a fillet's A or B face), and the corner-vertex → corner arc (where it is the end face).
func filletMaps(fils []edgeFillet) (map[*topo.Face]map[uint64]math.Point3, map[*topo.Face]map[uint64]corner) {
	ab := map[*topo.Face]map[uint64]math.Point3{}
	ends := map[*topo.Face]map[uint64]corner{}
	put := func(m map[*topo.Face]map[uint64]math.Point3, f *topo.Face, id uint64, p math.Point3) {
		if m[f] == nil {
			m[f] = map[uint64]math.Point3{}
		}
		m[f][id] = p
	}
	for _, ef := range fils {
		for _, c := range []corner{ef.c0, ef.c1} {
			put(ab, c.a, c.vertex.ID(), c.ta)
			put(ab, c.b, c.vertex.ID(), c.tb)
			if ends[c.endFace] == nil {
				ends[c.endFace] = map[uint64]corner{}
			}
			ends[c.endFace][c.vertex.ID()] = c
		}
	}
	return ab, ends
}

// transformFace rebuilds a face's loops, pulling A/B corners to their tangent points and
// expanding each end corner into a tangent-point-to-tangent-point arc. A face untouched by
// any fillet is copied unchanged.
func transformFace(f *topo.Face, subs map[uint64]math.Point3, ends map[uint64]corner) filletFace {
	ff := filletFace{surface: f.Geometry()}
	for _, l := range f.Loops() {
		ff.loops = append(ff.loops, transformLoop(f, l, subs, ends))
	}
	return ff
}

// transformLoop walks a loop's edge uses and applies the per-vertex fillet substitutions.
func transformLoop(f *topo.Face, l *topo.Loop, subs map[uint64]math.Point3, ends map[uint64]corner) filletLoop {
	uses := l.EdgeUses()
	n := len(uses)
	var fl filletLoop
	for i, u := range uses {
		v := useFromVertex(u)
		switch {
		case ends != nil && hasCorner(ends, v):
			c := ends[v.ID()]
			tIn := c.tOf(otherFace(uses[(i-1+n)%n].Edge(), f))
			tOut := c.tOf(otherFace(u.Edge(), f))
			arc, _ := geom.Arc3dByThreePoints(tIn, c.mid, tOut)
			fl.add(tIn, arc)
			fl.add(tOut, nil)
		case subs != nil && hasSubst(subs, v):
			fl.add(subs[v.ID()], nil)
		default:
			fl.add(v.Point(), nil)
		}
	}
	return fl
}

func hasCorner(ends map[uint64]corner, v *topo.Vertex) bool { _, ok := ends[v.ID()]; return ok }
func hasSubst(subs map[uint64]math.Point3, v *topo.Vertex) bool {
	_, ok := subs[v.ID()]
	return ok
}

// add appends a point and the curve of the segment leaving it (nil ⇒ straight).
func (l *filletLoop) add(p math.Point3, curve geom.Curve3) {
	l.pts = append(l.pts, p)
	l.curves = append(l.curves, curve)
}

// useFromVertex returns the from-vertex of an edge use (honouring reversal).
func useFromVertex(u *topo.EdgeUse) *topo.Vertex {
	if u.Reversed() {
		return u.Edge().EndVertex()
	}
	return u.Edge().StartVertex()
}

// otherFace returns the face sharing edge e that is not f.
func otherFace(e *topo.Edge, f *topo.Face) *topo.Face {
	for _, g := range e.Faces() {
		if g != f {
			return g
		}
	}
	return nil
}

// cylinderFace builds the rolling-ball cylinder face: tangent line on A, end arc, tangent
// line on B, end arc — wound so its geometric normal matches the cylinder's outward radial.
func cylinderFace(ef edgeFillet) filletFace {
	c0, c1 := ef.c0, ef.c1
	arc1, _ := geom.Arc3dByThreePoints(c1.ta, c1.mid, c1.tb) // TA1 → TB1 at end 1
	arc0, _ := geom.Arc3dByThreePoints(c0.tb, c0.mid, c0.ta) // TB0 → TA0 at end 0
	loop := filletLoop{
		pts:    []math.Point3{c0.ta, c1.ta, c1.tb, c0.tb},
		curves: []geom.Curve3{nil, arc1, nil, arc0},
	}
	if cylinderLoopFlipped(ef.cyl, loop) {
		loop = reverseFilletLoop(loop, ef)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loop}}
}

// cylinderLoopFlipped reports whether the loop winds against the cylinder's outward normal at
// the first end arc's midpoint (so it should be reversed for a consistent, outward face).
func cylinderLoopFlipped(cyl geom.Cylinder, loop filletLoop) bool {
	a, b, c := loop.pts[0], loop.pts[1], loop.pts[2]
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	u, v := cyl.ParamAt(centroidPts(loop.pts))
	return n.Dot(cyl.NormalAt(u, v)) < 0
}

// reverseFilletLoop reverses the cylinder loop (and rebuilds its arcs in the new direction).
func reverseFilletLoop(_ filletLoop, ef edgeFillet) filletLoop {
	c0, c1 := ef.c0, ef.c1
	arc0, _ := geom.Arc3dByThreePoints(c0.ta, c0.mid, c0.tb) // TA0 → TB0
	arc1, _ := geom.Arc3dByThreePoints(c1.tb, c1.mid, c1.ta) // TB1 → TA1
	return filletLoop{
		pts:    []math.Point3{c0.ta, c0.tb, c1.tb, c1.ta},
		curves: []geom.Curve3{arc0, nil, arc1, nil},
	}
}
