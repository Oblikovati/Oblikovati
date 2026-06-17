// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// Live in-canvas feature preview (Inventor's "the feature previews the expected results
// before you commit"). A tool that can build the feature it would commit evaluates it
// non-destructively (feature.PartFeatures.PreviewResult) and the session highlights just the
// faces the feature introduces, translucent — GREEN when the feature adds material, RED when
// it removes it.

// DraftPreviewable is an optional Tool capability: a feature tool that can build the
// unattached feature it would commit, so the session can evaluate it speculatively and show
// a translucent solid result preview. It supersedes the wireframe-only Previewable for tools
// that implement it (RenderFrame prefers the solid preview, falling back to the wireframe).
type DraftPreviewable interface {
	DraftFeature(s *Session) (feature.Feature, bool)
}

// Every part-feature tool drives the live preview through DraftPreviewable; these assertions
// pin the wiring so a tool that loses its DraftFeature method fails to compile.
var (
	_ DraftPreviewable = (*ExtrudeTool)(nil)
	_ DraftPreviewable = (*RevolveTool)(nil)
	_ DraftPreviewable = (*SweepTool)(nil)
	_ DraftPreviewable = (*LoftTool)(nil)
	_ DraftPreviewable = (*CoilTool)(nil)
	_ DraftPreviewable = (*HoleTool)(nil)
	_ DraftPreviewable = (*FilletTool)(nil)
	_ DraftPreviewable = (*ChamferTool)(nil)
	_ DraftPreviewable = (*ShellTool)(nil)
	_ DraftPreviewable = (*DraftTool)(nil)
	_ DraftPreviewable = (*ThreadTool)(nil)
	_ DraftPreviewable = (*RibTool)(nil)
	_ DraftPreviewable = (*EmbossTool)(nil)
	_ DraftPreviewable = (*ThickenTool)(nil)
	_ DraftPreviewable = (*SplitTool)(nil)
	_ DraftPreviewable = (*GrillTool)(nil)
	_ DraftPreviewable = (*CoreCavityTool)(nil)
	_ DraftPreviewable = (*FaceOffsetTool)(nil)
	_ DraftPreviewable = (*ReplaceFaceTool)(nil)
	_ DraftPreviewable = (*DeleteFaceTool)(nil)
	_ DraftPreviewable = (*PatchTool)(nil)
	_ DraftPreviewable = (*StitchTool)(nil)
	_ DraftPreviewable = (*SurfaceTrimTool)(nil)
	_ DraftPreviewable = (*SculptTool)(nil)
	_ DraftPreviewable = (*ExtendTool)(nil)
	_ DraftPreviewable = (*SheetMetalFaceTool)(nil)
	_ DraftPreviewable = (*SheetMetalFlangeTool)(nil)
	_ DraftPreviewable = (*SheetMetalLipTool)(nil)
	_ DraftPreviewable = (*SheetMetalRipTool)(nil)
	_ DraftPreviewable = (*SheetMetalPunchTool)(nil)
)

// previewAddColor / previewRemoveColor are the operation-coded preview tints: a feature that
// adds volume previews green, one that removes volume previews red. previewOpacity keeps the
// underlying model visible through the preview.
var (
	previewAddColor    = [4]float32{0.30, 0.85, 0.35, 1}
	previewRemoveColor = [4]float32{0.90, 0.30, 0.30, 1}
)

// previewOpacity keeps the ghost light so the model reads through it; the matching opaque
// edge lines give the volume a crisp outline (Inventor draws the preview's edges).
const previewOpacity = 0.25

