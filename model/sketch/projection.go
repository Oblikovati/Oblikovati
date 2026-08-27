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
// reference keys). Projected geometry is construction/reference by default.

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

// ProjectedCurve is a sketch curve projected from a model edge (or cut edge). points is the sampled
// polyline; shape is its analytic form (a projected line/arc/circle projects to one — Inventor keeps
// them analytic), re-derived from points on every Update, so a projected arc draws smooth and offsets
// as a concentric arc rather than a faceted chain.
type ProjectedCurve struct {
	id     ID
	source CurveSource
	plane  Plane
	curve  geom.Curve2 // the analytic 2D form (ADR-0055); nil for a non-analytic (sampled) source
	points []math.Point2
	shape  projectedShape
	linked bool
	// srcKind/srcID persist the source's identity for save/reload + rebind (see ProjectedPoint).
	srcKind, srcID string
}

// AnalyticCurve returns the projected curve's exact analytic 2D form (a geom.Line2d/Circle2d/Arc2d)
// and true, or ok=false when the projection fell back to a sampled polyline (a non-analytic source
// or an oblique conic not yet handled). Consumers (extrude, offset, serialization) use this to keep
// projected geometry analytic (ADR-0055).
func (c *ProjectedCurve) AnalyticCurve() (geom.Curve2, bool) {
	return c.curve, c.curve != nil
}

// EntityID implements [Entity].
func (c *ProjectedCurve) EntityID() ID { return c.id }

// Points returns the cached sketch-space polyline.
func (c *ProjectedCurve) Points() []math.Point2 {
	out := make([]math.Point2, len(c.points))
	copy(out, c.points)
	return out
}

// SourceID returns the source identity, or "" once the link is broken.
func (c *ProjectedCurve) SourceID() string {
	if !c.linked {
		return ""
	}
	return c.source.SourceID()
}

// IsReference reports that projected geometry is reference/construction geometry.
func (c *ProjectedCurve) IsReference() bool { return true }

// Linked reports whether the projection still tracks its source.
func (c *ProjectedCurve) Linked() bool { return c.linked }

// Update re-projects from the current source. It prefers the ANALYTIC path (the source's exact
// curve projected onto the plane, keeping a circle a circle — ADR-0055) and only samples a source
// with no analytic curve. A lost reference breaks the link, freezing the last geometry.
func (c *ProjectedCurve) Update() {
	if !c.linked {
		return
	}
	if c.updateAnalytic() {
		return
	}
	src, ok := c.source.SamplePoints()
	if !ok {
		c.linked = false
		return
	}
	c.curve = nil
	c.points = c.points[:0]
	for _, q := range src {
		c.points = append(c.points, c.plane.ToSketch(q))
	}
	c.shape = fitProjectedShape(c.points)
}

// updateAnalytic sets the analytic 2D curve from the source's exact curve, returning false when the
// source is not an [AnalyticCurveSource], its reference is lost, or the projection is not yet
// analytic (an oblique conic) — the caller then falls back to sampling.
func (c *ProjectedCurve) updateAnalytic() bool {
	as, ok := c.source.(AnalyticCurveSource)
	if !ok {
		return false
	}
	c3, ok := as.SourceCurve()
	if !ok {
		return false
	}
	c2, ok := geom.ProjectCurveToPlane(sketchPlaneToGeom(c.plane), c3)
	if !ok {
		return false
	}
	c.curve = c2
	c.points = sampleCurve2(c2, projectedRenderSegments)
	c.shape = projectedShape{}
	return true
}

// sketchPlaneToGeom builds the geom.Plane whose (u,v) frame is the sketch plane's (xAxis, yAxis), so
// geom.ProjectCurveToPlane projects into the sketch's own 2D coordinate system.
func sketchPlaneToGeom(p Plane) geom.Plane {
	pl, _ := geom.NewPlaneFromAxes(p.origin, p.xAxis.AsVector(), p.yAxis.AsVector())
	return pl
}

// sampleCurve2 samples an analytic 2D curve into n+1 points for drawing and hit-testing.
func sampleCurve2(c geom.Curve2, n int) []math.Point2 {
	lo, hi := c.Domain()
	pts := make([]math.Point2, n+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/float64(n))
	}
	return pts
}

