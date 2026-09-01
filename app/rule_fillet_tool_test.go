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
	t.Parallel()
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

// TestRuleFilletToolParams exercises the property-dialog surface: the name, the rule/radius
// accessors (radius rejects a non-positive value), and the Params model the head renders.
func TestRuleFilletToolParams(t *testing.T) {
	t.Parallel()
	tl := NewRuleFilletTool()
	if tl.Name() != "Rule Fillet" {
		t.Errorf("name = %q, want Rule Fillet", tl.Name())
	}
	tl.SetRule(int(feature.RuleFilletAllEdges))
	tl.SetRadiusMM(5)
	if tl.Rule() != int(feature.RuleFilletAllEdges) || tl.RadiusMM() != 5 {
		t.Errorf("rule/radius = %d/%g, want allEdges/5", tl.Rule(), tl.RadiusMM())
	}
	tl.SetRadiusMM(-1) // a non-positive radius is rejected, keeping the prior value
	if tl.RadiusMM() != 5 {
		t.Errorf("radius after -1 = %g, want it kept at 5", tl.RadiusMM())
	}

	p := tl.Params()
	if len(p.Choices) != 1 || len(p.Choices[0].Options) != 3 || len(p.Floats) != 1 {
		t.Fatalf("params = %d choices (%d options) / %d floats, want 1 (3) / 1", len(p.Choices), len(p.Choices[0].Options), len(p.Floats))
	}
	p.Choices[0].Set(int(feature.RuleFilletAllFillets))
	p.Floats[0].Set(8)
	if p.Choices[0].Get() != int(feature.RuleFilletAllFillets) || p.Floats[0].Get() != 8 {
		t.Errorf("param round-trip: rule %d radius %g, want allFillets/8", p.Choices[0].Get(), p.Floats[0].Get())
	}
}

// TestRuleFilletToolCommitNoPart covers the no-active-part error path.
func TestRuleFilletToolCommitNoPart(t *testing.T) {
	t.Parallel()
	if err := NewRuleFilletTool().Commit(NewSession()); err == nil {
		t.Error("commit with no active part should error")
	}
}

// TestRuleFilletViaRibbonCommand confirms the Surface-panel ribbon command starts the tool.
func TestRuleFilletViaRibbonCommand(t *testing.T) {
	t.Parallel()
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

// TestRuleFilletToolDraftFeature pins the #1626 commit-gate seam: no draft while the radius is
// invalid, and a non-nil draft — the same rule fillet Commit builds — once commit-ready.
func TestRuleFilletToolDraftFeature(t *testing.T) {
	t.Parallel()
	tl := NewRuleFilletTool()
	tl.radiusMM = 0 // force below the gate (SetRadiusMM rejects non-positive values)
	if _, ok := tl.DraftFeature(nil); ok {
		t.Error("DraftFeature must not build while the radius is non-positive")
	}
	tl.SetRadiusMM(2)
	if draft, ok := tl.DraftFeature(nil); !ok || draft == nil {
		t.Fatalf("DraftFeature = (%v, %v), want a non-nil draft once commit-ready", draft, ok)
	}
}
