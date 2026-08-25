// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// LoftCondition is the boundary tangency control at a loft's end section; the canonical
// definition lives in the Apache-2.0 api/types (see ADR-0018).
type LoftCondition = types.LoftCondition

// LoftAreaStop is one control point of a loft area graph (canonical in api/types).
type LoftAreaStop = types.LoftAreaStop

// Loft end conditions (aliases of the canonical api/types values).
const (
	LoftFree           = types.LoftFree
	LoftAngle          = types.LoftAngle
	LoftDirection      = types.LoftDirection
	LoftTangent        = types.LoftTangent
	LoftSmooth         = types.LoftSmooth
	LoftG3             = types.LoftG3
	LoftSharpPoint     = types.LoftSharpPoint
	LoftTangentToPlane = types.LoftTangentToPlane
)

// LoftEnd is the end-section condition for a loft start or end: how the surface leaves that
// section. Angle (radians, measured from the section's sketch plane) and Impact (takeoff
// weight, default 1) drive the Angle/Direction condition; Reversed flips the takeoff through
// the plane. The zero value is a Free end (the natural ruled/curved blend).
type LoftEnd struct {
	Condition LoftCondition
	Angle     float64
	Impact    float64
	Reversed  bool
}

// loftEnds carries the resolved start/end conditions plus the section-plane normals the skinner
// needs to build the angled takeoff tangents. firstSurf/lastSurf are the adjacent face surfaces for
// a face-continuity end (Tangent/Smooth on a face section): the skinner reads their real 1st/2nd
// derivatives so the loft continues that face's tangent (G1) and curvature (G2) across the section
// edge, instead of the normal-only approximation. They are nil for sketch/point sections.
type loftEnds struct {
	first, last         LoftEnd
	firstN, lastN       math.UnitVector3
	firstSurf, lastSurf geom.Surface
}

// loftGuides carries everything that shapes the OUTER skin beyond the plain blend: an explicit
// point correspondence (mapCurves; one anchor point per section), an area graph (cross-section
// area along the loft), rails (local pulls), and a centerline (bends the spine). Empty for the bore.
type loftGuides struct {
	mapCurves  [][]math.Point3
	areaGraph  []types.LoftAreaStop
	rails      [][]math.Point3
	centerline []math.Point3
}

// Sweep and loft generate real (faceted) solids through the shared swept-solid
// primitive. A sweep places the profile at each path point, oriented to the local
// path tangent; a loft blends through a list of profile sections (each on its own
// sketch plane), resampled to a common point count. Exact analytic/NURBS swept
// surfaces and guide-rail/centerline-twist control are later refinements.

// SweepDefinition is the recipe for a sweep: a profile swept along a 3D path. The path
// is a model-space point chain (a 3D sketch path) so the profile and path can lie on
// different planes — the meaningful sweep case.
type SweepDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Path         func() *sketch.Path3D // live path provider, re-derived each recompute
	Twist        func() float64        // total twist (radians) distributed along the path
	Operation    ops.PartFeatureOperation

	// The definition union (M08 PBI-094, #314) — see DefinitionType():
	Orientation   types.SweepProfileOrientation // 0 ⇒ normal to path
	AlignVector   math.Vector3                  // AlignToVector's fixed profile normal
	Taper         func() float64                // draft angle (radians) along the path
	TwistStations []SweepTwistStation           // pathAndSectionTwists rows (override Twist)
	GuideRail     func() *sketch.Path3D         // pathAndGuideRail steering/scaling rail
	// PathSketch / GuideRailSketch name the sketches the live Path / GuideRail providers read,
	// so the #1414 tail invalidation (and the browser's chronological nesting) can attribute a
	// parameter that drives the rail to this feature — the providers alone are opaque closures,
	// and without the attribution a rail-driving parameter edit re-solved the sketch but never
	// re-swept the body (silent stale geometry, Oblikovati#1693: the resized tubing kept its
	// old length). nil for a deserialized point-snapshot path (which nothing can re-drive).
	PathSketch      *sketch.Sketch
	GuideRailSketch *sketch.Sketch
	Scaling         types.SweepProfileScaling // rail scaling mode (0 ⇒ xy)
	GuideFaceKey    []byte                    // pathAndGuideSurface face (running bodies)
	// SolidToolIndex sweeps the running body at this index along the path
	// instead of a profile (the reference SolidSweepDefinition).
	SolidToolIndex *int
}

// SweepFeature sweeps a profile along a path.
type SweepFeature struct {
	def      *SweepDefinition
	featName string
	tool     *topo.Body // last swept solid, exposed so a pattern can replicate this feature
}

func (s *SweepFeature) Definition() *SweepDefinition { return s.def }
func (s *SweepFeature) Kind() string                 { return "sweep" }

