// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
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
	bld := topo.NewBuilder(true, cylLin("body", 0))
	vb := bld.AddVertex(vbp, cylLin("v", 0))
	vt := bld.AddVertex(vtp, cylLin("v", 1))
	eb := bld.AddEdge(bottom, vb, vb, cylLin("e", 0)) // closed bottom circle
	et := bld.AddEdge(top, vt, vt, cylLin("e", 1))    // closed top circle
	es := bld.AddEdge(geom.NewLineSegment(vbp, vtp), vb, vt, cylLin("e", 2))

	bld.AddFace(capBottom, cylLin("f", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, cylLin("f", 1), topo.OuterLoop(topo.Fwd(et)))
	// Periodic side: seam up, top circle (opposite the cap), seam down, bottom circle.
	bld.AddFace(side, cylLin("f", 2), topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build(), nil
}

func cylLin(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("cylinder", role, i)) }
