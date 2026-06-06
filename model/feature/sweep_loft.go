// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati/api/types"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// LoftCondition is the boundary tangency control at a loft's end section; the canonical
// definition lives in the Apache-2.0 api/types (see ADR-0018).
type LoftCondition = types.LoftCondition

// Loft end conditions (aliases of the canonical api/types values).
const (
	LoftFree           = types.LoftFree
	LoftAngle          = types.LoftAngle
	LoftDirection      = types.LoftDirection
	LoftTangent        = types.LoftTangent
	LoftSmooth         = types.LoftSmooth
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
// needs to build the angled takeoff tangents.
type loftEnds struct {
	first, last   LoftEnd
	firstN, lastN math.UnitVector3
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

// Recompute resolves the profile, places it along the path tangents into a faceted
// solid, and applies the operation against the running bodies.
func (s *SweepFeature) Recompute(in Input) (Output, error) {
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
	sections := sweepSections(prof, s.def.Sketch.Plane(), path.Points(), callOrZero(s.def.Twist))
	s.tool, err = sweptSolid(sections, path.IsClosed(), featOr(s.featName, "sweep"))
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, s.tool, s.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// sweepSections places the profile (recentered onto each path point) rotated so its
// normal follows the path tangent, with the total twist spread along the path.
func sweepSections(prof *sketch.Profile, plane sketch.Plane, path []math.Point3, twist float64) [][]math.Point3 {
	base := modelPolygon(prof, plane)
	centroid := centroidOf(base)
	normal := plane.Normal()
	tangents := pathTangents(path)
	sections := make([][]math.Point3, len(path))
	for k, pt := range path {
		align := math.RotateBetween(normal, tangents[k])
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			v := align.TransformVector(centroid.VectorTo(p))
			if twist != 0 {
				frac := float64(k) / float64(len(path)-1)
				v = math.Rotation4(twist*frac, tangents[k], math.P3(0, 0, 0)).TransformVector(v)
			}
			sec[i] = pt.TranslateBy(v)
		}
		sections[k] = sec
	}
	return sections
}

// pathTangents returns a unit tangent at each path point: the forward segment at the
// start, the backward segment at the end, and the average of the two interior segments.
func pathTangents(path []math.Point3) []math.UnitVector3 {
	n := len(path)
	out := make([]math.UnitVector3, n)
	for k := range path {
		var v math.Vector3
		switch {
		case k == 0:
			v = path[0].VectorTo(path[1])
		case k == n-1:
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
	return c.AddLive(skt, profileIndex, func() *sketch.Path3D { return path }, twist, op)
}

// AddLive is Add with a live path provider, re-derived on every recompute, so a
// parameter that drives the path sketch reshapes the sweep (the static Add snapshots
// the path and does not track edits).
func (c *SweepFeatures) AddLive(skt *sketch.Sketch, profileIndex int, path func() *sketch.Path3D, twist func() float64, op ops.PartFeatureOperation) *PartFeature {
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
}

// LoftType reports the loft mode derived from the definition — the analogue of Inventor's
// LoftTypeEnum (a regular loft, or one guided by rails).
func (d *LoftDefinition) LoftType() types.LoftType {
	if len(d.Rails) > 0 {
		return types.LoftWithRails
	}
	return types.RegularLoft
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
	outers, inners, normals, err := l.resolveSections(in.Bodies)
	if err != nil {
		return Output{}, err
	}
	tool, err := l.skinTool(outers, inners, l.endsWith(normals), l.resolveRails())
	if err != nil {
		return Output{}, err
	}
	l.tool = tool
	bodies, err := combine(in.Bodies, l.tool, l.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// endsWith pairs the definition's end conditions with the first/last section normals (a sketch
// plane, an apex tangent plane, or a source-face normal) the skinner needs to aim the takeoff.
func (l *LoftFeature) endsWith(normals []math.UnitVector3) loftEnds {
	first, last := l.def.First, l.def.Last
	if l.def.LiveEnds != nil {
		first, last = l.def.LiveEnds()
	}
	return loftEnds{first: first, last: last, firstN: normals[0], lastN: normals[len(normals)-1]}
}

// resolveRails evaluates the definition's rail providers into model-space polylines (dropping
// empty/degenerate ones), so a parameter driving a rail reshapes the loft each recompute.
func (l *LoftFeature) resolveRails() [][]math.Point3 {
	var rails [][]math.Point3
	for _, r := range l.def.Rails {
		if r == nil {
			continue
		}
		if pts := r(); len(pts) >= 2 {
			rails = append(rails, pts)
		}
	}
	return rails
}

// skinTool builds the lofted solid for the resolved loops: a plain skin (no holes), a
// directly-meshed tube (one hole — a pipe), or a multi-hole solid cut from the skin (rare). Guide
// rails apply to the OUTER surface only. The one-hole tube is meshed directly rather than via a
// bore Cut because a bore whose caps are coplanar with the body's caps leaves the Difference open.
func (l *LoftFeature) skinTool(outers [][]math.Point3, inners [][][]math.Point3, ends loftEnds, rails [][]math.Point3) (*topo.Body, error) {
	feat := featOr(l.featName, "loft")
	switch numHoles(inners) {
	case 0:
		return skinLoops(outers, l.def.Closed, feat, ends, rails)
	case 1:
		return tubeLoops(outers, holeRing(inners, 0), l.def.Closed, feat, ends, rails)
	default:
		return hollowByCut(outers, inners, l.def.Closed, feat, ends, rails)
	}
}

// hollowByCut skins the outer body and cuts each bore out of it — the fallback for lofts with
// more than one hole, where a multiply-connected end cap can't be a simple annular strip. Each
// bore is extended past the body's end caps (extendEnds) so the through-cut is not coplanar. Rails
// guide the outer skin only (the bores skin unguided).
func hollowByCut(outers [][]math.Point3, inners [][][]math.Point3, closed bool, feat string, ends loftEnds, rails [][]math.Point3) (*topo.Body, error) {
	tool, err := skinLoops(outers, closed, feat, ends, rails)
	if err != nil {
		return nil, err
	}
	eps := 0.0
	if !closed {
		eps = loftOvershoot(outers)
	}
	for h := 0; h < numHoles(inners); h++ {
		ring := extendEnds(holeRing(inners, h), eps)
		hole, herr := skinLoops(ring, closed, feat+"-hole", ends, nil)
		if herr != nil {
			return nil, herr
		}
		if tool, err = ops.Boolean(ops.Cut, tool, hole); err != nil {
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
func (l *LoftFeature) resolveSections(bodies []*topo.Body) (outers [][]math.Point3, inners [][][]math.Point3, normals []math.UnitVector3, err error) {
	if len(l.def.Sections) < 2 {
		return nil, nil, nil, fmt.Errorf("loft: %d sections, need at least 2", len(l.def.Sections))
	}
	if err := l.validatePointSections(); err != nil {
		return nil, nil, nil, err
	}
	for i, s := range l.def.Sections {
		outer, holes, n, e := resolveSection(s, bodies)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("loft section %d: %w", i, e)
		}
		outers, inners, normals = append(outers, outer), append(inners, holes), append(normals, n)
	}
	for i, h := range inners {
		if len(h) != len(inners[0]) {
			return nil, nil, nil, fmt.Errorf("loft: section %d has %d holes, want %d (hole counts must match across sections; a point section cannot pair with a hollow one)", i, len(h), len(inners[0]))
		}
	}
	return outers, inners, normals, nil
}

// resolveSection resolves one section's outer loop, inner (hole) loops, and normal.
func resolveSection(s LoftSection, bodies []*topo.Body) ([]math.Point3, [][]math.Point3, math.UnitVector3, error) {
	switch {
	case s.IsPoint():
		return []math.Point3{*s.Point}, nil, sectionNormal(s), nil
	case s.IsFace():
		f, ok := findFace(bodies, s.FaceKey)
		if !ok {
			return nil, nil, math.UnitVector3{}, fmt.Errorf("face reference is lost (no running body has it)")
		}
		outer, holes := faceLoopsModel(f)
		if len(outer) < 3 {
			return nil, nil, math.UnitVector3{}, fmt.Errorf("face has a degenerate boundary (%d points)", len(outer))
		}
		return outer, holes, faceNormal(f, outer), nil
	default:
		prof, e := resolveSingleProfile(s.Sketch, s.ProfileIndex, "loft")
		if e != nil {
			return nil, nil, math.UnitVector3{}, e
		}
		outer := loopToModel(prof.OuterLoop(), s.Sketch.Plane())
		var holes [][]math.Point3
		for _, il := range prof.InnerLoops() {
			holes = append(holes, loopToModel(il, s.Sketch.Plane()))
		}
		return outer, holes, s.Sketch.Plane().Normal(), nil
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
		if f, ok := b.FindFaceByKey(key); ok {
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

// loopUseStarts is the ordered start points of a loop's oriented edge uses.
func loopUseStarts(l *topo.Loop) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		v := u.Edge().StartVertex()
		if u.Reversed() {
			v = u.Edge().EndVertex()
		}
		pts = append(pts, v.Point())
	}
	return pts
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
func skinLoops(loops [][]math.Point3, closed bool, feat string, ends loftEnds, rails [][]math.Point3) (*topo.Body, error) {
	return sweptSolid(skinnedSections(loops, maxLoopCount(loops), closed, ends, rails), closed, feat)
}

// tubeLoops skins corresponding outer and inner section loops to a common point count, then
// meshes them directly into a hollow tube. Outer and inner share that point count so their
// rings pair up across the annular end caps; both run through the same correspondence + spline
// blend, so the pipe wall reads as smooth as a solid loft. Rails guide the outer wall only.
func tubeLoops(outers, inners [][]math.Point3, closed bool, feat string, ends loftEnds, rails [][]math.Point3) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeSolid(skinnedSections(outers, n, closed, ends, rails), skinnedSections(inners, n, closed, ends, nil), closed, feat)
}

// skinnedSections resamples loops to n points, corresponds them (minimize twist), blends them
// with a tangent-driven Hermite spline (Catmull-Rom interior, end conditions at the ends), and
// pulls the result toward any guide rails — the densified section sequence ready to mesh. See
// loft_skin.go.
func skinnedSections(loops [][]math.Point3, n int, closed bool, ends loftEnds, rails [][]math.Point3) [][]math.Point3 {
	resampled := make([][]math.Point3, len(loops))
	for i, lp := range loops {
		resampled[i] = resampleLoop(lp, n)
	}
	return railGuide(splineSections(alignSections(resampled), closed, ends), rails)
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
	return c.AddConditioned(sections, closed, op, LoftEnd{}, LoftEnd{})
}

// AddConditioned adds a loft with explicit start/end conditions, so the surface can curve away
// from a flat ruled blend (e.g. an Angle takeoff on a two-section loft). The conditions are
// ignored when closed (a closed loft has no end sections).
func (c *LoftFeatures) AddConditioned(sections []LoftSection, closed bool, op ops.PartFeatureOperation, first, last LoftEnd) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, First: first, Last: last})
}

// AddConditionedLive is AddConditioned with a live end-condition provider, re-read on every
// recompute, so an end angle/impact driven by a parameter reshapes the loft.
func (c *LoftFeatures) AddConditionedLive(sections []LoftSection, closed bool, op ops.PartFeatureOperation, liveEnds func() (first, last LoftEnd)) *PartFeature {
	return c.AddConditionedLiveRailed(sections, closed, op, liveEnds, nil)
}

// AddConditionedLiveRailed is AddConditionedLive plus live guide rails (the parametric entry used
// by the op handler — both the end angles and the rails re-derive on each recompute).
func (c *LoftFeatures) AddConditionedLiveRailed(sections []LoftSection, closed bool, op ops.PartFeatureOperation, liveEnds func() (first, last LoftEnd), rails []func() []math.Point3) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, LiveEnds: liveEnds, Rails: rails})
}

// AddRailed is AddConditioned plus guide rails (the kLoftWithRails mode): each rail is a live
// provider of a model-space polyline that touches the end sections; the loft's outer surface is
// pulled to follow them.
func (c *LoftFeatures) AddRailed(sections []LoftSection, closed bool, op ops.PartFeatureOperation, first, last LoftEnd, rails []func() []math.Point3) *PartFeature {
	return c.add(&LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op, First: first, Last: last, Rails: rails})
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

// resampleLoop returns n points spaced evenly by arc length around the closed polygon, so
// sections of differing vertex counts blend point-for-point. A degenerate loop (a point section,
// or all-coincident points) expands to n copies of its point — an apex the mesher welds to one
// vertex, so a loft to a point skins a cone/dome.
func resampleLoop(poly []math.Point3, n int) []math.Point3 {
	segLen, total := loopSegmentLengths(poly)
	if total == 0 {
		out := make([]math.Point3, n)
		for k := range out {
			out[k] = poly[0]
		}
		return out
	}
	m := len(poly)
	out := make([]math.Point3, n)
	step := total / float64(n)
	seg, acc := 0, 0.0
	for k := 0; k < n; k++ {
		target := step * float64(k)
		for seg < m-1 && acc+segLen[seg] < target {
			acc += segLen[seg]
			seg++
		}
		f := 0.0
		if segLen[seg] > 0 {
			f = (target - acc) / segLen[seg]
		}
		a, b := poly[seg], poly[(seg+1)%m]
		out[k] = a.TranslateBy(a.VectorTo(b).Scale(f))
	}
	return out
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