// Operation and ToolBody expose this feature's boolean op and tool so a pattern/mirror can
// replicate it correctly (cut/join the swept solid at each occurrence) — see [ToolFeature].
func (s *SweepFeature) Operation() ops.PartFeatureOperation { return s.def.Operation }
func (s *SweepFeature) ToolBody() *topo.Body                { return s.tool }

// Recompute resolves the profile, places it along the path under the
// definition union's behavior (orientation, twist, rail, surface, taper) into
// a faceted solid, and applies the operation against the running bodies. A
// solid sweep instead drags a tool body along the path.
func (s *SweepFeature) Recompute(in Input) (Output, error) {
	if s.def.SolidToolIndex != nil {
		return s.recomputeSolidSweep(in)
	}
	prof, err := resolveSingleProfile(s.def.Sketch, s.def.ProfileIndex, "sweep")
	if err != nil {
		return Output{}, err
	}
	if s.def.Path == nil {
		return Output{}, errors.New("sweep: path needs at least two points")
	}
	path := s.def.Path()
	if path == nil || path.Count() < 2 {
		return Output{}, errors.New("sweep: path needs at least two points")
	}
	cfg, err := s.resolveSweepConfig(in, path.Points())
	if err != nil {
		return Output{}, err
	}
	s.tool, err = s.buildSweepTool(prof, path, cfg)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, s.tool, s.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildSweepTool builds the swept tool body. A rigid straight sweep of a circle / line+arc profile
// keeps ANALYTIC faces — a straight NormalToPath sweep is exactly an extrude along the path tangent,
// so it reuses the extrude analytic prism (#2164 follow-up) and a projected face sees real arcs, not
// chords. Every other sweep (bent path, taper/twist/scaling/rail/guide, a surface sweep, or a profile
// with holes) uses the faceted section skin.
func (s *SweepFeature) buildSweepTool(prof *sketch.Profile, path *sketch.Path3D, cfg sweepConfig) (*topo.Body, error) {
	feat := featOr(s.featName, "sweep")
	if s.def.Operation != ops.Surface && s.sweepIsRigid() {
		if body := analyticStraightSweep(prof, s.def.Sketch.Plane(), path.Points(), feat); body != nil {
			return body, nil
		}
	}
	sections, err := sweepSectionsCfg(prof, s.def.Sketch.Plane(), path.Points(), cfg)
	if err != nil {
		return nil, err
	}
	return s.sweepTool(sections, path.IsClosed())
}

// sweepIsRigid reports whether the definition sweeps the profile rigidly — no taper, twist, scaling
// rail, or guide surface, and the profile kept normal to the path. Only a rigid sweep maps to an
// analytic prism; any deformation makes the section vary along the path (a non-analytic skin).
func (s *SweepFeature) sweepIsRigid() bool {
	d := s.def
	return normalToPathOrientation(d.Orientation) &&
		callOrZero(d.Taper) == 0 &&
		callOrZero(d.Twist) == 0 &&
		len(d.TwistStations) == 0 &&
		d.GuideRail == nil &&
		len(d.GuideFaceKey) == 0
}

// normalToPathOrientation reports whether the orientation keeps the profile perpendicular to the path
// tangent — the explicit NormalToPath, or the zero value the definition defaults to (sweep.go).
func normalToPathOrientation(o types.SweepProfileOrientation) bool {
	return o == 0 || o == types.NormalToPath
}

// sweepTool builds the swept body from its cross-sections: for the Surface operation
// (kSurfaceOperation, #1858) an OPEN swept sheet (the profile boundary swept, no end caps) via
// sweptShell; otherwise the swept solid. combine() adds a surface tool as a surface body (no
// boolean).
func (s *SweepFeature) sweepTool(sections [][]math.Point3, closed bool) (*topo.Body, error) {
	feat := featOr(s.featName, "sweep")
	if s.def.Operation == ops.Surface {
		return sweptShell(sections, closed, feat)
	}
	return sweptSolid(sections, closed, feat)
}

// pathTangents returns a unit tangent at each path point: the forward segment at the
// start, the backward segment at the end, and the average of the two interior segments.
func pathTangents(path []math.Point3) []math.UnitVector3 {
	n := len(path)
	out := make([]math.UnitVector3, n)
	for k := range path {
		var v math.Vector3
		switch k {
		case 0:
			v = path[0].VectorTo(path[1])
		case n - 1:
			v = path[n-2].VectorTo(path[n-1])
		default:
			v = path[k-1].VectorTo(path[k]).Add(path[k].VectorTo(path[k+1]))
		}
		u, err := math.UnitVector3FromVector(v)
		if err != nil {
			u = math.V3(0, 0, 1).AsUnit()
		}
		out[k] = u
	}
	return out
}

// SweepFeatures adds sweeps into the engine.
type SweepFeatures struct{ engine *PartFeatures }

// NewSweepFeatures binds the collection to an engine.
func NewSweepFeatures(engine *PartFeatures) *SweepFeatures { return &SweepFeatures{engine} }

// Add adds a sweep of the profile along path with the given total twist and operation.
func (c *SweepFeatures) Add(skt *sketch.Sketch, profileIndex int, path *sketch.Path3D, twist func() float64, op ops.PartFeatureOperation) *PartFeature {
	return c.addLiveSweep(skt, profileIndex, func() *sketch.Path3D { return path }, twist, op)
}

// AddDefinition adds a sweep from a fully-populated definition — the union
// variants (#314) construct through this.
func (c *SweepFeatures) AddDefinition(def *SweepDefinition) *PartFeature {
	sf := &SweepFeature{def: def}
	pf := c.engine.Add(sf)
	pf.SetName(c.engine.UniqueName("Sweep"))
	sf.featName = pf.name
	return pf
}

// addLiveSweep is Add with a live path provider, re-derived on every recompute, so a
// parameter that drives the path sketch reshapes the sweep (the static Add snapshots
// the path and does not track edits).
func (c *SweepFeatures) addLiveSweep(skt *sketch.Sketch, profileIndex int, path func() *sketch.Path3D, twist func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &SweepDefinition{Sketch: skt, ProfileIndex: profileIndex, Path: path, Twist: twist, Operation: op}
	sf := &SweepFeature{def: def}
	pf := c.engine.Add(sf)
	pf.SetName(c.engine.UniqueName("Sweep"))
	sf.featName = pf.name
	return pf
}

// LoftSection identifies one cross-section of a loft: a closed profile on a sketch; or — when
// Point is set — a single point (an apex), so the loft tapers to a cone or a domed tip (valid
// only first or last); or — when FaceKey is set — an existing body face, so the loft can leave
// it tangent (Tangent/Smooth conditions). The face's boundary is the section geometry and its
// surface gives the takeoff; the key is resolved against the running bodies each recompute.
type LoftSection struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Point        *math.Point3
	FaceKey      []byte
}

