// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	"sync/atomic"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/health"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/param"
	"oblikovati.org/model/seq"
)

// ID is the session-stable handle of a sketch or sketch entity, used by
// constraints and selection. Like document ids it is not persisted across
// sessions (regenerated on load); cross-session identity is via reference keys.
type ID uint64

var idSeq atomic.Uint64

func nextID() ID { return ID(idSeq.Add(1)) }

// raiseIDSeq lifts the entity id clock to at least n. Restoring a document pins its
// entity/sketch ids to their persisted (verbatim) values; raising the clock past the
// largest restored id ensures ids minted afterwards never collide with them (#153).
func raiseIDSeq(n uint64) {
	for {
		cur := idSeq.Load()
		if cur >= n {
			return
		}
		if idSeq.CompareAndSwap(cur, n) {
			return
		}
	}
}

// Entity is a piece of sketch geometry (line, arc, circle, point, …). The full
// interface — constrainable points, curve evaluation — is filled in by the entity
// types (M06-F02); here it is the minimum the container needs.
type Entity interface {
	// EntityID returns the entity's sketch-local id.
	EntityID() ID
}

// base holds the identity and edit/visibility/display state common to every sketch kind.
type base struct {
	id      ID
	name    string
	editing bool
	visible bool
	shared  bool   // Inventor PlanarSketch.Shared: stays visible at top level, may feed many features
	seq     uint64 // global creation stamp, shared with features/work features; see model/seq
	health  health.Health

	// Display + solve overrides (Inventor Sketch.Color/LineType/LineWeight/DeferUpdates).
	color        string  // empty ⇒ inherit the document default
	lineType     string  // empty ⇒ inherit (api/types.SketchLineType value)
	lineWeight   float64 // 0 ⇒ inherit
	deferUpdates bool    // true ⇒ batch edits, solve on resume (M21-F08)

	// Custom line type loaded from a .lin definition file (issue #161); non-nil
	// exactly when lineType is "custom". The pattern persists with the document;
	// the source file path is kept only for reporting.
	customLineType     *linetype.Definition
	customLineTypeFile string

	// Per-entity format overrides (#2015). They live on the shared base rather than on Sketch
	// because a 3D sketch carries the same overrides — its Format panel was registered from the
	// same list while only the planar half had storage, so every list on the 3D tab edited
	// nothing (#2039).
	formats   map[ID]EntityFormat // absent ⇒ sketch defaults
	formatRev uint64              // bumped on every format edit, so a drawing cache can see one
}

func newBase(name string) base {
	return base{id: nextID(), name: name, visible: true, seq: seq.Next(), health: health.Healthy}
}

// ID returns the sketch's session id.
func (b *base) ID() ID { return b.id }

// Name returns the sketch's display name.
func (b *base) Name() string { return b.name }

// SetName renames the sketch.
func (b *base) SetName(name string) { b.name = name }

// IsEditing reports whether the sketch is in edit mode (open for geometry changes).
func (b *base) IsEditing() bool { return b.editing }

// Edit enters edit mode; ExitEdit leaves it.
func (b *base) Edit()     { b.editing = true }
func (b *base) ExitEdit() { b.editing = false }

// Visible reports whether the sketch is shown; SetVisible toggles it.
func (b *base) Visible() bool     { return b.visible }
func (b *base) SetVisible(v bool) { b.visible = v }

// Shared reports whether the sketch is shared (Inventor PlanarSketch.Shared): a shared
// sketch stays at the browser's top level even after a feature consumes it, and may be
// consumed by several features. SetShared toggles it (browser "Share Sketch", issue #132).
func (b *base) Shared() bool     { return b.shared }
func (b *base) SetShared(s bool) { b.shared = s }

// Seq returns the sketch's global creation stamp (restoring a saved recipe pins it via
// seq.Restore so reopened documents keep their original interleaving).
func (b *base) Seq() uint64 { return b.seq }

// Health returns the sketch's solve health (set by the solver, M06-F05).
func (b *base) Health() health.Health { return b.health }

// Color returns the sketch's color override ("" ⇒ inherit); SetColor sets it.
func (b *base) Color() string     { return b.color }
func (b *base) SetColor(c string) { b.color = c }

// LineType returns the sketch's line-type override (api/types.SketchLineType value,
// "" ⇒ inherit); SetLineType sets it and drops any loaded custom definition when
// moving to a non-custom style.
func (b *base) LineType() string { return b.lineType }
func (b *base) SetLineType(t string) {
	b.lineType = t
	if t != string(types.SketchLineCustom) {
		b.customLineType, b.customLineTypeFile = nil, ""
	}
}

