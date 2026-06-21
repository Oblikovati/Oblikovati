// SPDX-License-Identifier: GPL-2.0-only

package benchgen

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// tinyProfile is a fast stand-in for Auto30k: the same 7-level shape and four tiers at
// ~1/100 the scale, so tests stay F.I.R.S.T while exercising every code path (pools,
// shared kit, deep tree, round-robin reuse, quota distribution).
func tinyProfile() Profile {
	return Profile{
		Name: "tiny", Systems: 2, Modules: 2, SubModules: 2, Bays: 2,
		Tiers: []TierSpec{
			{Fastener, 4, 240, 6, 0.4, 1.2},
			{Bracket, 20, 80, 12, 2.0, 0.4},
			{Machined, 24, 32, 16, 3.0, 4.0},
			{System, 8, 8, 16, 6.0, 8.0},
		},
	}
}

func newMemWorkspace() *doc.Workspace {
	return doc.NewWorkspace(persistence.NewPackageStore())
}

func TestGenerateRealizesTierCounts(t *testing.T) {
	ws := newMemWorkspace()
	root, stats, err := Generate(ws, "tiny", tinyProfile())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Non-fastener tiers hit their target exactly (quota distribution); fasteners come
	// from the shared kit so they land near target (kitSize×bays).
	if got := stats.PerTier[Bracket]; got != 80 {
		t.Errorf("bracket placements = %d, want 80", got)
	}
	if got := stats.PerTier[Machined]; got != 32 {
		t.Errorf("machined placements = %d, want 32", got)
	}
	if got := stats.PerTier[System]; got != 8 {
		t.Errorf("system placements = %d, want 8", got)
	}
	// LeafPlacements is ground truth from the flattened tree: it must equal the sum of
	// the realized per-tier counts.
	want := stats.PerTier[Fastener] + stats.PerTier[Bracket] + stats.PerTier[Machined] + stats.PerTier[System]
	if stats.LeafPlacements != want {
		t.Errorf("LeafPlacements = %d, want %d (sum of tiers)", stats.LeafPlacements, want)
	}
	if root.Content().(*compdef.AssemblyComponentDefinition).Occurrences().Count() != 2 {
		t.Errorf("root should place %d systems", tinyProfile().Systems)
	}
}

func TestGenerateMemoryScalesWithUniqueNotPlacements(t *testing.T) {
	ws := newMemWorkspace()
	_, stats, err := Generate(ws, "tiny", tinyProfile())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// The flyweight claim: far more leaf placements than unique meshes (heavy reuse,
	// dominated by the fastener kit), and the unique pool matches the profile.
	if stats.UniqueMeshes != 4+20+24+8 {
		t.Errorf("UniqueMeshes = %d, want %d", stats.UniqueMeshes, 4+20+24+8)
	}
	if stats.LeafPlacements <= stats.UniqueMeshes {
		t.Errorf("expected placements (%d) >> unique meshes (%d)", stats.LeafPlacements, stats.UniqueMeshes)
	}
}

func TestGenerateBodiesTessellate(t *testing.T) {
	ws := newMemWorkspace()
	root, _, err := Generate(ws, "tiny", tinyProfile())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Tessellation correctness is the top priority: every placed leaf body must mesh to
	// a non-empty facet set, or the assembly is invisible/broken downstream.
	placed := root.Content().(*compdef.AssemblyComponentDefinition).PlacedBodies()
	if len(placed) == 0 {
		t.Fatal("no placed bodies")
	}
	checked := 0
	for _, pb := range placed[:min(12, len(placed))] {
		mesh, _ := ops.TessellateBody(pb.Body, ops.DefaultQuality())
		if mesh == nil || mesh.TriangleCount() == 0 {
			t.Fatalf("placed body %v tessellated to no triangles", pb.Path)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no bodies checked")
	}
}

func TestDistributeSumsToTotal(t *testing.T) {
	cases := []struct{ total, n int }{{80, 16}, {2200, 320}, {7, 4}, {0, 5}, {5, 1}}
	for _, c := range cases {
		sum := 0
		for i := 0; i < c.n; i++ {
			sum += distribute(c.total, c.n, i)
		}
		if sum != c.total {
			t.Errorf("distribute(%d,%d) summed to %d, want %d", c.total, c.n, sum, c.total)
		}
	}
}

func TestProfileByName(t *testing.T) {
	if _, err := ProfileByName("auto30k"); err != nil {
		t.Errorf("auto30k: %v", err)
	}
	if _, err := ProfileByName("auto1m"); err != nil {
		t.Errorf("auto1m: %v", err)
	}
	if _, err := ProfileByName("nope"); err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestProfileAccessors(t *testing.T) {
	for _, p := range []Profile{Auto30k(), Auto1M()} {
		if p.BayCount() != p.Systems*p.Modules*p.SubModules*p.Bays {
			t.Errorf("%s BayCount mismatch", p.Name)
		}
		if p.TotalPlacements() == 0 || p.TotalUniqueMeshes() == 0 {
			t.Errorf("%s totals should be non-zero", p.Name)
		}
		if _, ok := p.Tier(Fastener); !ok {
			t.Errorf("%s should define the fastener tier", p.Name)
		}
	}
	// A tier the profile does not define is reported absent.
	empty := Profile{Name: "empty"}
	if _, ok := empty.Tier(Bracket); ok {
		t.Error("empty profile should not define any tier")
	}
}

func TestTierString(t *testing.T) {
	cases := map[Tier]string{Fastener: "fastener", Bracket: "bracket", Machined: "machined", System: "system"}
	for tier, want := range cases {
		if got := tier.String(); got != want {
			t.Errorf("Tier(%d).String() = %q, want %q", int(tier), got, want)
		}
	}
	if got := Tier(99).String(); got != "tier(99)" {
		t.Errorf("unknown tier String = %q", got)
	}
}

func TestGenerateRejectsDegenerateGeometry(t *testing.T) {
	// A profile whose tier has fewer than three polygon sides cannot form a profile, so
	// generation must fail with the offending value rather than build an empty body.
	bad := Profile{
		Name: "bad", Systems: 1, Modules: 1, SubModules: 1, Bays: 1,
		Tiers: []TierSpec{{Tier: Bracket, UniqueMeshes: 1, Placements: 1, Sides: 2, RadiusCm: 1, HeightCm: 1}},
	}
	if _, _, err := Generate(newMemWorkspace(), "bad", bad); err == nil {
		t.Error("expected error for a 2-sided polygon tier")
	}
}

func TestGridPositionSpreadsAndCenters(t *testing.T) {
	// A single cell sits at the volume center; many indices fan out to distinct points.
	c := gridPosition(0, 1, carBoundsCm)
	if c.X != carBoundsCm.X/2 {
		t.Errorf("single-cell X = %g, want center %g", c.X, carBoundsCm.X/2)
	}
	a := bayTransform(0, 64).Translation()
	b := bayTransform(40, 64).Translation()
	if a == b {
		t.Error("distinct bay indices should map to distinct positions")
	}
	if partTransform(1).Translation() == partTransform(2).Translation() {
		t.Error("distinct part slots should jitter to distinct offsets")
	}
}
