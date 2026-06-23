// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/renderer"
)

// Surface Analysis (M36-F12) is a live interrogation overlay: while the tool is active, every
// visible body's surface is drawn with reflection / highlight / isophote lines so a styling
// reviewer can judge fairness and continuity — a curvature discontinuity (a G1-only seam) bends
// the lines, a G2 seam keeps them smooth. It is a display mode, not an edit: it carries no clicks
// and its dialog Params are the controls (mode, line count, light/stripe direction).

const (
	interrogIsophote = iota
	interrogReflection
	interrogHighlight
)

// interrogationTolerance is the chord tolerance the surface is tessellated at for interrogation —
// fine enough for crisp lines, matching the client-graphics overlay resolver.
const interrogationTolerance = 0.01

// interrogationColor is the near-black line colour the interrogation curves draw in (high contrast
// on the default light-grey shaded surface).
var interrogationColor = [4]float32{0.05, 0.05, 0.08, 1}

// SurfaceInterrogationTool shows the live reflection/highlight/isophote overlay while active.
type SurfaceInterrogationTool struct {
	dialogTool
	mode             int
	count            int
	dirX, dirY, dirZ float64
}

// NewSurfaceInterrogationTool returns the tool defaulting to 12 isophote lines from an over-the-
// shoulder light direction.
func NewSurfaceInterrogationTool() *SurfaceInterrogationTool {
	return &SurfaceInterrogationTool{mode: interrogIsophote, count: 12, dirX: 0.3, dirY: 0.3, dirZ: 1}
}

// Name implements [Tool].
func (t *SurfaceInterrogationTool) Name() string { return "Surface Analysis" }

// Prompt guides the input.
func (t *SurfaceInterrogationTool) Prompt(*Session) string {
	return "Surface analysis overlay — choose the line family, count and direction; close to dismiss."
}

// CanCommit is false — it is a display overlay, not an edit.
func (t *SurfaceInterrogationTool) CanCommit() bool { return false }

// Commit is a no-op (display-only).
func (t *SurfaceInterrogationTool) Commit(*Session) error { return nil }

// Params exposes the interrogation controls for the generic dialog.
func (t *SurfaceInterrogationTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{{Label: "Line Count", Get: func() int { return t.count }, Set: func(v int) { t.count = v }}},
		Floats: []FloatParam{
			{Label: "Direction X", Get: func() float64 { return t.dirX }, Set: func(v float64) { t.dirX = v }},
			{Label: "Direction Y", Get: func() float64 { return t.dirY }, Set: func(v float64) { t.dirY = v }},
			{Label: "Direction Z", Get: func() float64 { return t.dirZ }, Set: func(v float64) { t.dirZ = v }},
		},
		Choices: []ChoiceParam{
			{Label: "Lines", Options: []string{"Isophotes", "Reflection", "Highlight"},
				Get: func() int { return t.mode }, Set: func(v int) { t.mode = v }},
		},
	}
}

// activeInterrogationTool returns the active Surface Analysis tool, if any.
func (s *Session) activeInterrogationTool() (*SurfaceInterrogationTool, bool) {
	if s.tool == nil {
		return nil, false
	}
	t, ok := s.tool.tool.(*SurfaceInterrogationTool)
	return t, ok
}

// SurfaceInterrogationItems returns the live interrogation line overlay for every visible body, or
// nil when the Surface Analysis tool is not active. The head appends it to the viewport draw list.
func (s *Session) SurfaceInterrogationItems() []renderer.DrawItem {
	t, ok := s.activeInterrogationTool()
	if !ok {
		return nil
	}
	dir := math.V3(math.Scalar(t.dirX), math.Scalar(t.dirY), math.Scalar(t.dirZ))
	eye := s.Camera().Eye
	var segs []analysis.Segment3
	for _, b := range s.VisibleBodies() {
		m := bodyInterrogationSamples(s, b)
		if len(m.Triangles) == 0 {
			continue
		}
		segs = append(segs, interrogationLines(t, m, eye, dir)...)
	}
	if len(segs) == 0 {
		return nil
	}
	return []renderer.DrawItem{interrogationDrawItem(segs)}
}

// bodyInterrogationSamples tessellates a body and packs it as analysis samples (positions, normals,
// triangle triples).
func bodyInterrogationSamples(s *Session, b *topo.Body) analysis.SurfaceSamples {
	fs := s.FacetStore().CalculateFacets(b, interrogationTolerance)
	return analysis.SurfaceSamples{
		Positions: fs.Mesh.Positions,
		Normals:   fs.Mesh.Normals,
		Triangles: trianglesFromIndices(fs.Mesh.Indices),
	}
}

// interrogationLines dispatches to the chosen interrogation family.
func interrogationLines(t *SurfaceInterrogationTool, m analysis.SurfaceSamples, eye math.Point3, dir math.Vector3) []analysis.Segment3 {
	switch t.mode {
	case interrogReflection:
		return analysis.ReflectionLines(m, eye, dir, t.count)
	case interrogHighlight:
		return analysis.HighlightLines(m, eye, dir, t.count)
	default:
		return analysis.Isophotes(m, dir, t.count)
	}
}

// trianglesFromIndices regroups a flat triangle-index list into index triples.
func trianglesFromIndices(idx []int) [][3]int {
	out := make([][3]int, 0, len(idx)/3)
	for i := 0; i+2 < len(idx); i += 3 {
		out = append(out, [3]int{idx[i], idx[i+1], idx[i+2]})
	}
	return out
}

// interrogationDrawItem flattens interrogation segments into one line draw item.
func interrogationDrawItem(segs []analysis.Segment3) renderer.DrawItem {
	pos := make([]math.Point3, 0, len(segs)*2)
	idx := make([]int, 0, len(segs)*2)
	for i, sg := range segs {
		pos = append(pos, sg.A, sg.B)
		idx = append(idx, 2*i, 2*i+1)
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: interrogationColor}
}
