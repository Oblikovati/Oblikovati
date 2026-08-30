// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sweep feature (M48 #2240 split of sweep_loft.go). Sweeps a sketch profile along a 3D path: the
// definition, the feature wrapper and Recompute, the rigid/normal-to-path orientation, the swept-section
// tool builder and the SweepFeatures adder collection. The shared profile/path binding helpers live in
// sweep_loft.go.

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

// buildSweepTool builds the swept tool body. A rigid sweep keeps ANALYTIC faces where it can: a
// straight run reuses the extrude analytic prism (a straight NormalToPath sweep is exactly an extrude
// along the path tangent), and a full circular path with a circle profile is a torus (#2164 follow-up)
// — so a projected face sees real arcs, not chords. Every other sweep (a bent/partial path,
// taper/twist/scaling/rail/guide, a surface sweep, or a profile with holes) uses the faceted skin.
func (s *SweepFeature) buildSweepTool(prof *sketch.Profile, path *sketch.Path3D, cfg sweepConfig) (*topo.Body, error) {
	feat := featOr(s.featName, "sweep")
	if s.def.Operation != ops.Surface && s.sweepIsRigid() {
		if body := analyticRigidSweep(prof, s.def.Sketch.Plane(), path, feat); body != nil {
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
