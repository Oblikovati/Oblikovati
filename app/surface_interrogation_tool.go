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
	interrogZebra = iota // filled black/white reflection bands (the zebra map)
	interrogIsophote
	interrogReflection
	interrogHighlight
)

// interrogModeLabels label the interrogation families (index = mode constant).
var interrogModeLabels = []string{"Zebra", "Isophotes", "Reflection", "Highlight"}

// interrogationTolerance is the chord tolerance the surface is tessellated at for interrogation —
// fine enough for crisp lines/bands, matching the client-graphics overlay resolver.
const interrogationTolerance = 0.01

// interrogationColor is the near-black line colour the interrogation curves draw in (high contrast
// on the default light-grey shaded surface).
var interrogationColor = [4]float32{0.05, 0.05, 0.08, 1}

// zebraDark and zebraLight are the two alternating zebra-band colours (unlit, drawn with per-vertex
// colour over the surface), matching the showroom black/white stripe environment.
var (
	zebraDark  = [4]float32{0.03, 0.03, 0.03, 1}
	zebraLight = [4]float32{0.97, 0.97, 0.97, 1}
)

// SurfaceInterrogationTool shows the live reflection/highlight/isophote overlay while active.
type SurfaceInterrogationTool struct {
	dialogTool
	mode             int
	count            int
	dirX, dirY, dirZ float64
}

// NewSurfaceInterrogationTool returns the tool defaulting to the zebra map with 16 bands from an
// over-the-shoulder stripe direction.
func NewSurfaceInterrogationTool() *SurfaceInterrogationTool {
	return NewSurfaceInterrogationToolMode(interrogZebra)
}

// NewSurfaceInterrogationToolMode returns the interrogation tool preset to one family — the per-mode
// ribbon buttons (Zebra / Isophotes / Reflection / Highlight) each launch it this way.
func NewSurfaceInterrogationToolMode(mode int) *SurfaceInterrogationTool {
	return &SurfaceInterrogationTool{mode: mode, count: 16, dirX: 0.3, dirY: 0.3, dirZ: 1}
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
		Ints: []IntParam{{Label: "Band/Line Count", Get: func() int { return t.count }, Set: func(v int) { t.count = v }}},
		Floats: []FloatParam{
			{Label: "Direction X", Get: func() float64 { return t.dirX }, Set: func(v float64) { t.dirX = v }},
			{Label: "Direction Y", Get: func() float64 { return t.dirY }, Set: func(v float64) { t.dirY = v }},
			{Label: "Direction Z", Get: func() float64 { return t.dirZ }, Set: func(v float64) { t.dirZ = v }},
		},
		Choices: []ChoiceParam{
			{Label: "Mode", Options: interrogModeLabels,
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
	if t.mode == interrogZebra {
		cam := s.Camera()
		return s.zebraItems(t, cam.Eye.VectorTo(cam.Target), dir)
	}
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

// zebraItems builds the filled zebra-band overlay: every visible body's surface is drawn as solid
// triangles coloured by their reflection band (black/white), so the band edges reveal continuity.
// viewDir is the (constant) camera view direction reflected to stripe the surface.
func (s *Session) zebraItems(t *SurfaceInterrogationTool, viewDir math.Vector3, dir math.Vector3) []renderer.DrawItem {
	var items []renderer.DrawItem
	for _, b := range s.VisibleBodies() {
		m := bodyInterrogationSamples(s, b)
		if len(m.Triangles) == 0 {
			continue
		}
		if item, ok := zebraDrawItem(m, analysis.ZebraTriangleBands(m, viewDir, dir, t.count)); ok {
			items = append(items, item)
		}
	}
	return items
}

// zebraOffsetFactor lifts the zebra mesh off the surface by this fraction of the body's bounding-box
// diagonal — a decal offset that places the bands just in front of the shaded body so they win the
// depth test cleanly (no z-fighting speckle), while being far too small to alter the silhouette.
const zebraOffsetFactor = 1e-3

// zebraDrawItem builds one unlit, flat-shaded triangle item: each triangle is emitted with its own
// three vertices coloured by its band parity, so the bands stay crisp (no per-vertex blending). The
// vertices are nudged out along their normals by a tiny decal offset so the bands sit cleanly over
// the shaded body instead of z-fighting it.
func zebraDrawItem(m analysis.SurfaceSamples, bands []bool) (renderer.DrawItem, bool) {
	off := zebraOffsetFactor * meshDiagonal(m.Positions)
	pos := make([]math.Point3, 0, len(m.Triangles)*3)
	col := make([][4]float32, 0, len(m.Triangles)*3)
	idx := make([]int, 0, len(m.Triangles)*3)
	for ti, tri := range m.Triangles {
		c := zebraLight
		if bands[ti] {
			c = zebraDark
		}
		for _, v := range tri {
			idx = append(idx, len(pos))
			pos = append(pos, liftAlongNormal(m.Positions[v], m.Normals[v], off))
			col = append(col, c)
		}
	}
	if len(pos) == 0 {
		return renderer.DrawItem{}, false
	}
	return renderer.DrawItem{Primitive: renderer.Triangles, Positions: pos, Indices: idx, Colors: col, Opacity: 1}, true
}

// liftAlongNormal returns p moved by off along the unit-normalized n (no move for a zero normal).
func liftAlongNormal(p math.Point3, n math.Vector3, off float64) math.Point3 {
	l := float64(n.Length())
	if l < 1e-12 {
		return p
	}
	return p.TranslateBy(n.Scale(math.Scalar(off / l)))
}

// meshDiagonal returns the bounding-box diagonal length of the points (0 for an empty mesh).
func meshDiagonal(pts []math.Point3) float64 {
	if len(pts) == 0 {
		return 0
	}
	lo, hi := pts[0], pts[0]
	for _, p := range pts {
		lo = math.P3(min(lo.X, p.X), min(lo.Y, p.Y), min(lo.Z, p.Z))
		hi = math.P3(max(hi.X, p.X), max(hi.Y, p.Y), max(hi.Z, p.Z))
	}
	return float64(lo.DistanceTo(hi))
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
