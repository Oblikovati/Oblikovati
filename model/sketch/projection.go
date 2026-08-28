// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Projection links model geometry into a sketch as reference geometry that updates
// when the source changes. The model side is reached through a seam — [PointSource]
// / [CurveSource] — so the sketch never depends on the B-rep kernel directly; the
// kernel's vertices/edges implement these in M07 (the same seam discipline used for
// reference keys). Projected geometry is reference geometry: grounded, but a first-class
// curve profiles and constraints use natively (ADR-0055 phase 3).

// PointSource is a model entity that yields a 3D position to project (e.g. a topo
// vertex). SourceID is a stable identity used to recognize the source across recompute
// for associative re-projection. Position returns ok=false when the reference is lost
// (the source no longer resolves against the current B-rep) so the projection can freeze
// rather than jump.
type PointSource interface {
	SourceID() string
	Position() (math.Point3, bool)
}

// CurveSource is a model entity that yields a sampled 3D polyline to project (e.g. a topo
// edge's tessellation, or a plane-cut edge). SamplePoints returns ok=false when the
// reference is lost.
type CurveSource interface {
	SourceID() string
	SamplePoints() ([]math.Point3, bool)
}

// AnalyticCurveSource is the optional [CurveSource] capability that yields the source edge's EXACT
// analytic curve, so the projection stays analytic — a projected circle is a circle, an arc an arc
// — instead of the sampled polyline SamplePoints returns (ADR-0055). A single model edge implements
// it; a multi-edge cut/silhouette loop does not (it has no one analytic curve) and falls back to
// SamplePoints.
type AnalyticCurveSource interface {
	SourceCurve() (geom.Curve3, bool)
}

// ProjectedPoint is a sketch point projected from a model vertex. It owns a fixed
// reference anchor (a Point in the sketch's refPts) so other sketch geometry can be
// constrained to it; the projection re-derives the anchor's position on Update until the
// link is broken. The anchor's id is the entity's id.
type ProjectedPoint struct {
	source PointSource
	plane  Plane
	anchor *Point
	linked bool
	// srcKind/srcID persist the source's identity (its kind tag + SourceID) so a saved sketch
	// round-trips: serialization writes them, restore keeps them, and compdef rebinds a live
	// self-resolving source from them after a reload (#1268).
	srcKind, srcID string
}

// EntityID is the anchor's id — constraining to this id binds to the fixed anchor.
func (p *ProjectedPoint) EntityID() ID { return p.anchor.id }

// Anchor returns the constrainable fixed reference point.
func (p *ProjectedPoint) Anchor() *Point { return p.anchor }

// Position returns the anchor's sketch-space position.
func (p *ProjectedPoint) Position() math.Point2 { return p.anchor.Position() }

// SourceID returns the projected source's identity, or "" if the link was broken.
func (p *ProjectedPoint) SourceID() string {
	if !p.linked {
		return ""
	}
	return p.source.SourceID()
}

// IsReference reports that projected geometry is reference/construction geometry.
func (p *ProjectedPoint) IsReference() bool { return true }

// Linked reports whether the projection still tracks its source.
func (p *ProjectedPoint) Linked() bool { return p.linked }

// Update re-projects from the current source position. It is a no-op once the link is
// broken; if the source's reference is lost it breaks the link, freezing the last
// projected position (the reference-lost behavior).
func (p *ProjectedPoint) Update() {
	if !p.linked {
		return
	}
	pos, ok := p.source.Position()
	if !ok {
		p.linked = false
		return
	}
	p.anchor.SetPosition(p.plane.ToSketch(pos))
}

// BreakLink detaches the projection from its source, freezing its current geometry
// (the "break link" / include-without-associativity option).
func (p *ProjectedPoint) BreakLink() { p.linked = false }

