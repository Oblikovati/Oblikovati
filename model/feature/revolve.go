// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Revolve feature (M48 #2239 split of sketched_features.go). Sweeps a sketch profile about an axis: the
// definition, the feature wrapper and Recompute, the axis resolution (work axis / sketch centreline) and
// the RevolveFeatures adder collection. The revolve tool geometry builders live in revolve_build.go; the
// shared sketch-profile binding helpers in sketched_features.go.

// revolveSegments is the angular facet count for a full revolution; partial angles
// use a proportional share (so a 90° revolve gets ~1/4 the facets).
const revolveSegments = 64

// RevolveDefinition is the recipe for a revolve: a profile spun about an axis. The axis is, in
// precedence: an explicit work axis; else a specific centerline (AxisCenterline on its sketch);
// else the profile sketch's single centerline (auto).
type RevolveDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	// ProfileSeed is an interior seed point (sketch 2-D, cm) selecting the region by
	// containment. When set it is resolved to a region index EVERY recompute so the selection
	// survives the sketch being re-solved between load and recompute (which reorders the DCEL
	// regions, stranding ProfileIndex on the wrong cell — #region-seed). nil ⇒ use ProfileIndex.
	ProfileSeed          []float64
	Axis                 *WorkAxis
	AxisCenterline       *sketch.Line   // a specific centerline to revolve about
	AxisCenterlineSketch *sketch.Sketch // the centerline's sketch (for its plane)
	Angle                func() float64 // 0 ⇒ full revolution
	// Angle2 is the second-direction sweep (radians, opposite sense), the
	// reference two-directional revolve (M08 PBI-093, #313). nil/0 ⇒ one-way.
	Angle2 func() float64
	// Direction is the side Angle sweeps to: default forward, flipped backward, or symmetric
	// (half each way) — Inventor's revolve Direction (#2019). Ignored once Angle2 is set, which
	// is the asymmetric mode and names both sides itself. See revolveSpan.
	Direction ExtentDirection
	// Extent is how the revolve terminates (#1860). The zero value, DistanceExtent, is the ANGLE
	// extent — a revolve's "distance" is its Angle (Inventor's kAngleExtent) — and the geometric
	// members terminate on ToPlane/FromPlane or on the next material instead. See revolveExtentSpan.
	Extent ExtentType
	// ToPlane is the to-face stop (and the "to" of from-to); FromPlane is the "from". Both must
	// contain the revolve axis — see radialHalfPlaneDir. Unused by the angle and to-next extents.
	ToPlane   *WorkPlane
	FromPlane *WorkPlane
	Operation ops.PartFeatureOperation
}

// RevolveFeature spins a profile about an axis.
type RevolveFeature struct {
	def      *RevolveDefinition
	featName string
	tool     *topo.Body // last solid of revolution, exposed so a pattern can replicate it
}

func (r *RevolveFeature) Definition() *RevolveDefinition { return r.def }
func (r *RevolveFeature) Kind() string                   { return "revolve" }

// Operation and ToolBody let a pattern/mirror replicate this feature with the right boolean
// (see [ToolFeature]).
func (r *RevolveFeature) Operation() ops.PartFeatureOperation { return r.def.Operation }
func (r *RevolveFeature) ToolBody() *topo.Body                { return r.tool }

