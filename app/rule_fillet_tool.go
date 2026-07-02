// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// RuleFilletTool is the interactive Rule Fillet command (#1076): round a whole CLASS of the active
// part's edges in one feature — every convex edge (All Rounds), every concave edge (All Fillets), or
// every manifold edge (All Edges) — at one radius. It needs no viewport picking, so it is a dialog
// tool: the user chooses the rule and radius in the property window, then OK. (The geometry — the
// edge-class classifier + the dress-up kernel — already shipped via #1020.)
type RuleFilletTool struct {
	dialogTool
	rule     feature.RuleFilletRule
	radiusMM float64
	added    *feature.PartFeature
}

// ruleFilletRuleOptions are the dropdown labels, indexed by [feature.RuleFilletRule].
var ruleFilletRuleOptions = []string{"All Rounds", "All Fillets", "All Edges"}

// NewRuleFilletTool returns a rule-fillet tool defaulting to All Rounds at a 2 mm radius.
func NewRuleFilletTool() *RuleFilletTool {
	return &RuleFilletTool{rule: feature.RuleFilletAllRounds, radiusMM: 2}
}

// Name implements [Tool].
func (t *RuleFilletTool) Name() string { return "Rule Fillet" }

// Rule / SetRule and RadiusMM / SetRadiusMM are the property-window accessors.
func (t *RuleFilletTool) Rule() int         { return int(t.rule) }
func (t *RuleFilletTool) SetRule(i int)     { t.rule = feature.RuleFilletRule(i) }
func (t *RuleFilletTool) RadiusMM() float64 { return t.radiusMM }
func (t *RuleFilletTool) SetRadiusMM(mm float64) {
	if mm > 0 {
		t.radiusMM = mm
	}
}

// Params exposes the rule selector and radius for the generic property dialog.
func (t *RuleFilletTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{{Label: "Rule", Options: ruleFilletRuleOptions, Get: t.Rule, Set: t.SetRule}},
		Floats:  []FloatParam{{Label: "Radius (mm)", Get: t.RadiusMM, Set: t.SetRadiusMM}},
	}
}

// CanCommit is true once a positive radius is set (always, given the default).
func (t *RuleFilletTool) CanCommit() bool { return t.radiusMM > 0 }

// addRuleFillet builds the rule-fillet feature into fs — the shared constructor used by both
// Commit (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *RuleFilletTool) addRuleFillet(fs *feature.PartFeatures) *feature.PartFeature {
	radiusCm := t.radiusMM / 10 // model/database length unit is the centimetre (1 unit = 10 mm)
	return feature.NewDressUpFeatures(fs).AddRuleFillet(t.rule, func() float64 { return radiusCm })
}

// DraftFeature implements [PartFeatureTool] (#1626): the rule fillet it would commit, built into
// a scratch engine so the commit gate and preview can evaluate it without touching the part.
func (t *RuleFilletTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addRuleFillet(fs), nil
	})
}

// Commit rounds the chosen edge class on the active part and recomputes; a sick feature (no matching
// edges, or a self-colliding radius) keeps the tool open with the reason.
func (t *RuleFilletTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addRuleFillet(part.Features())
	part.Recompute()
	s.recordEdit(part, "Rule Fillet")
	if !t.added.Health().OK() {
		return errors.New("rule fillet: " + t.added.Health().Reason)
	}
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RuleFilletTool) AddedFeature() *feature.PartFeature { return t.added }
