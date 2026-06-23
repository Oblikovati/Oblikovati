// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/analysis"
	"oblikovati.org/renderer"
)

// The Surface Deviation tool (M36-F14) maps the signed distance between the last two surface bodies:
// it reports min/max/RMS and the out-of-tolerance count to the Command Window, and draws a colour
// deviation heatmap over the sampled surface (blue under, green in-tolerance, red over). A display
// tool (no commit) — the Class-A "how far is this panel from its master / its neighbour" check.

// deviationGrid is the sampling resolution of the deviation map per surface direction.
const deviationGrid = 32

// SurfaceDeviationTool compares the last two surface bodies and overlays their deviation.
type SurfaceDeviationTool struct {
	tolerance float64 // out-of-tolerance band (database units; default 0.01 cm = 0.1 mm)
	report    *analysis.DeviationReport
	overlay   []renderer.DrawItem
}

// NewSurfaceDeviationTool returns the deviation tool with a 0.1 mm default tolerance.
func NewSurfaceDeviationTool() *SurfaceDeviationTool { return &SurfaceDeviationTool{tolerance: 0.01} }

// Name implements [Tool].
func (t *SurfaceDeviationTool) Name() string { return "Surface Deviation" }

// Start computes the deviation between the last two surface bodies and reports its summary.
func (t *SurfaceDeviationTool) Start(s *Session) { t.recompute(s) }

// AcceptedKinds: the tool acts on the running bodies, not a pick.
func (t *SurfaceDeviationTool) AcceptedKinds() []SelectionKind { return nil }

// Pick is a no-op — the deviation map reads the running bodies, not a selection.
func (t *SurfaceDeviationTool) Pick(*Session, Selectable) {}

// Cancel clears the overlay.
func (t *SurfaceDeviationTool) Cancel(*Session) { t.overlay = nil }

// Prompt guides the user.
func (t *SurfaceDeviationTool) Prompt(*Session) string {
	return "Surface Deviation: maps the last two surfaces' gap. Set the tolerance; the map updates."
}

// Params exposes the tolerance (re-mapped on change).
func (t *SurfaceDeviationTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{
		Label: "Tolerance", Get: func() float64 { return t.tolerance },
		Set: func(v float64) { t.tolerance = v },
	}}}
}

// CanCommit is false — a deviation map is a read-only overlay.
func (t *SurfaceDeviationTool) CanCommit() bool { return false }

// Commit is a no-op (display tool).
func (t *SurfaceDeviationTool) Commit(*Session) error { return nil }

// Report returns the last computed deviation report (for inspection/tests).
func (t *SurfaceDeviationTool) Report() *analysis.DeviationReport { return t.report }

// recompute samples the last two surface bodies' deviation, reports the summary, builds the overlay.
func (t *SurfaceDeviationTool) recompute(s *Session) {
	src, target, ok := lastTwoNurbsSurfaces(s)
	if !ok {
		s.SetNotice("Surface Deviation needs two surface bodies")
		return
	}
	r := analysis.SurfaceDeviationToSurface(src, target, deviationGrid, deviationGrid)
	t.report = &r
	t.overlay = deviationOverlay(r, t.tolerance)
	s.feedNotice(fmt.Sprintf("Deviation: min %.4g, max %.4g, RMS %.4g, |max| %.4g; %d/%d samples out of ±%.4g",
		r.Min, r.Max, r.RMS, r.AbsMax, r.OutOfTolerance(t.tolerance), len(r.Samples), t.tolerance))
}

// DeviationItems returns the active deviation overlay (empty when the tool is not running).
func (s *Session) DeviationItems() []renderer.DrawItem {
	ti := s.ActiveTool()
	if ti == nil {
		return nil
	}
	t, ok := ti.Tool().(*SurfaceDeviationTool)
	if !ok || t == nil {
		return nil
	}
	return t.overlay
}

// lastTwoNurbsSurfaces returns the NURBS surfaces of the active part's last two surface bodies
// (src = last, target = second-to-last).
func lastTwoNurbsSurfaces(s *Session) (src, target geom.Surface, ok bool) {
	part, err := activePart(s)
	if err != nil {
		return nil, nil, false
	}
	bodies := part.SurfaceBodies()
	if bodies.Count() < 2 {
		return nil, nil, false
	}
	sb, sok := nurbsFaceSurface(bodies.Item(bodies.Count() - 1))
	tb, tok := nurbsFaceSurface(bodies.Item(bodies.Count() - 2))
	if !sok || !tok {
		return nil, nil, false
	}
	return sb, tb, true
}

// deviationOverlay builds a colour-mapped triangle heatmap over the deviation grid: each sample is a
// vertex coloured by its signed distance (green within ±tolerance, ramping to red over / blue under),
// the grid quads triangulated. Samples are row-major (i over u, j over v).
func deviationOverlay(r analysis.DeviationReport, tol float64) []renderer.DrawItem {
	if len(r.Samples) != r.UCount*r.VCount || r.UCount < 2 || r.VCount < 2 {
		return nil
	}
	item := renderer.DrawItem{Primitive: renderer.Triangles, Opacity: 0.9}
	for _, sm := range r.Samples {
		item.Positions = append(item.Positions, sm.Point)
		item.Colors = append(item.Colors, deviationColor(sm.Distance, tol, r.AbsMax))
	}
	at := func(i, j int) int { return i*r.VCount + j }
	for i := 0; i+1 < r.UCount; i++ {
		for j := 0; j+1 < r.VCount; j++ {
			a, b, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			item.Indices = append(item.Indices, a, b, c, a, c, d)
		}
	}
	return []renderer.DrawItem{item}
}

// deviationColor maps a signed distance to a heatmap colour: green inside ±tol, ramping to red
// (over) / blue (under) by magnitude toward absMax.
func deviationColor(d, tol, absMax float64) [4]float32 {
	if absMax <= 0 || d <= tol && d >= -tol {
		return [4]float32{0.1, 0.85, 0.2, 1} // in tolerance: green
	}
	f := float32(0.3)
	if absMax > tol {
		over := (absAbs(d) - tol) / (absMax - tol)
		f = 0.3 + 0.7*float32(clamp01dev(over))
	}
	if d > 0 {
		return [4]float32{f, 1 - f, 0.1, 1} // over: toward red
	}
	return [4]float32{0.1, 1 - f, f, 1} // under: toward blue
}

func absAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func clamp01dev(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
