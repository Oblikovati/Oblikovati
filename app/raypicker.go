// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// RayPicker is the real headless hit-test: it casts a camera ray through the clicked
// pixel and finds the nearest face of the scene bodies (the same query the GPU
// ID-buffer answers in production) and the nearest origin work plane. It implements
// [Picker], so a test "clicks on" a modeled solid or a datum plane — screen coordinate
// → ray → face/plane — with no GPU.
type RayPicker struct {
	camera       scene.Camera
	bodies       func() []*topo.Body
	rayBodies    func(origin math.Point3, dir math.Vector3) []*topo.Body // spatial-index ray query (assemblies, M34-F5)
	planes       func() []*feature.WorkPlane
	points       func() []*feature.WorkPoint
	axes         func() []*feature.WorkAxis
	sketches     func() []*sketch.Sketch
	sketches3D   func() []*sketch.Sketch3D
	meshes       func() []*feature.MeshFeature                   // placed mesh references whose facets pick (#1776)
	meshIndex    map[*feature.MeshGeometry]*meshRayEntry         // per-mesh ray BVH, built once (immutable geometry)
	pointClouds  func() []*pointcloud.PointCloud                 // attached scans whose points snap (#645)
	occurrenceOf func(*topo.Body) (*occurrence.Occurrence, bool) // maps an assembly body hit to its component (#769)
	priorityRank func(SelectionKind) int                         // user pick-priority ranking for ambiguous hits (#1222); nil ⇒ default order
}

// SetPriorityRank installs the user-defined pick priority (lower rank = higher priority). The
// Session pushes its SelectionFilterState.Rank so reordering the Selection Filter window changes
// which kind wins an ambiguous pick (#1222). A nil ranker keeps the historical default order.
func (p *RayPicker) SetPriorityRank(rank func(SelectionKind) int) { p.priorityRank = rank }

// defaultRankByKind is the historical pick precedence used when no user ranking is installed:
// the snapPick exact targets (datum point/axis, cloud point, vertex, edge) outrank the
// depth-sorted occurrence/body/face/plane/profile/sketch hits, matching the old fixed snapPick
// sequence and RayPicker.Pick append order.
var defaultRankByKind = func() map[SelectionKind]int {
	m := make(map[SelectionKind]int)
	for i, k := range defaultFilterableKinds() {
		m[k] = i
	}
	return m
}()

// rankOf returns the priority rank of a kind (0 = highest); unranked kinds sort last.
func (p *RayPicker) rankOf(k SelectionKind) int {
	if p.priorityRank != nil {
		return p.priorityRank(k)
	}
	if r, ok := defaultRankByKind[k]; ok {
		return r
	}
	return len(defaultRankByKind)
}

// highestPriority returns the candidate with the lowest rank (the user's top priority among the
// co-located exact snaps), or false when there are none.
func (p *RayPicker) highestPriority(cands []Selectable) (Selectable, bool) {
	best, bestRank := -1, stdmath.MaxInt
	for i, c := range cands {
		if r := p.rankOf(c.SelectionKind()); r < bestRank {
			best, bestRank = i, r
		}
	}
	if best < 0 {
		return nil, false
	}
	return cands[best], true
}

// pickPixelRadius is how close (in pixels) the cursor must be to a datum point or axis to
// snap it — small explicit targets are picked by screen proximity, not by depth.
const pickPixelRadius = 8.0

// NewRayPicker builds a picker over a camera and a provider of the current scene
// bodies (e.g. the active part's SurfaceBodies).
func NewRayPicker(camera scene.Camera, bodies func() []*topo.Body) *RayPicker {
	return &RayPicker{camera: camera, bodies: bodies}
}

// WithPlanes adds a provider of selectable work planes (origin AND visible user datums),
// so the picker can resolve a click on a datum plane in empty space — e.g. to pick a
// ribbon-created plane as a new sketch's host.
func (p *RayPicker) WithPlanes(planes func() []*feature.WorkPlane) *RayPicker {
	p.planes = planes
	return p
}

// WithSketches adds a provider of the part's (visible) sketches, so the picker can
// resolve a click inside a sketch profile region — what an extrude/revolve consumes.
func (p *RayPicker) WithSketches(sketches func() []*sketch.Sketch) *RayPicker {
	p.sketches = sketches
	return p
}

