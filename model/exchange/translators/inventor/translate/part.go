// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// BodyFromIPT decodes an .ipt and reconstructs its solid body from the ACIS B-rep,
// welded into a watertight kernel body. Used for in-memory geometry validation.
func BodyFromIPT(iptBytes []byte) (*topo.Body, []string, error) {
	b, err := brepOf(iptBytes)
	if err != nil {
		return nil, nil, err
	}
	return BodyFromBrep(b, "import:ipt#0")
}

// FromInventor translates an .ipt into a native Oblikovati part package at outPath:
// user parameters become Oblikovati parameters, sketches are extracted and emitted, and the
// feature tree is rebuilt in history order. A part that can't be fully translated is saved in
// its PARTIAL parametric state (sketches + whatever features built) so its history stays
// inspectable in the browser — not replaced by Inventor's opaque display mesh (see
// meshFallbackEnabled). Returns any non-fatal translation warnings.
func FromInventor(iptBytes []byte, outPath string) ([]string, error) {
	d, err := ipt.Open(iptBytes)
	if err != nil {
		return nil, err
	}
	if d.IsAssembly() {
		return nil, assemblyError(d)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, warns, err := buildPart(ws, outPath, d, meshFallbackEnabled())
	if err != nil {
		return warns, err
	}
	if err := ws.Save(document); err != nil {
		return warns, err
	}
	return warns, nil
}

// meshFallbackEnabled reports whether to import Inventor's stored display tessellation for a
// part that doesn't fully rebuild parametrically. Default OFF: the translator saves the partial
// parametric state (sketches + built features) so the sketch/feature history is inspectable and
// a failed step can be diagnosed. Set OBK_IPT_MESH_FALLBACK=1 to instead import the opaque
// display-mesh body — useful when only the silhouette matters (e.g. an assembly preview).
func meshFallbackEnabled() bool {
	return os.Getenv("OBK_IPT_MESH_FALLBACK") == "1"
}

// buildPart adds a part document to ws at outPath and populates it from the decoded .ipt —
// parameters, then the extracted sketches, then the feature tree over those sketches. It stops
// short of saving so callers (a standalone part, or an assembly component) control that. The
// partial parametric state is kept as-is; the display mesh is imported only when meshFallback is
// set AND the parametric path produced no solid.
func buildPart(ws *doc.Workspace, outPath string, d *ipt.Document, meshFallback bool) (*doc.Document, []string, error) {
	document, err := compdef.AddPart(ws, outPath, true)
	if err != nil {
		return nil, nil, err
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	warns := addParameters(def, d)
	built, notes := addFeatures(def, d)
	warns = append(warns, notes...)
	warns = append(warns, applyGeometricConstraints(def, d)...)
	warns = append(warns, applyTangentConstraints(def, d)...)
	warns = append(warns, applyCircleRelations(def, d)...)
	warns = append(warns, applyGroundConstraints(def, d)...)
	warns = append(warns, applySymmetryConstraints(def, d)...)
	warns = append(warns, applyDistanceDimensions(def, d)...)
	warns = append(warns, applyRadiusDimensions(def, d)...)
	warns = append(warns, applyAngleDimensions(def, d)...)
	warns = append(warns, applyRevolveRadii(def, d)...)
	warns = append(warns, applyAxialLengths(def, d)...)
	warns = append(warns, applyCentrelineAnchor(def, d)...)
	// Offset (distance-from-line) dims last: on a revolve profile they can duplicate the
	// radius/axial dims above, so applying those first lets keptWithoutMoving's DOF-reduction check
	// drop a redundant offset instead of over-constraining (the shaft carries exactly such a pair).
	warns = append(warns, applyOffsetDimensions(def, d)...)
	def.Recompute()
	if built && !firstBodyIsSolid(def) && nearlySolid(def) {
		retryWithoutDissolve(def)
	}
	// A rebuilt body that escapes Inventor's own tessellation was mis-decoded; drop it rather than
	// ship wrong geometry (see gateBodyAgainstMesh).
	if built {
		if dropped := gateBodyAgainstMesh(def, d); len(dropped) > 0 {
			warns = append(warns, dropped...)
			built = false
		}
	}
	// Inventor's display mesh hides the sketch/feature history behind a single imported body, so for
	// a part that DID decode a partial parametric tree it is imported only on explicit opt-in — the
	// tree stands. But a body-only .ipt (a derived/imported solid, or a downloaded vendor part) has
	// NO sketches and NO features: there is no history to mask, and leaving it EMPTY discards the one
	// thing it carries. So when nothing parametric decoded, import its body unconditionally, turning
	// an empty translation into the real shape (31 of the corpus's ThirdParty + base-plate parts).
	noParametric := def.Sketches().Count() == 0 && def.Features().Count() == 0
	// A baseless extrude chain (extrudes exist but all cut/join, no New-Body — see
	// buildExtrudeFeatures) also can't rebuild parametrically, so its real shape must come from the
	// imported body too.
	ex := ipt.DecodeExtrudes(d)
	noBase := len(ex) > 0 && !hasBaseExtrude(ex)
	// A parametric attempt that produced NO body at all — every extrude declined, or a revolve/
	// loft/sweep base didn't reconstruct — leaves the def empty just like a body-only part. There is
	// no partial parametric body to mask, so import the real shape rather than ship an empty part
	// (HeadShield, CapstainFrontBody, MagneticShieldBlock and other decode-but-build-nothing parts).
	noBody := len(def.SurfaceBodies().All()) == 0
	if (meshFallback || noParametric || noBase || noBody) && (!built || !hasSolidBody(def)) {
		warns = append(warns, addBodyIfPresent(def, d)...)
		def.Recompute()
	}
	return document, warns, nil
}

// firstBodyIsSolid reports whether the part's first body is a closed solid — the exact test the
// corpus classifier uses to call a part SOLID (vs an open SURFACE body).
func firstBodyIsSolid(def *compdef.PartComponentDefinition) bool {
	bodies := def.SurfaceBodies().All()
	return len(bodies) > 0 && bodies[0].IsSolid()
}

// nearlySolidEdgeLimit gates the dissolve fallback: a body with more open mesh edges than this was
// never close to a solid, so per-region prisms cannot recover one. The largest dissolve REGRESSION
// observed (WheelSlider, a coincident-wall crack the merged tool tripped on an otherwise-solid part)
// is 3 open edges; a fundamentally-cracked part is far above this (BigChunkyPlate: 94). The limit sits
// well above the former and below the latter, so a genuine regression still triggers the fallback
// while a hopeless part skips it.
const nearlySolidEdgeLimit = 40

// nearlySolid reports whether the first body is close enough to closed that the dissolve fallback's
// per-region rebuild could plausibly recover a SOLID — i.e. the dissolve merely tripped a small
// coincident-wall crack. Used to skip the expensive off-recompute (~2.5 min on BigChunkyPlate) on a
// body whose many open edges mean it was never near-solid, where the fallback would only spend that
// time producing a worse, still-open mesh.
func nearlySolid(def *compdef.PartComponentDefinition) bool {
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		return false
	}
	return openEdgeCount(bodies[0]) <= nearlySolidEdgeLimit
}

// openEdgeCount is the number of boundary (single-triangle) mesh edges of the body — 0 for a
// watertight solid, growing with how cracked an open body is. Undirected edges are welded by
// coordinate so the count is independent of per-face vertex indexing.
func openEdgeCount(b *topo.Body) int {
	mesh, _ := ops.TessellateBody(b, ops.DefaultQuality())
	const grid = 1e-6
	key := func(p m.Point3) [3]int64 {
		return [3]int64{int64(p.X / grid), int64(p.Y / grid), int64(p.Z / grid)}
	}
	count := map[[2][3]int64]int{}
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		for e := 0; e < 3; e++ {
			a, bb := key(mesh.Positions[mesh.Indices[i+e]]), key(mesh.Positions[mesh.Indices[i+(e+1)%3]])
			if a == bb {
				continue
			}
			if lessKey(bb, a) {
				a, bb = bb, a
			}
			count[[2][3]int64{a, bb}]++
		}
	}
	open := 0
	for _, c := range count {
		if c == 1 {
			open++
		}
	}
	return open
}

