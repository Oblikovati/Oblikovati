// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Profiles, loops and paths are the only things a sketch exports to the feature
// engine (parametric-cad §10, modeling/00): solved geometry is turned into closed
// regions (with outer/inner loops) and connected chains. The feature engine never
// sees the solver or raw sketch entities beyond these.

// ProfileEntity is one curve in a loop or path, with the orientation in which it is
// traversed (Reversed when its natural direction opposes the traversal).
type ProfileEntity struct {
	Entity   Entity
	reversed bool
}

// Reversed reports whether the entity is traversed against its natural direction.
func (pe ProfileEntity) Reversed() bool { return pe.reversed }

// Loop is an ordered chain of entities. It is closed when its traversal returns to
// the start point.
type Loop struct {
	entities []ProfileEntity
	polygon  []math.Point2
	closed   bool
}

// Entities returns the loop's ordered entities.
func (l Loop) Entities() []ProfileEntity { return l.entities }

// IsClosed reports whether the loop forms a closed region.
func (l Loop) IsClosed() bool { return l.closed }

// Polygon returns the loop's representative polyline (arcs/circles sampled), used
// for containment tests.
func (l Loop) Polygon() []math.Point2 { return l.polygon }

// Profile is a region features extrude/revolve: an outer boundary loop and the
// inner loops (holes) it contains. An open profile (IsClosed false) has only its
// chain as the outer loop; features reject open profiles for solids but accept them
// for surfaces (enforced at the feature, not here).
type Profile struct {
	outer Loop
	inner []Loop
}

// OuterLoop returns the profile's outer boundary loop.
func (p *Profile) OuterLoop() Loop { return p.outer }

// InnerLoops returns the profile's hole loops.
func (p *Profile) InnerLoops() []Loop {
	out := make([]Loop, len(p.inner))
	copy(out, p.inner)
	return out
}

// IsClosed reports whether the profile encloses a region (its outer loop is closed).
func (p *Profile) IsClosed() bool { return p.outer.closed }

// Area returns the profile's enclosed area: the outer loop's area minus its holes' (0 for
// an open profile). Uses the loops' representative polygons (curves sampled), so it is the
// tessellated area, accurate to the sampling density.
func (p *Profile) Area() float64 {
	if !p.outer.closed {
		return 0
	}
	area := p.outer.Area()
	for _, h := range p.inner {
		area -= h.Area()
	}
	return area
}

// Area returns the loop's absolute enclosed area (shoelace over its polygon).
func (l Loop) Area() float64 { return stdmath.Abs(signedPolygonArea(l.polygon)) }

// signedPolygonArea returns the signed area of a polygon (positive for CCW winding).
func signedPolygonArea(poly []math.Point2) float64 {
	if len(poly) < 3 {
		return 0
	}
	var sum float64
	for i := range poly {
		j := (i + 1) % len(poly)
		sum += float64(poly[i].X*poly[j].Y - poly[j].X*poly[i].Y)
	}
	return sum / 2
}

// Contains reports whether the sketch-plane point q lies in the profile's region: inside
// the (closed) outer loop and outside every inner-loop hole. Used to hit-test a click on
// a profile so it can be picked for extrude/revolve. An open profile contains nothing.
func (p *Profile) Contains(q math.Point2) bool {
	if !p.outer.closed || !pointInPolygon(q, p.outer.polygon) {
		return false
	}
	for _, h := range p.inner {
		if pointInPolygon(q, h.polygon) {
			return false
		}
	}
	return true
}

// Profiles is the set of profiles detected in a sketch.
type Profiles struct {
	items []*Profile
}

// Count returns the number of profiles; Item returns the i-th.
func (ps *Profiles) Count() int          { return len(ps.items) }
func (ps *Profiles) Item(i int) *Profile { return ps.items[i] }

// All returns the detected profiles.
func (ps *Profiles) All() []*Profile {
	out := make([]*Profile, len(ps.items))
	copy(out, ps.items)
	return out
}

// Profiles detects regions in the sketch from its non-construction geometry: it builds
// a planar arrangement of the (faceted) curves and extracts the minimal closed cells —
// so a dividing curve splits a shape into several regions (arrangement.go / regions.go).
// maxProfileEntities caps profile/region detection. Above it the sketch is treated
// as reference geometry (no profiles): an imported drawing has no single extrudable
// region, and arranging that many segments every hover frame would freeze the UI.
const maxProfileEntities = 20000