// IsPoint reports whether this section is a point (apex) rather than a profile.
func (s LoftSection) IsPoint() bool { return s.Point != nil }

// IsFace reports whether this section is an existing body face (resolved by FaceKey).
func (s LoftSection) IsFace() bool { return len(s.FaceKey) > 0 }

// LoftDefinition is the recipe for a loft: a blend through ordered sections, optionally closed
// (the last section blends back to the first). First/Last carry the end-section conditions that
// let the surface curve away from a flat ruled blend (ignored when Closed — a closed loft has
// no end sections).
type LoftDefinition struct {
	Sections  []LoftSection
	Closed    bool
	Operation ops.PartFeatureOperation
	First     LoftEnd
	Last      LoftEnd
	// LiveEnds, when set, supplies the start/end conditions afresh on every recompute so a
	// parameter driving an end angle/impact reshapes the loft (the static First/Last are the
	// snapshot used otherwise). Mirrors SweepDefinition.Twist's live-provider pattern.
	LiveEnds func() (first, last LoftEnd)
	// Rails are optional guide curves (the kLoftWithRails mode): live providers of model-space
	// polylines that touch the end sections; the loft's outer surface is pulled to follow them.
	Rails []func() []math.Point3
	// Centerline is an optional spine curve (the kLoftWithCenterline mode): a live provider of a
	// model-space polyline the section centroids follow, so the loft bends along it. Mutually
	// exclusive with Rails (as in Inventor); if both are set the centerline is applied first.
	Centerline func() []math.Point3
	// AreaGraph is an optional cross-section area graph (the kLoftWithAreaGraphSections mode): area
	// scale stops along the loft; the section areas are scaled to follow it.
	AreaGraph []types.LoftAreaStop
	// MapCurves is an optional explicit point correspondence (MapPointCurves): live providers of one
	// anchor point per section; the first overrides the automatic minimum-twist alignment.
	MapCurves []func() []math.Point3
}

// LoftType reports the loft mode derived from the definition — the analogue of Inventor's
// LoftTypeEnum (regular, or guided by an area graph, centerline, or rails). MapCurves is a
// correspondence override, not a type, so it does not change this.
func (d *LoftDefinition) LoftType() types.LoftType {
	switch {
	case len(d.AreaGraph) > 0:
		return types.LoftWithAreaGraphSections
	case d.Centerline != nil:
		return types.LoftWithCenterline
	case len(d.Rails) > 0:
		return types.LoftWithRails
	default:
		return types.RegularLoft
	}
}

