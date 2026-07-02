// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/renderer"
)

// The Continuity Check tool (M36-F13) is the numeric companion to the Surface Analysis overlay:
// pick an edge shared by two faces and it reports the positional gap (G0), normal angle (G1) and
// curvature difference (G2) along the edge, and draws the edge coloured green→red by the worst
// local deviation. It is the quantitative acceptance gate behind match/blend/bridge/extend.

// continuityCheckSamples is how many points along the edge the checker measures.
const continuityCheckSamples = 25

// ContinuityCheckTool measures cross-edge continuity on a picked shared edge.
type ContinuityCheckTool struct {
	edge    *EdgeHandle
	report  *analysis.ContinuityReport
	overlay []renderer.DrawItem
}

// NewContinuityCheckTool returns the continuity-check tool.
func NewContinuityCheckTool() *ContinuityCheckTool { return &ContinuityCheckTool{} }

// Name implements [Tool].
func (t *ContinuityCheckTool) Name() string { return "Continuity Check" }

// Start is a no-op; the engine installs the edge filter from AcceptedKinds.
func (t *ContinuityCheckTool) Start(*Session) {}

// AcceptedKinds declares the tool picks edges (the shared seam to measure).
func (t *ContinuityCheckTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked edge for the unified highlight.
func (t *ContinuityCheckTool) Picks() []Selectable { return edgeSelectables(t.Edges()) }

// Edges returns the picked edge (one or none).
func (t *ContinuityCheckTool) Edges() []EdgeHandle {
	if t.edge == nil {
		return nil
	}
	return []EdgeHandle{*t.edge}
}

// Pick measures continuity across the picked edge and reports the result to the Command Window.
func (t *ContinuityCheckTool) Pick(s *Session, sel Selectable) {
	e, ok := sel.(EdgeHandle)
	if !ok {
		return
	}
	rep, overlay, err := measureEdgeContinuity(e.Edge)
	if err != nil {
		s.SetNotice(err.Error())
		return
	}
	ec := e
	t.edge, t.report, t.overlay = &ec, &rep, overlay
	s.feedNotice(continuitySummary(rep))
}

// CanCommit is false — the tool reports on pick, there is nothing to commit.
func (t *ContinuityCheckTool) CanCommit() bool { return false }

// Commit is a no-op (report-on-pick).
func (t *ContinuityCheckTool) Commit(*Session) error { return nil }

// Cancel restores the default selection filter.
func (t *ContinuityCheckTool) Cancel(*Session) {}

// Preview draws the edge coloured by local continuity deviation.
func (t *ContinuityCheckTool) Preview(*Session) []renderer.DrawItem { return t.overlay }

// Report returns the most recent continuity report (nil before a pick), for inspection/tests.
func (t *ContinuityCheckTool) Report() *analysis.ContinuityReport { return t.report }

// measureEdgeContinuity builds the edge→surface traces for the edge's two faces and runs the
// cross-edge continuity checker, returning the report and the coloured along-edge overlay.
func measureEdgeContinuity(edge *topo.Edge) (analysis.ContinuityReport, []renderer.DrawItem, error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return analysis.ContinuityReport{}, nil, fmt.Errorf("continuity check needs an edge shared by exactly two faces (got %d)", len(faces))
	}
	a, b := faces[0].Geometry(), faces[1].Geometry()
	curve := edge.Geometry()
	lo, hi := curve.Domain()
	at := func(t float64) math.Point3 { return curve.PointAt(lo + (hi-lo)*t) }
	ea := func(t float64) (u, v float64) { return a.ParamAt(at(t)) }
	eb := func(t float64) (u, v float64) { return b.ParamAt(at(t)) }
	rep := analysis.CrossEdgeContinuity(a, b, ea, eb, continuityCheckSamples)
	return rep, continuityOverlay(rep, at), nil
}

// continuityOverlay draws the edge as a polyline coloured green (continuous) → red (discontinuous)
// by each sample's worst normalized deviation.
func continuityOverlay(rep analysis.ContinuityReport, at func(float64) math.Point3) []renderer.DrawItem {
	n := len(rep.Samples)
	if n < 2 {
		return nil
	}
	pos := make([]math.Point3, n)
	col := make([][4]float32, n)
	for i, s := range rep.Samples {
		pos[i] = at(s.T)
		col[i] = severityColor(continuitySeverity(s))
	}
	idx := make([]int, 0, (n-1)*2)
	for i := 0; i < n-1; i++ {
		idx = append(idx, i, i+1)
	}
	return []renderer.DrawItem{{Primitive: renderer.Lines, Positions: pos, Indices: idx, Colors: col}}
}

// continuitySeverity maps one sample's deviations to [0,1] — the worst of a 0.1mm gap, a 1° normal
// break, or a 50% curvature jump reads as fully discontinuous.
func continuitySeverity(s analysis.ContinuitySample) float64 {
	return math.Clamp01(max(s.Gap/0.1, s.NormalDeg/1.0, s.CurvPct/50.0))
}

// severityColor blends green (0, continuous) to red (1, discontinuous).
func severityColor(sev float64) [4]float32 {
	return [4]float32{float32(0.1 + 0.85*sev), float32(0.7 * (1 - sev)), 0.05, 1}
}

// continuitySummary is the one-line Command Window report of the aggregate deviations.
func continuitySummary(rep analysis.ContinuityReport) string {
	return fmt.Sprintf("Continuity — G0 gap max %.4g (avg %.4g), G1 angle max %.3f° (avg %.3f°), G2 curvature max %.1f%% (avg %.1f%%)",
		rep.MaxGap, rep.AvgGap, rep.MaxNormalDeg, rep.AvgNormalDeg, rep.MaxCurvPct, rep.AvgCurvPct)
}
