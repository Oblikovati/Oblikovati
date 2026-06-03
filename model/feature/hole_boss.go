// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
)

// Hole and boss features place parametric cylindrical cuts/bosses on a picked placement
// face (held by reference key, re-resolved each recompute). A hole drills a faceted
// cylinder at the face centroid along the inward normal and subtracts it via the general
// boolean. Boss geometry still defers (ErrDeferred → Warning). A lost placement face →
// Sick. Point placement (vs. the face centroid) and counterbore/countersink profiles are
// follow-ups.

// HoleTapInfo carries thread data for a tapped hole, consumed by hole tables (M14).
type HoleTapInfo struct {
	Tapped      bool
	Designation string
}

// HoleType is the hole's bottom/profile style.
type HoleType uint8

const (
	DrilledHole HoleType = iota
	CounterboreHole
	CountersinkHole
)

// HoleDefinition is the recipe for a hole: a placement face, diameter and depth,
// type, and optional tap data.
type HoleDefinition struct {
	PlacementFaceKey []byte
	Diameter         func() float64
	Depth            func() float64
	Type             HoleType
	Tap              HoleTapInfo
}

// HoleFeature drills a hole into the running solid.
type HoleFeature struct {
	def      *HoleDefinition
	featName string
}

// Definition returns the hole recipe.
func (h *HoleFeature) Definition() *HoleDefinition { return h.def }

// Kind implements [Feature].
func (h *HoleFeature) Kind() string { return "hole" }

// Recompute resolves the placement face, drills a faceted cylinder at its centroid along
// the inward normal, and subtracts it from the running body via the boolean.
func (h *HoleFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, ok := body.FindFaceByKey(h.def.PlacementFaceKey)
	if !ok {
		return Output{}, fmt.Errorf("hole: placement face reference lost")
	}
	r, depth := callOrZero(h.def.Diameter)/2, callOrZero(h.def.Depth)
	if r <= 0 || depth <= 0 {
		return Output{}, fmt.Errorf("hole: diameter %g and depth %g must both be > 0", 2*r, depth)
	}
	into, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0).Scale(-1))
	if err != nil {
		return Output{}, fmt.Errorf("hole: placement face has no normal")
	}
	tool := drillTool(centroidOf(faceVertexPoints(face)), into, r, depth, featOr(h.featName, "hole"))
	res, err := ops.Boolean(ops.Cut, body, tool)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res)}, nil
}

// BossDefinition is the recipe for a boss: a raised cylinder on a placement face.
type BossDefinition struct {
	PlacementFaceKey []byte
	Diameter         func() float64
	Height           func() float64
}

// BossFeature adds a cylindrical boss to the running solid.
type BossFeature struct{ def *BossDefinition }

func (b *BossFeature) Definition() *BossDefinition { return b.def }
func (b *BossFeature) Kind() string                { return "boss" }
func (b *BossFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, [][]byte{b.def.PlacementFaceKey}, "boss")
}

// HoleFeatures and BossFeatures add hole/boss features into the engine.
type (
	HoleFeatures struct{ engine *PartFeatures }
	BossFeatures struct{ engine *PartFeatures }
)

// NewHoleFeatures / NewBossFeatures bind the collections to an engine.
func NewHoleFeatures(engine *PartFeatures) *HoleFeatures { return &HoleFeatures{engine} }
func NewBossFeatures(engine *PartFeatures) *BossFeatures { return &BossFeatures{engine} }

// AddDrilled adds a simple drilled hole on the placement face.
func (c *HoleFeatures) AddDrilled(faceKey []byte, diameter, depth func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole})
}

// AddTapped adds a tapped hole with thread data.
func (c *HoleFeatures) AddTapped(faceKey []byte, diameter, depth func() float64, designation string) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole, Tap: HoleTapInfo{Tapped: true, Designation: designation}})
}

// addHole adds a hole feature, naming it (Hole1, Hole2, …) so its generated topology has
// a stable, distinct lineage.
func (c *HoleFeatures) addHole(def *HoleDefinition) *PartFeature {
	hf := &HoleFeature{def: def}
	pf := c.engine.Add(hf)
	pf.SetName(c.engine.UniqueName("Hole"))
	hf.featName = pf.name
	return pf
}

// Add adds a cylindrical boss on the placement face.
func (c *BossFeatures) Add(faceKey []byte, diameter, height func() float64) *PartFeature {
	return c.engine.Add(&BossFeature{def: &BossDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Height: height}})
}