// WithSketches3D adds a provider of the part's (visible) 3D sketches, so the picker can
// resolve a click on a 3D-sketch curve or point — what the 3D constraint tools consume
// (issue #142).
func (p *RayPicker) WithSketches3D(sketches3D func() []*sketch.Sketch3D) *RayPicker {
	p.sketches3D = sketches3D
	return p
}

// WithMeshes adds a provider of the part's visible placed mesh references, so a click can select a
// mesh by hitting one of its facets (#1776). Ray-testing uses a cached per-mesh BVH.
func (p *RayPicker) WithMeshes(meshes func() []*feature.MeshFeature) *RayPicker {
	p.meshes = meshes
	return p
}

// WithPointClouds adds a provider of the part's visible attached scans, so the cursor snaps to a
// scan point and a point-collecting tool can model against it (M17-F06, #645).
func (p *RayPicker) WithPointClouds(clouds func() []*pointcloud.PointCloud) *RayPicker {
	p.pointClouds = clouds
	return p
}

// WithPoints / WithAxes add providers of the part's datum points and axes, so a click on
// one snaps to it (the reference inputs for point/axis-driven work planes).
func (p *RayPicker) WithPoints(points func() []*feature.WorkPoint) *RayPicker {
	p.points = points
	return p
}

func (p *RayPicker) WithAxes(axes func() []*feature.WorkAxis) *RayPicker {
	p.axes = axes
	return p
}

// WithOccurrenceLookup adds a body→occurrence map so a click on an assembly component's body
// resolves to that occurrence (component-level selection). Provided only for assemblies; without
// it the picker behaves as the part-only hit-test (#769).
func (p *RayPicker) WithOccurrenceLookup(lookup func(*topo.Body) (*occurrence.Occurrence, bool)) *RayPicker {
	p.occurrenceOf = lookup
	return p
}

// WithRayBodies adds a ray-aware body provider — a spatial index that returns only the bodies a
// given ray could hit — so face/edge/vertex hit-tests in a large assembly skip the placements
// the ray misses instead of scanning (and materializing) every world body (M34-F5). A nil
// result means "no index for this scene" and the picker falls back to the full body list.
func (p *RayPicker) WithRayBodies(query func(origin math.Point3, dir math.Vector3) []*topo.Body) *RayPicker {
	p.rayBodies = query
	return p
}

// candidateBodies returns the bodies to hit-test for one ray: the spatial-index candidates when
// a ray-body provider is set and answers for this scene (assemblies — only placements the ray
// crosses), otherwise the full scene body list (parts, which are small). A non-nil result, even
// when empty, is authoritative: it means the index ran and the ray crossed nothing.
func (p *RayPicker) candidateBodies(origin math.Point3, dir math.Vector3) []*topo.Body {
	if p.rayBodies != nil {
		if cand := p.rayBodies(origin, dir); cand != nil {
			return cand
		}
	}
	return p.bodies()
}

// SetCamera updates the view used for picking.
func (p *RayPicker) SetCamera(c scene.Camera) { p.camera = c }