// LoftFeature blends through sections.
type LoftFeature struct {
	def      *LoftDefinition
	featName string
	tool     *topo.Body // last lofted solid, exposed so a pattern can replicate this feature
}

func (l *LoftFeature) Definition() *LoftDefinition { return l.def }
func (l *LoftFeature) Kind() string                { return "loft" }

// Operation and ToolBody expose this feature's boolean op and tool so a pattern/mirror can
// replicate it correctly (cut/join the lofted solid at each occurrence) — see [ToolFeature].
func (l *LoftFeature) Operation() ops.PartFeatureOperation { return l.def.Operation }
func (l *LoftFeature) ToolBody() *topo.Body                { return l.tool }

// Recompute skins the loft solid through the sections. The outer loops skin the body; a single
// inner loop per section (the common pipe) is meshed directly into a hollow tube, so a loft of
// annulus sections is a watertight pipe rather than a filled cone.
func (l *LoftFeature) Recompute(in Input) (Output, error) {
	outers, inners, normals, surfs, err := l.resolveSections(in.Bodies)
	if err != nil {
		return Output{}, err
	}
	tool, err := l.skinTool(outers, inners, l.endsWith(normals, surfs), l.resolveGuides(), in.Diag)
	if err != nil {
		return Output{}, err
	}
	l.tool = tool
	bodies, err := combine(in, l.tool, l.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// endsWith pairs the definition's end conditions with the first/last section normals (a sketch
// plane, an apex tangent plane, or a source-face normal) the skinner needs to aim the takeoff, plus
// the adjacent face surfaces for a face-continuity (Tangent/Smooth) end so the skinner can read real
// tangent/curvature derivatives.
func (l *LoftFeature) endsWith(normals []math.UnitVector3, surfs []geom.Surface) loftEnds {
	first, last := l.def.First, l.def.Last
	if l.def.LiveEnds != nil {
		first, last = l.def.LiveEnds()
	}
	e := loftEnds{first: first, last: last, firstN: normals[0], lastN: normals[len(normals)-1]}
	if first.Condition.IsFaceContinuity() {
		e.firstSurf = surfs[0]
	}
	if last.Condition.IsFaceContinuity() {
		e.lastSurf = surfs[len(surfs)-1]
	}
	return e
}

// resolveGuides evaluates the definition's rail + centerline providers into model-space polylines
// (dropping empty/degenerate ones), so a parameter driving a guide reshapes the loft each recompute.
func (l *LoftFeature) resolveGuides() loftGuides {
	var g loftGuides
	for _, r := range l.def.Rails {
		if r == nil {
			continue
		}
		if pts := r(); len(pts) >= 2 {
			g.rails = append(g.rails, pts)
		}
	}
	if l.def.Centerline != nil {
		if pts := l.def.Centerline(); len(pts) >= 2 {
			g.centerline = pts
		}
	}
	g.areaGraph = l.def.AreaGraph
	for _, m := range l.def.MapCurves {
		if m == nil {
			continue
		}
		if pts := m(); len(pts) >= 2 {
			g.mapCurves = append(g.mapCurves, pts)
		}
	}
	return g
}

// skinTool builds the lofted solid for the resolved loops: a plain skin (no holes), a
// directly-meshed tube (one hole — a pipe), or a multi-hole solid cut from the skin (rare). Guides
// (rails + centerline) shape the OUTER surface only. The one-hole tube is meshed directly rather
// than via a bore Cut because a bore whose caps are coplanar with the body's caps leaves it open.
func (l *LoftFeature) skinTool(outers [][]math.Point3, inners [][][]math.Point3, ends loftEnds, guides loftGuides, rec *diag.Recorder) (*topo.Body, error) {
	feat := featOr(l.featName, "loft")
	// The Surface operation (kSurfaceOperation, #1858) skins an OPEN sheet — no end caps: a plain
	// skin for a solid section, an open pipe surface for a single-bore section. combine() adds it as
	// a surface body. A multi-bore surface loft (the cut-based hollow path) is a follow-up.
	surface := l.def.Operation == ops.Surface
	switch numHoles(inners) {
	case 0:
		if surface {
			return skinShell(outers, l.def.Closed, feat, ends, guides)
		}
		return skinLoops(outers, l.def.Closed, feat, ends, guides)
	case 1:
		if surface {
			return tubeShellLoops(outers, holeRing(inners, 0), l.def.Closed, feat, ends, guides)
		}
		return tubeLoops(outers, holeRing(inners, 0), l.def.Closed, feat, ends, guides)
	default:
		if surface {
			return nil, fmt.Errorf("loft: the surface operation supports at most one interior loop per section, got %d", numHoles(inners))
		}
		return hollowByCut(outers, inners, l.def.Closed, feat, ends, guides, rec)
	}
}

// hollowByCut skins the outer body and cuts each bore out of it — the fallback for lofts with
// more than one hole, where a multiply-connected end cap can't be a simple annular strip. Each
// bore is extended past the body's end caps (extendEnds) so the through-cut is not coplanar. Guides
// shape the outer skin only (the bores skin unguided). rec collects the bore cuts' boolean-fallback
// diagnostics (#1601; nil discards).
func hollowByCut(outers [][]math.Point3, inners [][][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides, rec *diag.Recorder) (*topo.Body, error) {
	tool, err := skinLoops(outers, closed, feat, ends, guides)
	if err != nil {
		return nil, err
	}
	eps := 0.0
	if !closed {
		eps = loftOvershoot(outers)
	}
	for h := 0; h < numHoles(inners); h++ {
		ring := extendEnds(holeRing(inners, h), eps)
		hole, herr := skinLoops(ring, closed, feat+"-hole", ends, loftGuides{})
		if herr != nil {
			return nil, herr
		}
		if tool, err = ops.BooleanWithDiagnostics(ops.Cut, tool, hole, rec); err != nil {
			return nil, err
		}
	}
	return tool, nil
}

// holeRing collects hole h's loop from every section into one section sequence.
func holeRing(inners [][][]math.Point3, h int) [][]math.Point3 {
	ring := make([][]math.Point3, len(inners))
	for i := range inners {
		ring[i] = inners[i][h]
	}
	return ring
}

// resolveSections resolves each section into its outer loop, inner (hole) loops, and plane/face
// normal in model space. A section is a sketch profile, a point (apex; single-point loop, no
// holes, valid only at an end), or an existing body face (resolved against bodies — its boundary
// is the loop, its surface gives the normal for Tangent/Smooth). At least one section must be a
// real profile/face, and all sections must share their inner-loop count.
func (l *LoftFeature) resolveSections(bodies []*topo.Body) (outers [][]math.Point3, inners [][][]math.Point3, normals []math.UnitVector3, surfs []geom.Surface, err error) {
	if len(l.def.Sections) < 2 {
		return nil, nil, nil, nil, fmt.Errorf("loft: %d sections, need at least 2", len(l.def.Sections))
	}
	if err := l.validatePointSections(); err != nil {
		return nil, nil, nil, nil, err
	}
	for i, s := range l.def.Sections {
		outer, holes, n, surf, e := resolveSection(s, bodies)
		if e != nil {
			return nil, nil, nil, nil, fmt.Errorf("loft section %d: %w", i, e)
		}
		outers, inners, normals, surfs = append(outers, outer), append(inners, holes), append(normals, n), append(surfs, surf)
	}
	for i, h := range inners {
		if len(h) != len(inners[0]) {
			return nil, nil, nil, nil, fmt.Errorf("loft: section %d has %d holes, want %d (hole counts must match across sections; a point section cannot pair with a hollow one)", i, len(h), len(inners[0]))
		}
	}
	return outers, inners, normals, surfs, nil
}

// resolveSection resolves one section's outer loop, inner (hole) loops, normal, and — for a face
// section — the face's surface (nil otherwise; the skinner reads it for real face continuity).
func resolveSection(s LoftSection, bodies []*topo.Body) ([]math.Point3, [][]math.Point3, math.UnitVector3, geom.Surface, error) {
	switch {
	case s.IsPoint():
		return []math.Point3{*s.Point}, nil, sectionNormal(s), nil, nil
	case s.IsFace():
		f, ok := findFace(bodies, s.FaceKey)
		if !ok {
			return nil, nil, math.UnitVector3{}, nil, fmt.Errorf("face reference is lost (no running body has it)")
		}
		outer, holes := faceLoopsModel(f)
		if len(outer) < 3 {
			return nil, nil, math.UnitVector3{}, nil, fmt.Errorf("face has a degenerate boundary (%d points)", len(outer))
		}
		return outer, holes, faceNormal(f, outer), f.Geometry(), nil
	default:
		prof, e := resolveSingleProfile(s.Sketch, s.ProfileIndex, "loft")
		if e != nil {
			return nil, nil, math.UnitVector3{}, nil, e
		}
		outer := loopToModel(prof.OuterLoop(), s.Sketch.Plane())
		var holes [][]math.Point3
		for _, il := range prof.InnerLoops() {
			holes = append(holes, loopToModel(il, s.Sketch.Plane()))
		}
		return outer, holes, s.Sketch.Plane().Normal(), nil, nil
	}
}

// sectionNormal is a sketch/point section's plane normal — for a point (apex) section the tangent
// plane a TangentToPlane condition domes against. Falls back to +Z for a bare 3D point.
func sectionNormal(s LoftSection) math.UnitVector3 {
	if s.Sketch != nil {
		return s.Sketch.Plane().Normal()
	}
	return math.V3(0, 0, 1).AsUnit()
}

// findFace resolves a face reference key against the running bodies (persistent naming).
func findFace(bodies []*topo.Body, key []byte) (*topo.Face, bool) {
	for _, b := range bodies {
		if f, ok := FindOrRecoverFace(b, key); ok {
			return f, true
		}
	}
	return nil, false
}

// faceLoopsModel returns a face's outer boundary loop and its inner (hole) loops as ordered
// model-space polygons (the "from" vertex of each oriented edge use).
func faceLoopsModel(f *topo.Face) (outer []math.Point3, inners [][]math.Point3) {
	for _, l := range f.Loops() {
		poly := loopUseStarts(l)
		if l.IsOuter() {
			outer = poly
		} else if len(poly) >= 3 {
			inners = append(inners, poly)
		}
	}
	return outer, inners
}

// loopUseStarts is the loop's boundary as an ordered point ring, each oriented edge sampled in
// traversal order. A straight edge contributes just its start point (the polygon corner); a curved
// edge (a circle/arc — e.g. the rim of an analytic cylinder cap) is sampled into many points so the
// ring is a real polygon, not a single vertex. The closing point is dropped (it equals the next
// edge's start).
func loopUseStarts(l *topo.Loop) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		c := u.Edge().Geometry()
		lo, hi := c.Domain()
		n := edgeRingSamples(c)
		for i := 0; i < n; i++ { // [0,n): exclude the endpoint shared with the next edge's start
			f := float64(i) / float64(n)
			t := lo + (hi-lo)*f
			if u.Reversed() {
				t = hi - (hi-lo)*f
			}
			pts = append(pts, c.PointAt(t))
		}
	}
	return pts
}

// edgeRingSamples is how many points to sample an edge into for a loop ring: 1 for a straight
// segment (the start corner), more for a curved edge so it reads as a polygon.
func edgeRingSamples(c geom.Curve3) int {
	if _, ok := c.(geom.LineSegment); ok {
		return 1
	}
	return 48
}

// faceNormal is the source face's surface normal used to aim a Tangent/Smooth takeoff: exact for
// a planar face (its plane normal), otherwise the boundary's best-fit (Newell) normal — so face
// continuity is exact for planar source faces and a sensible approximation for curved ones. Sign
// is irrelevant (the takeoff is re-oriented outward by the skinner).
func faceNormal(f *topo.Face, outer []math.Point3) math.UnitVector3 {
	if pl, ok := f.Geometry().(geom.Plane); ok {
		return pl.Normal().AsUnit()
	}
	return boundaryNormal(outer)
}

// boundaryNormal is the Newell normal of a (near-)planar polygon (+Z when degenerate).
func boundaryNormal(poly []math.Point3) math.UnitVector3 {
	var n math.Vector3
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		n = n.Add(math.V3((a.Y-b.Y)*(a.Z+b.Z), (a.Z-b.Z)*(a.X+b.X), (a.X-b.X)*(a.Y+b.Y)))
	}
	if n.Length() < 1e-12 {
		return math.V3(0, 0, 1).AsUnit()
	}
	return n.AsUnit()
}