func lessKey(a, b [3]int64) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[2] < b[2]
}

// retryWithoutDissolve is the whole-part fallback for the abutting-prism dissolve (#38). The dissolve
// fixes the coincident-wall crack a slot-with-corner-reliefs cut leaves and turns some parts from an
// open SURFACE into a closed SOLID — but on a few it trips a downstream boolean fragility (in a LATER
// feature, so it can't be caught per-feature) and OPENS a body that per-region prisms would have kept
// solid. When the part didn't come back a solid AND is near-solid (see nearlySolid), rebuild it with
// the dissolve OFF and keep that: per-region prisms restore the regressed solid (WheelSlider). A part
// far from solid never reaches here — the dissolve's (usually cleaner) mesh stands, and its ~2.5-min
// off-recompute is skipped. The dissolve is thus retained wherever it helps or is neutral, and undone
// only where per-region prisms recover a solid.
func retryWithoutDissolve(def *compdef.PartComponentDefinition) {
	if feature.SetExtrudeDissolve(def.Features(), false) == 0 {
		return // no extrude dissolved: nothing to undo
	}
	def.Recompute()
}

// hasSolidBody reports whether the definition holds a non-degenerate solid after recompute —
// the signal that the parametric rebuild actually produced geometry (vs a feature tree that
// silently computed to nothing on a real part the decoder couldn't fully bind).
func hasSolidBody(def *compdef.PartComponentDefinition) bool {
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		return false
	}
	return analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3 > 1e-6
}

