// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"strconv"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/sketch"
)

// WorkRef is a stable, serializable reference to a work feature or origin element —
// the way work features (and revolve) name the geometry they are built on, mirroring
// Inventor's reference model (every work feature is defined relative to references,
// not absolute coordinates). Origin elements have well-known refs; user work features
// are referenced by their position in their collection ("plane/2", "axis/0"), which is
// deterministic because they are recreated in creation order on recompute/reopen — the
// same index-as-stable-ref scheme sketches and patterns already use.
type WorkRef string

// Well-known origin references — the static coordinate system every part document has
// from creation (Inventor's "Origin" folder): a center point, the X/Y/Z axes, and the
// XY/XZ/YZ planes. They are grounded coordinate-system elements. The reference strings
// are the canonical, Apache-2.0 contract vocabulary ([types] WorkRef*), aliased here so
// /source and the public API can never drift (ADR-0018).
const (
	OriginCenter  WorkRef = types.WorkRefCenter
	OriginXAxis   WorkRef = types.WorkRefXAxis
	OriginYAxis   WorkRef = types.WorkRefYAxis
	OriginZAxis   WorkRef = types.WorkRefZAxis
	OriginXYPlane WorkRef = types.WorkRefXYPlane
	OriginXZPlane WorkRef = types.WorkRefXZPlane
	OriginYZPlane WorkRef = types.WorkRefYZPlane
)

// workResolver resolves a WorkRef to its current geometry. [WorkGeometry] implements
// it; work-feature definitions resolve their references through it at recompute.
type workResolver interface {
	plane(WorkRef) (sketch.Plane, error)
	axis(WorkRef) (*WorkAxis, error)
	point(WorkRef) (math.Point3, error)
	surface(WorkRef) (geom.Surface, error)
	edge(WorkRef) (*topo.Edge, error)
}

// WorkGeometry is a part's construction-geometry frame: the static origin coordinate
// system plus the user-created work planes, axes, points and coordinate systems. It is
// the single owner so references between work features (and from revolve) resolve in
// one place. A part component definition holds exactly one.
type WorkGeometry struct {
	planes  *WorkPlanes
	axes    *WorkAxes
	points  *WorkPoints
	ucs     *UserCoordinateSystems
	userSeq []userEntry  // user work features in global creation order (for serialization)
	bodies  []*topo.Body // the running solid result, so surface-tangent work planes can
	// resolve a picked B-rep face's reference key to its surface (set each Recompute).
	tracker FootprintTracker // captures each plane's parameter footprint (set by the part); nil ⇒ no capture
}

// FootprintTracker captures the dependency keys read while fn runs — satisfied by
// *param.Parameters (TrackKeys). The part injects it so each work plane records the
// parameter driving its offset/angle, attributing a work-plane-offset edit to the sketches
// hosted on that plane instead of forcing a wholesale rebuild (ADR-0044).
type FootprintTracker interface {
	TrackKeys(fn func()) []depend.Key
}

// SetFootprintTracker injects the per-recompute footprint capture (see [FootprintTracker]).
// With no tracker, plane recompute is unchanged and offset edits fall back to wholesale —
// safe, just not incremental.
func (g *WorkGeometry) SetFootprintTracker(t FootprintTracker) { g.tracker = t }

// userEntry records a user work feature's collection and index in global creation
// order, so serialization replays them in dependency order — a work feature may
// reference an earlier one (e.g. a point on a user plane).
type userEntry struct {
	collection string
	index      int
}

// recordUser appends a freshly added user work feature to the creation-order log.
func (g *WorkGeometry) recordUser(collection string, index int) {
	g.userSeq = append(g.userSeq, userEntry{collection: collection, index: index})
}

// NewWorkGeometry builds the frame pre-populated with the grounded origin coordinate
// system (center point, X/Y/Z axes, XY/XZ/YZ planes) — the absolute document reference
// frame every other work feature is built relative to.
func NewWorkGeometry() *WorkGeometry {
	g := &WorkGeometry{}
	g.planes = newWorkPlanes(g)
	g.axes = newWorkAxes(g)
	g.points = newWorkPoints(g)
	g.ucs = NewUserCoordinateSystems()
	g.seedOrigin()
	return g
}

// WorkPlanes/WorkAxes/WorkPoints/UserCoordinateSystems expose the collections (origin
// elements first, then user features).
func (g *WorkGeometry) WorkPlanes() *WorkPlanes                       { return g.planes }
func (g *WorkGeometry) WorkAxes() *WorkAxes                           { return g.axes }
func (g *WorkGeometry) WorkPoints() *WorkPoints                       { return g.points }
func (g *WorkGeometry) UserCoordinateSystems() *UserCoordinateSystems { return g.ucs }

// OriginPlanes returns the three origin coordinate-system planes (XY/XZ/YZ) — the
// browser's Origin folder and the default sketch hosts.
func (g *WorkGeometry) OriginPlanes() []*WorkPlane {
	out := make([]*WorkPlane, 0, 3)
	for i := 0; i < g.planes.Count(); i++ {
		if p := g.planes.Item(i); p.coordinateSystem {
			out = append(out, p)
		}
	}
	return out
}