// Cells are classified into outer boundaries and holes by even–odd nesting; geometry
// that bounds no cell (a connected but unclosed chain) becomes an open profile.
func (s *Sketch) Profiles() *Profiles {
	// A very dense sketch — an imported drawing can be hundreds of thousands of
	// segments — has no practical extrudable profile, and arranging that many
	// segments would stall the frame (the hover picker calls this). Skip detection
	// above the cap so picking/hover stay instant; reference geometry offers no
	// profiles. This is O(1) (len(ents)), checked before the geometry signature.
	if len(s.ents) > maxProfileEntities {
		if s.profilesCache == nil {
			s.profilesCache = &Profiles{}
		}
		return s.profilesCache
	}
	sig := s.geomSignature()
	if s.profilesCache != nil && s.profilesSig == sig {
		return s.profilesCache
	}
	ents := s.normalGeometry()
	loops := detectRegions(ents)
	ps := &Profiles{}
	ps.items = append(ps.items, groupRegions(loops)...)
	for _, chain := range openChainsOutside(ents, loops) {
		ps.items = append(ps.items, &Profile{outer: chain})
	}
	s.profilesCache, s.profilesSig = ps, sig
	return ps
}

// geomSignature is a cheap fingerprint of the sketch geometry — entity/point counts
// folded with every point coordinate (FNV-1a) — so Profiles() rebuilds after any
// add, remove or move but reuses its cache when nothing changed. It is O(points),
// far below the region detection it guards.
func (s *Sketch) geomSignature() uint64 {
	const prime = 1099511628211
	h := uint64(14695981039346656037)
	mix := func(v uint64) { h = (h ^ v) * prime }
	mix(uint64(len(s.ents)))
	mix(uint64(len(s.pts)))
	for _, p := range s.pts {
		mix(stdmath.Float64bits(p.X))
		mix(stdmath.Float64bits(p.Y))
	}
	for _, e := range s.ents {
		// Construction/centerline entities are excluded from profiles (normalGeometry), so
		// toggling that flag changes the regions without changing any coordinate — fold it in.
		if cg, ok := e.(interface{ IsConstruction() bool }); ok && cg.IsConstruction() {
			mix(1)
		} else {
			mix(2)
		}
		mixEntityScalars(mix, e)
	}
	return h
}

// mixEntityScalars folds an entity's intrinsic scalars that are NOT stored as points — a
// circle's radius, an ellipse's axes/angles. A dimension resizing one (e.g. radius = od/2)
// leaves the centre point, and so the point-coordinate signature, unchanged; without this the
// cache would return a stale profile and break parametric resize (the spacer/flange regressions).
func mixEntityScalars(mix func(uint64), e Entity) {
	bits := func(v math.Scalar) { mix(stdmath.Float64bits(float64(v))) }
	switch c := e.(type) {
	case *Circle:
		bits(c.Radius)
	case *Ellipse:
		bits(c.MajorRadius)
		bits(c.MinorRadius)
		bits(c.MajorAxis.X)
		bits(c.MajorAxis.Y)
	case *EllipticalArc:
		bits(c.MajorRadius)
		bits(c.MinorRadius)
		bits(c.MajorAxis.X)
		bits(c.MajorAxis.Y)
		bits(c.StartAngle)
		bits(c.EndAngle)
	}
}

// ClosedLoops returns the sketch's standalone closed loops detected by endpoint chaining
// (detectLoops), independent of the planar-arrangement region finder. Because each loop is a
// connected chain, crossing loops (e.g. a grid of bars whose rectangles intersect mid-edge but
// share no endpoints) are returned individually and intact — unlike Profiles, whose even–odd
// region nesting cannot represent overlapping interior loops. Callers that need a boolean of
// loops (grill: boundary − union(bars), #863) use this instead of profile inner loops.
func (s *Sketch) ClosedLoops() []Loop {
	closed, _ := detectLoops(s.normalGeometry())
	return closed
}

// openChainsOutside returns the open chains formed by entities that bound no detected
// region — the geometry an extrude rightly rejects but a surface may consume. Entities
// already used by a region are excluded so a dividing curve is not also reported open.
func openChainsOutside(ents []Entity, loops []Loop) []Loop {
	used := map[Entity]bool{}
	for _, l := range loops {
		for _, pe := range l.entities {
			used[pe.Entity] = true
		}
	}
	var leftover []Entity
	for _, e := range ents {
		if !used[e] {
			leftover = append(leftover, e)
		}
	}
	_, open := detectLoops(leftover)
	return open
}

// normalGeometry returns the sketch's non-construction curve entities.
func (s *Sketch) normalGeometry() []Entity {
	var out []Entity
	for _, e := range s.ents {
		if cg, ok := e.(interface{ IsConstruction() bool }); ok && cg.IsConstruction() {
			continue
		}
		out = append(out, e)
	}
	return out
}
