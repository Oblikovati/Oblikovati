// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	"slices"

	"oblikovati.org/kernel/ops/surface"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Stitch/knit and sculpt (M10-F01, PBI-110) combine the surface bodies accumulated
// in the running state into one quilt. Stitch welds coincident edges (closed quilt →
// solid unless held as a surface); sculpt requires the surfaces to enclose a volume
// and yields the filled solid. Both consume the running surface bodies and replace
// them with the single combined body.

// StitchDefinition is the recipe for a stitch/knit: the coincidence tolerance and
// whether to keep a closed quilt as a surface rather than promoting it to a solid.
type StitchDefinition struct {
	Tolerance         float64
	MaintainAsSurface bool
}

// StitchFeature welds the running surface bodies into one quilt (PBI-110).
type StitchFeature struct {
	def      *StitchDefinition
	featName string
}

// Definition returns the stitch recipe.
func (s *StitchFeature) Definition() *StitchDefinition { return s.def }

// Kind implements [Feature].
func (s *StitchFeature) Kind() string { return "stitch" }

// Recompute welds the running surface bodies; a closed quilt becomes a solid unless
// MaintainAsSurface is set.
func (s *StitchFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) == 0 {
		return Output{}, errors.New("stitch: no surface bodies in the running state")
	}
	quilt, err := heal.Stitch(in.Bodies, s.def.Tolerance, s.def.MaintainAsSurface, s.featName)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: []*topo.Body{quilt}}, nil
}

// StitchFeatures adds stitch/knit features into the engine.
type StitchFeatures struct{ engine *PartFeatures }

// NewStitchFeatures binds the collection to a feature engine.
func NewStitchFeatures(engine *PartFeatures) *StitchFeatures { return &StitchFeatures{engine: engine} }

// Add welds the running surface bodies with the given tolerance, promoting a closed
// quilt to a solid unless maintainAsSurface.
func (c *StitchFeatures) Add(tolerance float64, maintainAsSurface bool) *PartFeature {
	def := &StitchDefinition{Tolerance: tolerance, MaintainAsSurface: maintainAsSurface}
	sf := &StitchFeature{def: def, featName: "Stitch"}
	pf := c.engine.Add(sf)
	sf.featName = pf.name
	return pf
}

// KnitFeatures is Inventor's alias for stitching surfaces into a quilt — the same
// operation under the "knit" name.
type KnitFeatures = StitchFeatures

// NewKnitFeatures binds a knit (alias of stitch) collection to an engine.
func NewKnitFeatures(engine *PartFeatures) *KnitFeatures { return NewStitchFeatures(engine) }

// SculptDefinition is the recipe for a sculpt: the operation against existing material, the
// coincidence tolerance for closing the bounding surfaces, and the #1881 options. Directions gives
// a per-bounding-surface keep-positive side (empty ⇒ closed-quilt stitch); BodyIndices selects the
// bounding surfaces (empty ⇒ all running bodies); AffectedIndex targets a body for join/cut.
type SculptDefinition struct {
	Operation     ops.PartFeatureOperation
	Tolerance     float64
	Directions    []bool
	BodyIndices   []int
	AffectedIndex *int
}

// SculptFeature fills a volume bounded by the running surface bodies, producing a solid (PBI-110).
// Closed quilts weld into the solid; open bounding surfaces are closed by their per-surface
// directions (#1881). new adds a body; join/cut boolean it into the affected solid.
type SculptFeature struct {
	def      *SculptDefinition
	featName string
}

// Definition returns the sculpt recipe.
func (s *SculptFeature) Definition() *SculptDefinition { return s.def }

// Kind implements [Feature].
func (s *SculptFeature) Kind() string { return "sculpt" }