// validatePointSections enforces that point (apex) sections only sit at the ends and that the
// loft still has at least one real profile to skin from (a loft of only points is a line).
func (l *LoftFeature) validatePointSections() error {
	secs := l.def.Sections
	profiles := 0
	for i, s := range secs {
		if !s.IsPoint() {
			profiles++
			continue
		}
		if i != 0 && i != len(secs)-1 {
			return fmt.Errorf("loft: section %d is a point; point sections are only allowed first or last", i)
		}
	}
	if profiles == 0 {
		return fmt.Errorf("loft: every section is a point; need at least one profile section")
	}
	return nil
}

// numHoles returns the per-section inner-loop count (all sections share it).
func numHoles(inners [][][]math.Point3) int {
	if len(inners) == 0 {
		return 0
	}
	return len(inners[0])
}

// loopToModel maps a sketch loop's polygon into model space on the sketch plane.
func loopToModel(loop sketch.Loop, plane sketch.Plane) []math.Point3 {
	poly := loop.Polygon()
	out := make([]math.Point3, len(poly))
	for i, p := range poly {
		out[i] = plane.ToModel(p)
	}
	return out
}

// skinLoops resamples a set of section loops to a common point count, corresponds them
// (minimize twist), blends them with a Catmull-Rom spline (a loft is not a straight blend),
// and meshes the swept solid. See loft_skin.go.
func skinLoops(loops [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	return sweptSolid(skinnedSections(loops, maxLoopCount(loops), closed, ends, guides), closed, feat)
}

// tubeLoops skins corresponding outer and inner section loops to a common point count, then
// meshes them directly into a hollow tube. Outer and inner share that point count so their
// rings pair up across the annular end caps; both run through the same correspondence + spline
// blend, so the pipe wall reads as smooth as a solid loft. Guides shape the outer wall only.
func tubeLoops(outers, inners [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeSolid(skinnedSections(outers, n, closed, ends, guides), skinnedSections(inners, n, closed, ends, loftGuides{}), closed, feat)
}

// skinShell is skinLoops' open counterpart for the surface operation: the skinned sections meshed
// as an OPEN sheet (no end caps) via sweptShell (#1858).
func skinShell(loops [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	return sweptShell(skinnedSections(loops, maxLoopCount(loops), closed, ends, guides), closed, feat)
}

// tubeShellLoops is tubeLoops' open counterpart: the nested outer/inner skinned sections meshed as
// an open pipe surface (no annular end caps) via tubeShell (#1858).
func tubeShellLoops(outers, inners [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeShell(skinnedSections(outers, n, closed, ends, guides), skinnedSections(inners, n, closed, ends, loftGuides{}), closed, feat)
}

// skinnedSections resamples loops to n points, corresponds them (minimize twist), blends them with
// a tangent-driven Hermite spline (Catmull-Rom interior, end conditions at the ends), bends the
// spine to any centerline, then pulls the result toward any guide rails — the densified section
// sequence ready to mesh. See loft_skin.go.
func skinnedSections(loops [][]math.Point3, n int, closed bool, ends loftEnds, guides loftGuides) [][]math.Point3 {
	resampled := make([][]math.Point3, len(loops))
	for i, lp := range loops {
		resampled[i] = resampleLoop(lp, n)
	}
	aligned := mapAlign(resampled, guides.mapCurves)
	// A twisting loft's skin quads are a full section-edge wide, so they warp steeply and facet
	// even when finely sampled along the length. Subdivide each section edge (corner-preserving)
	// proportional to the twist, so the skin is narrow enough across its width to read smooth.
	// The wrap's correspondence is offset by the monodromy (a 180° half-twist returns shifted by
	// half the points); the around-subdivision and the spline blend both measure the wrap twist
	// against the start REINDEXED by that shift, else a Möbius/twisted closure reads its seam as a
	// ~180° twist and over-subdivides every section (the seam notch's cousin — a 12× mesh blow-up).
	shift := closureShift(aligned, closed)
	aligned = densifyAround(aligned, aroundSubdivisions(aligned, closed, shift))
	secs := splineSections(aligned, closed, ends, closureShift(aligned, closed))
	secs = areaGraphScale(secs, guides.areaGraph)
	secs = centerlineGuide(secs, guides.centerline)
	return railGuide(secs, guides.rails)
}

// maxLoopCount returns the largest point count across every loop in the given sets, the common
// resample resolution so the densest section is not coarsened.
func maxLoopCount(loopSets ...[][]math.Point3) int {
	n := 0
	for _, set := range loopSets {
		for _, lp := range set {
			if len(lp) > n {
				n = len(lp)
			}
		}
	}
	return n
}

// LoftFeatures adds lofts into the engine.
type LoftFeatures struct{ engine *PartFeatures }

// NewLoftFeatures binds the collection to an engine.
func NewLoftFeatures(engine *PartFeatures) *LoftFeatures { return &LoftFeatures{engine} }

// Add adds a loft blending through the sections (optionally closed) under op, with Free end
// conditions (a two-section loft is ruled).
func (c *LoftFeatures) Add(sections []LoftSection, closed bool, op ops.PartFeatureOperation) *PartFeature {
	return c.addConditioned(sections, closed, op, LoftEnd{}, LoftEnd{})
}

// addConditioned adds a loft with explicit start/end conditions, so the surface can curve away
// from a flat ruled blend (e.g. an Angle takeoff on a two-section loft). The conditions are
// ignored when closed (a closed loft has no end sections).
func (c *LoftFeatures) addConditioned(sections []LoftSection, closed bool, op ops.PartFeatureOperation, first, last LoftEnd) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, First: first, Last: last})
}

// LoftGuideSet bundles every optional loft guide so the general constructors take one argument
// rather than a growing parameter list: rails, a centerline, an area graph, and explicit map
// curves (all the guides beyond the end conditions).
type LoftGuideSet struct {
	Rails      []func() []math.Point3
	Centerline func() []math.Point3
	AreaGraph  []types.LoftAreaStop
	MapCurves  []func() []math.Point3
}

// AddConditionedLiveGuided is AddConditionedLive plus the full live guide set (the parametric entry
// used by the op handler; the end angles and guides re-derive on each recompute).
func (c *LoftFeatures) AddConditionedLiveGuided(sections []LoftSection, closed bool, op ops.PartFeatureOperation, liveEnds func() (first, last LoftEnd), g LoftGuideSet) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, LiveEnds: liveEnds, Rails: g.Rails, Centerline: g.Centerline, AreaGraph: g.AreaGraph, MapCurves: g.MapCurves})
}