// Recompute re-derives every work feature in order (origin first, grounded; then user
// features, which may reference earlier ones). bodies is the part's current solid
// result, against which surface-tangent work planes resolve their picked faces; pass
// nil when no body exists yet (those planes then go Sick until one does). The part
// recomputes work geometry once before the feature program (so features can reference
// work axes/planes) and once after (so tangent planes see the freshly built body).
func (g *WorkGeometry) Recompute(bodies []*topo.Body) {
	g.bodies = bodies
	for i := 0; i < g.points.Count(); i++ {
		g.points.Item(i).recompute(g)
	}
	for i := 0; i < g.axes.Count(); i++ {
		g.axes.Item(i).recompute(g)
	}
	for i := 0; i < g.planes.Count(); i++ {
		g.recomputePlane(g.planes.Item(i))
	}
}

// recomputePlane recomputes one plane, capturing the parameters its definition read (its
// offset/angle) into the plane's footprint when a tracker is set. The capture nests inside
// the part's whole-recompute tracking, so the union still reaches the wholesale calculation
// while each plane also gets its own footprint (ADR-0044).
func (g *WorkGeometry) recomputePlane(p *WorkPlane) {
	if g.tracker == nil {
		p.recompute(g)
		return
	}
	p.paramFootprint = g.tracker.TrackKeys(func() { p.recompute(g) })
}

// seedOrigin creates the grounded origin coordinate-system elements with their
// well-known refs. Their geometry is fixed (the document's absolute frame).
func (g *WorkGeometry) seedOrigin() {
	center := math.P3(0, 0, 0)
	g.points.addOrigin(OriginCenter, "Center Point", center)
	g.axes.addOrigin(OriginXAxis, "X Axis", center, mustUnit(1, 0, 0))
	g.axes.addOrigin(OriginYAxis, "Y Axis", center, mustUnit(0, 1, 0))
	g.axes.addOrigin(OriginZAxis, "Z Axis", center, mustUnit(0, 0, 1))
	g.planes.addOrigin(OriginXYPlane, "XY Plane", sketch.XYPlane())
	g.planes.addOrigin(OriginXZPlane, "XZ Plane", sketch.XZPlane())
	g.planes.addOrigin(OriginYZPlane, "YZ Plane", sketch.YZPlane())
}

// plane/axis/point implement workResolver, resolving a ref to its current geometry.
func (g *WorkGeometry) plane(ref WorkRef) (sketch.Plane, error) {
	if pl, isFace, err := g.facePlane(ref); isFace {
		return pl, err // a planar B-rep face used as a plane input (offset-from-face, etc.)
	}
	if i, ok := userIndex(ref, "plane"); ok {
		if i < 0 || i >= g.planes.Count() {
			return sketch.Plane{}, fmt.Errorf(errNoWorkPlane, ref)
		}
		wp := g.planes.Item(i)
		if wp.deleted {
			return sketch.Plane{}, fmt.Errorf("work geometry: work plane %q was deleted", ref)
		}
		return wp.Plane(), nil
	}
	w, ok := g.planes.byKey[ref]
	if !ok {
		return sketch.Plane{}, fmt.Errorf("work geometry: unknown plane reference %q", ref)
	}
	return w.Plane(), nil
}

// ResolvePlaneRef resolves any plane reference — a planar B-rep face key, a user work plane
// ("plane/N"), or an origin plane ("origin/plane/xy") — to its plane (origin + normal). It is
// the public entry the feature-edit re-pick (#163) uses to turn a wire plane ref into a
// mirror's plane; errors when the ref names no resolvable plane.
func (g *WorkGeometry) ResolvePlaneRef(ref WorkRef) (sketch.Plane, error) { return g.plane(ref) }

// WorkPlaneByRef resolves a WorkRef to its *WorkPlane (origin or user) — for features
// that reference a datum plane, e.g. an extrude's to-face / from-to target.
func (g *WorkGeometry) WorkPlaneByRef(ref WorkRef) (*WorkPlane, error) {
	if i, ok := userIndex(ref, "plane"); ok {
		if i < 0 || i >= g.planes.Count() {
			return nil, fmt.Errorf(errNoWorkPlane, ref)
		}
		wp := g.planes.Item(i)
		if wp.deleted {
			return nil, fmt.Errorf("work geometry: work plane %q was deleted", ref)
		}
		return wp, nil
	}
	w, ok := g.planes.byKey[ref]
	if !ok {
		return nil, fmt.Errorf("work geometry: unknown plane reference %q", ref)
	}
	return w, nil
}

func (g *WorkGeometry) axis(ref WorkRef) (*WorkAxis, error) {
	if i, ok := userIndex(ref, "axis"); ok {
		if i < 0 || i >= g.axes.Count() {
			return nil, fmt.Errorf("work geometry: no work axis %q", ref)
		}
		wa := g.axes.Item(i)
		if wa.deleted {
			return nil, fmt.Errorf("work geometry: work axis %q was deleted", ref)
		}
		return wa, nil
	}
	w, ok := g.axes.byKey[ref]
	if !ok {
		return nil, fmt.Errorf("work geometry: unknown axis reference %q", ref)
	}
	return w, nil
}

