// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General curved half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). The arrangement pipeline
// the planar boolean has, lifted to curved faces: split every face by the cutting plane, keep the
// sub-faces on the negative side, build a planar lid bounded by the section curves, and curved-stitch
// the result. Composing this over a convex tool's planes is the box boolean. This file wires the stages
// that are robust today — whole/drop faces and the boundary-less (sphere) cap — and defers a looped
// face crossed by the imprint (the cap-cutting step) to ErrUnsupportedHalfSpace, so the caller keeps the
// CSG fallback until that split lands.

// generalHalfSpace cuts an arbitrary analytic body by one plane via split → classify → lid → stitch.
func generalHalfSpace(body *topo.Body, plane geom.Plane, n math.Vector3, faces []curvedFace) (*topo.Body, error) {
	var kept []curvedFace
	var section []loopEdge
	for _, f := range faces {
		pieces, arcs, err := splitFaceByPlane(f, plane, n)
		if err != nil {
			return nil, err
		}
		kept = append(kept, pieces...)
		section = append(section, arcs...)
	}
	if len(kept) == 0 {
		return topo.MergeBodies(topo.NewLineage(topo.Tok("halfspace", "empty", 0)), true), nil
	}
	if len(section) == 0 {
		return body, nil // no face was cut: the plane clears the body and the whole of it is kept
	}
	lids, ok := buildLids(plane, n, section)
	if !ok {
		return nil, ErrUnsupportedHalfSpace
	}
	return curvedStitch(append(kept, lids...)), nil
}

// splitFaceByPlane splits one face by the cutting plane, returning the kept (negative-side) sub-faces
// and the section arcs that bound the lid. A face the plane does not cross is kept whole or dropped; a
// boundary-less face (sphere) splits into the kept cap; a looped face crossed by the imprint defers.
func splitFaceByPlane(f curvedFace, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	curves, handled := curvedImprint(f, curvedFace{surface: plane})
	if !handled {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	if len(curves) == 0 {
		if signedDistance(faceSample(f), plane, n) <= 0 {
			return []curvedFace{f}, nil, nil // whole face on the negative side
		}
		return nil, nil, nil // whole face on the positive side: dropped
	}
	if len(f.loops) == 0 {
		return capSplit(f, curves[0])
	}
	return nil, nil, ErrUnsupportedHalfSpace // looped face crossed by the imprint: next increment
}

// capSplit splits a boundary-less face (a bare sphere) by its imprint circle into the kept negative cap
// plus the section circle. The cap loop is the circle REVERSED so the −n cap is the enclosed region
// (the cut circle's normal is the plane normal, planeSphereCurve); the section circle is forward so the
// lid, with outward +n, traverses it counter-clockwise.
func capSplit(f curvedFace, c geom.Curve3) ([]curvedFace, []loopEdge, error) {
	circle, ok := c.(geom.Circle)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	cap := curvedFace{
		surface: f.surface,
		lineage: f.lineage, // the cap keeps the sphere's reference key (K1a/K1b)
		loops:   []curvedLoop{{edges: []loopEdge{{curve: circle, t0: 1, t1: 0}}}},
	}
	return []curvedFace{cap}, []loopEdge{{curve: circle, t0: 0, t1: 1}}, nil
}

// buildLids chains the section arcs into closed loops and builds one planar lid face per loop, on the
// cutting plane with outward normal +n. ok=false when the section arcs do not close into loops (a case
// the looped split will complete).
func buildLids(plane geom.Plane, n math.Vector3, section []loopEdge) ([]curvedFace, bool) {
	loops := chainSectionLoops(section)
	if loops == nil {
		return nil, false
	}
	lidPlane, err := geom.NewPlane(plane.Origin, n)
	if err != nil {
		return nil, false
	}
	out := make([]curvedFace, 0, len(loops))
	for i, loop := range loops {
		out = append(out, curvedFace{
			surface: lidPlane,
			loops:   []curvedLoop{{edges: loop}},
			lineage: topo.NewLineage(topo.Tok("halfspace", "lid", i)),
		})
	}
	return out, true
}

// chainSectionLoops groups section arcs into closed boundary loops: a closed-circle section is its own
// loop; open arcs chain end-to-start. Returns nil if open arcs are present but cannot be closed (the
// looped split's job).
func chainSectionLoops(section []loopEdge) [][]loopEdge {
	var loops [][]loopEdge
	var open []loopEdge
	for _, e := range section {
		if samePoint(e.start(), e.end()) {
			loops = append(loops, []loopEdge{e}) // a full section circle is a loop on its own
		} else {
			open = append(open, e)
		}
	}
	if len(open) > 0 {
		return nil // chaining arcs into loops lands with the looped split
	}
	return loops
}

// faceSample returns a point on the face (a boundary vertex, or a surface point for a boundary-less
// face) whose plane side reports the whole face's side when the plane does not cross it.
func faceSample(f curvedFace) math.Point3 {
	if len(f.loops) > 0 && len(f.loops[0].edges) > 0 {
		return f.loops[0].edges[0].start()
	}
	return f.surface.PointAt(0, 0)
}

// signedDistance returns n·(p − plane.Origin): negative on the kept side.
func signedDistance(p math.Point3, plane geom.Plane, n math.Vector3) float64 {
	return float64(plane.Origin.VectorTo(p).Dot(n))
}

// samePoint reports whether two points coincide within the weld tolerance.
func samePoint(a, b math.Point3) bool {
	return float64(a.DistanceTo(b)) < 1e-7
}
