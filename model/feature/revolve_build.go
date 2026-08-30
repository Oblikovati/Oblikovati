// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Revolve feature — the TOOL GEOMETRY builders (M48 #2239 split of sketched_features.go). Builds the
// revolve solid/sheet from the profile: the analytic torus/sphere fast paths, the meridian vertices from
// the profile loop, and the general revolved-section sweep. The feature and axis resolution live in
// revolve.go.

// buildRevolveSheet revolves the profile boundary into an open surface of revolution (no caps) —
// Inventor's Surface-operation revolve (kSurfaceOperation, #1858). Uses the faceted section sweep
// (sweptShell), matching the faceted swept-solid path; combine() adds the result as a surface body.
func buildRevolveSheet(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64, feat string) (*topo.Body, error) {
	sections, closed := revolveSectionsFrom(prof, plane, axis, angle, start)
	return sweptShell(sections, closed, feat)
}

// buildRevolveSolid revolves a profile (already projected onto plane) about axis over the
// swept angle starting at start. A full 360° revolve becomes a TRUE analytic solid whenever every
// profile edge revolves to an exact surface — straight edges → cylinder/cone/plane faces, an off-axis
// arc → a torus, and an on-axis arc closing at the pole → a spherical cap (the domed end of a tapered
// roller, #129) — so thread/chamfer/fillet attach to its revolved faces and the mesh carries true
// curvature (not the 48-facet-per-arc swept solid that starves the frame-loop pick). Every other case
// (partial angle, a spline edge, a sphere zone we do not build analytically) keeps the faceted swept
// solid; SolidOfRevolutionMeridian returns nil to signal that fallback. Booleans re-facet an analytic
// revolve body on demand (combine → planarized).
func buildRevolveSolid(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64, feat string) (*topo.Body, error) {
	if fullRevolution(angle) {
		if body, ok := sphereProfileSolid(prof, plane, axis, feat); ok {
			return body, nil // a pole-to-pole semicircle on the axis revolves to an analytic sphere (#129 follow-up)
		}
		if body, ok := circleProfileTorus(prof, plane, axis, feat); ok {
			return body, nil // a circle clear of the axis revolves to an analytic torus (#129 follow-up)
		}
		if verts, ok := meridianVertsFromProfile(prof, plane, axis); ok {
			if body, err := brep.SolidOfRevolutionMeridian(axis.Origin(), axis.Direction().AsVector(), verts, feat); err == nil && body != nil {
				return body, nil
			}
		}
	} else if body, ok := buildPartialRevolveSolid(prof, plane, axis, angle, start, feat); ok {
		return body, nil // a partial revolve of a line-only meridian keeps analytic sector walls (#2019)
	}
	sections, closed := revolveSectionsFrom(prof, plane, axis, angle, start)
	return sweptSolid(sections, closed, feat)
}

// circleProfileTorus builds an analytic torus when the profile is a single CIRCLE clear of the axis: the
// circle (the tube cross-section) revolved a full turn is a torus whose MAJOR radius is the circle centre's
// distance from the axis and MINOR radius the circle's own radius. This is the #129 curved-meridian
// follow-up for the torus case — a circle profile otherwise facets into hundreds of cone slivers. ok=false
// for any other profile (the caller keeps the straight/faceted path), including a circle that reaches the
// axis (minor ≥ major), which would revolve to a self-intersecting spindle/horn torus we do not build.
func circleProfileTorus(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, feat string) (*topo.Body, bool) {
	ents := prof.OuterLoop().Entities()
	if len(ents) != 1 {
		return nil, false
	}
	circle, ok := ents[0].Entity.(*sketch.Circle)
	if !ok {
		return nil, false
	}
	o, a := axis.Origin(), axis.Direction().AsVector()
	v := o.VectorTo(plane.ToModel(circle.CenterPoint().Position()))
	z := v.Dot(a)
	major := float64(v.Sub(a.Scale(z)).Length())
	minor := float64(circle.CurveRadius())
	if minor >= major {
		return nil, false
	}
	body, err := brep.SolidTorus(o.TranslateBy(a.Scale(z)), a, major, minor, feat)
	if err != nil {
		return nil, false
	}
	return body, true
}