// Projection is the associative link from a model edge to the concrete grounded reference entity it
// drives in the sketch (ADR-0055 phase 3). Unlike the old ProjectedCurve wrapper it is NOT itself an
// entity: the sketch carries a real reference Line/Circle/Arc/Ellipse/EllipticalArc/Spline (in
// s.ents and its typed collection, grounded and driven), so every consumer — constraints, offset,
// profiles, extrude — uses it natively with no projected-curve special case. The Projection
// re-derives that entity's geometry from the source on Update, and persists the source link.
type Projection struct {
	source CurveSource
	entity Entity // the concrete grounded reference entity this projection drives
	plane  Plane
	linked bool
	// srcKind/srcID persist the source's identity for save/reload + rebind (see ProjectedPoint).
	srcKind, srcID string
}

// Entity returns the concrete grounded reference entity the projection drives.
func (p *Projection) Entity() Entity { return p.entity }

// SourceID returns the source identity, or "" once the link is broken.
func (p *Projection) SourceID() string {
	if !p.linked {
		return ""
	}
	return p.source.SourceID()
}

// Linked reports whether the projection still tracks its source.
func (p *Projection) Linked() bool { return p.linked }

// BreakLink detaches the projection from its source, freezing the current reference entity.
func (p *Projection) BreakLink() { p.linked = false }

// SourceDescriptor returns the (kind, id) a host uses to rebuild this projection's live source after
// a reload; an empty kind means there is nothing to rebind (the projection stays frozen).
func (p *Projection) SourceDescriptor() (kind, id string) { return p.srcKind, p.srcID }

// Rebind attaches a freshly built live source to a projection restored frozen, making it associative
// again; the next UpdateProjections re-projects from it.
func (p *Projection) Rebind(src CurveSource) { p.source, p.linked = src, true }

// Update re-drives the reference entity from the current source. It prefers the ANALYTIC path (the
// source's exact curve projected onto the plane, keeping a circle a circle — ADR-0055) and only
// samples a source with no analytic curve. Geometry is written IN PLACE when the curve type is
// unchanged, so the entity's id and points survive; a type change or a lost reference is handled by
// rebuilding or freezing.
func (p *Projection) Update(s *Sketch) {
	if !p.linked {
		return
	}
	if c2, ok := projectSourceCurve(p.plane, p.source); ok {
		if setReferenceGeometry(p.entity, c2) {
			return
		}
		p.rebuild(s, s.addReferenceCurve(c2))
		return
	}
	pts, ok := p.source.SamplePoints()
	if !ok {
		p.linked = false
		return
	}
	proj := projectPointsToPlane(p.plane, pts)
	if sp, isSpline := p.entity.(*Spline); isSpline && setReferenceSplinePoints(sp, proj) {
		return
	}
	p.rebuild(s, s.addReferencePolyline(proj))
}

// rebuild swaps in a freshly built reference entity when Update cannot drive the old one in place
// (its curve type changed). The old entity and its points are dropped; the new entity takes the
// projection. A recompute type change is rare (an edge projected head-on then obliquely), and a
// constraint bound to the old entity is necessarily invalidated by the geometry change anyway.
func (p *Projection) rebuild(s *Sketch, next Entity) {
	if next == nil {
		p.linked = false
		return
	}
	s.deleteEntity(p.entity)
	p.entity = next
}

// projectSourceCurve projects the source's exact analytic 3D curve onto the plane, keeping its type
// (circle→circle, arc→arc, line→line, oblique conic→ellipse). ok is false when the source is not an
// [AnalyticCurveSource], its reference is lost, or the projection is edge-on/free-form.
func projectSourceCurve(plane Plane, src CurveSource) (geom.Curve2, bool) {
	as, ok := src.(AnalyticCurveSource)
	if !ok {
		return nil, false
	}
	c3, ok := as.SourceCurve()
	if !ok {
		return nil, false
	}
	return geom.ProjectCurveToPlane(sketchPlaneToGeom(plane), c3)
}