// Recompute resolves the profile, spins it about the axis into a faceted solid of
// revolution, and applies the operation against the running bodies.
func (r *RevolveFeature) Recompute(in Input) (Output, error) {
	// Resolve the seed against the CURRENT regions each recompute (region ordering is a DCEL
	// artifact that shifts when the sketch re-solves — #region-seed); fall back to the index.
	profileIndex := r.def.ProfileIndex
	if len(r.def.ProfileSeed) > 0 {
		profileIndex = resolveSeed(r.def.Sketch, r.def.ProfileSeed, r.def.ProfileIndex)
	}
	prof, err := resolveSingleProfile(r.def.Sketch, profileIndex, "revolve")
	if err != nil {
		return Output{}, err
	}
	axis, err := r.revolveAxis()
	if err != nil {
		return Output{}, err
	}
	r.tool, err = r.buildRevolveTool(prof, axis, in.Bodies)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, r.tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildRevolveTool spins the profile into the tool body, resolving the swept span from the
// definition. For the Surface operation (kSurfaceOperation, #1858) it revolves the profile
// boundary into an OPEN surface of revolution (a sheet); otherwise it builds the solid of
// revolution via the shared [buildRevolveSolid] (the assembly-context revolve, #735, reuses it).
func (r *RevolveFeature) buildRevolveTool(prof *sketch.Profile, axis *WorkAxis,
	bodies []*topo.Body) (*topo.Body, error) {
	angle, start, err := r.resolveRevolveSpan(prof, axis, bodies)
	if err != nil {
		return nil, err
	}
	plane, feat := r.def.Sketch.Plane(), featOr(r.featName, "revolve")
	if r.def.Operation == ops.Surface {
		return buildRevolveSheet(prof, plane, axis, angle, start, feat)
	}
	return buildRevolveSolid(prof, plane, axis, angle, start, feat)
}

// revolveAxis resolves the axis of revolution: an explicit work axis if set, otherwise the
// sketch's single centerline (Inventor's "revolve about the sketch centerline"). No axis and no
// (or several) centerlines → Sick.
func (r *RevolveFeature) revolveAxis() (*WorkAxis, error) {
	if r.def.Axis != nil {
		return r.def.Axis, nil
	}
	if r.def.AxisCenterline != nil {
		return centerlineAxis(r.def.AxisCenterline, r.def.AxisCenterlineSketch)
	}
	return sketchCenterlineAxis(r.def.Sketch, "revolve")
}

// sketchCenterlineAxis resolves a sketch's single centerline into a revolve axis (Inventor's
// "revolve about the sketch centerline"). No centerline, or more than one, is ambiguous and
// reported as feature health — shared by the part and assembly-context revolves (#735).
func sketchCenterlineAxis(sk *sketch.Sketch, feat string) (*WorkAxis, error) {
	cls := sk.Centerlines()
	if len(cls) == 0 {
		return nil, fmt.Errorf("%s: no axis of revolution (set an axis or add a sketch centerline)", feat)
	}
	if len(cls) > 1 {
		return nil, fmt.Errorf("%s: ambiguous axis — the sketch has multiple centerlines; pick one", feat)
	}
	return centerlineAxis(cls[0], sk)
}

// centerlineAxis turns a centerline line on its sketch into a transient axis of revolution.
func centerlineAxis(line *sketch.Line, sk *sketch.Sketch) (*WorkAxis, error) {
	o, d := line.Axis3D(sk.Plane())
	dir, err := math.UnitVector3FromVector(d)
	if err != nil {
		return nil, errors.New("revolve: the centerline is degenerate")
	}
	return &WorkAxis{origin: o, dir: dir}, nil
}

// RevolveFeatures adds revolves into the engine.
type RevolveFeatures struct{ engine *PartFeatures }

// NewRevolveFeatures binds the collection to an engine.
func NewRevolveFeatures(engine *PartFeatures) *RevolveFeatures { return &RevolveFeatures{engine} }

// Add adds a revolve of the profile about axis through angle (nil ⇒ full).
func (c *RevolveFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{Sketch: skt, ProfileIndex: profileIndex, Axis: axis, Angle: angle, Operation: op}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}

// AddTwoDirectional adds a revolve sweeping angle forward and angle2 in the
// opposite sense about axis — the reference two-directional revolve (#313).
func (c *RevolveFeatures) AddTwoDirectional(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, angle, angle2 func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{Sketch: skt, ProfileIndex: profileIndex, Axis: axis, Angle: angle, Angle2: angle2, Operation: op}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}

// AddAboutCenterline adds a revolve that spins the profile about the sketch's own centerline
// (the common case: profile + centerline in one sketch). The sketch must hold exactly one.
func (c *RevolveFeatures) AddAboutCenterline(skt *sketch.Sketch, profileIndex int, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	return c.Add(skt, profileIndex, nil, angle, op)
}

// AddAboutCenterlineLine adds a revolve about a SPECIFIC centerline (on axisSketch) — used when
// the chosen axis isn't simply the profile sketch's lone centerline (several centerlines, or one
// in another sketch).
func (c *RevolveFeatures) AddAboutCenterlineLine(profileSketch *sketch.Sketch, profileIndex int, axisSketch *sketch.Sketch, axisLine *sketch.Line, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{
		Sketch: profileSketch, ProfileIndex: profileIndex,
		AxisCenterline: axisLine, AxisCenterlineSketch: axisSketch, Angle: angle, Operation: op,
	}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}