// RenderPolyline returns the curve as a smooth polyline for drawing and hit-testing: the analytic
// shape sampled finely when the projection is a line/arc/circle (so a projected arc is not the 16
// source facets), else the raw projected points.
func (c *ProjectedCurve) RenderPolyline() []math.Point2 {
	if p := c.shape.polyline(); p != nil {
		return p
	}
	return c.Points()
}

// BreakLink detaches the projection from its source, freezing the current polyline.
func (c *ProjectedCurve) BreakLink() { c.linked = false }

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

// curveProjectionOfSource is [Sketch.projectionOfSource] for projected curves, with the same
// identity rule.
func (s *Sketch) curveProjectionOfSource(kind, id string) (*ProjectedCurve, bool) {
	if kind == "" || id == "" {
		return nil, false
	}
	for _, e := range s.ents {
		if c, ok := e.(*ProjectedCurve); ok && c.srcKind == kind && c.srcID == id {
			return c, true
		}
	}
	return nil, false
}

// ProjectCurve projects a model edge into the sketch as reference geometry. Like
// [Sketch.ProjectPoint] it returns an existing projection of the same source rather than
// duplicating it (#2016).
func (s *Sketch) ProjectCurve(src CurveSource) *ProjectedCurve {
	if c, ok := s.curveProjectionOfSource(curveSourceKind(src), src.SourceID()); ok {
		return c
	}
	c := &ProjectedCurve{
		id: nextID(), source: src, plane: s.plane, linked: true,
		srcKind: curveSourceKind(src), srcID: src.SourceID(),
	}
	c.Update()
	s.add(c)
	return c
}

// ProjectCutEdges projects the edges where the sketch plane cuts the model. The cut
// edges are computed by the kernel (M07) and supplied as sources; the sketch maps
// them to its plane and adds them as reference geometry.
func (s *Sketch) ProjectCutEdges(sources []CurveSource) []*ProjectedCurve {
	out := make([]*ProjectedCurve, 0, len(sources))
	for _, src := range sources {
		out = append(out, s.ProjectCurve(src))
	}
	return out
}

// UpdateProjections re-projects every linked projected entity from its source — the hook a
// recompute calls so projected (include) geometry tracks the model as it changes. With
// self-resolving sources (keyed by reference) this re-binds the projection to the freshly
// rebuilt B-rep, making projection associative.
func (s *Sketch) UpdateProjections() {
	for _, e := range s.ents {
		switch v := e.(type) {
		case *ProjectedPoint:
			v.Update()
		case *ProjectedCurve:
			v.Update()
		}
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

// SourceDescriptor / Rebind for a projected curve mirror ProjectedPoint's.
func (c *ProjectedCurve) SourceDescriptor() (kind, id string) { return c.srcKind, c.srcID }
func (c *ProjectedCurve) Rebind(src CurveSource)              { c.source, c.linked = src, true }

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

// RestoreProjectedCurve rebuilds a projected curve frozen at the given polyline, pinning its id
// and keeping the source descriptor for a later Rebind. This is the LEGACY path for documents that
// stored the sampled coords; new documents store the analytic curve (RestoreProjectedCurveAnalytic).
func (s *Sketch) RestoreProjectedCurve(id ID, pts []math.Point2, srcKind, srcID string) *ProjectedCurve {
	c := &ProjectedCurve{id: id, plane: s.plane, points: pts, shape: fitProjectedShape(pts), linked: false, srcKind: srcKind, srcID: srcID}
	s.add(c)
	return c
}

// RestoreProjectedCurveAnalytic rebuilds a projected curve frozen at its persisted analytic 2D curve
// (ADR-0055), pinning its id and keeping the source descriptor for a later Rebind. The polyline is
// re-derived from the curve for display; it is not stored.
func (s *Sketch) RestoreProjectedCurveAnalytic(id ID, curve geom.Curve2, srcKind, srcID string) *ProjectedCurve {
	c := &ProjectedCurve{
		id: id, plane: s.plane, curve: curve, points: sampleCurve2(curve, projectedRenderSegments),
		linked: false, srcKind: srcKind, srcID: srcID,
	}
	s.add(c)
	return c
}

// analyticCurveData encodes a projected curve's analytic 2D form as a compact shape tag + params for
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
		return nil, false
	}
}