// projectPointsToPlane maps sampled model-space points onto the sketch plane's 2D frame.
func projectPointsToPlane(plane Plane, pts []math.Point3) []math.Point2 {
	out := make([]math.Point2, len(pts))
	for i, q := range pts {
		out[i] = plane.ToSketch(q)
	}
	return out
}

// setReferenceSplinePoints drives a reference spline's points in place, returning false when the
// projected point count changed (the caller rebuilds the spline).
func setReferenceSplinePoints(sp *Spline, pts []math.Point2) bool {
	if len(sp.Points) != len(pts) {
		return false
	}
	for i, p := range pts {
		sp.Points[i].SetPosition(p)
	}
	return true
}

// sketchPlaneToGeom builds the geom.Plane whose (u,v) frame is the sketch plane's (xAxis, yAxis), so
// geom.ProjectCurveToPlane projects into the sketch's own 2D coordinate system.
func sketchPlaneToGeom(p Plane) geom.Plane {
	pl, _ := geom.NewPlaneFromAxes(p.origin, p.xAxis.AsVector(), p.yAxis.AsVector())
	return pl
}

// ProjectPoint projects a model vertex into the sketch as a fixed, constrainable reference
// point and returns the associative projected point. Projecting a source the sketch already
// carries returns that projection instead of stacking a second, coincident reference on top of
// it — so projecting the origin centre by hand does not double the auto-projected one (#2016).
func (s *Sketch) ProjectPoint(src PointSource) *ProjectedPoint {
	if p, ok := s.projectionOfSource(pointSourceKind(src), src.SourceID()); ok {
		return p
	}
	pos, _ := src.Position() // resolved now; a later lost reference freezes via Update
	p := &ProjectedPoint{
		source: src, plane: s.plane, anchor: s.newRefPoint(s.plane.ToSketch(pos)), linked: true,
		srcKind: pointSourceKind(src), srcID: src.SourceID(),
	}
	p.Update()
	s.add(p)
	return p
}

// projectionOfSource finds an existing projected point built from the same source. Identity is
// the (kind, id) pair persistence rebinds through, so two projections that would rebind to one
// source ARE one projection. A source with no stable identity — either half empty, as an
// anonymous or frozen source has — never matches: distinct unnamed sources must not collapse
// into each other.
func (s *Sketch) projectionOfSource(kind, id string) (*ProjectedPoint, bool) {
	if kind == "" || id == "" {
		return nil, false
	}
	for _, e := range s.ents {
		if p, ok := e.(*ProjectedPoint); ok && p.srcKind == kind && p.srcID == id {
			return p, true
		}
	}
	return nil, false
}

// projectionOfCurveSource finds an existing curve projection built from the same source, with the
// same (kind, id) identity rule as [Sketch.projectionOfSource].
func (s *Sketch) projectionOfCurveSource(kind, id string) (*Projection, bool) {
	if kind == "" || id == "" {
		return nil, false
	}
	for _, p := range s.projections {
		if p.srcKind == kind && p.srcID == id {
			return p, true
		}
	}
	return nil, false
}

// ProjectCurve projects a model edge into the sketch as a concrete grounded reference entity and
// returns the associative projection that drives it. Like [Sketch.ProjectPoint] it returns an
// existing projection of the same source rather than duplicating it (#2016).
func (s *Sketch) ProjectCurve(src CurveSource) *Projection {
	kind, id := curveSourceKind(src), src.SourceID()
	if p, ok := s.projectionOfCurveSource(kind, id); ok {
		return p
	}
	p := &Projection{source: src, plane: s.plane, linked: true, srcKind: kind, srcID: id}
	p.entity = s.buildProjectionEntity(src)
	s.projections = append(s.projections, p)
	return p
}