// assemblyError reports that the input is an assembly (.iam), not a part. FromInventor
// works on part bytes; an assembly references component files on disk, so it must be
// translated by AssemblyFromInventor(iamPath, outPath) instead.
func assemblyError(d *ipt.Document) error {
	seg, _ := d.Segment("AmDcSegment")
	occ := ipt.DecodeOccurrences(seg)
	return fmt.Errorf("input is an Inventor assembly (.iam), not a part: %d occurrence(s) of component(s) %v — use AssemblyFromInventor(iamPath, outPath)",
		len(occ), ipt.ComponentRefs(occ))
}

// placedSketch is a decoded 2D sketch together with the plane it lives on — the product of
// sketch EXTRACTION, deliberately independent of any feature that will consume it.
type placedSketch struct {
	geom  ipt.Sketch
	plane sketch.Plane
}

// emittedSketch is a placedSketch after it has been added to the document: the sketch handle and
// its line entities in decode order (a revolve/extrude references these by index).
type emittedSketch struct {
	sk    *sketch.Sketch
	lines []*sketch.Line
}

// addFeatures translates the part in two decoupled passes: first EXTRACT and emit every sketch
// (so the geometry always reaches the document), then BUILD features that reference those
// sketches by decode order. A feature that can't be translated is skipped with an explanatory
// note, leaving the emitted sketches and any earlier features intact — the partial parametric
// state is what gets saved. Returns whether any feature built, plus per-step notes.
func addFeatures(def *compdef.PartComponentDefinition, d *ipt.Document) (bool, []string) {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return false, nil
	}
	// Loft and sweep are self-contained whole-part builders that own their sketch placement
	// (offset section planes / a 3D sweep path). When one applies it fully defines the part, so
	// it runs before — and instead of — the general extract-then-build path.
	sketches := ipt.DecodeSketches(seg)
	if heights, ok := ipt.LoftSectionHeights(seg, len(sketches)); ok {
		if addLoft(def, sketches, heights) {
			emitDroppedCurveSketches(def, d) // keep splines/ellipses the line-only loft profile drops
			return true, nil
		}
	}
	if sw, ok := ipt.DecodeSweep(seg); ok {
		if addSweep(def, sw) {
			emitDroppedCurveSketches(def, d) // keep splines/ellipses the line-only sweep profile drops
			return true, nil
		}
	}
	// Decoupled path: extract + emit all sketches unconditionally, then build features over them.
	placed := extractSketches(d, seg)
	if ipt.HasRevolve(seg) {
		// The revolve path owns its own emission: a machined revolve part is rebuilt from the node
		// graph (profile+centreline+cuts kept whole) when the incidence line set can't close it.
		return buildRevolveDispatch(def, d, seg, placed)
	}
	emitted := emitSketches(def, placed)
	return buildExtrudeFeatures(def, d, seg, placed, emitted)
}

// addBodyIfPresent reconstructs the solid body when the part has one we can rebuild.
// A missing/curved/empty body is reported as a warning, not a failure — a parameter-
// or sketch-only part is still a valid translation. It prefers Inventor's own stored display
// tessellation (PmGraphicsSegment: curved faces and holes already meshed), and falls back to
// the analytic ACIS planar reconstruction when the graphics mesh is absent/degenerate.
func addBodyIfPresent(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	// Prefer Inventor's display tessellation, but only when it is a real 3-D body — some parts store
	// a flat footprint placeholder there and keep the true body in the SAB, so a degenerate mesh
	// falls through to the B-rep (or to no body) rather than importing a wrong flat sheet.
	if raw := SoupFromMesh(ipt.GraphicsMesh(d)); raw.TriangleCount() >= 4 && meshIsSpatial(raw) {
		return importSoup(def, raw)
	}
	seg, ok := d.Segment("PmBRepSegment")
	if !ok {
		return nil
	}
	raw, err := SoupFromBrep(ipt.ExtractBrep(seg))
	if err != nil {
		return []string{"body: " + err.Error()}
	}
	return importSoup(def, raw)
}

// importSoup writes a triangle soup to a temp STL and imports it as the part's body (welded
// to a solid on the way in). Warnings (e.g. a non-watertight mesh brought in as a surface)
// are surfaced to the caller.
func importSoup(def *compdef.PartComponentDefinition, raw meshio.RawMesh) []string {
	stlPath, cleanup, err := writeTempSTL(encodeBinarySTL(raw))
	if err != nil {
		return []string{"body: " + err.Error()}
	}
	defer cleanup()
	res, err := exchange.Import(def, stlPath, types.FormatSTL)
	if err != nil {
		return []string{"body: " + err.Error()}
	}
	return res.Warnings
}

func brepOf(iptBytes []byte) (ipt.Brep, error) {
	d, err := ipt.Open(iptBytes)
	if err != nil {
		return ipt.Brep{}, err
	}
	seg, ok := d.Segment("PmBRepSegment")
	if !ok {
		return ipt.Brep{}, fmt.Errorf("build: .ipt has no PmBRepSegment (no solid body)")
	}
	return ipt.ExtractBrep(seg), nil
}

func writeTempSTL(data []byte) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "ipt-body-*.stl")
	if err != nil {
		return "", func() {}, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", func() {}, err
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}