// SetCustomLineType installs a loaded .lin definition as the sketch's line style
// (lineType becomes "custom"); from is the source file, kept for reporting.
//
//	sk.SetCustomLineType(def, "styles.lin")
func (b *base) SetCustomLineType(d linetype.Definition, from string) {
	b.customLineType, b.customLineTypeFile = &d, from
	b.lineType = string(types.SketchLineCustom)
}

// CustomLineType returns the loaded custom definition, its source file, and
// whether one is present.
func (b *base) CustomLineType() (linetype.Definition, string, bool) {
	if b.customLineType == nil {
		return linetype.Definition{}, "", false
	}
	return *b.customLineType, b.customLineTypeFile, true
}

// LineWeight returns the sketch's line-weight override in cm (0 ⇒ inherit);
// SetLineWeight sets it.
func (b *base) LineWeight() float64     { return b.lineWeight }
func (b *base) SetLineWeight(w float64) { b.lineWeight = w }

// DeferUpdates reports whether the sketch batches edits (solving on resume);
// SetDeferUpdates toggles it (the solve gate is wired in M21-F08).
func (b *base) DeferUpdates() bool     { return b.deferUpdates }
func (b *base) SetDeferUpdates(d bool) { b.deferUpdates = d }

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
	// (#1419); the solver consumes it like any other, but it is not a user-facing relation.
	for _, a := range s.arcs.items {
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

// add appends an entity to the sketch's geometry list.
func (s *Sketch) add(e Entity) { s.ents = append(s.ents, e) }

// removeEntity drops an entity from the geometry list (used by delete/trim). It does not
// touch constraints; callers handle those. Returns whether it was present.
func (s *Sketch) removeEntity(e Entity) {
	for i, x := range s.ents {
		if x == e {
			s.ents = append(s.ents[:i], s.ents[i+1:]...)
			return
		}
	}
}

// deleteEntity removes e from the entity list AND its typed collection, so an edit that
// drops a curve (e.g. a whole-curve trim) leaves no dangling collection entry that
// Count/Item/serialization would still report.
func (s *Sketch) deleteEntity(e Entity) {
	s.removeEntity(e)
	s.dropFromCollection(e)
	s.ClearEntityFormat(e.EntityID()) // the format dies with its entity (#2015)
}

// dropFromCollection removes e from its typed collection (the deleteEntity
// half that knows every entity family).
func (s *Sketch) dropFromCollection(e Entity) {
	switch t := e.(type) {
	case *Line:
		s.lines.remove(t)
	case *Circle:
		s.circles.remove(t)
	case *Arc:
		s.arcs.remove(t)
	case *TextBox:
		s.texts.remove(t)
		s.deleteTextBoxAnchor(t) // the anchor record dies with its text (M06-F11)
	case *Ellipse:
		s.ellipses.remove(t)
	case *EllipticalArc:
		s.ellArcs.remove(t)
	case *Spline:
		s.splines.remove(t)
	case *Point:
		s.points.remove(t)
	case *BlockInstance:
		s.blocks.remove(t) // also detaches the definition back-reference (M06-F07)
	}
}

// newPoint creates a constrainable point at pos and registers it as a solver
// variable. Curve factories use it for endpoints/centers (not added to Entities).
func (s *Sketch) newPoint(pos math.Point2) *Point {
	p := s.ptArena.alloc()
	p.id, p.X, p.Y = nextID(), pos.X, pos.Y
	s.pts = append(s.pts, p)
	return p
}

// pointArenaBlock is how many Points one arena block holds. Large enough that a dense imported
// polyline (thousands of vertices) costs a handful of allocations, small enough that a sketch
// with a few points wastes little: 1024 * sizeof(Point) ≈ 24 KB per block.
const pointArenaBlock = 1024

// pointArena hands out stable *Point from fixed-size blocks, so bulk authoring — importing a
// drawing's tens of thousands of polyline vertices (#1549) — allocates one block per
// pointArenaBlock points instead of one heap object per point, cutting the import's allocation
// count and steady-state GC scan cost. Blocks are never reallocated, so every handed-out
// pointer stays valid; points removed from a sketch simply leave a stale slot (rare, and freed
// with the whole sketch). Identity is still by pointer, since each &block[i] is unique.
type pointArena struct {
	blocks [][]Point
	used   int // points taken from the current (last) block
}

// alloc returns a pointer to the next free, zeroed Point, starting a fresh block when full.
func (a *pointArena) alloc() *Point {
	if len(a.blocks) == 0 || a.used == pointArenaBlock {
		a.blocks = append(a.blocks, make([]Point, pointArenaBlock))
		a.used = 0
	}
	p := &a.blocks[len(a.blocks)-1][a.used]
	a.used++
	return p
}

// NewPoint creates a sketch vertex point for use as a shared endpoint of lines or
// arcs (it is not a standalone point marker). It lets a bulk authoring caller — the
// DWG importer — connect a polyline's segments through shared vertices instead of
// duplicating endpoints, roughly a third less geometry on dense drawings.
func (s *Sketch) NewPoint(pos math.Point2) *Point { return s.newPoint(pos) }

// removePoint drops a solver point (a deactivated spline-handle end or a
// point moved into a block definition).
func (s *Sketch) removePoint(p *Point) {
	for i, x := range s.pts {
		if x == p {
			s.pts = append(s.pts[:i], s.pts[i+1:]...)
			return
		}
	}
}

// newRefPoint creates a fixed reference point (a projected anchor): a real Point other
// geometry can be constrained to, but excluded from the solver's free variables, so the
// solver holds it in place while other geometry moves to meet it.
func (s *Sketch) newRefPoint(pos math.Point2) *Point {
	p := &Point{id: nextID(), X: pos.X, Y: pos.Y}
	s.refPts = append(s.refPts, p)
	return p
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

// Sketches is the collection of planar sketches owned by a component definition.
type Sketches struct {
	items []*Sketch
	byID  map[ID]*Sketch
	seq   int // running counter behind the Sketch1, Sketch2, … auto-names
	// blockDefs is the part-level block-definition registry every sketch of
	// the part places instances from (M06-F07, #622).
	blockDefs *BlockDefinitions
	// params is the document's parameter DAG, shared into every sketch added to the
	// collection so dimension expressions referencing user parameters resolve. Nil for
	// a bare collection (tests), which leaves each sketch with its own empty set.
	params *param.Parameters
}

// ShareParameters makes the collection hand the document's parameter DAG to every
// sketch it creates (live or on restore), so a dimension expression like "width"
// resolves against user parameters. Wiring at creation matters: the restore path adds a
// sketch's dimensions immediately, and SetParameters must precede them.
func (c *Sketches) ShareParameters(ps *param.Parameters) { c.params = ps }

// NewSketches returns an empty collection.
func NewSketches() *Sketches {
	return &Sketches{byID: map[ID]*Sketch{}, blockDefs: &BlockDefinitions{}}
}

// BlockDefinitions returns the part-level block-definition registry.
func (sc *Sketches) BlockDefinitions() *BlockDefinitions { return sc.blockDefs }

// Add creates a planar sketch on plane and adds it to the collection, giving it the
// next free auto-name (Sketch1, Sketch2, …) like the reference API.
func (c *Sketches) Add(plane Plane) *Sketch {
	return c.AddNamed(c.nextSketchName(), plane)
}

// nextSketchName mints the first unused Sketch{N} name, advancing the counter past
// names already taken (so a restored "Sketch3" doesn't collide with a later Add).
func (c *Sketches) nextSketchName() string {
	for {
		c.seq++
		name := fmt.Sprintf("Sketch%d", c.seq)
		if !c.nameTaken(name) {
			return name
		}
	}
}

// nameTaken reports whether a sketch in the collection already uses name.
func (c *Sketches) nameTaken(name string) bool {
	for _, s := range c.items {
		if s.Name() == name {
			return true
		}
	}
	return false
}

// AddNamed creates a named planar sketch on plane.
func (c *Sketches) AddNamed(name string, plane Plane) *Sketch {
	s := &Sketch{base: newBase(name), plane: plane}
	s.initCollections()
	if c.params != nil {
		s.SetParameters(c.params) // before any dimensions are added (live or on restore)
	}
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
}

// restoreSketchID pins a freshly-added sketch's local id to its persisted value so the
// sketch's document-derived persistent reference key (#153) is stable across load, re-keying
// the byID index and raising the id clock past it. A zero saved id (a legacy recipe with no
// persisted sketch id) keeps the minted one.
func (c *Sketches) restoreSketchID(s *Sketch, saved uint64) {
	if saved == 0 {
		return
	}
	delete(c.byID, s.id)
	s.id = ID(saved)
	c.byID[s.id] = s
	raiseIDSeq(saved)
}

// Count returns the number of sketches.
func (c *Sketches) Count() int { return len(c.items) }

// Item returns the sketch at index i (0-based).
func (c *Sketches) Item(i int) *Sketch { return c.items[i] }

// ByID returns the sketch with the given id.
func (c *Sketches) ByID(id ID) (*Sketch, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// Remove deletes the sketch with the given id, reporting whether it was found. The
// auto-name counter is not rewound — Inventor does not reuse a deleted sketch's number.
func (c *Sketches) Remove(id ID) bool {
	if _, ok := c.byID[id]; !ok {
		return false
	}
	delete(c.byID, id)
	for i, s := range c.items {
		if s.id == id {
			c.items = append(c.items[:i], c.items[i+1:]...)
			break
		}
	}
	return true
}