// buildProjectionEntity resolves the source's current geometry and builds the concrete reference
// entity: the source's exact analytic curve when it has one (a circle → a reference Circle, ADR-0055),
// otherwise a reference Spline through its sampled polyline.
func (s *Sketch) buildProjectionEntity(src CurveSource) Entity {
	if c2, ok := projectSourceCurve(s.plane, src); ok {
		if e := s.addReferenceCurve(c2); e != nil {
			return e
		}
	}
	pts, _ := src.SamplePoints() // resolved now; a later lost reference freezes via Update
	return s.addReferencePolyline(projectPointsToPlane(s.plane, pts))
}

// Projections returns the sketch's curve projections — the associative links that drive its concrete
// grounded reference entities (ADR-0055 phase 3), in creation order. The driven entities are also in
// [Sketch.Entities]; this is the view a host uses to reach the source link a projection carries.
func (s *Sketch) Projections() []*Projection {
	out := make([]*Projection, len(s.projections))
	copy(out, s.projections)
	return out
}

// ProjectCutEdges projects the edges where the sketch plane cuts the model. The cut
// edges are computed by the kernel (M07) and supplied as sources; the sketch maps
// them to its plane and adds them as reference geometry.
func (s *Sketch) ProjectCutEdges(sources []CurveSource) []*Projection {
	out := make([]*Projection, 0, len(sources))
	for _, src := range sources {
		out = append(out, s.ProjectCurve(src))
	}
	return out
}

// UpdateProjections re-projects every linked projection from its source — the hook a recompute calls
// so projected (include) geometry tracks the model as it changes. With self-resolving sources (keyed
// by reference) this re-binds the projection to the freshly rebuilt B-rep, making projection
// associative.
func (s *Sketch) UpdateProjections() {
	for _, e := range s.ents {
		if v, ok := e.(*ProjectedPoint); ok {
			v.Update()
		}
	}
	for _, p := range s.projections {
		p.Update(s)
	}
	s.updateCloudAnchors() // re-project scan-anchored points so they follow their clouds (#645)
}

// sourceKinded is the optional interface a projection source implements to declare its kind
// ("vertex"/"edge"/"workPoint"/"workAxis"/"workPlane"), so the sketch can persist enough to let
// compdef rebuild a live source on restore (#1268). The model-side reference sources implement
// it; a source that does not is persisted with an empty kind and stays frozen after a reload.
type sourceKinded interface{ SourceKind() string }

func pointSourceKind(src PointSource) string {
	if k, ok := src.(sourceKinded); ok {
		return k.SourceKind()
	}
	return ""
}

func curveSourceKind(src CurveSource) string {
	if k, ok := src.(sourceKinded); ok {
		return k.SourceKind()
	}
	return ""
}

// SourceDescriptor returns the (kind, id) a host uses to rebuild this projection's live source
// after a reload; an empty kind means there is nothing to rebind (the projection stays frozen).
func (p *ProjectedPoint) SourceDescriptor() (kind, id string) { return p.srcKind, p.srcID }

// Rebind attaches a freshly built live source to a projection restored frozen, making it
// associative again; the next UpdateProjections re-projects from it.
func (p *ProjectedPoint) Rebind(src PointSource) { p.source, p.linked = src, true }

// RestoreProjectedPoint rebuilds a projected point frozen at pos, pinning the anchor's id so
// constraints that reference it survive, and keeping the source descriptor for a later Rebind.
// The host re-attaches the live source after load (compdef.rebindSketchProjections, #1268).
func (s *Sketch) RestoreProjectedPoint(anchorID ID, pos math.Point2, srcKind, srcID string) *ProjectedPoint {
	anchor := &Point{id: anchorID, X: pos.X, Y: pos.Y}
	s.refPts = append(s.refPts, anchor)
	p := &ProjectedPoint{plane: s.plane, anchor: anchor, linked: false, srcKind: srcKind, srcID: srcID}
	s.add(p)
	return p
}