// featurePreviewItems evaluates a tool's draft feature against the current model without
// committing it and returns translucent triangle items over the feature's new faces — green
// when it adds material, red when it removes it. Empty when the draft is not ready, the
// active document is not a part, or the speculative build fails (a sick preview shows nothing
// rather than partial geometry).
func (s *Session) featurePreviewItems(t DraftPreviewable) []renderer.DrawItem {
	draft, ok := t.DraftFeature(s)
	if !ok {
		return nil
	}
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	base := s.VisibleBodies()
	// Evaluating the draft both produces the result bodies and, for an operational feature,
	// populates its tool body (built before the boolean, so it survives a failing op).
	result, err := part.Features().PreviewResult(draft)
	// Prefer the feature's tool body (the swept/drilled solid it contributes): rendering it
	// whole and translucent shows the pending feature's volume as a ghost colored by its
	// operation — crucially for a CUT, whose volume is interior and would be hidden.
	if items := toolBodyPreview(draft); items != nil {
		return items
	}
	// Dress-ups (fillet/chamfer/shell/…) have no separable tool body: show the solid volume
	// they add or remove, computed as the boolean difference between the result and the base.
	if err != nil || len(result) == 0 {
		return nil
	}
	return deltaPreviewItems(base, result)
}

// faceKeys returns the reference keys of picked faces — the input the modify features
// (delete/replace/offset face) consume, shared by their tools' commit and preview builders.
func faceKeys(faces []FaceHandle) [][]byte {
	keys := make([][]byte, len(faces))
	for i, f := range faces {
		keys[i] = f.Face.ReferenceKey()
	}
	return keys
}

// draftFromScratch builds a tool's feature into a throwaway engine — so the part's real
// feature program is untouched — and returns the underlying Feature value for the preview to
// evaluate non-destructively. build runs the tool's exact commit-time construction (every
// variant) against the scratch engine, so the previewed feature is identical to what OK
// creates with no duplicated builder. The scratch engine needs no params/keys: Add only stores
// and names the feature; resolution happens later in PreviewResult against the part's engine.
func draftFromScratch(build func(*feature.PartFeatures) (*feature.PartFeature, error)) (feature.Feature, bool) {
	pf, err := build(feature.NewPartFeatures(nil, nil))
	if err != nil || pf == nil {
		return nil, false
	}
	return pf.Definition(), true
}

// toolBodyPreview tessellates an operational feature's tool body — the solid it sweeps/drills,
// populated by the preview recompute — as a translucent ghost, GREEN when the operation adds
// material (new body / join) and RED when it removes it (cut / intersect). Nil when the
// feature exposes no tool body (e.g. a dress-up feature).
func toolBodyPreview(draft feature.Feature) []renderer.DrawItem {
	tf, ok := draft.(feature.ToolFeature)
	if !ok || tf.ToolBody() == nil {
		return nil
	}
	return bodyPreviewItems(tf.ToolBody(), operationColor(tf.Operation()))
}

// operationColor maps a boolean operation to the preview tint: material-adding operations
// (new body / join) preview green, material-removing ones (cut / intersect) preview red.
func operationColor(op ops.PartFeatureOperation) [4]float32 {
	if op == ops.NewBody || op == ops.Join {
		return previewAddColor
	}
	return previewRemoveColor
}

// bodyPreviewItems renders a preview solid as a translucent ghost (one triangle item per
// face) plus its edges as opaque lines in the same hue — the outline makes the light
// translucent volume legible, the way Inventor draws a feature preview.
func bodyPreviewItems(b *topo.Body, color [4]float32) []renderer.DrawItem {
	q := ops.DefaultQuality()
	mesh, _ := ops.TessellateBody(b, q)
	if mesh == nil || mesh.TriangleCount() == 0 {
		return nil
	}
	// Smooth the normals across facets meeting below the crease angle so a curved delta (a
	// fillet's round wedge) reads as one surface instead of flat-shaded stripes.
	mesh.Normals = ops.SmoothShadeNormals(mesh, ops.DefaultCreaseAngle())
	items := []renderer.DrawItem{previewBodyFill(mesh, color)}
	if line := previewEdgeLines(b, q, color); line != nil {
		items = append(items, *line)
	}
	return items
}

