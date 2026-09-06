// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SolidCylinder builds a closed analytic cylinder B-rep (K1b): one true cylindrical side
// face (not faceted) plus two planar circular caps, sharing two closed-circle edges and a
// vertical seam edge. axis must be a unit vector. It is the first analytic curved solid in
// the kernel and the clean drill-tool body the curved boolean will subtract.
//
// Topology (the classic periodic cylinder): the side is a periodic face made simply-
// connected by the seam — its loop runs up the seam, around the top circle, down the seam,
// around the bottom circle, so the seam carries two opposite uses and each circle is shared
// with its cap in opposite orientations (a valid manifold solid per ops.Validate).
//
// The full 2π-periodic side tessellates over its true trim via ops.periodicBandGrid (so its
// area/volume are correct, a hair under exact from chord inscription). This constructor is
// not yet wired into the boolean — that is K1b slice 3 (a cylinder Cut → a clean drilled hole).
func SolidCylinder(baseCenter math.Point3, axis math.Vector3, radius, height float64) (*topo.Body, error) {
	return SolidCylinderNamed(baseCenter, axis, radius, height, "cylinder")
}

// SolidCylinderNamed is SolidCylinder with the feature token its lineages carry, so two cylinders one
// part builds — two holes' tools — are two namespaces rather than the same three keys twice
// (ADR-0043: one namespace per feature instance).
//
//	tool, err := brep.SolidCylinderNamed(base, axis, r, depth, "Hole2")
func SolidCylinderNamed(baseCenter math.Point3, axis math.Vector3, radius, height float64, feat string) (*topo.Body, error) {
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
	bottom, err := geom.NewCircle(baseCenter, axis, radius)
	if err != nil {
		return nil, err
	}
	topCenter := baseCenter.TranslateBy(axis.Scale(math.Scalar(height)))
	// Share the bottom circle's frame so the seam is a single vertical line at angle 0.
	top := geom.Circle{Center: topCenter, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius}

	side, err := geom.NewCylinder(baseCenter, axis, radius)
	if err != nil {
		return nil, err
	}
	capBottom, err := geom.NewPlane(baseCenter, axis.Scale(-1)) // outward = −axis
	if err != nil {
		return nil, err
	}
	capTop, err := geom.NewPlane(topCenter, axis) // outward = +axis
	if err != nil {
		return nil, err
	}

	vbp, vtp := bottom.PointAt(0), top.PointAt(0) // seam endpoints (angle 0 on each circle)
	bld := topo.NewBuilder(true, lin("body", 0))
	vb := bld.AddVertex(vbp, lin("v", 0))
	vt := bld.AddVertex(vtp, lin("v", 1))
	eb := bld.AddEdge(bottom, vb, vb, lin("e", 0)) // closed bottom circle
	et := bld.AddEdge(top, vt, vt, lin("e", 1))    // closed top circle
	es := bld.AddEdge(geom.NewLineSegment(vbp, vtp), vb, vt, lin("e", 2))

	bld.AddFace(capBottom, lin("f", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, lin("f", 1), topo.OuterLoop(topo.Fwd(et)))
	// Periodic side: seam up, top circle (opposite the cap), seam down, bottom circle.
	bld.AddFace(side, lin("f", 2), topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build(), nil
}
