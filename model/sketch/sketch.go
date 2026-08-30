// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/param"
)

// Sketch is a planar 2D sketch hosted on a [Plane]. It owns its entities and (from
// F03/F04) its constraints, and resolves to profiles/paths via the solver (F05/F06).
type Sketch struct {
	base
	plane     Plane
	planeHost func() Plane // when set, RefreshPlane re-reads the host (e.g. a work plane)
	// hostWorkRef is the datum reference ("plane/N") this sketch was created on, empty for an
	// origin/fixed plane. Kept as a plain string (no feature dependency) so the host can tell which
	// sketches consume a construction work plane and auto-delete it with its last consumer (#1849).
	hostWorkRef string
	ents        []Entity
	pts         []*Point              // every constrainable point (endpoints, centers, standalone) — the solver's variables
	refPts      []*Point              // fixed reference points (projected anchors): constrainable but not solved
	cloudPts    []*cloudAnchoredPoint // sketch points anchored on scan points (datum-cloud provenance, #645)
	ptArena     pointArena            // block allocator backing newPoint (one alloc per block, not per vertex)
	// projections are the associative links from model edges to the concrete grounded reference
	// entities they drive (ADR-0055 phase 3). The driven entities live in ents/the typed collections
	// like any geometry; a projection re-derives one on UpdateProjections and persists its source.
	projections []*Projection

	lines         *Lines
	arcs          *Arcs
	circles       *Circles
	ellipses      *Ellipses
	ellArcs       *EllipticalArcs
	splines       *Splines
	points        *Points
	images        *SketchImages
	fills         *FillRegions
	texts         *TextBoxes
	eqCurves      *EquationCurves
	fixedSpl      *FixedSplines
	offSpl        *OffsetSplines
	blocks        *Blocks
	splineHandles *SplineHandles
	geomCons      *GeometricConstraints
	dimCons       *DimensionConstraints
	params        *param.Parameters

	// profilesCache memoises Profiles() — region detection is O(n log n) over the
	// geometry and was rerun on every call (the hover picker calls it each frame),
	// freezing on a dense imported sketch. profilesSig is the geometry signature it
	// was built for (counts + point coordinates), so any edit invalidates it.
	profilesCache *Profiles
	profilesSig   uint64

	// paramFootprint is the dependency footprint this sketch's last solve read (its
	// dimension targets), captured by the part recompute as depend.Keys (Oblikovati#1414,
	// ADR-0044). A parameter edit re-solves and rebuilds only the features whose consumed
	// sketch footprint it touches, instead of the whole program.
	paramFootprint []depend.Key

	// hostFootprint, when set, returns the dependency footprint of the sketch's host plane
	// (a work plane's offset/angle parameter reads). The sketch stays content-agnostic
	// about WHAT hosts it (ADR-0036): this is an opaque provider the part wires to the work
	// plane, so a work-plane-offset edit reaches features through the hosted sketch's
	// footprint instead of forcing a wholesale rebuild (ADR-0044).
	hostFootprint func() []depend.Key
}

// SetParameterFootprint records the dependency keys the sketch's solve read; the part
// recompute captures the footprint around each solve (param.TrackKeys) so the feature
// engine can dirty only the features a parameter edit actually affects.
func (s *Sketch) SetParameterFootprint(keys []depend.Key) { s.paramFootprint = keys }

// SetHostFootprint registers an opaque provider of the host plane's dependency footprint
// (nil to clear). Paired with [Sketch.SetPlaneHost] when the host is a parametric work
// plane, so the plane's offset parameter is attributed to this sketch (ADR-0044).
func (s *Sketch) SetHostFootprint(footprint func() []depend.Key) { s.hostFootprint = footprint }

// ParameterFootprint returns the sketch's full dependency footprint: its own solve reads
// plus, when hosted on a parametric work plane, that plane's footprint. The union is what
// a consuming feature is attributed against (ADR-0044).
func (s *Sketch) ParameterFootprint() []depend.Key {
	if s.hostFootprint == nil {
		return s.paramFootprint
	}
	return append(append([]depend.Key(nil), s.paramFootprint...), s.hostFootprint()...)
}

// Plane returns the sketch's host plane.
func (s *Sketch) Plane() Plane { return s.plane }

