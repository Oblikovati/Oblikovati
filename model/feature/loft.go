// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft feature (M48 #2240 split of sweep_loft.go). Skins a surface through a series of sections (sketch
// loops, points or model faces): the loft types, the definition, the feature wrapper and Recompute, the
// end/guide conditions and the LoftFeatures adder collection. The section resolution and skin helpers
// live in loft_sections.go; the shared binding helpers in sweep_loft.go.

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
