// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/model/health"
)

func TestHoleAndBossResolvePlacementFace(t *testing.T) {
	body := prismBody()
	face := body.Faces()[0].ReferenceKey()

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	holes := NewHoleFeatures(fs)
	drilled := holes.AddDrilled(face, func() float64 { return 0.3 }, func() float64 { return 0.5 })
	tapped := holes.AddTapped(face, func() float64 { return 0.2 }, func() float64 { return 0.4 }, "M5x0.8")
	boss := NewBossFeatures(fs).Add(face, func() float64 { return 0.5 }, func() float64 { return 1 })
	fs.Recompute()

	for name, pf := range map[string]*PartFeature{"hole": drilled, "boss": boss} {
		if pf.Health().Status != health.Warning {
			t.Errorf("%s health = %v, want warning (placement resolved, cut deferred)", name, pf.Health().Status)
		}
	}
	if tap := tapped.Definition().(*HoleFeature).Definition().Tap; !tap.Tapped || tap.Designation != "M5x0.8" {
		t.Errorf("tap info = %+v, want tapped M5x0.8", tap)
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
