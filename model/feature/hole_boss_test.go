// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

func TestHoleGeneratesRealGeometryBossDefers(t *testing.T) {
	// A drilled hole now generates real geometry (healthy). Each feature is tested on its
	// own body: a boolean rebuilds the topology with new lineage, so a reference to a
	// pre-cut face does not survive (chaining across a boolean is a follow-up).
	hb := prismBody()
	fsHole := NewPartFeatures(nil, nil)
	NewBaseFeatures(fsHole).AddBase(hb)
	drilled := NewHoleFeatures(fsHole).AddDrilled(hb.Faces()[0].ReferenceKey(), func() float64 { return 0.3 }, func() float64 { return 2 })
	fsHole.Recompute()
	if !drilled.Health().OK() {
		t.Errorf("hole health = %v, want OK (real cut)", drilled.Health())
	}

	bb := prismBody()
	fsBoss := NewPartFeatures(nil, nil)
	NewBaseFeatures(fsBoss).AddBase(bb)
	boss := NewBossFeatures(fsBoss).Add(bb.Faces()[0].ReferenceKey(), func() float64 { return 0.5 }, func() float64 { return 1 })
	fsBoss.Recompute()
	if boss.Health().Status != health.Warning {
		t.Errorf("boss health = %v, want warning (geometry deferred)", boss.Health().Status)
	}

	tapped := NewHoleFeatures(NewPartFeatures(nil, nil)).AddTapped([]byte("f"), func() float64 { return 0.2 }, func() float64 { return 0.4 }, "M5x0.8")
	if tap := tapped.Definition().(*HoleFeature).Definition().Tap; !tap.Tapped || tap.Designation != "M5x0.8" {
		t.Errorf("tap info = %+v, want tapped M5x0.8", tap)
	}
}

func TestHoleDrillsThroughForReal(t *testing.T) {
	// A 4×4×2 block (vol 32). Drill a Ø2 hole through the top face → removes the
	// 32-gon cylinder of radius 1 over the full thickness 2.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // end cap (z=2), normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 3 }) // depth 3 > thickness 2 ⇒ through
	fs.Recompute()

	if !hole.Health().OK() {
		t.Fatalf("through hole sick: %+v", hole.Health())
	}
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("hole result = %d bodies, want 1", len(res))
	}
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	want := 32 - regularPolygonArea(1, holeFacets)*2 // block − through cylinder
	if got := ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("drilled volume = %g, want %g (32 − Ø2 through hole)", got, want)
	}
}

func TestHoleGoesSickOnLostFace(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	hole := NewHoleFeatures(fs).AddDrilled([]byte("ghost-face"), func() float64 { return 0.3 }, func() float64 { return 0.5 })
	fs.Recompute()
	if hole.Health().Status != health.Sick {
		t.Errorf("hole on a lost placement face = %v, want sick", hole.Health().Status)
	}
}

func TestHoleBossDefinitionAccessors(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	h := NewHoleFeatures(fs).AddDrilled([]byte("f"), func() float64 { return 1 }, func() float64 { return 2 })
	if h.Definition().(*HoleFeature).Definition().Type != DrilledHole {
		t.Error("hole definition not accessible")
	}
	b := NewBossFeatures(fs).Add([]byte("f"), func() float64 { return 1 }, func() float64 { return 2 })
	if b.Definition().(*BossFeature).Definition().Diameter() != 1 {
		t.Error("boss definition not accessible")
	}
}