// AxisByRef returns the work axis with the given reference (e.g. [OriginYAxis]) — the
// exported lookup the UI uses to resolve a chosen revolve/coil axis. Reports false when
// no such axis exists.
func (g *WorkGeometry) AxisByRef(ref WorkRef) (*WorkAxis, bool) {
	a, err := g.axis(ref)
	if err != nil {
		return nil, false
	}
	return a, true
}

// WorkPointByRef returns the datum work point with the given reference (e.g. [OriginCenter]),
// or false when none exists — the exported lookup projecting the origin centre point into a
// sketch uses (#1262). It resolves only datum points (origin frame + user work points), not
// B-rep vertex references.
func (g *WorkGeometry) WorkPointByRef(ref WorkRef) (*WorkPoint, bool) {
	if i, ok := userIndex(ref, "point"); ok {
		if i < 0 || i >= g.points.Count() {
			return nil, false
		}
		wp := g.points.Item(i)
		if wp.deleted {
			return nil, false
		}
		return wp, true
	}
	w, ok := g.points.byKey[ref]
	return w, ok
}

func (g *WorkGeometry) point(ref WorkRef) (math.Point3, error) {
	if p, isVertex, err := g.vertexPoint(ref); isVertex {
		return p, err
	}
	if i, ok := userIndex(ref, "point"); ok {
		if i < 0 || i >= g.points.Count() {
			return math.Point3{}, fmt.Errorf("work geometry: no work point %q", ref)
		}
		wp := g.points.Item(i)
		if wp.deleted {
			return math.Point3{}, fmt.Errorf("work geometry: work point %q was deleted", ref)
		}
		return wp.Point(), nil
	}
	w, ok := g.points.byKey[ref]
	if !ok {
		return math.Point3{}, fmt.Errorf("work geometry: unknown point reference %q", ref)
	}
	return w.Point(), nil
}

// validateRedefineRef rejects ref as a redefine input for the work feature keyed self when
// binding it would create a reference cycle: the feature depending on itself, directly or
// through other work features. A cyclic definition re-derives from its own previous result
// and drifts on EVERY recompute — an offset plane offset from itself moves by the offset
// each pass — silently corrupting geometry. It also rejects a user reference that names no
// existing work feature, so a typo'd wire repick fails loudly instead of leaving a silently
// sick plane. Origin elements and B-rep face/vertex references are terminal (a cycle through
// the feature program, e.g. tangent to a face extruded off this plane, is out of this walk's
// scope — the same exposure the creation flow has).
func (g *WorkGeometry) validateRedefineRef(ref WorkRef, self WorkRef) error {
	seen := map[WorkRef]bool{}
	var walk func(r WorkRef) error
	walk = func(r WorkRef) error {
		if r == self {
			return fmt.Errorf("work geometry: %q would make %q depend on itself (a cyclic definition drifts on every recompute)", ref, self)
		}
		if seen[r] {
			return nil
		}
		seen[r] = true
		refs, isUser, err := g.userFeatureRefs(r)
		if err != nil {
			return err
		}
		if !isUser {
			return nil
		}
		for _, rr := range refs {
			if err := walk(rr); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(ref)
}

// userFeatureRefs returns the definition references of the user work feature ref names;
// isUser is false for anything else (origin elements, B-rep faces/vertices). A user
// reference whose index does not exist is an error naming the offending ref.
func (g *WorkGeometry) userFeatureRefs(ref WorkRef) (refs []WorkRef, isUser bool, err error) {
	if i, ok := userIndex(ref, "plane"); ok {
		if i < 0 || i >= g.planes.Count() {
			return nil, true, fmt.Errorf(errNoWorkPlane, ref)
		}
		return g.planes.Item(i).def.refs(), true, nil
	}
	if i, ok := userIndex(ref, "axis"); ok {
		if i < 0 || i >= g.axes.Count() {
			return nil, true, fmt.Errorf("work geometry: no work axis %q", ref)
		}
		return g.axes.Item(i).def.refs(), true, nil
	}
	if i, ok := userIndex(ref, "point"); ok {
		if i < 0 || i >= g.points.Count() {
			return nil, true, fmt.Errorf("work geometry: no work point %q", ref)
		}
		return g.points.Item(i).def.refs(), true, nil
	}
	return nil, false, nil
}

// userIndex parses a "<collection>/<i>" user-feature reference into its index.
func userIndex(ref WorkRef, collection string) (int, bool) {
	prefix := collection + "/"
	s := string(ref)
	if !strings.HasPrefix(s, prefix) {
		return 0, false
	}
	i, err := strconv.Atoi(strings.TrimPrefix(s, prefix))
	if err != nil {
		return 0, false
	}
	return i, true
}

// userRef builds the stable reference for the i-th user feature in a collection.
func userRef(collection string, i int) WorkRef {
	return WorkRef(collection + "/" + strconv.Itoa(i))
}

func mustUnit(x, y, z float64) math.UnitVector3 {
	u, _ := math.NewUnitVector3(x, y, z)
	return u
}