// sphereProfileSolid builds an analytic geom.Sphere (one boundary-less face, brep.SolidSphere) when the
// profile is a pole-to-pole semicircle on the axis: an ARC whose centre AND both endpoints lie on the
// revolve axis, closed by a straight LINE along the axis (the flat side of a half-disc). Revolving it a
// full turn is a whole sphere centred at the arc centre with the arc radius. This is the #129 curved-
// meridian follow-up for the FULL-sphere case — otherwise the half-disc facets into a ~1600-face UV
// sphere that shatters the frame-loop hover-pick (e.g. every representational bearing ball). ok=false for
// any other profile (the caller keeps the meridian/faceted path), including an off-axis arc (that is the
// torus/zone case) or an arc that does not close pole-to-pole.
func sphereProfileSolid(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, feat string) (*topo.Body, bool) {
	ents := prof.OuterLoop().Entities()
	if len(ents) != 2 {
		return nil, false
	}
	arc, line := arcAndLine(ents[0].Entity, ents[1].Entity)
	if arc == nil || line == nil {
		return nil, false
	}
	o, a := axis.Origin(), axis.Direction().AsVector()
	radius := float64(arc.CurveRadius())
	if radius <= 0 {
		return nil, false
	}
	onAxis := func(p2 math.Point2) bool { // perpendicular distance from the revolve axis, relative to the sphere radius
		v := o.VectorTo(plane.ToModel(p2))
		return float64(v.Sub(a.Scale(v.Dot(a))).Length()) <= 1e-7*radius
	}
	if !halfDiscPolesOnAxis(arc, line, onAxis) {
		return nil, false // centre/pole or the closing side off the axis: not a full sphere (torus/zone/cap case)
	}
	body, err := brep.SolidSphere(plane.ToModel(arc.CenterPoint().Position()), radius, feat)
	if err != nil {
		return nil, false
	}
	return body, true
}

// halfDiscPolesOnAxis reports whether the half-disc's arc centre, both arc poles, and both endpoints
// of the closing line all lie on the revolve axis — the full-sphere signature (a pole/centre off the
// axis is instead the torus/zone/cap case).
func halfDiscPolesOnAxis(arc *sketch.Arc, line *sketch.Line, onAxis func(math.Point2) bool) bool {
	return onAxis(arc.CenterPoint().Position()) && onAxis(arc.Start.Position()) && onAxis(arc.End.Position()) &&
		onAxis(line.StartPoint().Position()) && onAxis(line.EndPoint().Position())
}

// arcAndLine returns (arc, line) from an unordered pair of profile entities, or (nil, nil) unless the
// pair is exactly one arc and one line — the half-disc signature sphereProfileSolid recognizes.
func arcAndLine(e0, e1 sketch.Entity) (*sketch.Arc, *sketch.Line) {
	if a, ok := e0.(*sketch.Arc); ok {
		if l, ok := e1.(*sketch.Line); ok {
			return a, l
		}
	}
	if a, ok := e1.(*sketch.Arc); ok {
		if l, ok := e0.(*sketch.Line); ok {
			return a, l
		}
	}
	return nil, nil
}

// revolveSpan and fullRevolution — which way and how far a revolve sweeps — live in
// revolve_span.go (#2019).

// meridianVertsFromProfile walks the profile's outer loop into brep.RevolveVertex meridian vertices in
// the axis's (radius, height) half-plane — radius = perpendicular distance from the axis, height =
// signed distance along it — one vertex per edge, at the point that edge SHARES with the next, carrying
// the arc CENTRE when the edge is a circular arc (so it revolves to a torus/sphere face, not chorded
// cone facets). ok is false when the loop holds a non-line/arc entity (a spline) or is not a contiguous
// chain: the caller keeps the faceted revolve.
//
// The ring is built from the entities' GEOMETRY (their shared endpoints), NOT from each entity's loop
// Reversed flag: the region extractor does not always set those flags as a consistent directed
// traversal for a mixed line/arc loop, and a wrong flag put the groove arc's endpoints on the wrong
// span — the revolve then built the groove torus with the wrong minor radius and the 532xx housing
// washer's mesh volume collapsed to ~28% (Oblikovati.AddIns.PartDesigner #54). Chaining by shared
// points is flag-independent, matching how the loop's polygon is already assembled.
func meridianVertsFromProfile(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis) ([]brep.RevolveVertex, bool) {
	o, a := axis.Origin(), axis.Direction().AsVector()
	toRZ := func(p2 math.Point2) math.Point2 {
		v := o.VectorTo(plane.ToModel(p2))
		z := v.Dot(a)
		return math.P2(v.Sub(a.Scale(z)).Length(), z)
	}
	shapes, ok := loopEntityShapes(prof.OuterLoop().Entities())
	if !ok {
		return nil, false
	}
	tol := loopChainTolerance(shapes)
	verts := make([]brep.RevolveVertex, len(shapes))
	for k := range shapes {
		end, ok := sharedEndpoint(shapes[k], shapes[(k+1)%len(shapes)], tol)
		if !ok {
			return nil, false // entities do not chain end-to-end: not a simple ring, keep faceted
		}
		rv := brep.RevolveVertex{P: toRZ(end)}
		if shapes[k].center != nil {
			c := toRZ(*shapes[k].center)
			rv.ArcCenter = &c
		}
		verts[k] = rv
	}
	return verts, true
}

