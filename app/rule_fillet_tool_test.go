// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// TestRuleFilletToolEndToEnd drives the Rule Fillet UI: a block, choose All Rounds + a radius, OK —
// and asserts a valid solid feature was added and the convex edges were rounded (volume dropped from
// the block's). The tool needs no picking, so it commits straight from its parameters (#1076).
func TestRuleFilletToolEndToEnd(t *testing.T) {
	s, _ := newPartWithBlock(t, 6) // 6×6×2 block, vol 72
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	rf := NewRuleFilletTool()
	rf.SetRule(int(feature.RuleFilletAllRounds))
	rf.SetRadiusMM(2)
	s.StartTool(rf)
	if !rf.CanCommit() {
		t.Fatal("rule fillet tool not ready with a positive radius")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	if rf.AddedFeature() == nil || !rf.AddedFeature().Health().OK() {
		t.Fatalf("rule fillet feature not healthy: %+v", rf.AddedFeature())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("rule-filleted body not a valid solid: %+v", r)
	}
	if after := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; after >= before {
		t.Errorf("volume %g did not drop from %g — All Rounds should round the convex edges", after, before)
	}
}

// TestRuleFilletViaRibbonCommand confirms the Surface-panel ribbon command starts the tool.
func TestRuleFilletViaRibbonCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 6)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.RuleFillet"); err != nil {
		t.Fatalf("execute Surface.RuleFillet: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*RuleFilletTool); !ok {
		t.Fatal("Rule Fillet command did not start the rule fillet tool")
	}
}
