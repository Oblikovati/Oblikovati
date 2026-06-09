// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"os"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// analyticCurvesEnabled gates the analytic-cylinder extrude (#129) behind OBK_ANALYTIC_CURVES while the
// downstream stack is finished. ON, an extruded circle is a TRUE cylinder (thread/chamfer/fillet on it
// work) and curved bodies are re-faceted on demand for the planar boolean/hull/dress-up; OFF (default),
// extrude stays faceted exactly as before — so the feature lands incrementally without regressing
// topology-index-sensitive callers. Remove the gate once topology-stable re-faceting + curved-aware
// dress-up are complete (see #127/#129).
func analyticCurvesEnabled() bool { return os.Getenv("OBK_ANALYTIC_CURVES") != "" }

// Extruding a circular profile used to produce a faceted prism (a 64-gon), so the "cylinder"
// had only planar faces — which blocks thread (no cylindrical face to attach) and makes
// chamfer/fillet on the rim non-manifold (Oblikovati/Oblikovati#129, #127). When a profile's
// outer loop is a single full circle (no holes, no taper) we instead build an ANALYTIC cylinder:
// a true geom.Cylinder side face bounded by two circular edges, with planar disk caps — the same
// construction as kernel/brep.SolidCylinder. Downstream features that need a real cylindrical
// face then work. Other profiles (arcs, polygons, holes, tapered) keep the faceted path for now.

// circleLoop returns the loop's single full-circle entity, or nil if the loop is not exactly
// one Circle (e.g. it is a polygon, an arc chain, or a multi-entity loop).
func circleLoop(l sketch.Loop) *sketch.Circle {
	ents := l.Entities()
	if len(ents) != 1 {
		return nil
	}
	c, _ := ents[0].Entity.(*sketch.Circle)
	return c
}

// buildAnalyticCylinder builds a true-cylinder solid from a full-circle profile swept over the
// span along the sketch-plane normal. nil if the geometry is degenerate (caller falls back to the
// faceted prism). Mirrors kernel/brep.SolidCylinder, with this feature's lineage tokens.
func buildAnalyticCylinder(c *sketch.Circle, plane sketch.Plane, sp span, feat string) *topo.Body {
	radius := float64(c.Radius)
	height := stdmath.Abs(sp.far - sp.near)
	if radius <= 0 || height <= 0 {
		return nil
	}
	normal := plane.Normal().AsVector()
	lo := stdmath.Min(sp.near, sp.far)
	base := plane.ToModel(c.Center.Position()).TranslateBy(normal.Scale(math.Scalar(lo)))
	topCenter := base.TranslateBy(normal.Scale(math.Scalar(height)))

	lin := func(kind string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, kind, i)) }
	bottom, err := geom.NewCircle(base, normal, radius)
	if err != nil {
		return nil
	}
	// Share the bottom circle's frame so the seam is a single vertical line at angle 0.
	top := geom.Circle{Center: topCenter, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius}
	side, err := geom.NewCylinder(base, normal, radius)
	if err != nil {
		return nil
	}
	capBottom, err := geom.NewPlane(base, normal.Scale(-1)) // outward = −axis
	if err != nil {
		return nil
	}
	capTop, err := geom.NewPlane(topCenter, normal) // outward = +axis
	if err != nil {
		return nil
	}

	vbp, vtp := bottom.PointAt(0), top.PointAt(0)
	bld := topo.NewBuilder(true, lin("body", 0))
	vb := bld.AddVertex(vbp, lin("vertex", 0))
	vt := bld.AddVertex(vtp, lin("vertex", 1))
	eb := bld.AddEdge(bottom, vb, vb, lin("bottom-edge", 0)) // closed bottom circle
	et := bld.AddEdge(top, vt, vt, lin("top-edge", 0))       // closed top circle
	es := bld.AddEdge(geom.NewLineSegment(vbp, vtp), vb, vt, lin("side-edge", 0))

	bld.AddFace(capBottom, lin("cap", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, lin("cap", 1), topo.OuterLoop(topo.Fwd(et)))
	bld.AddFace(side, lin("side", 0), topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build()
}