// Pick returns the nearest selectable under the pixel honoring the filter: a face hit
// (or its owning body) when a solid is in front, a sketch profile region the ray lands
// in, or the nearest origin work plane whose finite display square the ray crosses —
// whichever the filter accepts and is closest. Ties favor faces, then planes, then
// profiles (the append order), so a solid in front wins over the sketch on its face.
func (p *RayPicker) Pick(x, y float64, filter *SelectionFilter) (Selectable, bool) {
	origin, dir := p.camera.RayThrough(x, y)
	if sel, ok := p.snapPick(origin, dir, filter); ok {
		return sel, true
	}
	var cands []pickCandidate
	if face, body, t := p.nearestFace(origin, dir); face != nil {
		// In an assembly a click selects the whole component by default, so the occurrence
		// candidate is appended BEFORE the face — it wins the depth tie under an all-accepting
		// filter. A machining tool that sets a face/edge filter excludes SelectOccurrence and so
		// gets the face instead.
		if occ, ok := p.occurrenceForBody(body, filter); ok {
			cands = append(cands, pickCandidate{t, OccurrenceHandle{Occurrence: occ}})
		}
		if sel, ok := facePick(face, body, filter); ok {
			cands = append(cands, pickCandidate{t, sel})
		}
	}
	if plane, t := p.nearestPlane(origin, dir); plane != nil && filter.Accepts(SelectWorkPlane) {
		cands = append(cands, pickCandidate{t, WorkPlaneHandle{Plane: plane}})
	}
	if sel, t, ok := p.nearestProfile(origin, dir, filter); ok {
		cands = append(cands, pickCandidate{t, sel})
	}
	if sel, t, ok := p.nearestSketchCurve(origin, dir, filter); ok {
		cands = append(cands, pickCandidate{t, sel})
	}
	if sel, t, ok := p.nearestSketch3DEntity(origin, dir, filter); ok {
		cands = append(cands, pickCandidate{t, sel})
	}
	if sel, t, ok := p.nearestMeshFacet(origin, dir, filter); ok {
		cands = append(cands, pickCandidate{t, sel})
	}
	return p.nearestCandidate(cands)
}

// meshRayEntry caches a placed mesh's ray BVH and its triangle→facet map. A mesh geometry is
// immutable, so the index is built once per geometry and reused across picks.
type meshRayEntry struct {
	index   *tessellate.MeshRayIndex
	facetOf []int
}

// nearestMeshFacet returns the placed-mesh facet the ray hits nearest, or ok=false. The per-mesh BVH
// keeps this cheap enough to run on hover every frame even for a dense scan (#1776).
func (p *RayPicker) nearestMeshFacet(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, float64, bool) {
	if p.meshes == nil || !filter.Accepts(SelectMeshFace) {
		return nil, stdmath.Inf(1), false
	}
	var hit Selectable
	best := stdmath.Inf(1)
	for _, mf := range p.meshes() {
		e := p.meshRayEntry(mf.Geometry())
		if e.index == nil {
			continue
		}
		if tri, t, ok := e.index.Nearest(origin, dir); ok && t < best {
			best, hit = t, MeshFaceHandle{Mesh: mf, Facet: e.facetOf[tri]}
		}
	}
	return hit, best, hit != nil
}

// meshRayEntry returns g's cached ray index, building it on first use.
func (p *RayPicker) meshRayEntry(g *feature.MeshGeometry) *meshRayEntry {
	if p.meshIndex == nil {
		p.meshIndex = map[*feature.MeshGeometry]*meshRayEntry{}
	}
	if e, ok := p.meshIndex[g]; ok {
		return e
	}
	tris, facetOf := meshTriangles(g)
	e := &meshRayEntry{index: tessellate.NewMeshRayIndex(g.Vertices, tris), facetOf: facetOf}
	p.meshIndex[g] = e
	return e
}

