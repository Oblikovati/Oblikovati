// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Torus half-space cut (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). Trims a bare torus by a plane
// PERPENDICULAR to its axis — the only torus cut with an analytic section (two concentric circles; an
// oblique or axis-parallel plane cuts a quartic SPIRIC curve, deferred to CSG). The kept solid is the tube
// arc on the plane's negative side (a singly-trimmed torus BAND) capped by the planar ANNULUS between the
// two section circles. It is the torus analogue of cylinderHalfSpace/coneHalfSpace, built directly here
// because the section is a nested circle pair the general arrangement's lid chainer does not assemble.

// torusSolidParams recovers a bare torus body's geom.Torus. ok=false unless the body is exactly one torus
// face (a boundary-less analytic torus, as SolidTorus builds it) — the shape this fast path trims.
func torusSolidParams(faces []curvedFace) (geom.Torus, bool) {
	if len(faces) != 1 {
		return geom.Torus{}, false
	}
	t, ok := faces[0].surface.(geom.Torus)
	return t, ok
}

// perpendicularToTorusAxis reports whether the cut plane normal is parallel to the torus axis — the
// constant-axial-level cut whose section is the two concentric circles this path handles.
func perpendicularToTorusAxis(n math.Vector3, t geom.Torus) bool {
	return stdmath.Abs(float64(n.Dot(t.AxisDir.AsVector()))) >= 1-cylinderAxisCosTol
}

// torusSectionTol is the model-relative length margin for a torus spiric-section offset test (#1399):
// the radius/offset comparisons that classify the section topology (single oval, two ovals, figure-eight)
// scale with the torus's own extent (major + minor radius) instead of a cm-anchored 1e-7.
func torusSectionTol(t geom.Torus) float64 {
	return geom.ResolutionForSize(t.MajorRadius + t.MinorRadius).Plane()
}

// torusHalfSpace keeps the tube arc of a torus on the plane's negative side, rebuilt as a trimmed torus
// band plus a planar annular lid. The plane must be perpendicular to the axis. A plane clear of the tube
// (|d| ≥ minor radius) keeps the whole torus or empties it, by which side the kept half-space faces.
func torusHalfSpace(body *topo.Body, t geom.Torus, plane geom.Plane) (*topo.Body, error) {
	n := unit(plane.Normal())
	axis := t.AxisDir.AsVector()
	nAxis := float64(n.Dot(axis))
	d := float64(t.Center.VectorTo(plane.Origin).Dot(axis)) // section level along the axis
	r := t.MinorRadius
	radialTol := geom.ResolutionForBox(body.RangeBox()).Plane() // model-relative clearance (#1399)
	if d >= r-radialTol {
		return keepWholeOrEmpty(body, nAxis > 0) // the whole tube lies below d: kept iff the low side is kept
	}
	if d <= -r+radialTol {
		return keepWholeOrEmpty(body, nAxis < 0)
	}
	half := stdmath.Sqrt(r*r - d*d)
	level := t.Center.TranslateBy(axis.Scale(math.Scalar(d)))
	outer := geom.Circle{Center: level, Normal: t.AxisDir, RefDir: t.Ref, Radius: t.MajorRadius + half}
	inner := geom.Circle{Center: level, Normal: t.AxisDir, RefDir: t.Ref, Radius: t.MajorRadius - half}
	vMid := 1.5 * stdmath.Pi // the kept seam runs through the tube bottom when the LOW side is kept (nAxis>0)
	if nAxis < 0 {
		vMid = 0.5 * stdmath.Pi // through the tube top when the high side is kept
	}
	return buildTorusBandSolid(t, level, n, outer, inner, vMid)
}

// keepWholeOrEmpty returns the whole body when keep is true, else an empty body — the plane-clears case.
func keepWholeOrEmpty(body *topo.Body, keep bool) (*topo.Body, error) {
	if keep {
		return body, nil
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok("halfspace", "empty", 0)), true), nil
}

// buildTorusBandSolid assembles the kept solid: the trimmed torus BAND (a seam-bridged loop between the
// two section circles, mirroring the rim-fillet band so each circle is shared with the lid in the opposite
// sense) and the planar ANNULUS lid (outer circle, inner-circle hole) on the cut plane with outward normal
// +n. The seam is the tube-circle arc at u=0 over the kept v-range (through vMid).
func buildTorusBandSolid(t geom.Torus, level math.Point3, n math.Vector3, outer, inner geom.Circle, vMid float64) (*topo.Body, error) {
	lidPlane, err := geom.NewPlane(level, n)
	if err != nil {
		return nil, err
	}
	seam, err := geom.Arc3dByThreePoints(outer.PointAt(0), t.PointAt(0, vMid), inner.PointAt(0))
	if err != nil {
		return nil, err
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "torusbody", 0)))
	vo := bld.AddVertex(outer.PointAt(0), torusLin("vo"))
	vi := bld.AddVertex(inner.PointAt(0), torusLin("vi"))
	outerE := bld.AddEdge(outer, vo, vo, torusLin("outer"))
	innerE := bld.AddEdge(inner, vi, vi, torusLin("inner"))
	seamE := bld.AddEdge(seam, vo, vi, torusLin("seam"))
	bld.AddFace(t, torusLin("band"),
		topo.OuterLoop(topo.Fwd(seamE), topo.Rev(innerE), topo.Rev(seamE), topo.Fwd(outerE)))
	bld.AddFace(lidPlane, torusLin("lid"),
		topo.OuterLoop(topo.Rev(outerE)), topo.InnerLoop(topo.Fwd(innerE)))
	return bld.Build(), nil
}

// torusLin builds a torus-halfspace lineage token.
func torusLin(role string) topo.Lineage { return topo.NewLineage(topo.Tok("halfspace", role, 0)) }