// AddGuided is AddConditioned plus the full guide set (static ends) — the general builder the tool
// and tests use; AddRailed/AddCenterlined are thin wrappers for the single-guide cases.
func (c *LoftFeatures) AddGuided(sections []LoftSection, closed bool, op ops.PartFeatureOperation, first, last LoftEnd, g LoftGuideSet) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, First: first, Last: last, Rails: g.Rails, Centerline: g.Centerline, AreaGraph: g.AreaGraph, MapCurves: g.MapCurves})
}

func (c *LoftFeatures) add(def *LoftDefinition) *PartFeature {
	lf := &LoftFeature{def: def}
	pf := c.engine.Add(lf)
	pf.SetName(c.engine.UniqueName("Loft"))
	lf.featName = pf.name
	return pf
}

// centroidOf returns the average of a point set.
func centroidOf(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}

// resampleLoop returns n points around the closed polygon that PRESERVE its original vertices
// (corners) and fill the remaining budget with collinear points along the edges (proportional to
// edge length), so an elongated polygon keeps its exact shape and area. A plain arc-length
// resample lands samples mid-edge and cuts the corners off any non-square polygon: a 16×2 mm
// rectangle resampled to 4 points became an 18 mm² quad (0.5625× its 32 mm² area), which then
// under-filled every swept solid's volume by the same factor (Oblikovati loft volume deficit,
// found 2026-06-15). Because interior points are collinear, the shape (and area) is unchanged;
// because every loop is resampled to the same n, sections of differing vertex counts still blend
// point-for-point. maxLoopCount guarantees n >= len(poly), so the densest loop (n == m) is
// returned verbatim. A degenerate loop (a point section) expands to n copies of its point — an
// apex the mesher welds to one vertex, so a loft to a point skins a cone/dome.
func resampleLoop(poly []math.Point3, n int) []math.Point3 {
	m := len(poly)
	segLen, total := loopSegmentLengths(poly)
	if m == 0 || total == 0 { // a point/apex section: n copies of its point
		out := make([]math.Point3, n)
		for k := range out {
			if m > 0 {
				out[k] = poly[0]
			}
		}
		return out
	}
	if n <= m { // the densest loop (n == m): keep its corners exactly (n < m can't occur, see above)
		return append([]math.Point3(nil), poly...)
	}
	interior := edgeInteriorCounts(segLen, total, n-m)
	out := make([]math.Point3, 0, n)
	for i := 0; i < m; i++ {
		a, b := poly[i], poly[(i+1)%m]
		out = append(out, a) // the corner — always preserved
		for t := 1; t <= interior[i]; t++ {
			f := math.Scalar(float64(t) / float64(interior[i]+1))
			out = append(out, a.TranslateBy(a.VectorTo(b).Scale(f)))
		}
	}
	return out
}