// RestoreProjection rebuilds a curve projection frozen at its persisted reference entity: the
// concrete entity is passed already built (with its ids pinned by the restore codec), and this pairs
// it with a frozen Projection carrying the source descriptor for a later Rebind (#1268, ADR-0055).
func (s *Sketch) RestoreProjection(entity Entity, srcKind, srcID string) *Projection {
	p := &Projection{entity: entity, plane: s.plane, linked: false, srcKind: srcKind, srcID: srcID}
	s.projections = append(s.projections, p)
	return p
}

// analyticCurveData encodes a reference entity's analytic 2D form as a compact shape tag + params for
// persistence (ADR-0055): line[ax,ay,bx,by], circle[cx,cy,r], arc[cx,cy,r,start,sweep]. ok=false for
// a form with no compact encoding yet (the caller then persists the sampled polyline).
func analyticCurveData(c geom.Curve2) (shape string, params []float64, ok bool) {
	switch k := c.(type) {
	case geom.LineSegment2d:
		return "line", []float64{float64(k.StartPoint.X), float64(k.StartPoint.Y), float64(k.EndPoint.X), float64(k.EndPoint.Y)}, true
	case geom.Circle2d:
		return "circle", []float64{float64(k.Center.X), float64(k.Center.Y), k.Radius}, true
	case geom.Arc2d:
		return "arc", []float64{float64(k.Center.X), float64(k.Center.Y), k.Radius, k.StartAngle, k.SweepAngle}, true
	case geom.EllipseFull2d:
		return "ellipse", []float64{float64(k.Center.X), float64(k.Center.Y),
			float64(k.MajorAxis.X()), float64(k.MajorAxis.Y()), k.MajorRadius, k.MinorRadius}, true
	case geom.EllipticalArc2d:
		return "ellipticalArc", []float64{float64(k.Center.X), float64(k.Center.Y),
			float64(k.MajorAxis.X()), float64(k.MajorAxis.Y()), k.MajorRadius, k.MinorRadius, k.StartAngle, k.SweepAngle}, true
	default:
		return "", nil, false
	}
}

// analyticCurveFromData rebuilds the geom.Curve2 a projected curve persisted via analyticCurveData,
// or ok=false when the shape/params do not describe one.
func analyticCurveFromData(shape string, p []float64) (geom.Curve2, bool) {
	switch shape {
	case "line":
		if len(p) != 4 {
			return nil, false
		}
		return geom.NewLineSegment2d(math.P2(math.Scalar(p[0]), math.Scalar(p[1])), math.P2(math.Scalar(p[2]), math.Scalar(p[3]))), true
	case "circle":
		if len(p) != 3 {
			return nil, false
		}
		return geom.NewCircle2d(math.P2(math.Scalar(p[0]), math.Scalar(p[1])), p[2]), true
	case "arc":
		if len(p) != 5 {
			return nil, false
		}
		return geom.NewArc2d(math.P2(math.Scalar(p[0]), math.Scalar(p[1])), p[2], p[3], p[4]), true
	default:
		return conicCurveFromData(shape, p)
	}
}

// conicCurveFromData rebuilds a projected ellipse / elliptical arc persisted by analyticCurveData
// (ADR-0055 phase 2). ellipse=[cx,cy,mx,my,majorR,minorR]; ellipticalArc adds [start,sweep].
func conicCurveFromData(shape string, p []float64) (geom.Curve2, bool) {
	switch shape {
	case "ellipse":
		if len(p) != 6 {
			return nil, false
		}
		e, err := geom.NewEllipseFull2d(math.P2(math.Scalar(p[0]), math.Scalar(p[1])), math.V2(math.Scalar(p[2]), math.Scalar(p[3])), p[4], p[5])
		return e, err == nil
	case "ellipticalArc":
		if len(p) != 8 {
			return nil, false
		}
		e, err := geom.NewEllipticalArc2d(math.P2(math.Scalar(p[0]), math.Scalar(p[1])), math.V2(math.Scalar(p[2]), math.Scalar(p[3])), p[4], p[5], p[6], p[7])
		return e, err == nil
	default:
		return nil, false
	}
}
