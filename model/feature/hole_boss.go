// SPDX-License-Identifier: GPL-2.0-only

package feature

// Hole and boss features place parametric cylindrical cuts/bosses on a picked
// placement face (held by reference key, re-resolved each recompute). The drill/
// counterbore/boss B-rep requires the general boolean against the running solid
// (kernel phase C), so once the placement face resolves these defer geometry
// (ErrDeferred → Warning); a lost placement face → Sick.

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
type HoleFeature struct{ def *HoleDefinition }

// Definition returns the hole recipe.
func (h *HoleFeature) Definition() *HoleDefinition { return h.def }

// Kind implements [Feature].
func (h *HoleFeature) Kind() string { return "hole" }

// Recompute resolves the placement face, then defers the cut geometry.
func (h *HoleFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, [][]byte{h.def.PlacementFaceKey}, "hole")
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
	return c.engine.Add(&HoleFeature{def: &HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole}})
}

// AddTapped adds a tapped hole with thread data.
func (c *HoleFeatures) AddTapped(faceKey []byte, diameter, depth func() float64, designation string) *PartFeature {
	def := &HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole, Tap: HoleTapInfo{Tapped: true, Designation: designation}}
	return c.engine.Add(&HoleFeature{def: def})
}

// Add adds a cylindrical boss on the placement face.
func (c *BossFeatures) Add(faceKey []byte, diameter, height func() float64) *PartFeature {
	return c.engine.Add(&BossFeature{def: &BossDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Height: height}})
}
