// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// KnownPartKind labels the analytic fast-path cases the closed-form catalog builds exactly, without
// marching — OCCT's ChFiKPart_ComputeData short-circuit. Anything not recognized is NotKnownPart and
// routes to the general marcher (Phase 4).
type KnownPartKind uint8

const (
	// NotKnownPart: no closed form applies — the general marcher must build it.
	NotKnownPart KnownPartKind = iota
	// KnownPlanarEdge: a straight edge between two planes → a cylindrical rolling-ball blend.
	KnownPlanarEdge
	// KnownCylinderRim: a full circular rim between a cylinder and a planar cap → a toroidal band.
	KnownCylinderRim
	// KnownCylinderArc: a circular arc between a cylinder and a plane → a torus with setback caps.
	KnownCylinderArc
)

// String returns a stable diagnostic name.
func (k KnownPartKind) String() string {
	switch k {
	case KnownPlanarEdge:
		return "planar-edge"
	case KnownCylinderRim:
		return "cylinder-rim"
	case KnownCylinderArc:
		return "cylinder-arc"
	default:
		return "not-known-part"
	}
}

// ClassifyKnownPart recognizes, from a spine's geometry alone, which analytic fast path builds its
// blend — mirroring the routing the ops catalog already performs (loneRimPick / loneArcPick / the
// planar analytic path). It classifies GEOMETRY only; radius admissibility is the builder's concern.
// A multi-edge or curved-neighbour spine is NotKnownPart, so the marcher owns it (Phase 4).
func ClassifyKnownPart(sp *Spine) KnownPartKind {
	if sp.NbEdges() != 1 {
		return NotKnownPart
	}
	e := sp.Edge(0)
	faces := e.Faces()
	if len(faces) != 2 {
		return NotKnownPart
	}
	switch e.Geometry().(type) {
	case geom.LineSegment:
		if bothPlanar(faces) {
			return KnownPlanarEdge
		}
	case geom.Circle:
		if cylinderAndPlane(faces) && e.StartVertex() == e.EndVertex() {
			return KnownCylinderRim
		}
	case geom.Arc3d:
		if cylinderAndPlane(faces) {
			return KnownCylinderArc
		}
	}
	return NotKnownPart
}

// bothPlanar reports whether both faces carry a planar surface.
func bothPlanar(faces []*topo.Face) bool {
	_, a := faces[0].Geometry().(geom.Plane)
	_, b := faces[1].Geometry().(geom.Plane)
	return a && b
}

// cylinderAndPlane reports whether the two faces are one cylinder and one plane (in either order).
func cylinderAndPlane(faces []*topo.Face) bool {
	_, cyl0 := faces[0].Geometry().(geom.Cylinder)
	_, pl0 := faces[0].Geometry().(geom.Plane)
	_, cyl1 := faces[1].Geometry().(geom.Cylinder)
	_, pl1 := faces[1].Geometry().(geom.Plane)
	return (cyl0 && pl1) || (pl0 && cyl1)
}
