// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	Path         *sketch.Path3D
	Twist        func() float64 // total twist (radians) distributed along the path
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
	if s.def.Path == nil || s.def.Path.Count() < 2 {
		return Output{}, errors.New("sweep: path needs at least two points")
	}
	sections := sweepSections(prof, s.def.Sketch.Plane(), s.def.Path.Points(), callOrZero(s.def.Twist))
	s.tool, err = sweptSolid(sections, s.def.Path.IsClosed(), featOr(s.featName, "sweep"))
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

// Recompute resolves each section profile, resamples them to a common point count, and
// blends them into a faceted solid.
func (l *LoftFeature) Recompute(in Input) (Output, error) {
	sections, err := l.resolveSections()
	if err != nil {
		return Output{}, err
	}
	l.tool, err = sweptSolid(sections, l.def.Closed, featOr(l.featName, "loft"))
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, l.tool, l.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// resolveSections maps each loft section's profile into model space and resamples them
// all to the largest section's point count so the blend connects matching points.
func (l *LoftFeature) resolveSections() ([][]math.Point3, error) {
	if len(l.def.Sections) < 2 {
		return nil, fmt.Errorf("loft: %d sections, need at least 2", len(l.def.Sections))
	}
	raw := make([][]math.Point3, len(l.def.Sections))
	n := 0
	for i, s := range l.def.Sections {
		prof, err := resolveSingleProfile(s.Sketch, s.ProfileIndex, "loft")
		if err != nil {
			return nil, err
		}
		raw[i] = modelPolygon(prof, s.Sketch.Plane())
		if len(raw[i]) > n {
			n = len(raw[i])
		}
	}
	for i := range raw {
		raw[i] = resampleLoop(raw[i], n)
	}
	return raw, nil
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
