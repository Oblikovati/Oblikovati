// SPDX-License-Identifier: GPL-2.0-only

// Package benchgen synthesizes large, deeply-nested assemblies for performance
// benchmarking (M34, large-assembly scale: 100k unique / 1M total). It reproduces
// the geometric-weight distribution and DAG depth of an automotive assembly without
// modeling real car parts: four geometry tiers (fasteners, brackets, machined parts,
// systems) placed into a 6–8 level hierarchy whose dominant flyweight — a shared
// fastener kit reached through thousands of occurrence paths — exercises exactly the
// definition/occurrence sharing the renderer's instancing relies on (ADR-0038).
//
// The four tiers' unique-mesh counts and placement counts are produced exactly by
// sizing each tier's unique-definition pool to its target and round-robin placing it
// to the target count, so the per-tier reuse ratio (e.g. brackets 5×, machined ~1.5×,
// fasteners 1500×) falls out of placements/unique. Profiles parameterize the tree's
// branching and the tiers, so the committed 30k automotive spec (Auto30k) and the
// 100k-unique/1M-total goal (Auto1M) differ only by configuration.
package benchgen

import "fmt"

// Tier identifies one of the four automotive geometry classes the benchmark scatters
// through the assembly; each stresses a distinct engine hot path (instancing, frustum
// culling, raycast/selection, VRAM upload).
type Tier int

const (
	// Fastener is a low-poly primitive cluster (bolt) instanced thousands of times —
	// the Vulkan-instancing target.
	Fastener Tier = iota
	// Bracket is a medium-poly formed part scattered across the car bounds — the
	// frustum-culling / spatial-partition target.
	Bracket
	// Machined is a high-poly part (dense facets standing in for fillets/chamfers/
	// bores) — the raycast & selection-precision target.
	Machined
	// System is a massive memory-heavy unique part — the VRAM / cold-load target.
	System
)

// String renders a tier name for stats and document paths.
func (t Tier) String() string {
	switch t {
	case Fastener:
		return "fastener"
	case Bracket:
		return "bracket"
	case Machined:
		return "machined"
	case System:
		return "system"
	default:
		return fmt.Sprintf("tier(%d)", int(t))
	}
}

// TierSpec is one tier's contribution to the assembly. UniqueMeshes sizes the pool of
// distinct part definitions; Placements is the total number of occurrences of that
// pool across the whole flattened tree (so reuse = Placements/UniqueMeshes). Sides and
// HeightCm shape the extruded n-gon prism that stands in for the tier's geometry —
// Sides drives facet count (poly weight), so high-poly tiers carry many sides.
type TierSpec struct {
	Tier         Tier
	UniqueMeshes int
	Placements   int
	Sides        int
	RadiusCm     float64
	HeightCm     float64
}

// Profile is a complete synthetic-assembly recipe: the structural branching of the
// intermediate sub-assembly tree (Systems→Modules→SubModules→Bays) plus the four
// tiers. Bays is the count of leaf bays under each sub-module; the product of the four
// branching factors is the total number of leaf bays the placements are spread across.
type Profile struct {
	Name       string
	Systems    int // L1: vehicle systems placed by the root
	Modules    int // L2: modules placed by each system
	SubModules int // L3: sub-modules placed by each module
	Bays       int // L4: leaf bays placed by each sub-module
	Tiers      []TierSpec
}

// BayCount is the number of leaf bays the tree expands to — the breadth the flatten
// traversal walks at its deepest non-fastener level.
func (p Profile) BayCount() int { return p.Systems * p.Modules * p.SubModules * p.Bays }

// Tier returns the spec for tier t and whether the profile defines it.
func (p Profile) Tier(t Tier) (TierSpec, bool) {
	for _, s := range p.Tiers {
		if s.Tier == t {
			return s, true
		}
	}
	return TierSpec{}, false
}

// TotalPlacements sums every tier's placement count — the target leaf-occurrence count
// of the flattened assembly.
func (p Profile) TotalPlacements() int {
	total := 0
	for _, s := range p.Tiers {
		total += s.Placements
	}
	return total
}

// TotalUniqueMeshes sums every tier's unique-definition pool — the count memory should
// scale with (not TotalPlacements), the central claim the benchmark validates.
func (p Profile) TotalUniqueMeshes() int {
	total := 0
	for _, s := range p.Tiers {
		total += s.UniqueMeshes
	}
	return total
}

// Auto30k is the committed automotive benchmark: ~30,000 leaf placements (75% / 16% /
// 7% / 2% across the four tiers) over a 7-level DAG (~320 leaf bays), with ~2,815
// unique meshes. The branching 5×4×4×4 and the per-tier pools are chosen so the
// realized counts land within ~1% of the 22,500 / 5,000 / 2,200 / 300 spec.
func Auto30k() Profile {
	return Profile{
		Name: "auto30k", Systems: 5, Modules: 4, SubModules: 4, Bays: 4,
		Tiers: []TierSpec{
			{Fastener, 15, 22500, 6, 0.4, 1.2},
			{Bracket, 1000, 5000, 12, 2.0, 0.4},
			{Machined, 1500, 2200, 64, 3.0, 4.0},
			{System, 300, 300, 96, 18.0, 22.0},
		},
	}
}

// Auto1M scales the same shape to the headline goal — ~1,000,000 leaf placements over
// ~10,240 leaf bays with ~101,500 unique meshes (100k-unique class). Generating it is
// heavy by design (it is the cold-load / VRAM stress); the committed baseline targets
// Auto30k.
func Auto1M() Profile {
	return Profile{
		Name: "auto1m", Systems: 10, Modules: 8, SubModules: 8, Bays: 16,
		Tiers: []TierSpec{
			{Fastener, 60, 750000, 6, 0.4, 1.2},
			{Bracket, 50000, 165000, 12, 2.0, 0.4},
			{Machined, 41000, 75000, 64, 3.0, 4.0},
			{System, 10000, 10000, 96, 18.0, 22.0},
		},
	}
}

// ProfileByName resolves a built-in profile by its --profile flag value, erroring with
// the offending name and the known set so a typo is self-explaining.
func ProfileByName(name string) (Profile, error) {
	switch name {
	case "auto30k":
		return Auto30k(), nil
	case "auto1m":
		return Auto1M(), nil
	default:
		return Profile{}, fmt.Errorf("benchgen: unknown profile %q; known profiles are auto30k, auto1m", name)
	}
}