// edgeInteriorCounts apportions `extra` interior points across edges of the given lengths,
// proportional to length (largest-remainder rounding so the counts sum to exactly extra) — longer
// edges get more in-between samples. Used by resampleLoop to upsample a loop without moving its
// corners.
func edgeInteriorCounts(segLen []float64, total float64, extra int) []int {
	counts := make([]int, len(segLen))
	if extra <= 0 || total <= 0 {
		return counts
	}
	rema := make([]float64, len(segLen))
	assigned := 0
	for i, l := range segLen {
		ideal := float64(extra) * l / total
		counts[i] = int(ideal) // floor (ideal >= 0)
		rema[i] = ideal - float64(counts[i])
		assigned += counts[i]
	}
	for ; assigned < extra; assigned++ { // hand the leftover to the biggest remainders
		best := 0
		for i := 1; i < len(rema); i++ {
			if rema[i] > rema[best] {
				best = i
			}
		}
		counts[best]++
		rema[best] = -1 // don't pick the same edge twice
	}
	return counts
}

// loopSegmentLengths returns each edge length of the closed polygon and their total.
func loopSegmentLengths(poly []math.Point3) ([]float64, float64) {
	m := len(poly)
	segLen := make([]float64, m)
	total := 0.0
	for i := 0; i < m; i++ {
		segLen[i] = poly[i].DistanceTo(poly[(i+1)%m])
		total += segLen[i]
	}
	return segLen, total
}