// Recompute builds the sculpted solid from the selected bounding surfaces (welding a closed quilt,
// or intersecting the directed halfspaces of open surfaces) and combines it per the operation.
func (s *SculptFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) == 0 {
		return Output{}, errors.New("sculpt: no bounding surfaces in the running state")
	}
	surfaces, others := s.partition(in.Bodies)
	if len(surfaces) == 0 {
		return Output{}, errors.New("sculpt: no bounding surfaces selected")
	}
	solid, err := s.buildSolid(surfaces)
	if err != nil {
		return Output{}, err
	}
	if s.def.Operation == ops.Cut || s.def.Operation == ops.Join {
		return sculptBoolean(others, solid, s.def.Operation, s.def.AffectedIndex)
	}
	return Output{Bodies: append(others, solid)}, nil // new body
}

// partition splits the running bodies into the selected bounding surfaces and the rest (which carry
// through and hold the affected body for join/cut). No selection ⇒ every body is a bounding surface.
func (s *SculptFeature) partition(bodies []*topo.Body) (surfaces, others []*topo.Body) {
	if len(s.def.BodyIndices) == 0 {
		return bodies, nil
	}
	picked := map[int]bool{}
	for _, i := range s.def.BodyIndices { // preserve the caller's order so Directions[i] aligns
		if i >= 0 && i < len(bodies) {
			surfaces = append(surfaces, bodies[i])
			picked[i] = true
		}
	}
	for i, b := range bodies {
		if !picked[i] {
			others = append(others, b)
		}
	}
	return surfaces, others
}

// buildSolid welds a closed quilt into a solid, or — with per-surface directions — intersects the
// directed halfspaces of the (possibly open) bounding surfaces.
func (s *SculptFeature) buildSolid(surfaces []*topo.Body) (*topo.Body, error) {
	if len(s.def.Directions) > 0 {
		return surface.SculptDirected(surfaces, s.def.Directions, s.featName)
	}
	solid, err := heal.Stitch(surfaces, s.def.Tolerance, false, s.featName)
	if err != nil {
		return nil, err
	}
	if !solid.IsSolid() {
		return nil, errors.New("sculpt: bounding surfaces do not enclose a volume")
	}
	return solid, nil
}

// sculptBoolean combines the sculpted solid into the affected body (index into the carried bodies,
// default the last solid), leaving the other carried bodies untouched.
func sculptBoolean(others []*topo.Body, solid *topo.Body, op ops.PartFeatureOperation, affected *int) (Output, error) {
	target := sculptTarget(others, affected)
	if target < 0 {
		return Output{}, fmt.Errorf("sculpt %s: no solid body to modify", op)
	}
	result, err := ops.Boolean(op, others[target], solid)
	if err != nil {
		return Output{}, fmt.Errorf("sculpt %s: %w", op, err)
	}
	out := append([]*topo.Body(nil), others...)
	out[target] = result
	return Output{Bodies: out}, nil
}

// sculptTarget returns the index (into others) of the affected body: the given index if it is a
// valid solid, else the most recent solid, or -1.
func sculptTarget(others []*topo.Body, affected *int) int {
	if affected != nil && *affected >= 0 && *affected < len(others) && others[*affected].IsSolid() {
		return *affected
	}
	for i, other := range slices.Backward(others) {
		if other.IsSolid() {
			return i
		}
	}
	return -1
}

// SculptFeatures adds sculpt features into the engine.
type SculptFeatures struct{ engine *PartFeatures }

// NewSculptFeatures binds the collection to a feature engine.
func NewSculptFeatures(engine *PartFeatures) *SculptFeatures { return &SculptFeatures{engine: engine} }

// Add fills the volume bounded by the running surfaces, combining via op (closed-quilt form).
func (c *SculptFeatures) Add(op ops.PartFeatureOperation, tolerance float64) *PartFeature {
	return c.AddSculpt(&SculptDefinition{Operation: op, Tolerance: tolerance})
}

// AddSculpt fills the volume bounded by the (optionally directed / selected) surfaces per def (#1881).
func (c *SculptFeatures) AddSculpt(def *SculptDefinition) *PartFeature {
	sf := &SculptFeature{def: def, featName: "Sculpt"}
	pf := c.engine.Add(sf)
	sf.featName = pf.name
	return pf
}
