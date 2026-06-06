// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

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

// LoftSection identifies one cross-section of a loft: a closed profile on a sketch.
type LoftSection struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
}

// LoftDefinition is the recipe for a loft: a blend through ordered sections, optionally
// closed (the last section blends back to the first).
type LoftDefinition struct {
	Sections  []LoftSection
	Closed    bool
	Operation ops.PartFeatureOperation
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
	outers, inners, err := l.loopSets()
	if err != nil {
		return Output{}, err
	}
	tool, err := l.skinTool(outers, inners)
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

// skinTool builds the lofted solid for the resolved loops: a plain skin (no holes), a
// directly-meshed tube (one hole — a pipe), or a multi-hole solid cut from the skin (rare). The
// one-hole tube is meshed directly rather than via a bore Cut because a bore whose caps are
// coplanar with the body's caps leaves the Difference open. See [tubeSolid] and [hollowByCut].
func (l *LoftFeature) skinTool(outers [][]math.Point3, inners [][][]math.Point3) (*topo.Body, error) {
	feat := featOr(l.featName, "loft")
	switch numHoles(inners) {
	case 0:
		return skinLoops(outers, l.def.Closed, feat)
	case 1:
		return tubeLoops(outers, holeRing(inners, 0), l.def.Closed, feat)
	default:
		return hollowByCut(outers, inners, l.def.Closed, feat)
	}
}

// hollowByCut skins the outer body and cuts each bore out of it — the fallback for lofts with
// more than one hole, where a multiply-connected end cap can't be a simple annular strip. Each
// bore is extended past the body's end caps (extendEnds) so the through-cut is not coplanar.
func hollowByCut(outers [][]math.Point3, inners [][][]math.Point3, closed bool, feat string) (*topo.Body, error) {
	tool, err := skinLoops(outers, closed, feat)
	if err != nil {
		return nil, err
	}
	eps := 0.0
	if !closed {
		eps = loftOvershoot(outers)
	}
	for h := 0; h < numHoles(inners); h++ {
		ring := extendEnds(holeRing(inners, h), eps)
		hole, herr := skinLoops(ring, closed, feat+"-hole")
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

// loopSets resolves each section into its outer loop and inner (hole) loops in model space. All
// sections must have the same number of inner loops so holes correspond across the loft.
func (l *LoftFeature) loopSets() (outers [][]math.Point3, inners [][][]math.Point3, err error) {
	if len(l.def.Sections) < 2 {
		return nil, nil, fmt.Errorf("loft: %d sections, need at least 2", len(l.def.Sections))
	}
	for _, s := range l.def.Sections {
		prof, e := resolveSingleProfile(s.Sketch, s.ProfileIndex, "loft")
		if e != nil {
			return nil, nil, e
		}
		outers = append(outers, loopToModel(prof.OuterLoop(), s.Sketch.Plane()))
		var holes [][]math.Point3
		for _, il := range prof.InnerLoops() {
			holes = append(holes, loopToModel(il, s.Sketch.Plane()))
		}
		inners = append(inners, holes)
	}
	for i, h := range inners {
		if len(h) != len(inners[0]) {
			return nil, nil, fmt.Errorf("loft: section %d has %d holes, want %d (hole counts must match across sections)", i, len(h), len(inners[0]))
		}
	}
	return outers, inners, nil
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
func skinLoops(loops [][]math.Point3, closed bool, feat string) (*topo.Body, error) {
	return sweptSolid(skinnedSections(loops, maxLoopCount(loops), closed), closed, feat)
}

// tubeLoops skins corresponding outer and inner section loops to a common point count, then
// meshes them directly into a hollow tube. Outer and inner share that point count so their
// rings pair up across the annular end caps; both run through the same correspondence + spline
// blend, so the pipe wall reads as smooth as a solid loft.
func tubeLoops(outers, inners [][]math.Point3, closed bool, feat string) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeSolid(skinnedSections(outers, n, closed), skinnedSections(inners, n, closed), closed, feat)
}

// skinnedSections resamples loops to n points, corresponds them (minimize twist), and blends
// them with a Catmull-Rom spline — the densified section sequence ready to mesh. See loft_skin.go.
func skinnedSections(loops [][]math.Point3, n int, closed bool) [][]math.Point3 {
	resampled := make([][]math.Point3, len(loops))
	for i, lp := range loops {
		resampled[i] = resampleLoop(lp, n)
	}
	return splineSections(alignSections(resampled), closed)
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

// Add adds a loft blending through the sections (optionally closed) under op.
func (c *LoftFeatures) Add(sections []LoftSection, closed bool, op ops.PartFeatureOperation) *PartFeature {
	def := &LoftDefinition{Sections: append([]LoftSection(nil), sections...), Closed: closed, Operation: op}
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

// resampleLoop returns n points spaced evenly by arc length around the closed polygon,
// so sections of differing vertex counts blend point-for-point.
func resampleLoop(poly []math.Point3, n int) []math.Point3 {
	segLen, total := loopSegmentLengths(poly)
	if total == 0 {
		return poly
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