// SetPlaneHost makes the sketch track a host plane (a work plane): RefreshPlane will
// re-read it on each recompute so the sketch — and anything built on it — follows the
// work plane when it moves. A nil host detaches (the plane stays fixed). The current
// plane is updated immediately.
func (s *Sketch) SetPlaneHost(host func() Plane) {
	s.planeHost = host
	if host != nil {
		s.plane = host()
	}
}

// SetHostWorkRef records the datum reference ("plane/N") this sketch is hosted on, so the host can
// count the sketch as a consumer of that work plane (#1849). Empty for an origin/fixed plane.
func (s *Sketch) SetHostWorkRef(ref string) { s.hostWorkRef = ref }

// HostWorkRef returns the datum reference this sketch was created on, or "" for an origin/fixed plane.
func (s *Sketch) HostWorkRef() string { return s.hostWorkRef }

// RefreshPlane re-reads the host plane if the sketch tracks one (no-op otherwise).
// Called by the part recompute after work geometry is recomputed, so a moved work
// plane carries its sketch (and dependent features) with it.
func (s *Sketch) RefreshPlane() {
	if s.planeHost != nil {
		s.plane = s.planeHost()
	}
}

// Entities returns the sketch's geometry in insertion order.
func (s *Sketch) Entities() []Entity {
	out := make([]Entity, len(s.ents))
	copy(out, s.ents)
	return out
}

// EntityCount returns the number of entities.
func (s *Sketch) EntityCount() int { return len(s.ents) }

// EntityByID returns the entity with the given session id, or false if none matches.
func (s *Sketch) EntityByID(id ID) (Entity, bool) {
	for _, e := range s.ents {
		if e.EntityID() == id {
			return e, true
		}
	}
	return nil, false
}

// PointByID returns the constrainable point with the given id — including curve
// endpoints/centers, which are not standalone entities — or false if none matches.
func (s *Sketch) PointByID(id ID) (*Point, bool) {
	for _, p := range s.pts {
		if p.id == id {
			return p, true
		}
	}
	for _, p := range s.refPts {
		if p.id == id {
			return p, true // a fixed projected/reference anchor can be constrained to
		}
	}
	return nil, false
}

// AllPoints returns every constrainable point in the sketch — free points (endpoints,
// centers, standalone) AND the fixed projected/reference anchors (refPts). It is the
// pick/snap/inference candidate set, so projected geometry (e.g. the origin centre point) can
// be selected and constrained to; the solver's free-variable universe is variables(), which
// deliberately excludes the fixed anchors. Before #1268 the anchors were omitted here, so a
// coincident constraint to a projected point could never be picked.
func (s *Sketch) AllPoints() []*Point {
	out := make([]*Point, 0, len(s.pts)+len(s.refPts))
	out = append(out, s.pts...)
	out = append(out, s.refPts...)
	return out
}

// Lines/Arcs/Circles/Ellipses/Splines/Points/Blocks return the typed entity
// factories (the Lines etc. collections).
func (s *Sketch) Lines() *Lines     { return s.lines }
func (s *Sketch) Arcs() *Arcs       { return s.arcs }
func (s *Sketch) Circles() *Circles { return s.circles }

// CircularCenters returns the centre position of every circle and arc. A sketch overlay marks
// these so a circular entity's centre is a visible hover/snap target: an arc's centre sits off the
// curve in empty space, so without a marker the user cannot aim a coincident constraint at it — it
// looks like the arc "has no centre" (#2159). Keeping the type knowledge here lets the head draw
// the markers without inspecting sketch entity types (archguard I1, #1624).
func (s *Sketch) CircularCenters() []math.Point2 {
	out := make([]math.Point2, 0, len(s.circles.items)+len(s.arcs.items))
	for _, c := range s.circles.items {
		out = append(out, c.Center.Position())
	}
	for _, a := range s.arcs.items {
		out = append(out, a.Center.Position())
	}
	return out
}

func (s *Sketch) Ellipses() *Ellipses { return s.ellipses }
func (s *Sketch) Splines() *Splines   { return s.splines }

// EllipticalArcs returns the elliptical-arc collection.
func (s *Sketch) EllipticalArcs() *EllipticalArcs { return s.ellArcs }

