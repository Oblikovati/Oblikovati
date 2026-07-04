// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial curved-on-planar boss certification (#1591, ADR-0049 Slice B). A cylindrical boss whose base circle
// CLIPS the seat face edge (a spigot straddling / overhanging a plate edge) unions through the planeUV
// operand — the seat face is trimmed by the base conic, an overhang underside cap closes the cantilever, and
// the split-base wall welds to both. Certified by an INDEPENDENT analytic mass (the boss sits at z≥2, the
// plate at z≤2, so the union has no volume overlap: 10·10·2 + π·2²·3 = 200 + 12π) plus the top-priority
// tessellation-watertightness gate (the user only ever sees the mesh).

// straddlingBoss builds the fixture: a 10×10×2 plate unioned with a boss (r=2, h=3) seated on the top face at
// (4,0) so its base circle (x∈[2,6]) overhangs the plate edge at x=5.
func straddlingBoss(t *testing.T) *topo.Body {
	t.Helper()
	plate, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	boss, err := brep.SolidCylinder(math.P3(4, 0, 2), math.V3(0, 0, 1), 2, 3)
	if err != nil {
		t.Fatalf("boss: %v", err)
	}
	res, ok := brep.JoinPartialBoss(plate, boss)
	if !ok {
		t.Fatal("JoinPartialBoss declined the straddling boss")
	}
	return res
}

func TestPartialBossIsWatertightManifold(t *testing.T) {
	res := straddlingBoss(t)
	if !res.IsSolid() {
		t.Fatal("straddling boss is not a solid")
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("straddling boss has %d shells, want 1", n)
	}
	free := 0
	for _, e := range res.Edges() {
		if len(e.Uses()) != 2 {
			free++
		}
	}
	if free != 0 {
		t.Errorf("straddling boss B-rep has %d free edges, want 0 (a closed manifold)", free)
	}
}

func TestPartialBossVolumeMatchesAnalytic(t *testing.T) {
	res := straddlingBoss(t)
	want := 200 + 12*stdmath.Pi // plate + cantilevered boss, no overlap
	got := BodyGeometryProperties(res, DefaultQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > 0.01 {
		t.Errorf("straddling boss volume %.4f vs analytic %.4f (200+12π); rel %.4f > 0.01", got, want, rel)
	}
}

// TestPartialBossTessellationIsWatertight is the top-priority tessellation gate (CLAUDE.md: the user only
// ever sees the mesh). The planeUV-trimmed seat, the overhang cap and the split-base wall must weld into a
// crack-free surface — every triangle edge shared by exactly two triangles — so the rendered frame shows no
// tear where the base conic crosses the plate edge.
func TestPartialBossTessellationIsWatertight(t *testing.T) {
	res := straddlingBoss(t)
	mesh, _ := TessellateBody(res, DefaultQuality())
	if free := cornerMeshFreeEdges(mesh); free != 0 {
		t.Errorf("straddling boss tessellation has %d free edges (want 0) — a visible crack at the clipped seat", free)
	}
}
