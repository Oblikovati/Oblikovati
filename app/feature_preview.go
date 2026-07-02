// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
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

// Every part-feature tool drives the live preview and the sick-config commit gate
// through DraftPreviewable. The per-type assertion list that used to pin this is
// gone (#1626): every activation site now goes through Session.StartFeatureTool,
// whose PartFeatureTool parameter makes a tool that loses its DraftFeature method
// fail to compile at its command registration — the interface subsumes the list.

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
	if items := deltaPreviewItems(base, result); items != nil {
		return items
	}
	// Fallback for features with no clean solid delta (surface features, multi-body splits):
	// highlight the faces the feature introduced.
	if items := changedFacePreview(base, result); items != nil {
		return items
	}
	// Last resort for features that retrim/weld/fill existing surfaces without introducing a new
	// surface or volume delta (delete/replace face, stitch, trim, sculpt, extend): show the
	// resulting body itself, so the preview always shows the outcome.
	return resultBodyPreview(base, result)
}

// resultBodyPreview renders every result body as a translucent ghost — the fallback when there
// is no localizable delta. Green when the model grew, red when it shrank.
func resultBodyPreview(base, result []*topo.Body) []renderer.DrawItem {
	color := previewAddColor
	if totalVolume(result) < totalVolume(base) {
		color = previewRemoveColor
	}
	var items []renderer.DrawItem
	for _, b := range result {
		items = append(items, bodyPreviewItems(b, color)...)
	}
	return items
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
	pf, err := build(feature.NewPartFeatures(nil))
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

// changedFacePreview highlights the faces a feature introduced — result faces whose underlying
// surface is absent from the base — as one translucent mesh plus their feature edges, colored
// green when the model grew and red when it shrank. It is the fallback when a feature exposes
// no tool body and produces no clean solid delta (surface features, multi-body splits, and
// coincident-face edits), so every feature shows *something* meaningful.
func changedFacePreview(base, result []*topo.Body) []renderer.DrawItem {
	mesh, newFaces := collectNewFaces(base, result)
	if mesh.TriangleCount() == 0 {
		return nil
	}
	color := previewAddColor
	if totalVolume(result) < totalVolume(base) {
		color = previewRemoveColor
	}
	mesh.Normals = ops.SmoothShadeNormals(mesh, ops.DefaultCreaseAngle())
	items := []renderer.DrawItem{previewBodyFill(mesh, color)}
	if line := changedFaceEdges(result, newFaces, color); line != nil {
		items = append(items, *line)
	}
	return items
}

// collectNewFaces tessellates every result face whose surface is absent from the base into one
// merged mesh, and returns the set of those new faces (for edge selection).
func collectNewFaces(base, result []*topo.Body) (*ops.Mesh, map[*topo.Face]bool) {
	baseSurf := faceSurfacesOf(base)
	q := ops.DefaultQuality()
	mesh := &ops.Mesh{}
	newFaces := map[*topo.Face]bool{}
	for _, b := range result {
		for _, f := range b.Faces() {
			if surfaceInBase(baseSurf, f) {
				continue
			}
			if m := ops.TessellateFace(f, q); m != nil && m.TriangleCount() > 0 {
				newFaces[f] = true
				appendMesh(mesh, m)
			}
		}
	}
	return mesh, newFaces
}

// changedFaceEdges draws the feature edges that bound a new face (tangent seams suppressed) as
// one opaque line item in the preview hue.
func changedFaceEdges(result []*topo.Body, newFaces map[*topo.Face]bool, color [4]float32) *renderer.DrawItem {
	q := ops.DefaultQuality()
	var pts []math.Point3
	var idx []int
	for _, b := range result {
		for _, e := range b.Edges() {
			if !edgeTouchesNewFace(e, newFaces) || isTangentEdge(e) {
				continue
			}
			pl := ops.TessellateEdge(e, q)
			base := len(pts)
			pts = append(pts, pl...)
			for i := 0; i+1 < len(pl); i++ {
				idx = append(idx, base+i, base+i+1)
			}
		}
	}
	if len(idx) == 0 {
		return nil
	}
	return &renderer.DrawItem{Primitive: renderer.Lines, Positions: pts, Indices: idx, Color: [4]float32{color[0], color[1], color[2], 1}, OnTop: true}
}

// edgeTouchesNewFace reports whether any face adjacent to e is one the feature introduced.
func edgeTouchesNewFace(e *topo.Edge, newFaces map[*topo.Face]bool) bool {
	for _, f := range e.Faces() {
		if newFaces[f] {
			return true
		}
	}
	return false
}

// isTangentEdge reports whether an edge's two faces meet smoothly (within the crease angle), so
// it should not be drawn as a feature edge — mirrors ops.VisibleEdges' tangent suppression.
func isTangentEdge(e *topo.Edge) bool {
	faces := e.Faces()
	if len(faces) != 2 {
		return false
	}
	mid := e.Geometry().PointAt(0.5)
	cos := math.Scalar(stdmath.Cos(ops.DefaultCreaseAngle()))
	return faceNormalAt(faces[0], mid).Dot(faceNormalAt(faces[1], mid)) > cos
}

// faceNormalAt returns a face's outward unit normal at point p.
func faceNormalAt(f *topo.Face, p math.Point3) math.Vector3 {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	return n.AsUnit().AsVector()
}

// appendMesh appends src into dst, offsetting indices (a local mesh merge).
func appendMesh(dst, src *ops.Mesh) {
	b := len(dst.Positions)
	dst.Positions = append(dst.Positions, src.Positions...)
	dst.Normals = append(dst.Normals, src.Normals...)
	for _, i := range src.Indices {
		dst.Indices = append(dst.Indices, b+i)
	}
}

// faceSig fingerprints a face's underlying surface, trim-independently, so an unchanged face is
// recognised across a rebuild while a feature's new face is not. Analytic surfaces use their
// intrinsic parameters; an unrecognised surface falls back to its area-weighted centroid.
type faceSig struct {
	kind     int
	dir      math.Vector3
	anchor   math.Point3
	r1, r2   float64
	centroid math.Point3
	area     float64
}

func surfaceSigOf(f *topo.Face) faceSig {
	switch g := f.Geometry().(type) {
	case geom.Plane:
		n := g.UAxis.AsVector().Cross(g.VAxis.AsVector()).AsUnit().AsVector()
		return faceSig{kind: 1, dir: signNorm(n), anchor: math.P3(0, 0, 0).TranslateBy(n.Scale(n.Dot(g.Origin.AsVector())))}
	case geom.Cylinder:
		return faceSig{kind: 2, dir: signNorm(g.AxisDir.AsVector()), anchor: axisFoot(g.Origin, g.AxisDir.AsVector()), r1: g.Radius}
	case geom.Cone:
		return faceSig{kind: 3, dir: signNorm(g.AxisDir.AsVector()), anchor: g.Apex, r1: g.HalfAngle}
	case geom.Sphere:
		return faceSig{kind: 4, anchor: g.Center, r1: g.Radius}
	case geom.Torus:
		return faceSig{kind: 5, dir: signNorm(g.AxisDir.AsVector()), anchor: g.Center, r1: g.MajorRadius, r2: g.MinorRadius}
	default:
		c, a := faceAreaCentroid(f)
		return faceSig{kind: 0, centroid: c, area: a}
	}
}

func faceSurfacesOf(bodies []*topo.Body) []faceSig {
	var out []faceSig
	for _, b := range bodies {
		for _, f := range b.Faces() {
			out = append(out, surfaceSigOf(f))
		}
	}
	return out
}

func surfaceInBase(base []faceSig, f *topo.Face) bool {
	s := surfaceSigOf(f)
	for _, b := range base {
		if sigEqual(b, s) {
			return true
		}
	}
	return false
}

func sigEqual(a, b faceSig) bool {
	if a.kind != b.kind {
		return false
	}
	if a.kind == 0 {
		return a.centroid.IsEqualTo(b.centroid, 1e-4) && stdmath.Abs(a.area-b.area) <= 1e-3*max(a.area, b.area)+1e-9
	}
	return a.dir.IsEqualTo(b.dir, 1e-4) && a.anchor.IsEqualTo(b.anchor, 1e-4) && stdmath.Abs(a.r1-b.r1) <= 1e-4 && stdmath.Abs(a.r2-b.r2) <= 1e-4
}

// signNorm flips v so its dominant component is positive (a surface and its reverse share a sig).
func signNorm(v math.Vector3) math.Vector3 {
	ax, ay, az := stdmath.Abs(v.X), stdmath.Abs(v.Y), stdmath.Abs(v.Z)
	if (ax >= ay && ax >= az && v.X < 0) || (ay > ax && ay >= az && v.Y < 0) || (az > ax && az > ay && v.Z < 0) {
		return v.Scale(-1)
	}
	return v
}

// axisFoot returns the point on the axis line nearest the world origin (invariant to where
// along the axis origin sits).
func axisFoot(origin math.Point3, dir math.Vector3) math.Point3 {
	ov := origin.AsVector()
	return math.P3(0, 0, 0).TranslateBy(ov.Sub(dir.Scale(ov.Dot(dir))))
}

// faceAreaCentroid returns a face's tessellated area and area-weighted centroid.
func faceAreaCentroid(f *topo.Face) (math.Point3, float64) {
	m := ops.TessellateFace(f, ops.DefaultQuality())
	if m == nil {
		return math.P3(0, 0, 0), 0
	}
	var area, cx, cy, cz float64
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a, b, c := m.Positions[m.Indices[t]], m.Positions[m.Indices[t+1]], m.Positions[m.Indices[t+2]]
		ar := float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) * 0.5
		area += ar
		cx += ar * (a.X + b.X + c.X) / 3
		cy += ar * (a.Y + b.Y + c.Y) / 3
		cz += ar * (a.Z + b.Z + c.Z) / 3
	}
	if area == 0 {
		return math.P3(0, 0, 0), 0
	}
	return math.P3(cx/area, cy/area, cz/area), area
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
