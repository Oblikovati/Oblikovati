// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic curved geometry (#129) is now ALWAYS ON: an extruded circle / straight-edged revolve is a
// TRUE cylinder/cone/plane/torus B-rep (thread + analytic chamfer/fillet attach to it), and a curved
// body is re-faceted on demand for the planar boolean/hull (combine → planarized). The migration ran
// behind an OBK_ANALYTIC_CURVES gate through steps 1–3; the gate was removed once topology-stable
// re-faceting, analytic dress-up (#127), and through-all extent were all in place.

// Extruding a circular profile used to produce a faceted prism (a 64-gon), so the "cylinder"
// had only planar faces — which blocks thread (no cylindrical face to attach) and makes
// chamfer/fillet on the rim non-manifold (Oblikovati/Oblikovati#129, #127). When a profile's
// outer loop is a single full circle (no holes, no taper) we instead build an ANALYTIC cylinder:
// a true geom.Cylinder side face bounded by two circular edges, with planar disk caps — the same
// construction as kernel/brep.SolidCylinder. Downstream features that need a real cylindrical
// face then work. Other profiles (arcs, polygons, holes, tapered) keep the faceted path for now.

// circleLoop returns the centre and radius of the loop's single full-circle entity, and ok=false
// when the loop is not exactly one full circle (a polygon, an arc chain, or a multi-entity loop). A
// projected circle is a native sketch.Circle (ADR-0055 phase 3), so it matches here directly.
func circleLoop(l sketch.Loop) (center math.Point2, radius float64, ok bool) {
	ents := l.Entities()
	if len(ents) != 1 {
		return math.Point2{}, 0, false
	}
	if c, isCircle := ents[0].Entity.(*sketch.Circle); isCircle {
		return c.Center.Position(), float64(c.Radius), true
	}
	return math.Point2{}, 0, false
}

// buildAnalyticCylinder builds a true-cylinder solid from a full-circle profile (centre, radius)
// swept over the span along the sketch-plane normal. nil if the geometry is degenerate (caller
// falls back to the faceted prism). Mirrors kernel/brep.SolidCylinder, with this feature's lineage.
func buildAnalyticCylinder(center math.Point2, radius float64, plane sketch.Plane, sp span, feat string) *topo.Body {
	height := stdmath.Abs(sp.far - sp.near)
	if radius <= 0 || height <= 0 {
		return nil
	}
	normal := plane.Normal().AsVector()
	refDir := plane.XAxis() // angle-0 = sketch +X, so this cylinder records its generating frame (#129)
	lo := stdmath.Min(sp.near, sp.far)
	base := plane.ToModel(center).TranslateBy(normal.Scale(math.Scalar(lo)))
	topCenter := base.TranslateBy(normal.Scale(math.Scalar(height)))

	// Pin the analytic frame to the sketch plane (RefDir = sketch +X, winding CCW about the
	// normal) so it matches sampleCircle's faceted phase. planarizeSimpleCylinder then re-facets
	// this cylinder into a prism topologically identical to a direct faceted extrude (#129).
	bottom := geom.Circle{Center: base, Normal: plane.Normal(), RefDir: refDir, Radius: radius}
	// Share the bottom circle's frame so the seam is a single vertical line at angle 0.
	top := geom.Circle{Center: topCenter, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius}
	side, capBottom, capTop, ok := analyticCylinderFaces(base, topCenter, normal, refDir, radius)
	if !ok {
		return nil
	}
	return assembleAnalyticCylinder(bottom, top, side, capBottom, capTop, feat)
}

// analyticCylinderFaces builds the side cylinder (sharing refDir's angle-0 frame) and the two
// outward-facing planar caps. ok is false if any surface is degenerate.
func analyticCylinderFaces(base, topCenter math.Point3, normal math.Vector3, refDir math.UnitVector3, radius float64) (side geom.Cylinder, capBottom, capTop geom.Plane, ok bool) {
	side, err := geom.NewCylinderWithRef(base, normal, refDir.AsVector(), radius)
	if err != nil {
		return geom.Cylinder{}, geom.Plane{}, geom.Plane{}, false
	}
	if capBottom, err = geom.NewPlane(base, normal.Scale(-1)); err != nil { // outward = −axis
		return geom.Cylinder{}, geom.Plane{}, geom.Plane{}, false
	}
	if capTop, err = geom.NewPlane(topCenter, normal); err != nil { // outward = +axis
		return geom.Cylinder{}, geom.Plane{}, geom.Plane{}, false
	}
	return side, capBottom, capTop, true
}

// assembleAnalyticCylinder wires the two closed circle edges, the vertical seam, the two disk caps,
// and the periodic side face into the watertight cylinder body (the SolidCylinder topology).
func assembleAnalyticCylinder(bottom, top geom.Circle, side geom.Cylinder, capBottom, capTop geom.Plane, feat string) *topo.Body {
	lin := func(kind string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, kind, i)) }
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