// loopEntityShape is a line/arc loop entity reduced to what the meridian ring needs: its two endpoints
// (a, b — undirected) and, for an arc, its centre. Reading the endpoints geometrically keeps the
// entity's stored direction and the loop Reversed flag out of the ring construction.
type loopEntityShape struct {
	a, b   math.Point2
	center *math.Point2
}

// loopEntityShapes reduces the loop's entities to their shapes; ok is false when the loop is shorter
// than a triangle or holds a non-line/arc entity (a spline), so the caller keeps the faceted revolve.
func loopEntityShapes(ents []sketch.ProfileEntity) ([]loopEntityShape, bool) {
	if len(ents) < 3 {
		return nil, false
	}
	shapes := make([]loopEntityShape, len(ents))
	for i, pe := range ents {
		s, ok := shapeOfLoopEntity(pe)
		if !ok {
			return nil, false
		}
		shapes[i] = s
	}
	return shapes, true
}

// shapeOfLoopEntity reads a line/arc loop entity through the ShapedEntity capability (Kind +
// ShapePoints — [start, end] for a line, [centre, start, end] for an arc), per the sketch-entity
// type-switch ban (#1624, audit I1). ok is false for a spline (keeps the profile on the faceted path).
func shapeOfLoopEntity(pe sketch.ProfileEntity) (loopEntityShape, bool) {
	shaped, isShaped := pe.Entity.(sketch.ShapedEntity)
	if !isShaped {
		return loopEntityShape{}, false
	}
	pts := shaped.ShapePoints()
	switch shaped.Kind() {
	case sketch.LineKind:
		if len(pts) < 2 {
			return loopEntityShape{}, false
		}
		return loopEntityShape{a: pts[0], b: pts[1]}, true
	case sketch.ArcKind:
		if len(pts) < 3 {
			return loopEntityShape{}, false
		}
		c := pts[0]
		return loopEntityShape{a: pts[1], b: pts[2], center: &c}, true
	}
	return loopEntityShape{}, false
}

// sharedEndpoint returns the point entity s shares with the next entity t — s's END in the ring,
// derived from geometry rather than a direction flag. ok is false when they do not touch (the loop is
// not a contiguous chain).
func sharedEndpoint(s, t loopEntityShape, tol float64) (math.Point2, bool) {
	for _, p := range [2]math.Point2{s.a, s.b} {
		if near2D(p, t.a, tol) || near2D(p, t.b, tol) {
			return p, true
		}
	}
	return math.Point2{}, false
}

// near2D reports whether two sketch points coincide within tol.
func near2D(p, q math.Point2, tol float64) bool {
	return stdmath.Hypot(float64(p.X-q.X), float64(p.Y-q.Y)) <= tol
}

// loopChainTolerance is the endpoint-coincidence tolerance for chaining, scaled to the loop's extent so
// it holds across model scales (ADR-0042) instead of a cm-anchored constant.
func loopChainTolerance(shapes []loopEntityShape) float64 {
	ext := 1.0
	for _, s := range shapes {
		ext = stdmath.Max(ext, stdmath.Max(stdmath.Abs(float64(s.a.X)), stdmath.Abs(float64(s.a.Y))))
	}
	return ext * 1e-6
}

// revolveSectionsFrom is revolveSections starting at a signed angular offset —
// the two-directional revolve sweeps [start, start+angle] (#313).
func revolveSectionsFrom(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64) ([][]math.Point3, bool) {
	return revolvePointsAbout(modelPolygon(prof, plane), axis, angle, start)
}

// revolvePointsAbout sweeps a base section's points about the axis from start through angle,
// returning the rotated cross-sections (revolveSegments facets per full turn) and whether the
// revolution is full. Shared by the part revolve and the sheet-metal contour roll.
func revolvePointsAbout(base []math.Point3, axis *WorkAxis, angle, start float64) ([][]math.Point3, bool) {
	full := angle <= 0 || angle >= 2*stdmath.Pi-1e-9
	k, step := revolveSegments, 2*stdmath.Pi/float64(revolveSegments)
	if full {
		start = 0
	} else {
		segs := stdmath.Max(3, stdmath.Round(revolveSegments*angle/(2*stdmath.Pi)))
		k, step = int(segs)+1, angle/segs
	}
	sections := make([][]math.Point3, k)
	for s := 0; s < k; s++ {
		m := math.Rotation4(start+step*float64(s), axis.Direction(), axis.Origin())
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			sec[i] = m.TransformPoint(p)
		}
		sections[s] = sec
	}
	return sections, full
}

// modelPolygon returns a profile's outer-loop polygon mapped into model space.
func modelPolygon(prof *sketch.Profile, plane sketch.Plane) []math.Point3 {
	poly := prof.OuterLoop().Polygon()
	out := make([]math.Point3, len(poly))
	for i, p := range poly {
		out[i] = plane.ToModel(p)
	}
	return out
}
