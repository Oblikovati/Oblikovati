// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Extrude feature — the PRISM BUILDER (M48 #2235 split of extrude.go). Builds the extrusion tool body
// from a 2D profile: the capped/uncapped prism shell, the tapered/offset loops, the profile sheets
// (the sheet/surface variant), and the low-level cap/side face construction (C6-a: guarded edge reuse).
// The feature, extent resolution and merge/dissolve policy live in extrude.go and extrude_combine.go.

// buildPrism extrudes a closed polygon over the span (near→far offsets along the plane
// normal) into a watertight solid: a near cap, a far cap, and one planar side face per
// profile edge. A non-zero taper offsets the far loop by depth·tan(taper) so the sides
// draft outward (positive) or inward (negative); the far cap stays perpendicular to the
// normal, and each side stays a planar trapezoid.
func buildPrism(poly []math.Point2, plane sketch.Plane, sp span, taper float64, feat string) *topo.Body {
	return buildExtrusionShell(poly, plane, sp, taper, feat, true)
}

// buildExtrusionShell extrudes a closed polygon over the span (near→far offsets along the plane
// normal). With caps=true it is a watertight SOLID prism: a near cap, a far cap, and one planar
// side face per profile edge. With caps=false it is an OPEN, non-solid wall SHEET — the side
// faces only, no caps — Inventor's Surface-operation extrude (kSurfaceOperation, #1858). A
// non-zero taper offsets the far loop by depth·tan(taper) so the sides draft outward (positive)
// or inward (negative); the far cap (when present) stays perpendicular to the normal, and each
// side stays a planar trapezoid.
func buildExtrusionShell(poly []math.Point2, plane sketch.Plane, sp span, taper float64, feat string, caps bool) *topo.Body {
	// Normalise the cross-section to CCW: a CW input (a chamfer wedge whose edge frame happens to
	// wind that way) previously produced a topologically INSIDE-OUT prism — outward face normals
	// with loops traversed clockwise about them — which the orientation-faithful winding
	// classifier (#1599) reads as an empty solid and the boolean then mangles (#1600). One
	// canonical winding makes every downstream orientation decision (caps, sides, taper offset)
	// consistent by construction.
	if outwardSign(poly) < 0 {
		poly = reversePoly(poly)
	}
	n := len(poly)
	normal := plane.Normal().AsVector()
	topPoly := taperedLoop(poly, sp.depth(), taper)
	bld := topo.NewBuilder(caps, topo.NewLineage(topo.Tok(feat, "body", 0)))
	bottom := make([]*topo.Vertex, n)
	top := make([]*topo.Vertex, n)
	for i := range n {
		b := plane.ToModel(poly[i]).TranslateBy(normal.Scale(sp.near))
		t := plane.ToModel(topPoly[i]).TranslateBy(normal.Scale(sp.far))
		bottom[i] = bld.AddVertex(b, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(t, topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}
	be, te, ve := prismEdges(bld, bottom, top, feat)
	if caps {
		addCaps(bld, bottom, top, be, te, normal, feat)
	}
	addSides(bld, bottom, top, be, te, ve, outwardSign(poly), feat)
	return bld.Build()
}

// buildProfileSheets extrudes each selected profile into an open wall sheet (no caps), merging
// them into one non-solid surface body — the tool for an extrude with the Surface operation
// (kSurfaceOperation, #1858). There is no boolean; combine() adds the result as a surface body.
func buildProfileSheets(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, _ *diag.Recorder) *topo.Body {
	sheets := make([]*topo.Body, 0, len(profiles))
	for i, p := range profiles {
		name := feat
		if len(profiles) > 1 {
			name = fmt.Sprintf("%s/p%d", feat, i)
		}
		sheets = append(sheets, buildProfileSheet(p, plane, sp, taper, name))
	}
	if len(sheets) == 1 {
		return sheets[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(feat, "merged", 0)), false, sheets...)
}

// buildProfileSheet builds one profile's open wall sheet: the outer loop plus each inner loop,
// each as an uncapped tube, merged into a single non-solid body. Inner (hole) loops become their
// own open tubes rather than being booleaned out (there is no solid to cut). #1858.
func buildProfileSheet(p *sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string) *topo.Body {
	walls := []*topo.Body{buildExtrusionShell(p.OuterLoop().Polygon(), plane, sp, taper, feat, false)}
	for j, loop := range p.InnerLoops() {
		walls = append(walls, buildExtrusionShell(loop.Polygon(), plane, sp, taper, fmt.Sprintf("%s/hole%d", feat, j), false))
	}
	if len(walls) == 1 {
		return walls[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(feat, "sheet", 0)), false, walls...)
}

// reversePoly returns the polygon with its winding reversed.
func reversePoly(poly []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(poly))
	for i, p := range poly {
		out[len(poly)-1-i] = p
	}
	return out
}

// taperedLoop returns the far-loop polygon: poly unchanged when taper is 0, else each
// vertex offset along its outward bisector by depth·tan(taper) (positive widens).
func taperedLoop(poly []math.Point2, depth, taper float64) []math.Point2 {
	if taper == 0 {
		return poly
	}
	delta := depth * stdmath.Tan(taper) * outwardSign(poly)
	return offsetPolygon2D(poly, delta)
}

// offsetPolygon2D offsets each vertex outward by delta along the angle bisector of its two
// incident edges (a simple miter offset, exact for convex corners).
func offsetPolygon2D(poly []math.Point2, delta float64) []math.Point2 {
	n := len(poly)
	out := make([]math.Point2, n)
	for i := range n {
		prev := poly[(i-1+n)%n]
		next := poly[(i+1)%n]
		nIn := edgeNormal2D(prev, poly[i])
		nOut := edgeNormal2D(poly[i], next)
		bisect := math.V2(nIn.X+nOut.X, nIn.Y+nOut.Y)
		u, err := math.UnitVector2FromVector(bisect)
		if err != nil {
			out[i] = poly[i]
			continue
		}
		scale := delta / cosHalf(nIn, nOut)
		out[i] = math.P2(poly[i].X+u.AsVector().X*scale, poly[i].Y+u.AsVector().Y*scale)
	}
	return out
}

// edgeNormal2D returns the left-hand unit normal of edge a→b (points outward for a CCW loop).
func edgeNormal2D(a, b math.Point2) math.Vector2 {
	d := math.V2(b.X-a.X, b.Y-a.Y)
	u, err := math.UnitVector2FromVector(math.V2(d.Y, -d.X))
	if err != nil {
		return math.V2(0, 0)
	}
	return u.AsVector()
}

// cosHalf returns the cosine of the half-angle between two edge normals, the miter
// factor that keeps the bisector offset's perpendicular distance equal to delta (clamped
// so a near-straight corner does not blow up the offset).
func cosHalf(nIn, nOut math.Vector2) float64 {
	c := stdmath.Sqrt(stdmath.Max(0, (1+nIn.X*nOut.X+nIn.Y*nOut.Y)/2))
	if c < 0.2 {
		return 0.2
	}
	return c
}

// outwardSign returns +1 when the profile polygon is wound counter-clockwise in the
// sketch plane and −1 when clockwise. Side-wall normals are built as edgeDir×normal,
// which points away from the interior only for a CCW loop; profile detection does not
// guarantee a winding, so a clockwise loop must flip the side normals to stay coherent
// with the (fixed up/down) caps — otherwise the prism is "inside-out" and breaks
// rendering, volume, and downstream booleans.
func outwardSign(poly []math.Point2) float64 {
	area := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		j := (i + 1) % n
		area += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	if area < 0 {
		return -1
	}
	return 1
}

// prismEdges builds the bottom, top and vertical edges and returns them.
func prismEdges(bld *topo.Builder, bottom, top []*topo.Vertex, feat string) (be, te, ve []*topo.Edge) {
	n := len(bottom)
	be, te, ve = make([]*topo.Edge, n), make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := range n {
		j := (i + 1) % n
		be[i] = bld.AddEdge(geom.NewLineSegment(bottom[i].Point(), bottom[j].Point()), bottom[i], bottom[j], topo.NewLineage(topo.Tok(feat, "bottom-edge", i)))
		te[i] = bld.AddEdge(geom.NewLineSegment(top[i].Point(), top[j].Point()), top[i], top[j], topo.NewLineage(topo.Tok(feat, "top-edge", i)))
		ve[i] = bld.AddEdge(geom.NewLineSegment(bottom[i].Point(), top[i].Point()), bottom[i], top[i], topo.NewLineage(topo.Tok(feat, "side-edge", i)))
	}
	return be, te, ve
}

// addCaps builds the near (downward) and far (upward) cap faces, perpendicular to the
// extrude normal at each end.
func addCaps(bld *topo.Builder, bottom, top []*topo.Vertex, be, te []*topo.Edge, normal math.Vector3, feat string) {
	n := len(bottom)
	bottomPlane, _ := geom.NewPlane(bottom[0].Point(), normal.Negate())
	topPlane, _ := geom.NewPlane(top[0].Point(), normal)
	bottomLoop := make([]topo.Use, n)
	topLoop := make([]topo.Use, n)
	for i := range n {
		bottomLoop[i] = topo.Rev(be[n-1-i]) // reverse order & direction → outward-down
		topLoop[i] = topo.Fwd(te[i])
	}
	bld.AddFace(bottomPlane, topo.NewLineage(topo.Tok(feat, "start-cap", 0)), topo.OuterLoop(bottomLoop...))
	bld.AddFace(topPlane, topo.NewLineage(topo.Tok(feat, "end-cap", 0)), topo.OuterLoop(topLoop...))
}

// addSides builds one planar side wall per profile edge, through the wall's corners so
// the face tilts correctly when the extrude is tapered. sign flips the outward normal for
// a clockwise profile (see outwardSign) so every wall faces away from the interior.
func addSides(bld *topo.Builder, bottom, top []*topo.Vertex, be, te, ve []*topo.Edge, sign float64, feat string) {
	n := len(bottom)
	for i := range n {
		j := (i + 1) % n
		surf := sideSurface(bottom[i].Point(), bottom[j].Point(), top[i].Point(), sign)
		loop := topo.OuterLoop(topo.Fwd(be[i]), topo.Fwd(ve[j]), topo.Rev(te[i]), topo.Rev(ve[i]))
		bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "side", i)), loop)
	}
}

// sideSurface returns the side face's plane through corners b0,b1,t0 (b0→b1 along the
// profile edge, b0→t0 up the wall), oriented outward by sign. Falls back to a degenerate
// plane on a zero-area corner, which the validator then flags.
func sideSurface(b0, b1, t0 math.Point3, sign float64) geom.Surface {
	normal, err := math.UnitVector3FromVector(b0.VectorTo(b1).Cross(b0.VectorTo(t0)).Scale(sign))
	if err != nil {
		p, _ := geom.NewPlane(b0, math.V3(0, 0, 1))
		return p
	}
	p, _ := geom.NewPlane(b0, normal.AsVector())
	return p
}
