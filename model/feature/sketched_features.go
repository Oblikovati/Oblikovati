// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// The remaining sketched features carry their full Definition (the triangle) and
// extent/operation surface. Revolve and coil generate real (faceted) solids of
// revolution via the shared swept-solid primitive; sweep/loft generation is below.
// Curved profile edges and exact analytic/NURBS surfaces are a later refinement —
// phase A approximates the swept surfaces as planar facets, a real watertight solid.

// revolveSegments is the angular facet count for a full revolution; partial angles
// use a proportional share (so a 90° revolve gets ~1/4 the facets).
const revolveSegments = 64

// RevolveDefinition is the recipe for a revolve: a profile spun about an axis.
type RevolveDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Angle        func() float64 // 0 ⇒ full revolution
	Operation    ops.PartFeatureOperation
}

// RevolveFeature spins a profile about an axis.
type RevolveFeature struct {
	def      *RevolveDefinition
	featName string
}

func (r *RevolveFeature) Definition() *RevolveDefinition { return r.def }
func (r *RevolveFeature) Kind() string                   { return "revolve" }

// Recompute resolves the profile, spins it about the axis into a faceted solid of
// revolution, and applies the operation against the running bodies.
func (r *RevolveFeature) Recompute(in Input) (Output, error) {
	prof, err := resolveSingleProfile(r.def.Sketch, r.def.ProfileIndex, "revolve")
	if err != nil {
		return Output{}, err
	}
	if r.def.Axis == nil {
		return Output{}, errors.New("revolve: no axis of revolution")
	}
	sections, closed := revolveSections(prof, r.def.Sketch.Plane(), r.def.Axis, callOrZero(r.def.Angle))
	tool, err := sweptSolid(sections, closed, featOr(r.featName, "revolve"))
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// revolveSections places the profile (in model space) at evenly-spaced angles about the
// axis. A zero or ≥2π angle is a full revolution (a closed loop, no caps).
func revolveSections(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle float64) ([][]math.Point3, bool) {
	base := modelPolygon(prof, plane)
	full := angle <= 0 || angle >= 2*stdmath.Pi-1e-9
	k, step := revolveSegments, 2*stdmath.Pi/float64(revolveSegments)
	if !full {
		segs := stdmath.Max(3, stdmath.Round(revolveSegments*angle/(2*stdmath.Pi)))
		k, step = int(segs)+1, angle/segs
	}
	sections := make([][]math.Point3, k)
	for s := 0; s < k; s++ {
		m := math.Rotation4(step*float64(s), axis.Direction(), axis.Origin())
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			sec[i] = m.TransformPoint(p)
		}
		sections[s] = sec
	}
	return sections, full
}

// modelPolygon returns a profile's outer-loop polygon mapped into model space.
func modelPolygon(prof *sketch.Profile, plane sketch.Plane) []math.Point3 {
	poly := prof.OuterLoop().Polygon()
	out := make([]math.Point3, len(poly))
	for i, p := range poly {
		out[i] = plane.ToModel(p)
	}
	return out
}

// resolveSingleProfile re-derives one closed region from a sketch, erroring (→ sick)
// when it is missing or open.
func resolveSingleProfile(skt *sketch.Sketch, index int, feat string) (*sketch.Profile, error) {
	all := skt.Profiles()
	if index < 0 || index >= all.Count() {
		return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", feat, index, all.Count())
	}
	p := all.Item(index)
	if !p.IsClosed() {
		return nil, fmt.Errorf("%s: profile is open (cannot form a solid)", feat)
	}
	return p, nil
}

// featOr returns name if set, else fallback — the lineage feature id for generated
// topology (a unique per-feature name keeps reference keys distinct).
func featOr(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
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

// CoilDefinition is the recipe for a coil (helical sweep).
type CoilDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Pitch        func() float64
	Revolutions  func() float64
	Taper        float64
	Operation    ops.PartFeatureOperation
}

// CoilFeature sweeps a profile along a helix.
type CoilFeature struct {
	def      *CoilDefinition
	featName string
}

func (c *CoilFeature) Definition() *CoilDefinition { return c.def }
func (c *CoilFeature) Kind() string                { return "coil" }

// Recompute resolves the profile and sweeps it along a helix about the axis (pitch per
// revolution × revolutions) into a faceted solid, then applies the operation.
func (c *CoilFeature) Recompute(in Input) (Output, error) {
	prof, err := resolveSingleProfile(c.def.Sketch, c.def.ProfileIndex, "coil")
	if err != nil {
		return Output{}, err
	}
	if c.def.Axis == nil {
		return Output{}, errors.New("coil: no axis")
	}
	revs := callOrZero(c.def.Revolutions)
	if revs <= 0 {
		return Output{}, fmt.Errorf("coil: revolutions must be > 0, got %g", revs)
	}
	sections := coilSections(prof, c.def.Sketch.Plane(), c.def.Axis, callOrZero(c.def.Pitch), revs)
	tool, err := sweptSolid(sections, false, featOr(c.featName, "coil"))
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, tool, c.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// coilSections places the profile along a helix: at each step it is rotated about the
// axis by the running angle and translated along the axis by pitch·(angle/2π).
func coilSections(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, pitch, revolutions float64) [][]math.Point3 {
	base := modelPolygon(prof, plane)
	axisVec := axis.Direction().AsVector()
	k := int(stdmath.Max(3, stdmath.Round(revolveSegments*revolutions)))
	total := 2 * stdmath.Pi * revolutions
	sections := make([][]math.Point3, k+1)
	for s := 0; s <= k; s++ {
		angle := total * float64(s) / float64(k)
		rise := pitch * angle / (2 * stdmath.Pi)
		rot := math.Rotation4(angle, axis.Direction(), axis.Origin())
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			sec[i] = rot.TransformPoint(p).TranslateBy(axisVec.Scale(rise))
		}
		sections[s] = sec
	}
	return sections
}

// CoilFeatures adds coils into the engine.
type CoilFeatures struct{ engine *PartFeatures }

// NewCoilFeatures binds the collection to an engine.
func NewCoilFeatures(engine *PartFeatures) *CoilFeatures { return &CoilFeatures{engine} }

// Add adds a coil of the profile about axis with the given pitch (per revolution),
// number of revolutions, taper, and boolean operation.
func (c *CoilFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, pitch, revolutions func() float64, taper float64, op ops.PartFeatureOperation) *PartFeature {
	def := &CoilDefinition{
		Sketch: skt, ProfileIndex: profileIndex, Axis: axis,
		Pitch: pitch, Revolutions: revolutions, Taper: taper, Operation: op,
	}
	cf := &CoilFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Coil"))
	cf.featName = pf.name
	return pf
}

// RibDefinition is the recipe for a rib: a thin support from an open profile.
type RibDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Thickness    func() float64
	Operation    ops.PartFeatureOperation
}

// RibFeature thickens an open profile into a support.
type RibFeature struct{ def *RibDefinition }

func (r *RibFeature) Definition() *RibDefinition { return r.def }
func (r *RibFeature) Kind() string               { return "rib" }
func (r *RibFeature) Recompute(Input) (Output, error) {
	return Output{}, build.NotYetImplemented("PBI-096-rib-generation")
}
