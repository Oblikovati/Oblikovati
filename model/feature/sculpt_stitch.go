// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
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
	quilt, err := ops.Stitch(in.Bodies, s.def.Tolerance, s.def.MaintainAsSurface, s.featName)
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

// SculptDefinition is the recipe for a sculpt: the operation against existing
// material and the coincidence tolerance for closing the bounding surfaces.
type SculptDefinition struct {
	Operation ops.PartFeatureOperation
	Tolerance float64
}

// SculptFeature fills a volume bounded by the running surface bodies, producing a
// solid (PBI-110). The bounding surfaces must enclose a closed cell.
type SculptFeature struct {
	def      *SculptDefinition
	featName string
}

// Definition returns the sculpt recipe.
func (s *SculptFeature) Definition() *SculptDefinition { return s.def }

// Kind implements [Feature].
func (s *SculptFeature) Kind() string { return "sculpt" }

// Recompute welds the bounding surfaces into a solid; if they do not enclose a
// volume (the quilt is open) the feature goes sick — it cannot define a region.
func (s *SculptFeature) Recompute(in Input) (Output, error) {
	if len(in.Bodies) == 0 {
		return Output{}, errors.New("sculpt: no bounding surfaces in the running state")
	}
	solid, err := ops.Stitch(in.Bodies, s.def.Tolerance, false, s.featName)
	if err != nil {
		return Output{}, err
	}
	if !solid.IsSolid() {
		return Output{}, errors.New("sculpt: bounding surfaces do not enclose a volume")
	}
	return Output{Bodies: []*topo.Body{solid}}, nil
}

// SculptFeatures adds sculpt features into the engine.
type SculptFeatures struct{ engine *PartFeatures }

// NewSculptFeatures binds the collection to a feature engine.
func NewSculptFeatures(engine *PartFeatures) *SculptFeatures { return &SculptFeatures{engine: engine} }

// Add fills the volume bounded by the running surfaces, combining via op.
func (c *SculptFeatures) Add(op ops.PartFeatureOperation, tolerance float64) *PartFeature {
	def := &SculptDefinition{Operation: op, Tolerance: tolerance}
	sf := &SculptFeature{def: def, featName: "Sculpt"}
	pf := c.engine.Add(sf)
	sf.featName = pf.name
	return pf
}
