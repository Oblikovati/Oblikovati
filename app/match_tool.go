// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/feature"
)

// The Match Surface tool (M36-F05) rebuilds the running surface body against the previous one to a
// chosen continuity (G0/G1/G2/G3) — the defining Class-A move. It is parameter-only (the two most
// recent surface bodies are the inputs); on commit it reports the achieved cross-edge continuity,
// measured by the F13 checker, so the user sees the match landed within tolerance.

// matchEdgeOptions are the selectable boundary edges, in dialog order.
var matchEdgeOptions = []geom.Boundary{geom.UMinEdge, geom.UMaxEdge, geom.VMinEdge, geom.VMaxEdge}

// matchEdgeLabels label the boundary edges for the dialog.
var matchEdgeLabels = []string{"U Min", "U Max", "V Min", "V Max"}

// MatchTool matches the running surface body to the previous one.
type MatchTool struct {
	dialogTool
	order      int
	sourceEdge int // index into matchEdgeOptions
	targetEdge int
	added      *feature.PartFeature
}

// NewMatchTool returns a match tool defaulting to G2 across the canonical side-by-side seam (the
// source's U-min edge to the target's U-max edge).
func NewMatchTool() *MatchTool {
	return &MatchTool{order: 2, sourceEdge: 0, targetEdge: 1}
}

// Name implements [Tool].
func (t *MatchTool) Name() string { return "Match Surface" }

// Prompt guides the input.
func (t *MatchTool) Prompt(*Session) string {
	return "Match the running surface to the previous one: pick the continuity and the two edges, then OK."
}

// Params exposes the continuity order and the two edges for the generic dialog.
func (t *MatchTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "Continuity", Options: []string{"G0", "G1", "G2", "G3"}, Get: func() int { return t.order }, Set: func(v int) { t.order = v }},
		{Label: "Source Edge", Options: matchEdgeLabels, Get: func() int { return t.sourceEdge }, Set: func(v int) { t.sourceEdge = v }},
		{Label: "Target Edge", Options: matchEdgeLabels, Get: func() int { return t.targetEdge }, Set: func(v int) { t.targetEdge = v }},
	}}
}

// CanCommit is always true (the continuity order and edges are always valid choices); a missing
// target body surfaces as a feature-health error on commit.
func (t *MatchTool) CanCommit() bool { return true }

// Commit adds the match feature, recomputes, and reports the achieved continuity.
func (t *MatchTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	se, te := matchEdgeOptions[t.sourceEdge], matchEdgeOptions[t.targetEdge]
	t.added = feature.NewMatchFeatures(part.Features()).Add(t.order, se, te)
	part.Recompute()
	s.recordEdit(part, "Match Surface")
	if !t.added.Health().OK() {
		return errors.New("match surface: " + t.added.Health().Reason)
	}
	s.feedNotice(matchContinuityNotice(part, se, te))
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *MatchTool) AddedFeature() *feature.PartFeature { return t.added }

// matchContinuityNotice measures the matched seam with the F13 cross-edge checker and formats the
// achieved G1/G2 deviation for the Command Window.
func matchContinuityNotice(part interface{ SurfaceBodies() *topo.SurfaceBodies }, srcEdge, tgtEdge geom.Boundary) string {
	bodies := part.SurfaceBodies()
	if bodies.Count() < 2 {
		return "Matched surface"
	}
	src, sok := nurbsFaceSurface(bodies.Item(bodies.Count() - 1))
	tgt, tok := nurbsFaceSurface(bodies.Item(bodies.Count() - 2))
	if !sok || !tok {
		return "Matched surface"
	}
	rep := analysis.CrossEdgeContinuity(src, tgt, edgeParamOf(srcEdge), edgeParamOf(tgtEdge), 21)
	return fmt.Sprintf("Matched surface — G1 angle max %.3f°, G2 curvature max %.1f%%, G0 gap max %.4g",
		rep.MaxNormalDeg, rep.MaxCurvPct, rep.MaxGap)
}

// nurbsFaceSurface returns a body's first NURBS face surface.
func nurbsFaceSurface(b *topo.Body) (geom.BSplineSurface, bool) {
	for _, f := range b.Faces() {
		if s, ok := f.Geometry().(geom.BSplineSurface); ok {
			return s, true
		}
	}
	return geom.BSplineSurface{}, false
}

// edgeParamOf maps a boundary to the analysis edge-trace (t∈[0,1] → surface (u,v)) along that edge.
func edgeParamOf(edge geom.Boundary) analysis.EdgeParam {
	switch edge {
	case geom.UMaxEdge:
		return func(t float64) (u, v float64) { return 1, t }
	case geom.UMinEdge:
		return func(t float64) (u, v float64) { return 0, t }
	case geom.VMaxEdge:
		return func(t float64) (u, v float64) { return t, 1 }
	default: // VMinEdge
		return func(t float64) (u, v float64) { return t, 0 }
	}
}