// previewEdgeLines draws the body's CREASE edges (tangent/facet seams suppressed, like the
// model renderer) as one opaque line item in the preview hue — so a curved delta shows only
// its real boundary outline, not a web of tessellation seams.
func previewEdgeLines(b *topo.Body, q ops.Quality, color [4]float32) *renderer.DrawItem {
	polylines := ops.VisibleEdges(b, q, ops.DefaultCreaseAngle())
	var pts []math.Point3
	var idx []int
	for _, pl := range polylines {
		base := len(pts)
		pts = append(pts, pl...)
		for i := 0; i+1 < len(pl); i++ {
			idx = append(idx, base+i, base+i+1)
		}
	}
	if len(idx) == 0 {
		return nil
	}
	return &renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: pts,
		Indices:   idx,
		Color:     [4]float32{color[0], color[1], color[2], 1}, // same hue, fully opaque
		OnTop:     true,
	}
}

// totalVolume sums the enclosed volume (cm³) of the solid bodies; open surface bodies add 0.
func totalVolume(bodies []*topo.Body) float64 {
	q := ops.DefaultQuality()
	var v float64
	for _, b := range bodies {
		if b.IsSolid() {
			v += ops.BodyGeometryProperties(b, q).Volume
		}
	}
	return v
}

// deltaPreviewItems renders the solid volume a dress-up feature adds or removes — the boolean
// difference between the base and the speculative result — as a translucent ghost: RED for the
// material removed (base − result, e.g. a fillet's corner wedge, a shell's hollowed interior),
// GREEN for material added (result − base, e.g. an outward draft). This is Inventor's solid
// feature preview: the changed VOLUME, not just the changed faces. Nil when neither difference
// is a solid (a sick or no-op preview shows nothing). Dress-ups act on one body; multi-body
// states are skipped.
func deltaPreviewItems(base, result []*topo.Body) []renderer.DrawItem {
	b, r := singleSolid(base), singleSolid(result)
	if b == nil || r == nil {
		return nil
	}
	var items []renderer.DrawItem
	if removed := solidDifference(b, r); removed != nil {
		items = append(items, bodyPreviewItems(removed, previewRemoveColor)...)
	}
	if added := solidDifference(r, b); added != nil {
		items = append(items, bodyPreviewItems(added, previewAddColor)...)
	}
	return items
}

// singleSolid returns the lone solid body in bodies, or nil when there isn't exactly one.
func singleSolid(bodies []*topo.Body) *topo.Body {
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		return nil
	}
	return bodies[0]
}

// solidDifference returns target − tool as a solid body, or nil when the difference is empty
// or the boolean fails (preview must never show partial/garbage geometry).
func solidDifference(target, tool *topo.Body) *topo.Body {
	diff, err := ops.Boolean(ops.Cut, target, tool)
	if err != nil || diff == nil || len(diff.Faces()) == 0 || !diff.IsSolid() {
		return nil
	}
	if ops.BodyGeometryProperties(diff, ops.DefaultQuality()).Volume < 1e-9 {
		return nil
	}
	return diff
}

// previewBodyFill wraps a body mesh as a translucent, on-top triangle item — drawn ignoring
// the depth test so the preview ghost is always visible over the model, even where it sits
// inside or behind existing material (a cut's volume is interior, a join boss may be occluded).
// The translucency rides in the COLOR's alpha (the viewport flattens vertex color, not the
// Opacity field; see head/viewport/flatten.go), so we bake previewOpacity in.
func previewBodyFill(m *ops.Mesh, color [4]float32) renderer.DrawItem {
	fill := [4]float32{color[0], color[1], color[2], previewOpacity}
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: m.Positions,
		Normals:   m.Normals,
		Indices:   m.Indices,
		Color:     fill,
		Opacity:   previewOpacity,
		Shading:   renderer.ShadeFlat,
		OnTop:     true,
	}
}
