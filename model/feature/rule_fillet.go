// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// RuleFilletRule selects which of the running body's edges a rule fillet rounds, by dihedral class —
// the M20-F10 plastic-part "rule fillet" (#486), where you round a whole class of edges in one
// feature instead of picking them individually.
type RuleFilletRule int

const (
	// RuleFilletAllRounds rounds every CONVEX (outside) edge — Inventor's "All Rounds".
	RuleFilletAllRounds RuleFilletRule = iota
	// RuleFilletAllFillets fills every CONCAVE (inside) edge — Inventor's "All Fillets".
	RuleFilletAllFillets
	// RuleFilletAllEdges rounds every manifold edge, convex and concave.
	RuleFilletAllEdges
)

var ruleFilletNames = map[RuleFilletRule]string{
	RuleFilletAllRounds:  "allRounds",
	RuleFilletAllFillets: "allFillets",
	RuleFilletAllEdges:   "allEdges",
}

// String returns the rule's wire spelling.
func (r RuleFilletRule) String() string {
	if s, ok := ruleFilletNames[r]; ok {
		return s
	}
	return fmt.Sprintf("RuleFilletRule(%d)", int(r))
}

// ParseRuleFilletRule resolves a wire spelling back to its rule.
func ParseRuleFilletRule(s string) (RuleFilletRule, bool) {
	for r, name := range ruleFilletNames {
		if name == s {
			return r, true
		}
	}
	return 0, false
}

// RuleFilletDefinition rounds the edges of the running body that match Rule, all at one Radius.
type RuleFilletDefinition struct {
	Rule   RuleFilletRule
	Radius func() float64
}

// RuleFilletFeature rounds a rule-selected class of edges.
//
// Example: round every outside edge of the running body at 1 mm —
//
//	NewDressUpFeatures(part.Features()).AddRuleFillet(RuleFilletAllRounds, func() float64 { return 1 })
type RuleFilletFeature struct{ def *RuleFilletDefinition }

// Definition returns the feature's definition.
func (f *RuleFilletFeature) Definition() *RuleFilletDefinition { return f.def }

// Kind names the feature type.
func (f *RuleFilletFeature) Kind() string { return "rule-fillet" }

// Recompute rounds the rule-selected edges of the running body.
func (f *RuleFilletFeature) Recompute(in Input) (Output, error) {
	return ruleFilletBody(in, f.def.Rule, callOrZero(f.def.Radius), "rule-fillet")
}

// ruleFilletBody selects the running body's edges matching rule and rounds them with the constant-
// radius edge-fillet kernel. A body with no matching edge is a no-op (e.g. a rule fillet of an
// already-rounded body), not an error. Concave edges fill outward (the standard rule-fillet result).
func ruleFilletBody(in Input, rule RuleFilletRule, radius float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if radius <= 0 {
		return Output{}, fmt.Errorf("%s: radius %g must be > 0", feat, radius)
	}
	keys := ruleEdgeKeys(body, rule)
	if len(keys) == 0 {
		return Output{Bodies: in.Bodies}, nil
	}
	return filletBody(in, keys, radius, types.FilletCornerMiter, types.FilletConcaveOutward, blendProfile{}, feat)
}

// ruleEdgeKeys returns the reference keys of every edge whose dihedral class matches rule.
func ruleEdgeKeys(body *topo.Body, rule RuleFilletRule) [][]byte {
	var out [][]byte
	for _, e := range body.Edges() {
		if matchesFilletRule(rule, ops.ClassifyEdgeConvexity(e)) {
			out = append(out, e.ReferenceKey())
		}
	}
	return out
}

// matchesFilletRule reports whether a dihedral class is selected by rule.
func matchesFilletRule(rule RuleFilletRule, c ops.EdgeConvexity) bool {
	switch rule {
	case RuleFilletAllRounds:
		return c == ops.EdgeConvex
	case RuleFilletAllFillets:
		return c == ops.EdgeConcave
	case RuleFilletAllEdges:
		return c == ops.EdgeConvex || c == ops.EdgeConcave
	}
	return false
}