// Images returns the sketch-image collection.
func (s *Sketch) Images() *SketchImages { return s.images }

// FillRegions returns the fill-region collection; TextBoxes the sketch-text collection.
func (s *Sketch) FillRegions() *FillRegions { return s.fills }
func (s *Sketch) TextBoxes() *TextBoxes     { return s.texts }

// EquationCurves/FixedSplines/OffsetSplines return the derived-curve collections.
func (s *Sketch) EquationCurves() *EquationCurves { return s.eqCurves }
func (s *Sketch) FixedSplines() *FixedSplines     { return s.fixedSpl }
func (s *Sketch) OffsetSplines() *OffsetSplines   { return s.offSpl }
func (s *Sketch) Points() *Points                 { return s.points }
func (s *Sketch) Blocks() *Blocks                 { return s.blocks }

// SplineHandles returns the sketch's active spline tangency handles (M06-F11).
func (s *Sketch) SplineHandles() *SplineHandles { return s.splineHandles }

// GeometricConstraints returns the sketch's geometric-constraint collection.
func (s *Sketch) GeometricConstraints() *GeometricConstraints { return s.geomCons }

// DimensionConstraints returns the sketch's dimensional-constraint collection.
func (s *Sketch) DimensionConstraints() *DimensionConstraints { return s.dimCons }

// Parameters returns the parameter store backing this sketch's dimensions. By
// default a sketch owns its own; a component definition swaps in the shared store
// via [Sketch.SetParameters] so dimensions join the document's parameter DAG.
func (s *Sketch) Parameters() *param.Parameters { return s.params }

// SetParameters replaces the parameter store (and re-points the dimension
// collection at it). Call before adding dimensions.
func (s *Sketch) SetParameters(ps *param.Parameters) {
	s.params = ps
	s.dimCons.params = ps
}

// Constraints returns every residual-bearing constraint — all geometric plus the
// driving dimensions — which is exactly what the solver (F05) consumes. Driven
// dimensions are excluded (they report, they do not constrain).
func (s *Sketch) Constraints() []Constraint {
	out := s.geomCons.All()
	// Every arc carries an internal circularity constraint keeping its End on the circle
	// (#1419); the solver consumes it like any other, but it is not a user-facing relation. A
	// grounded reference arc (a projected edge) is held fixed by its refPts, so its circularity
	// would be a redundant all-fixed equation that trips the over-constrained check — skip it
	// (ADR-0055 phase 3).
	for _, a := range s.arcs.items {
		if a.reference {
			continue
		}
		out = append(out, a.circularity)
		for _, ft := range a.filletTangents {
			out = append(out, ft) // fillet tangency to each blended edge (#69)
		}
	}
	for _, d := range s.dimCons.items {
		if !d.driven && d.drivable() {
			out = append(out, d)
		}
	}
	return out
}

// initCollections wires the typed entity factories to this sketch.
func (s *Sketch) initCollections() {
	s.lines = &Lines{s: s}
	s.arcs = &Arcs{s: s}
	s.circles = &Circles{s: s}
	s.ellipses = &Ellipses{s: s}
	s.ellArcs = &EllipticalArcs{s: s}
	s.splines = &Splines{s: s}
	s.points = &Points{s: s}
	s.images = &SketchImages{s: s}
	s.fills = &FillRegions{s: s}
	s.texts = &TextBoxes{s: s}
	s.eqCurves = &EquationCurves{s: s}
	s.fixedSpl = &FixedSplines{s: s}
	s.offSpl = &OffsetSplines{s: s}
	s.blocks = &Blocks{s: s}
	s.splineHandles = &SplineHandles{s: s}
	s.geomCons = &GeometricConstraints{}
	s.params = param.NewParameters()
	s.dimCons = &DimensionConstraints{params: s.params}
}

// ToModel maps a sketch-space point to model space via the host plane.
func (s *Sketch) ToModel(p math.Point2) math.Point3 { return s.plane.ToModel(p) }

// ToSketch maps a model-space point onto the sketch plane.
func (s *Sketch) ToSketch(p math.Point3) math.Point2 { return s.plane.ToSketch(p) }