// nearestSketch3DEntity returns the 3D-sketch entity (curve or standalone point) the
// ray passes nearest within the pick radius, in any visible 3D sketch, when the
// filter accepts sketch entities — how the 3D constraint tools receive their picks
// (issue #142). Curves test against the same sampled polyline the overlay draws.
func (p *RayPicker) nearestSketch3DEntity(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, float64, bool) {
	if p.sketches3D == nil || !filter.Accepts(SelectSketchEntity) {
		return nil, stdmath.Inf(1), false
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit Selectable
	best := stdmath.Inf(1)
	for _, sk := range p.sketches3D() {
		for _, e := range sk.Entities() {
			if d, t, ok := raySketch3DEntityDistance(origin, dir, e); ok && d <= tol && t < best {
				best, hit = t, SketchEntityHandle{Entity: e}
			}
		}
	}
	return hit, best, hit != nil
}

// raySketch3DEntityDistance returns the closest approach between the ray and an
// entity's sampled polyline (a standalone point is its single sample).
func raySketch3DEntityDistance(origin math.Point3, dir math.Vector3, e sketch.Entity) (dist, t float64, ok bool) {
	pts := sketch.SamplePolyline3D(e, pick3DCurveSegments)
	if len(pts) == 0 {
		return 0, 0, false
	}
	if len(pts) == 1 {
		return rayPointDistance(origin, dir, pts[0])
	}
	dist, t = stdmath.Inf(1), stdmath.Inf(1)
	for i := 0; i+1 < len(pts); i++ {
		if d, s, segOK := raySegmentDistance(origin, dir, pts[i], pts[i+1]); segOK && d < dist {
			dist, t, ok = d, s, true
		}
	}
	return dist, t, ok
}

// rayPointDistance returns the distance from the forward ray to a point and the ray
// parameter of the closest approach; ok is false behind the ray origin.
func rayPointDistance(origin math.Point3, dir math.Vector3, pt math.Point3) (dist, t float64, ok bool) {
	t = float64(origin.VectorTo(pt).Dot(dir))
	if t <= 0 {
		return 0, 0, false
	}
	return float64(origin.TranslateBy(dir.Scale(math.Scalar(t))).DistanceTo(pt)), t, true
}

// pick3DCurveSegments is the sample budget for picking curved 3D entities — matches
// the overlay's draw sampling so what you see is what you pick.
const pick3DCurveSegments = 48

// nearestSketchCurve returns the sketch entity the ray passes nearest (within the pick radius, in
// any visible sketch), when the filter accepts sketch entities — so any imported curve (line,
// arc, circle, ellipse, elliptical arc, spline) can be picked in the part view, e.g. a line as a
// revolve axis. The spatial index facets every entity, so curves are hit along their true sweep.
func (p *RayPicker) nearestSketchCurve(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, float64, bool) {
	if p.sketches == nil || !filter.Accepts(SelectSketchEntity) {
		return nil, stdmath.Inf(1), false
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit Selectable
	best := stdmath.Inf(1)
	for _, sk := range p.sketches() {
		// Test only the segments near the ray (cached spatial index), not every
		// line — a dense imported sketch has hundreds of thousands.
		pickIndexFor(sk).forCandidates(origin, dir, tol, func(seg sketchSeg) {
			if d, t, ok := raySegmentDistance(origin, dir, seg.a, seg.b); ok && d <= tol && t < best {
				best, hit = t, SketchEntityHandle{Entity: seg.entity}
			}
		})
	}
	return hit, best, hit != nil
}

// snapPick returns the precise snap the cursor lands on — a datum point, datum axis, cloud
// point, body vertex, or edge — each an exact target within the pixel-snap radius that wins over
// the face behind it. When several are co-located the user's priority order decides which wins
// (#1222); by default that is the historical point→axis→cloud→vertex→edge precedence.
func (p *RayPicker) snapPick(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, bool) {
	var snaps []Selectable
	if pt, _ := p.nearestPoint(origin, dir); pt != nil && filter.Accepts(SelectWorkPoint) {
		snaps = append(snaps, WorkPointHandle{Point: pt})
	}
	if ax, _ := p.nearestAxis(origin, dir); ax != nil && filter.Accepts(SelectWorkAxis) {
		snaps = append(snaps, WorkAxisHandle{Axis: ax})
	}
	if pt, cloud, ok := p.nearestCloudPoint(origin, dir); ok && filter.Accepts(SelectPointCloudPoint) {
		snaps = append(snaps, PointCloudPointHandle{Cloud: cloud, Point: pt})
	}
	if v := p.nearestVertex(origin, dir); v != nil && filter.Accepts(SelectVertex) {
		snaps = append(snaps, VertexHandle{Vertex: v})
	}
	if e := p.nearestEdge(origin, dir); e != nil && filter.Accepts(SelectEdge) {
		snaps = append(snaps, EdgeHandle{Edge: e})
	}
	return p.highestPriority(snaps)
}

// pickCandidate is one ray hit: its forward parameter and the selectable it resolves to.
type pickCandidate struct {
	t   float64
	sel Selectable
}

// nearestCandidate returns the candidate the click resolves to: the one nearest along the ray,
// breaking near-coincident ties (within a screen-scaled depth window) by the user's pick priority
// so the Selection Filter ordering decides an ambiguous pick (#1222). With the default order the
// window collapses to the historical occurrence→face→plane→profile append precedence, and a
// candidate genuinely in front (more than the window nearer) always wins regardless of priority.
func (p *RayPicker) nearestCandidate(cands []pickCandidate) (Selectable, bool) {
	if len(cands) == 0 {
		return nil, false
	}
	minT := stdmath.Inf(1)
	for _, c := range cands {
		if c.t < minT {
			minT = c.t
		}
	}
	eps := p.depthTieEpsilon()
	best, bestRank := -1, stdmath.MaxInt
	for i, c := range cands {
		if c.t > minT+eps {
			continue // genuinely behind the front candidate — proximity wins over priority
		}
		if r := p.rankOf(c.sel.SelectionKind()); r < bestRank {
			best, bestRank = i, r
		}
	}
	return cands[best].sel, true
}

// depthTieEpsilon is the world-space depth window within which two hits count as the same pick
// for priority resolution — one pixel-snap radius at the view scale. A zero/unset camera yields
// 0, so only exact ties resolve by priority (the historical behaviour).
func (p *RayPicker) depthTieEpsilon() float64 {
	return pickPixelRadius * p.camera.WorldPerPixel()
}

// nearestProfile returns the closest sketch profile region the ray lands inside (mapped
// through each visible sketch's plane), and the ray parameter — when the filter accepts
// profiles. ok is false when no sketches are provided, profiles aren't accepted, or the
// ray misses every region.
func (p *RayPicker) nearestProfile(origin math.Point3, dir math.Vector3, filter *SelectionFilter) (Selectable, float64, bool) {
	if p.sketches == nil || !filter.Accepts(SelectProfile) {
		return nil, stdmath.Inf(1), false
	}
	var hit Selectable
	best := stdmath.Inf(1)
	for _, sk := range p.sketches() {
		t, uv, ok := rayPlanePoint(origin, dir, sk.Plane())
		if !ok || t >= best {
			continue
		}
		if idx, found := profileAt(sk, uv); found {
			best, hit = t, ProfileHandle{Sketch: sk, ProfileIndex: idx}
		}
	}
	return hit, best, hit != nil
}

// profileAt returns the index of the first profile in sk whose region contains the
// sketch-plane point uv, or false when uv is outside every profile.
func profileAt(sk *sketch.Sketch, uv math.Point2) (int, bool) {
	profiles := sk.Profiles()
	for i := 0; i < profiles.Count(); i++ {
		if profiles.Item(i).Contains(uv) {
			return i, true
		}
	}
	return 0, false
}

// rayPlanePoint intersects a ray with an (infinite) sketch plane, returning the forward
// ray parameter and the hit point in sketch (2D) coordinates.
func rayPlanePoint(origin math.Point3, dir math.Vector3, plane sketch.Plane) (float64, math.Point2, bool) {
	n := plane.Normal().AsVector()
	denom := dir.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, math.Point2{}, false
	}
	t := origin.VectorTo(plane.Origin()).Dot(n) / denom
	if t <= 0 {
		return 0, math.Point2{}, false
	}
	return t, plane.ToSketch(origin.TranslateBy(dir.Scale(t))), true
}

// nearestFace returns the closest ray-hit face, its body, and the ray parameter.
func (p *RayPicker) nearestFace(origin math.Point3, dir math.Vector3) (*topo.Face, *topo.Body, float64) {
	var hitFace *topo.Face
	var hitBody *topo.Body
	best := stdmath.Inf(1)
	for _, b := range p.candidateBodies(origin, dir) {
		if f, t, ok := query.RayCastFaces(b, origin, dir, ops.DefaultQuality()); ok && t < best {
			best, hitFace, hitBody = t, f, b
		}
	}
	return hitFace, hitBody, best
}

// nearestVertex returns the closest body vertex within the pixel-snap radius of the ray
// (nearest by ray depth on ties), or nil — the hit-test for picking model vertices (e.g. a
// three-point work plane through solid corners).
func (p *RayPicker) nearestVertex(origin math.Point3, dir math.Vector3) *topo.Vertex {
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *topo.Vertex
	best := stdmath.Inf(1)
	for _, b := range p.candidateBodies(origin, dir) {
		for _, v := range b.Vertices() {
			t := origin.VectorTo(v.Point()).Dot(dir)
			if t <= 0 || t >= best {
				continue
			}
			if origin.TranslateBy(dir.Scale(t)).DistanceTo(v.Point()) <= tol {
				best, hit = t, v
			}
		}
	}
	return hit
}

// nearestEdge returns the closest body edge whose polyline passes within the pixel-snap
// radius of the ray (nearest by ray depth on ties), or nil. Edges are sampled via the
// tessellator, so curved edges (arcs) pick too.
func (p *RayPicker) nearestEdge(origin math.Point3, dir math.Vector3) *topo.Edge {
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *topo.Edge
	best := stdmath.Inf(1)
	for _, b := range p.candidateBodies(origin, dir) {
		for _, e := range b.Edges() {
			pts := tessellate.TessellateEdge(e, ops.DefaultQuality())
			for i := 0; i+1 < len(pts); i++ {
				if d, t, ok := raySegmentDistance(origin, dir, pts[i], pts[i+1]); ok && d <= tol && t < best {
					best, hit = t, e
				}
			}
		}
	}
	return hit
}

// raySegmentDistance returns the closest distance between the forward ray (origin + t·dir,
// t ≥ 0; dir unit) and the segment [a, b], plus the ray parameter t at that closest point.
// ok is false for a degenerate segment.
func raySegmentDistance(origin math.Point3, dir math.Vector3, a, b math.Point3) (dist, t float64, ok bool) {
	e := a.VectorTo(b)
	eLen := e.Dot(e)
	if eLen < 1e-18 {
		return 0, 0, false
	}
	s, u := closestRaySegParams(dir, e, a.VectorTo(origin), eLen)
	p1 := origin.TranslateBy(dir.Scale(s))
	p2 := a.TranslateBy(e.Scale(u))
	return p1.DistanceTo(p2), s, true
}

// closestRaySegParams returns the ray parameter s (≥0) and segment parameter u (∈[0,1]) of
// the closest approach between ray (dir from origin) and segment direction e, where r is
// origin−a and eLen = e·e.
func closestRaySegParams(dir, e, r math.Vector3, eLen float64) (s, u float64) {
	bDot, c, f := dir.Dot(e), dir.Dot(r), e.Dot(r)
	if denom := dir.Dot(dir)*eLen - bDot*bDot; denom > 1e-12 {
		s = (bDot*f - c*eLen) / denom
	}
	if s < 0 {
		s = 0
	}
	switch u = (bDot*s + f) / eLen; {
	case u < 0:
		u, s = 0, max(0, -c)
	case u > 1:
		u, s = 1, max(0, bDot-c)
	}
	return s, u
}

// nearestPlane returns the closest origin work plane whose display square the ray
// crosses, and the ray parameter (or nil/Inf if none).
func (p *RayPicker) nearestPlane(origin math.Point3, dir math.Vector3) (*feature.WorkPlane, float64) {
	if p.planes == nil {
		return nil, stdmath.Inf(1)
	}
	var hit *feature.WorkPlane
	best := stdmath.Inf(1)
	for _, wp := range p.planes() {
		if t, ok := rayWorkPlane(origin, dir, wp); ok && t < best {
			best, hit = t, wp
		}
	}
	return hit, best
}

// occurrenceForBody resolves a body hit to its component occurrence, when the picker has a
// lookup (an assembly) and the filter accepts occurrence selection.
func (p *RayPicker) occurrenceForBody(body *topo.Body, filter *SelectionFilter) (*occurrence.Occurrence, bool) {
	if p.occurrenceOf == nil || !filter.Accepts(SelectOccurrence) {
		return nil, false
	}
	return p.occurrenceOf(body)
}

// facePick wraps a face hit as the handle the filter wants (face or owning body),
// reporting ok=false when there is no face or the filter admits neither kind.
func facePick(face *topo.Face, body *topo.Body, filter *SelectionFilter) (Selectable, bool) {
	switch {
	case face == nil:
		return nil, false
	case filter.Accepts(SelectFace):
		return FaceHandle{Face: face, Body: body}, true
	case filter.Accepts(SelectBody):
		return BodyHandle{Body: body}, true
	default:
		return nil, false
	}
}

// nearestPoint returns the closest datum point within the pixel-snap radius of the ray,
// and the forward ray parameter (or nil/Inf if none).
func (p *RayPicker) nearestPoint(origin math.Point3, dir math.Vector3) (*feature.WorkPoint, float64) {
	if p.points == nil {
		return nil, stdmath.Inf(1)
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *feature.WorkPoint
	best := stdmath.Inf(1)
	for _, wp := range p.points() {
		t := origin.VectorTo(wp.Point()).Dot(dir)
		if t <= 0 || t >= best {
			continue
		}
		if origin.TranslateBy(dir.Scale(t)).DistanceTo(wp.Point()) <= tol {
			best, hit = t, wp
		}
	}
	return hit, best
}

// nearestCloudPoint returns the displayed scan point closest to the ray within the pixel-snap
// radius, across every visible attached cloud, and its owning cloud (or ok=false if none). Only
// the budgeted, cropped DisplayedPoints are tested — what the user actually sees and can snap to.
func (p *RayPicker) nearestCloudPoint(origin math.Point3, dir math.Vector3) (math.Point3, *pointcloud.PointCloud, bool) {
	if p.pointClouds == nil {
		return math.Point3{}, nil, false
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit math.Point3
	var owner *pointcloud.PointCloud
	best := stdmath.Inf(1)
	for _, cloud := range p.pointClouds() {
		if !cloud.Visible() {
			continue
		}
		for _, pt := range cloud.DisplayedPoints() {
			t := origin.VectorTo(pt).Dot(dir)
			if t <= 0 || t >= best {
				continue
			}
			if origin.TranslateBy(dir.Scale(t)).DistanceTo(pt) <= tol {
				best, hit, owner = t, pt, cloud
			}
		}
	}
	return hit, owner, owner != nil
}

// nearestAxis returns the closest datum axis within the pixel-snap radius of the ray, and
// the forward ray parameter at the closest approach (or nil/Inf if none).
func (p *RayPicker) nearestAxis(origin math.Point3, dir math.Vector3) (*feature.WorkAxis, float64) {
	if p.axes == nil {
		return nil, stdmath.Inf(1)
	}
	tol := pickPixelRadius * p.camera.WorldPerPixel()
	var hit *feature.WorkAxis
	best := stdmath.Inf(1)
	for _, ax := range p.axes() {
		if t, d, ok := rayAxisDistance(origin, dir, ax); ok && t < best && d <= tol {
			best, hit = t, ax
		}
	}
	return hit, best
}

// rayAxisDistance returns the forward ray parameter and world distance at the closest
// approach between the camera ray (origin, unit dir) and an infinite work axis. ok is
// false when the ray is parallel to the axis or the closest approach is behind the camera.
func rayAxisDistance(origin math.Point3, dir math.Vector3, ax *feature.WorkAxis) (float64, float64, bool) {
	u := ax.Direction().AsVector()
	w0 := ax.Origin().VectorTo(origin) // O − A
	b := dir.Dot(u)
	denom := 1 - b*b
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, 0, false
	}
	d, e := dir.Dot(w0), u.Dot(w0)
	sc := (b*e - d) / denom // ray parameter at closest approach
	if sc <= 0 {
		return 0, 0, false
	}
	tc := (e - b*d) / denom // axis parameter at closest approach
	rayPoint := origin.TranslateBy(dir.Scale(sc))
	axisPoint := ax.Origin().TranslateBy(u.Scale(tc))
	return sc, rayPoint.DistanceTo(axisPoint), true
}

// rayWorkPlane intersects a ray with a work plane and reports the forward parameter
// when the hit lies within the plane's finite display square.
func rayWorkPlane(origin math.Point3, dir math.Vector3, wp *feature.WorkPlane) (float64, bool) {
	plane := wp.Plane()
	n := plane.Normal().AsVector()
	denom := dir.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return 0, false
	}
	t := origin.VectorTo(plane.Origin()).Dot(n) / denom
	if t <= 0 {
		return 0, false
	}
	uv := plane.ToSketch(origin.TranslateBy(dir.Scale(t)))
	if stdmath.Abs(uv.X) > wp.DisplaySize() || stdmath.Abs(uv.Y) > wp.DisplaySize() {
		return 0, false
	}
	return t, true
}
