// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"math"
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

// addParameters emits each decoded user parameter (value in database cm).
func addParameters(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	var warns []string
	for _, p := range ipt.DecodeParameters(seg) {
		// TODO: decode the parameter's unit kind; length (cm) is assumed for now.
		if _, err := def.Parameters().AddUserParameter(p.Name, fmt.Sprintf("%g cm", p.Value)); err != nil {
			warns = append(warns, fmt.Sprintf("parameter %q: %v", p.Name, err))
		}
	}
	return warns
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
	emitted := emitSketches(def, placed)
	if ipt.HasRevolve(seg) {
		built, notes := buildRevolve(def, seg, placed, emitted)
		emitDroppedCurveSketches(def, d) // keep splines/ellipses the line-only revolve profile drops
		return built, notes
	}
	return buildExtrudeFeatures(def, d, seg, placed, emitted)
}

// extractSketches decodes the part's 2D sketch geometry into placed sketches, WITHOUT regard to
// the feature that will consume them. A revolve profile is reconstructed from point incidence
// (exact connectivity that reunites a profile split across the 800-byte cluster gap); everything
// else uses the clustered decode. All land on the XY origin plane. Keeping extraction separate
// from the feature build is what lets a part's sketches always reach the document.
func extractSketches(d *ipt.Document, seg []byte) []placedSketch {
	decoded := ipt.DecodeSketches(seg)
	// The node graph states sketch membership and curve connectivity outright, so prefer it over
	// the byte-offset clustering, which both splits one sketch (Linkage1: 1 sketch decoded as 3)
	// and loses most of the geometry (BigChunkyPlate: 2143 entities decoded as 19). Scoped to the
	// extrude path: a revolve profile still goes through the incidence reconstruction below, whose
	// closed-loop gate the revolve build depends on (RevolveProfile needs the centreline SPLIT into
	// its own sketch, which the graph decode keeps in the profile sketch — see buildRevolve).
	//
	// Taken only when it actually decoded entities: some parts hold Sketch2D nodes whose entities
	// this layout can't read (an older save generation, presumably), and an empty graph result must
	// not replace geometry the cluster decode did find.
	if !ipt.HasRevolve(seg) {
		if graph := ipt.GraphSketches(d); sketchEntityCount(graph) > 0 {
			decoded = graph
		}
	}
	if ipt.HasRevolve(seg) {
		// Incidence connectivity beats the clustered decode for a revolve profile; fall back to
		// the clustered sketches when it declines (e.g. an arc-bearing profile).
		if profiles := ipt.LineProfiles(seg); len(profiles) > 0 {
			// Reunite a separate vertical centreline into the profile sketch (as Inventor authored
			// it) so the revolve's radius dimensions can bind to the profile in one sketch.
			lineSet := ipt.ReuniteRevolveAxis(profiles)
			decoded = lineSet
			// Incidence sometimes SPLITS the profile into isolated single-line sketches (EncoderWheel:
			// a 13-line profile decoded as three 1-line sketches, none closed), so no revolve binds and
			// the part falls back to its non-manifold display mesh. The node GRAPH keeps that profile
			// whole; when the fragmented line set forms no valid revolve but the graph does (a vertical
			// in-profile centreline, case A), revolve the graph instead. Preferring the line set first
			// keeps every currently-building revolve — and the horizontal-axis test fixtures — unchanged.
			if !revolveBinds(lineSet) {
				if graph := ipt.GraphSketches(d); sketchEntityCount(graph) > 0 && revolveBinds(graph) {
					decoded = graph
				}
			}
		}
	}
	placed := make([]placedSketch, len(decoded))
	for i := range decoded {
		placed[i] = placedSketch{geom: decoded[i], plane: sketchPlaneOf(decoded[i])}
	}
	return placed
}

// revolveBinds reports whether RevolveProfile finds a valid closed profile + axis in this sketch
// set — the gate that decides whether a revolve can be built from it at all.
func revolveBinds(sketches []ipt.Sketch) bool {
	_, ok := ipt.RevolveProfile(sketches)
	return ok
}

// sketchPlaneOf places a sketch where the file says it lives. A sketch's entity coordinates are 2D
// IN ITS OWN PLANE, so putting them all on XY builds any feature authored elsewhere in the wrong
// place: BigChunkyPlate has 9 sketches on its top face at z=3.00 whose bosses grow DOWN into the
// plate, and forced onto XY they hung BELOW it instead — a 5.60 cm body against a true 3.00, which
// cost the part all 46 features to the body gate (#29).
//
// Falls back to XY when the file states no placement this layout can read, so an unreadable
// transform declines to today's behaviour rather than inventing a plane.
func sketchPlaneOf(s ipt.Sketch) sketch.Plane {
	if !s.PlaneOK {
		return sketch.XYPlane()
	}
	p, ok := planeOf(s.Plane)
	if !ok {
		return sketch.XYPlane()
	}
	return p
}

// planeOf builds a sketch plane from a decoded placement (origin + two in-plane axes), or ok=false
// when the axes do not make one. Shared by the sketch placement and the "To" extent's target, which
// the file states in the same form.
func planeOf(pl ipt.SketchPlacement) (sketch.Plane, bool) {
	x, xerr := m.NewUnitVector3(m.Scalar(pl.XAxis[0]), m.Scalar(pl.XAxis[1]), m.Scalar(pl.XAxis[2]))
	y, yerr := m.NewUnitVector3(m.Scalar(pl.YAxis[0]), m.Scalar(pl.YAxis[1]), m.Scalar(pl.YAxis[2]))
	if xerr != nil || yerr != nil {
		return sketch.Plane{}, false
	}
	p, err := sketch.NewPlane(m.P3(m.Scalar(pl.Origin[0]), m.Scalar(pl.Origin[1]), m.Scalar(pl.Origin[2])), x, y)
	if err != nil {
		return sketch.Plane{}, false
	}
	return p, true
}

// emitDroppedCurveSketches re-emits the freeform curves — splines and ellipses — that a whole-part
// feature build (revolve, sweep, loft) drops because its profile extraction is line-only. Those
// features replace the emitted set with a reconstructed line profile, silently discarding every
// spline/ellipse the part also carries (Hose-Screen-Adapter, a revolve, lost all 3 of its sketch
// splines this way). Only the curves are emitted — never the co-resident lines — so nothing can
// duplicate the feature's profile: a revolve/loft profile is line-only and never holds a spline or
// ellipse, so these sketches are disjoint from it. The extrude path already emits the full graph
// set, so this is called ONLY on the whole-part feature branches.
func emitDroppedCurveSketches(def *compdef.PartComponentDefinition, d *ipt.Document) {
	for _, s := range ipt.GraphSketches(d) {
		if len(s.Splines) == 0 && len(s.Ellipses) == 0 {
			continue
		}
		curves := ipt.Sketch{
			Splines: s.Splines, SplineConstruction: s.SplineConstruction,
			Ellipses: s.Ellipses,
			Plane:    s.Plane, PlaneOK: s.PlaneOK,
		}
		emitSketchOn(def, curves, sketchPlaneOf(curves))
	}
}

// emitSketches adds every extracted sketch to the document and returns the handles in the same
// order (an empty sketch yields a nil handle so indices still line up with the decode). Runs
// unconditionally, before any feature is attempted.
func emitSketches(def *compdef.PartComponentDefinition, placed []placedSketch) []emittedSketch {
	out := make([]emittedSketch, len(placed))
	for i, ps := range placed {
		sk, lines := emitSketchOn(def, ps.geom, ps.plane)
		out[i] = emittedSketch{sk: sk, lines: lines}
	}
	return out
}

// buildRevolve builds the revolve over the already-emitted sketches — it emits no geometry
// itself. It picks the profile + axis by the binding RevolveProfile validates (a CLOSED,
// one-sided profile about an unambiguous centreline, which may live in a different sketch than
// the profile) and revolves about that line. When no such profile/axis is found, or the chosen
// sketch/line came out empty, it builds nothing and returns a note; the emitted sketches remain
// for inspection. angle is nil ⇒ full 360°.
func buildRevolve(def *compdef.PartComponentDefinition, seg []byte, placed []placedSketch, emitted []emittedSketch) (bool, []string) {
	geoms := make([]ipt.Sketch, len(placed))
	for i := range placed {
		geoms[i] = placed[i].geom
	}
	b, ok := ipt.RevolveProfile(geoms)
	if !ok {
		return false, []string{"revolve: no unambiguous closed profile + axis — sketches emitted, revolve not built"}
	}
	if b.ProfileSketch >= len(emitted) || emitted[b.ProfileSketch].sk == nil {
		return false, []string{"revolve: chosen profile sketch is empty — revolve not built"}
	}
	axis := emitted[b.AxisSketch]
	if b.AxisLine >= len(axis.lines) || axis.lines[b.AxisLine] == nil {
		return false, []string{"revolve: axis line not emitted — revolve not built"}
	}
	var angle func() float64 // nil ⇒ full 360°
	if a, ok := ipt.RevolveAngle(seg); ok {
		a := a
		angle = func() float64 { return a }
	}
	feature.NewRevolveFeatures(def.Features()).AddAboutCenterlineLine(
		emitted[b.ProfileSketch].sk, 0, axis.sk, axis.lines[b.AxisLine], angle, ops.NewBody)
	return true, nil
}

// buildExtrudeFeatures builds the extrude chain over the already-emitted sketches: each extrude
// consumes the sketch at its index, then a hole cuts the base, then a pattern/mirror replicates
// the last extrude. Each stage that is decoded but can't be built appends a note and is skipped;
// whatever built stays. Returns whether any feature built.
// hasBaseExtrude reports whether any extrude starts a body (New-Body). Without one the extrude chain
// is all cut/join with nothing to act on, so it cannot rebuild a solid — its base is a feature type
// this decoder does not produce.
func hasBaseExtrude(extrudes []ipt.Extrude) bool {
	for _, e := range extrudes {
		if e.Operation == ipt.OpNewBody {
			return true
		}
	}
	return false
}

func buildExtrudeFeatures(def *compdef.PartComponentDefinition, d *ipt.Document, seg []byte, placed []placedSketch, emitted []emittedSketch) (bool, []string) {
	built := false
	var notes []string
	extrudes := ipt.DecodeExtrudes(d)
	// A part whose extrudes are ALL cut/join/intersect has no base body to apply them to: its base
	// is a feature this decoder does not produce (a sheet-metal face on MainBaseSheet, say). Applying
	// cuts to nothing builds a garbage sliver — 1% of the true volume — so build no extrude and leave
	// the sketches standing; buildPart then imports the real body. Only a New-Body extrude starts a
	// solid, so its absence means the whole extrude chain is baseless.
	if len(extrudes) > 0 && !hasBaseExtrude(extrudes) {
		return false, []string{fmt.Sprintf("%d extrude(s) but none starts a body — no parametric base; imported body used", len(extrudes))}
	}
	// Each extrude names the profile it consumes (see ipt.ExtrudeProfiles); "extrude i uses sketch
	// i" only ever held for the generated corpus.
	profiles := ipt.ExtrudeProfiles(d)
	regions := ipt.ExtrudeRegions(d)
	// patternSource is the last feature a pattern/mirror may legitimately replicate: a cut or a
	// secondary boss (join), NEVER the base solid. Inventor's PatternFeature always references a
	// feature placed AFTER the base; replicating the base itself only stamps coincident duplicates
	// of the whole body — a centred base disk rotated about its own axis stacks N identical copies,
	// inflating the volume N× (SmartKnobConnectingPlate: a Ø46 plate patterned 5× → 5×2098 mm³).
	var patternSource *feature.PartFeature
	for i, ex := range extrudes {
		p := profileIndex(profiles, i)
		if p < 0 || p >= len(emitted) || emitted[p].sk == nil {
			notes = append(notes, fmt.Sprintf("extrude %d: no profile sketch resolved — skipped", i))
			continue
		}
		region := extrudeRegionAt(regions, i)
		idx := regionProfileIndices(emitted[p].sk, region)
		if len(idx) == 0 {
			// The sketch holds several regions and we can't tell which this extrude names, so any
			// choice would be a guess; leave the sketch standing without a body.
			notes = append(notes, fmt.Sprintf("extrude %d: could not match its region (%d loops) to any rebuilt profile — skipped", i, len(region)))
			continue
		}
		fx := feature.NewExtrudeFeatures(def.Features()).AddExtrude(
			emitted[p].sk, idx, operationOf(ex.Operation), extentOf(ex), 0)
		if ex.Operation != ipt.OpNewBody {
			patternSource = fx // only a cut/join extrude is a valid pattern target
		}
		built = true
	}
	// A drilled hole cuts the base solid: place it on the extrude's top face (analytic), drilling
	// at the profile centroid. Needs the base extrude to have built the body first.
	if h, ok := ipt.DecodeHole(seg); ok {
		if len(extrudes) > 0 && len(placed) > 0 && len(emitted) > 0 && emitted[0].sk != nil {
			cx, cy := profileCentroid(placed[0].geom)
			addHole(def, h, cx, cy, extrudes[0].Distance)
			built = true
		} else {
			notes = append(notes, "hole decoded but no base extrude to cut — skipped")
		}
	}
	// A pattern or mirror replicates a cut/boss feature; it must run after that source so its
	// occurrences re-cut the running body. When the only feature built is the base solid,
	// patternSource is nil and the pattern is skipped rather than stamping N coincident bodies.
	// Rectangular / circular / mirror are mutually exclusive.
	if rp, ok := ipt.DecodeRectPattern(d); ok {
		if patternSource != nil {
			addRectPattern(def, patternSource, rp)
			built = true
		} else {
			notes = append(notes, "rectangular pattern decoded but only a base solid to replicate — skipped")
		}
	} else if cp, ok := ipt.DecodeCircPattern(d); ok {
		if patternSource != nil {
			addCircPattern(def, patternSource, cp)
			built = true
		} else {
			notes = append(notes, "circular pattern decoded but only a base solid to replicate — skipped")
		}
	} else if mir, ok := ipt.DecodeMirror(d); ok {
		if patternSource != nil {
			addMirror(def, patternSource, mir)
			built = true
		} else {
			notes = append(notes, "mirror decoded but no source feature — skipped")
		}
	}
	return built, notes
}

// addMirror reflects the source feature across the decoded mirror plane (origin + normal in
// model space, cm). The plane is authored geometrically (no lineage key), the externally-
// authored path the MirrorFeature supports.
func addMirror(def *compdef.PartComponentDefinition, source *feature.PartFeature, mir ipt.Mirror) {
	feature.NewPatternFeatures(def.Features()).AddMirror(
		[]feature.ID{source.ID()}, nil,
		m.P3(m.Scalar(mir.Origin[0]), m.Scalar(mir.Origin[1]), m.Scalar(mir.Origin[2])),
		m.Vector3{X: m.Scalar(mir.Normal[0]), Y: m.Scalar(mir.Normal[1]), Z: m.Scalar(mir.Normal[2])},
	)
}

// addRectPattern replicates the source feature into a 1D grid of rp.Count occurrences
// stepping rp.Spacing cm along +X (the corpus direction). countY is 1 (single row); the
// second grid axis, arbitrary direction, and circular/mirror patterns are future work.
func addRectPattern(def *compdef.PartComponentDefinition, source *feature.PartFeature, rp ipt.RectPattern) {
	count, spacing := rp.Count, rp.Spacing
	feature.NewPatternFeatures(def.Features()).AddRectangular(
		[]feature.ID{source.ID()},
		func() int { return count }, func() int { return 1 },
		m.Vector3{X: m.Scalar(spacing)}, m.Vector3{},
	)
}

// addCircPattern replicates the source feature into cp.Count occurrences spread over
// cp.Angle radians about the Z axis through the origin (the corpus default). Arbitrary
// axis and partial-angle spacing modes are future work.
func addCircPattern(def *compdef.PartComponentDefinition, source *feature.PartFeature, cp ipt.CircPattern) {
	count, angle := cp.Count, cp.Angle
	feature.NewPatternFeatures(def.Features()).AddCircular(
		[]feature.ID{source.ID()},
		func() int { return count }, func() float64 { return angle },
		m.P3(0, 0, 0), m.Vector3{Z: 1},
	)
}

// addHole drills a decoded hole into the running solid on the base extrude's top face — a
// geometric face reference (centroid at the profile centre, +Z normal) that re-binds after
// recompute (the externally-authored placement path, ADR-0040). Drilled, counterbore, and
// countersink holes are built; v1 drills at the face centroid (explicit off-centre placement
// is future work).
func addHole(def *compdef.PartComponentDefinition, h ipt.Hole, cx, cy, thickness float64) {
	holes := feature.NewHoleFeatures(def.Features())
	dia, depth := constF(h.Diameter), constF(h.Depth)
	var pf *feature.PartFeature
	switch {
	case h.Tapped:
		pf = holes.AddTapped(nil, dia, depth, h.Designation)
	case h.Type == ipt.CounterboreHole:
		pf = holes.AddCounterbore(nil, dia, depth, constF(h.CounterDiameter), constF(h.CounterDepth))
	case h.Type == ipt.CountersinkHole:
		pf = holes.AddCountersink(nil, dia, depth, constF(h.CounterDiameter), constF(h.CounterAngle))
	case h.ThroughAll:
		pf = holes.AddDrilledThrough(nil, dia)
	default:
		pf = holes.AddDrilled(nil, dia, depth)
	}
	hd := pf.Definition().(*feature.HoleFeature).Definition()
	hd.ThroughAll = h.ThroughAll
	hd.GeomFace = &topo.GeometricFaceRef{Centroid: m.P3(m.Scalar(cx), m.Scalar(cy), m.Scalar(thickness)), Normal: m.Vector3{X: 0, Y: 0, Z: 1}}
}

// constF returns a closure yielding the fixed value v — the func() float64 hole/pattern
// dimensions take.
func constF(v float64) func() float64 { return func() float64 { return v } }

// profileCentroid is the average of a sketch profile's corner/centre points — the drill
// spot for a centred hole.
func profileCentroid(s ipt.Sketch) (float64, float64) {
	var sx, sy float64
	n := 0
	for _, l := range s.Lines {
		sx, sy, n = sx+l.A.X+l.B.X, sy+l.A.Y+l.B.Y, n+2
	}
	for _, c := range s.Circles {
		sx, sy, n = sx+c.Center.X, sy+c.Center.Y, n+1
	}
	if n == 0 {
		return 0, 0
	}
	return sx / float64(n), sy / float64(n)
}

// addLoft blends the decoded profiles into a solid — each on an XY-parallel plane at its +Z
// height (the first at z=0). Returns false if any profile is empty or fewer than two sections
// resolve. v1 is a ruled loft with free ends; guides and non-parallel section planes are
// future work.
func addLoft(def *compdef.PartComponentDefinition, sections []ipt.Sketch, heights []float64) bool {
	if len(sections) != len(heights) || len(sections) < 2 {
		return false
	}
	secs := make([]feature.LoftSection, 0, len(sections))
	for i, s := range sections {
		sk, _ := emitSketchOn(def, s, offsetXYPlane(heights[i]))
		if sk == nil {
			return false
		}
		secs = append(secs, feature.LoftSection{Sketch: sk})
	}
	feature.NewLoftFeatures(def.Features()).Add(secs, false, ops.NewBody)
	return true
}

// addSweep pushes the circular profile (on the XY origin plane) along the decoded path. The
// path's 2D points (u, v) map to model space (u, 0, v) — the XZ path plane (v1). Returns
// false if the profile is empty.
func addSweep(def *compdef.PartComponentDefinition, sw ipt.Sweep) bool {
	prof, _ := emitSketchOn(def, ipt.Sketch{Circles: []ipt.Circle{sw.Profile}}, sketch.XYPlane())
	if prof == nil {
		return false
	}
	pts := make([]*sketch.Point3D, len(sw.Path))
	for i, p := range sw.Path {
		pts[i] = sketch.NewPoint3D(m.P3(m.Scalar(p.X), 0, m.Scalar(p.Y)))
	}
	feature.NewSweepFeatures(def.Features()).Add(prof, 0, sketch.NewPath3D(pts, false), nil, ops.NewBody)
	return true
}

// offsetXYPlane returns the XY-parallel sketch plane whose origin is z cm along +Z.
func offsetXYPlane(z float64) sketch.Plane {
	xu, _ := m.UnitVector3FromVector(m.Vector3{X: 1})
	yu, _ := m.UnitVector3FromVector(m.Vector3{Y: 1})
	p, err := sketch.NewPlane(m.P3(0, 0, m.Scalar(z)), xu, yu)
	if err != nil {
		return sketch.XYPlane()
	}
	return p
}

// emitSketch adds a decoded 2D sketch (points/lines/circles, in cm) on the XY origin plane.
func emitSketch(def *compdef.PartComponentDefinition, s ipt.Sketch) *sketch.Sketch {
	sk, _ := emitSketchOn(def, s, sketch.XYPlane())
	return sk
}

// emitSketchOn adds a decoded 2D sketch on the given plane (in cm), or returns nil when it
// has no geometry. It also returns the emitted line entities in decode order (so a revolve can
// pick out its centreline). Loft sections use it to place profiles on parallel offset planes.
//
// Corners that share a coordinate share a single sketch Point (sharedPoints): in Inventor a
// profile's touching endpoints carry an explicit coincident constraint, so two lines meeting at a
// corner are ONE point, not two free ones. Reproducing that — rather than minting fresh endpoints
// per line (AddByTwoPoints) — gives the rebuilt sketch the same degrees of freedom as the original
// (a closed N-gon: 2N free DOF, not 4N).
func emitSketchOn(def *compdef.PartComponentDefinition, s ipt.Sketch, plane sketch.Plane) (*sketch.Sketch, []*sketch.Line) {
	if len(s.Points) == 0 && len(s.Lines) == 0 && len(s.Circles) == 0 && len(s.Arcs) == 0 && len(s.Ellipses) == 0 && len(s.Splines) == 0 {
		return nil, nil
	}
	sk := def.Sketches().Add(plane)
	pointAt := sharedPoints(sk)
	for _, p := range s.Points {
		pointAt(p) // a standalone point still shares a coordinate with a coincident corner
	}
	lines := make([]*sketch.Line, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = sk.Lines().Add(pointAt(l.A), pointAt(l.B))
		// Construction geometry must stay construction: as real geometry it CUTS the regions around
		// it, and the kernel excludes it from profiles only when marked.
		lines[i].SetConstruction(s.LineIsConstruction(i))
	}
	for i, a := range s.Arcs {
		// Share the arc's endpoints with the adjacent lines (pointAt) so a filleted profile stays a
		// closed loop, exactly as a shared line corner does. Consumers decode the MINOR arc, so its
		// sweep direction (CCW when the CCW span start→end is the shorter one) is derived here.
		sk.Arcs().Add(pointAt(a.Center), pointAt(a.Start), pointAt(a.End), minorArcCCW(a)).
			SetConstruction(s.ArcIsConstruction(i))
	}
	for i, c := range s.Circles {
		sk.Circles().AddByCenterRadius(m.P2(c.Center.X, c.Center.Y), m.Scalar(c.Radius)).
			SetConstruction(s.CircleIsConstruction(i))
	}
	for _, e := range s.Ellipses {
		// Share the centre with any coincident corner (pointAt), matching how Inventor stores an
		// ellipse's centre by reference. The major-axis direction and both semi-axes are verbatim.
		sk.Ellipses().AddWithCenter(pointAt(e.Center), m.V2(e.MajorAxis.X, e.MajorAxis.Y), m.Scalar(e.MajorR), m.Scalar(e.MinorR))
	}
	for i, sp := range s.Splines {
		// Share the fit points with any coincident corner (pointAt), matching how Inventor references
		// them. fit=true: Inventor's sketch spline interpolates these points rather than approximating.
		pts := make([]*sketch.Point, len(sp.Points))
		for j, p := range sp.Points {
			pts[j] = pointAt(p)
		}
		sk.Splines().AddWithPoints(pts, sp.Closed, true).SetConstruction(s.SplineIsConstruction(i))
	}
	return sk, lines
}

// minorArcCCW reports whether the minor arc from Start to End sweeps counter-clockwise about the
// centre — true when the CCW span (start angle → end angle) is the shorter (≤ π) way round.
func minorArcCCW(a ipt.Arc) bool {
	a0 := math.Atan2(a.Start.Y-a.Center.Y, a.Start.X-a.Center.X)
	a1 := math.Atan2(a.End.Y-a.Center.Y, a.End.X-a.Center.X)
	sweep := a1 - a0
	for sweep < 0 {
		sweep += 2 * math.Pi
	}
	return sweep <= math.Pi
}

// applyGeometricConstraints binds each decoded geometric constraint (horizontal / vertical /
// parallel / perpendicular — ipt.DecodeGeometricConstraints) onto the emitted sketch that holds its
// line(s), by coordinate. These constraints carry no value and are already satisfied by the solved
// geometry, so applying them removes degrees of freedom WITHOUT moving a point — the profile the
// feature consumes is unchanged. A constraint is applied only while the sketch still has free DOF,
// so a redundant one is never piled on to over-constrain it. Reports how many were applied.
func applyGeometricConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, gc := range ipt.DecodeGeometricConstraints(seg) {
		if applyGeoConstraint(def, gc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d geometric constraint(s)", applied)}
}

// applyTangentConstraints binds each decoded line↔circle tangent (ipt.DecodeTangentConstraints)
// onto the sketch that holds both the line and the circle, as an AddTangent. The geometry is
// already tangent (validated at decode — perpendicular distance from centre == radius), so it only
// removes degrees of freedom, never moves a point. DOF-guarded.
func applyTangentConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, tc := range ipt.DecodeTangentConstraints(seg) {
		if applyTangent(def, tc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d tangent constraint(s)", applied)}
}

// applyTangent binds one tangent to the first sketch that holds both its line and its circle.
func applyTangent(def *compdef.PartComponentDefinition, tc ipt.TangentConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		l := lineAtCoords(sk, tc.Line)
		c := circleAtCoord(sk, tc.Center, tc.Radius)
		if l == nil || c == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddTangent(l, c)
		return true
	}
	return false
}

// applyCircleRelations binds each decoded concentric / equal-radius constraint
// (ipt.DecodeCircleRelations) onto the sketch holding both circles. The relation already holds in
// the geometry (validated at decode), so it only removes degrees of freedom. DOF-guarded.
func applyCircleRelations(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, cr := range ipt.DecodeCircleRelations(seg) {
		if applyCircleRelation(def, cr) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d circle relation(s)", applied)}
}

// applyCircleRelation binds one circle relation to the first sketch holding both its circles.
func applyCircleRelation(def *compdef.PartComponentDefinition, cr ipt.CircleRelation) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		c1 := circleAtCoord(sk, cr.C1, cr.R1)
		c2 := circleAtCoord(sk, cr.C2, cr.R2)
		if c1 == nil || c2 == nil || c1 == c2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if cr.Kind == ipt.GeoConcentric {
			sk.GeometricConstraints().AddConcentric(c1, c2)
		} else {
			sk.GeometricConstraints().AddEqualRadius(c1, c2)
		}
		return true
	}
	return false
}

// applyGroundConstraints binds each decoded ground constraint (ipt.DecodeGroundConstraints) onto
// the sketch holding its entity, as an AddGround. Grounding freezes the entity at its current
// position, so it only removes degrees of freedom — no geometry moves. DOF-guarded.
func applyGroundConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, gc := range ipt.DecodeGroundConstraints(seg) {
		if applyGround(def, gc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d ground constraint(s)", applied)}
}

// applyGround grounds one decoded entity on the first sketch that holds it.
func applyGround(def *compdef.PartComponentDefinition, gc ipt.GroundConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		if sk.DegreesOfFreedom() <= 0 {
			continue
		}
		switch gc.Kind {
		case ipt.GroundLine:
			if l := lineAtCoords(sk, gc.Line); l != nil {
				sk.GeometricConstraints().AddGround(l)
				return true
			}
		case ipt.GroundCircle:
			if c := circleAtCoord(sk, gc.Center, gc.Radius); c != nil {
				sk.GeometricConstraints().AddGround(c)
				return true
			}
		case ipt.GroundPoint:
			if p := pointAtCoord(sk, gc.Pt); p != nil {
				sk.GeometricConstraints().AddGround(p)
				return true
			}
		}
	}
	return false
}

// applySymmetryConstraints binds each decoded symmetry (ipt.DecodeSymmetryConstraints) onto the
// sketch holding both points and the axis line, as an AddSymmetry. The geometry is already
// symmetric (validated at decode — each point reflects onto the other across the axis), so it only
// removes degrees of freedom, never moves a point. DOF-guarded.
func applySymmetryConstraints(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, sc := range ipt.DecodeSymmetryConstraints(seg) {
		if applySymmetry(def, sc) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d symmetry constraint(s)", applied)}
}

// applySymmetry binds one symmetry to the first sketch that holds both points and its axis line.
func applySymmetry(def *compdef.PartComponentDefinition, sc ipt.SymmetryConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p1 := pointAtCoord(sk, sc.P1)
		p2 := pointAtCoord(sk, sc.P2)
		ax := lineAtCoords(sk, sc.Axis)
		if p1 == nil || p2 == nil || ax == nil || p1 == p2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddSymmetry(p1, p2, ax)
		return true
	}
	return false
}

// circleAtCoord returns the sketch circle whose centre matches c and radius matches r (within
// coincideEps), or nil.
func circleAtCoord(sk *sketch.Sketch, c ipt.Point2D, r float64) *sketch.Circle {
	circles := sk.Circles()
	for i := 0; i < circles.Count(); i++ {
		if q := circles.Item(i); samePt(q.Center, c) && math.Abs(float64(q.Radius)-r) < coincideEps {
			return q
		}
	}
	return nil
}

// applyGeoConstraint applies one geometric constraint to the first sketch containing its geometry.
func applyGeoConstraint(def *compdef.PartComponentDefinition, gc ipt.GeoConstraint) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		switch gc.Kind {
		case ipt.GeoHorizontal, ipt.GeoVertical:
			pa, pb := pointAtCoord(sk, gc.L1[0]), pointAtCoord(sk, gc.L1[1])
			if pa == nil || pb == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			if gc.Kind == ipt.GeoHorizontal {
				sk.GeometricConstraints().AddHorizontal(pa, pb)
			} else {
				sk.GeometricConstraints().AddVertical(pa, pb)
			}
			return true
		case ipt.GeoParallel, ipt.GeoPerpendicular, ipt.GeoCollinear, ipt.GeoEqualLength:
			l1, l2 := lineAtCoords(sk, gc.L1), lineAtCoords(sk, gc.L2)
			if l1 == nil || l2 == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			switch gc.Kind {
			case ipt.GeoParallel:
				sk.GeometricConstraints().AddParallel(l1, l2)
			case ipt.GeoPerpendicular:
				sk.GeometricConstraints().AddPerpendicular(l1, l2)
			case ipt.GeoCollinear:
				sk.GeometricConstraints().AddCollinear(l1, l2)
			case ipt.GeoEqualLength:
				sk.GeometricConstraints().AddEqualLength(l1, l2)
			}
			return true
		case ipt.GeoMidpoint:
			// Bind only when a sketch point actually sits at the line's midpoint (the pinned point,
			// whether a standalone point or another line's endpoint at a T-junction). If none does,
			// the constraint isn't reproduced rather than inventing a point.
			l := lineAtCoords(sk, gc.L1)
			p := pointAtCoord(sk, gc.Pt)
			if l == nil || p == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			sk.GeometricConstraints().AddMidpoint(p, l)
			return true
		case ipt.GeoPointOnLine:
			// The pinned vertex lies on the line's interior (validated at decode). Bind it only when
			// both the line and a sketch point at that vertex are present, so it never invents a point.
			l := lineAtCoords(sk, gc.L1)
			p := pointAtCoord(sk, gc.Pt)
			if l == nil || p == nil || sk.DegreesOfFreedom() <= 0 {
				continue
			}
			sk.GeometricConstraints().AddPointOnLine(p, l)
			return true
		}
	}
	return false
}

// applyDistanceDimensions binds each decoded point-to-point distance dimension
// (ipt.DecodeDistanceDimensions) onto the emitted sketch holding both endpoints, as a driving
// AddDistance. The value equals the current separation, so it pins DOF without moving geometry. It
// is applied only while the sketch has free DOF, so a redundant dimension can't over-constrain.
func applyDistanceDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, dm := range ipt.DecodeDistanceDimensions(seg) {
		if applyDistanceDim(def, dm) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d distance dimension(s)", applied)}
}

// applyDistanceDim adds one distance dimension to the first sketch containing both endpoints, kept
// only if it does not move geometry (keptWithoutMoving) — the value equals the current separation,
// so a well-decoded dimension is a pure DOF reduction, but the guard makes that a checked invariant
// rather than an assumption.
func applyDistanceDim(def *compdef.PartComponentDefinition, dm ipt.DistanceDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		pa, pb := pointAtCoord(sk, dm.A), pointAtCoord(sk, dm.B)
		if pa == nil || pb == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddDistance(pa, pb, fmt.Sprintf("%g cm", dm.Value))
		})
	}
	return false
}

// applyAngleDimensions binds each decoded angle dimension (ipt.DecodeAngleDimensions) onto the
// sketch holding both its lines, as an AddAngle of their current angle. The value equals the
// present geometric angle, so it pins the angle DOF without moving geometry. Applied only while the
// sketch has free DOF.
func applyAngleDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, ad := range ipt.DecodeAngleDimensions(seg) {
		if applyAngleDim(def, ad) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d angle dimension(s)", applied)}
}

// applyAngleDim adds one angle dimension to the first sketch that holds both its lines — but only
// if it does not move the geometry. An angle dimension pins the UNSIGNED angle between two lines; on
// an under-constrained sketch the solver can satisfy it by rotating/flipping the profile into a
// different configuration, silently corrupting a revolve/extrude profile (observed on a chamfered
// revolve — the profile drifted and the swept volume changed). So it is applied speculatively, the
// sketch is solved, and the dimension is kept only when every point stayed put; otherwise it is
// removed and the points restored. This keeps the correctness-first invariant: a decoded dimension
// is reproduced only when it is a pure degree-of-freedom reduction, never a geometry edit.
func applyAngleDim(def *compdef.PartComponentDefinition, ad ipt.AngleDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		l1, l2 := lineAtCoords(sk, ad.L1), lineAtCoords(sk, ad.L2)
		if l1 == nil || l2 == nil || l1 == l2 || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddAngle(l1, l2, fmt.Sprintf("%g deg", ad.Degrees))
		})
	}
	return false
}

// applyOffsetDimensions binds each decoded offset (distance-from-line) dimension
// (ipt.DecodeOffsetDimensions) onto the sketch holding its point and reference line, as an
// AddOffsetDim of the current perpendicular distance. Kept only if it does not move geometry.
func applyOffsetDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, od := range ipt.DecodeOffsetDimensions(seg) {
		if applyOffsetDim(def, od) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d offset dimension(s)", applied)}
}

// applyOffsetDim adds one offset dimension to the first sketch that holds its point and line.
func applyOffsetDim(def *compdef.PartComponentDefinition, od ipt.OffsetDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p, l := pointAtCoord(sk, od.Pt), lineAtCoords(sk, od.Line)
		if p == nil || l == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddOffsetDim(p, l, false, fmt.Sprintf("%g cm", od.Value))
		})
	}
	return false
}

// keptWithoutMoving adds a dimension via add and keeps it only when it is a faithful reproduction:
// it must STRICTLY REDUCE the sketch's degrees of freedom (a redundant dimension that duplicates an
// existing constraint would only over-constrain — e.g. the shaft's offset dim vs its radius dims)
// AND leave every point where it was after a solve (a dimension whose solve admits a different
// configuration, like a two-line angle flip, would silently edit the geometry). If either fails the
// dimension is deleted and the points restored. Reports whether it was kept.
func keptWithoutMoving(sk *sketch.Sketch, add func() (*sketch.DimensionConstraint, error)) bool {
	pts := sk.Points()
	snap := make([]m.Point2, pts.Count())
	for i := 0; i < pts.Count(); i++ {
		snap[i] = pts.Item(i).Position()
	}
	dofBefore := sk.DegreesOfFreedom()
	dim, err := add()
	if err != nil {
		return false
	}
	sk.Solve()
	if sk.DegreesOfFreedom() < dofBefore && !anyPointMoved(pts, snap) {
		return true
	}
	sk.DimensionConstraints().Delete(dim)
	for i := 0; i < pts.Count(); i++ {
		pts.Item(i).SetPosition(snap[i])
	}
	return false
}

// anyPointMoved reports whether any sketch point drifted from its snapshot beyond coincideEps.
func anyPointMoved(pts *sketch.Points, snap []m.Point2) bool {
	for i := 0; i < pts.Count(); i++ {
		if pts.Item(i).Position().DistanceTo(snap[i]) > coincideEps {
			return true
		}
	}
	return false
}

// applyRadiusDimensions binds each decoded radius/diameter dimension (ipt.DecodeRadiusDimensions)
// onto the sketch holding its circle or arc, as an AddRadius of the curve's own radius. Radius and
// diameter are indistinguishable in the file and pin the same DOF, so both apply as a radius
// dimension — the value equals the current radius, so nothing moves. Applied only while the sketch
// has free DOF.
func applyRadiusDimensions(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, rd := range ipt.DecodeRadiusDimensions(seg) {
		if applyRadiusDim(def, rd) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d radius/diameter dimension(s)", applied)}
}

// applyRadiusDim adds one radius dimension to the first sketch that holds its circle or arc, kept
// only if it does not move geometry (keptWithoutMoving). An arc's radius has no DOF of its own (it
// is |centre − start|), so pinning it drives the centre/start points — the guard ensures the solve
// pins the radius in place rather than sliding those points to a different arc.
func applyRadiusDim(def *compdef.PartComponentDefinition, rd ipt.RadiusDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		if sk.DegreesOfFreedom() <= 0 {
			continue
		}
		var c sketch.CircularCurve
		if rd.Arc {
			if a := arcAtCoord(sk, rd.Center, rd.Radius); a != nil {
				c = a
			}
		} else if cc := circleAtCoord(sk, rd.Center, rd.Radius); cc != nil {
			c = cc
		}
		if c == nil {
			continue
		}
		return keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) {
			return sk.DimensionConstraints().AddRadius(c, fmt.Sprintf("%g cm", rd.Radius))
		})
	}
	return false
}

// arcAtCoord returns the sketch arc whose centre matches center and radius matches r (within
// coincideEps), or nil.
func arcAtCoord(sk *sketch.Sketch, center ipt.Point2D, r float64) *sketch.Arc {
	arcs := sk.Arcs()
	for i := 0; i < arcs.Count(); i++ {
		if a := arcs.Item(i); samePt(a.Center, center) && math.Abs(float64(a.Radius())-r) < coincideEps {
			return a
		}
	}
	return nil
}

// applyRevolveRadii binds each decoded revolve radius dimension (ipt.DecodeRevolveRadii) as a
// HORIZONTAL distance from the x=0 centreline to the vertical profile edge at x=V, in the sketch
// holding both (the reunited profile+centreline sketch). The value equals the edge's current x, so
// it pins the radius without moving geometry. Applied only while the sketch has free DOF.
func applyRevolveRadii(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, x := range ipt.DecodeRevolveRadii(seg) {
		if applyRevolveRadius(def, x) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d revolve radius dimension(s)", applied)}
}

// applyRevolveRadius adds one radius dimension: a horizontal distance from a centreline point (x≈0)
// to an edge point (x≈radius) in the first sketch that holds both.
func applyRevolveRadius(def *compdef.PartComponentDefinition, radius float64) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p0, pv := pointAtX(sk, 0), pointAtX(sk, radius)
		if p0 == nil || pv == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if _, err := sk.DimensionConstraints().AddDistanceOriented(p0, pv, fmt.Sprintf("%g cm", radius), sketch.HorizontalDistance); err != nil {
			return false
		}
		return true
	}
	return false
}

// applyAxialLengths binds each decoded axial step-length dimension (ipt.DecodeAxialLengths) as a
// VERTICAL distance between the two horizontal profile edges it spans, in the sketch holding both.
// The value equals the edges' current separation, so it pins the step length without moving
// geometry. Applied only while the sketch has free DOF.
func applyAxialLengths(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	applied := 0
	for _, ax := range ipt.DecodeAxialLengths(seg) {
		if applyAxialLength(def, ax) {
			applied++
		}
	}
	if applied == 0 {
		return nil
	}
	return []string{fmt.Sprintf("applied %d axial length dimension(s)", applied)}
}

// applyAxialLength adds one vertical distance between a point at y≈Y1 and a point at y≈Y2.
func applyAxialLength(def *compdef.PartComponentDefinition, ax ipt.AxialLength) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p1, p2 := pointAtY(sk, ax.Y1), pointAtY(sk, ax.Y2)
		if p1 == nil || p2 == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if _, err := sk.DimensionConstraints().AddDistanceOriented(p1, p2, fmt.Sprintf("%g cm", ax.Value), sketch.VerticalDistance); err != nil {
			return false
		}
		return true
	}
	return false
}

// applyCentrelineAnchor fixes a revolve sketch's point at the sketch origin (0,0). A revolve's
// centreline runs from the origin, and geometry drawn AT the sketch origin is coincident with the
// origin — a fixed reference in every CAD sketch — so pinning that point reproduces the anchoring
// the file leaves implicit (there is no explicit fix node; the axis line carries no vertical/fix
// constraint of its own). Fixing a point at its current position never moves geometry, and it is
// applied only while the sketch has free DOF. Gated to revolves so it only ever pins a centreline
// origin, not an incidental origin-touching corner of an extrude profile.
func applyCentrelineAnchor(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok || !ipt.HasRevolve(seg) {
		return nil
	}
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		p := pointAtCoord(sk, ipt.Point2D{X: 0, Y: 0})
		if p == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		sk.GeometricConstraints().AddFix(p)
		return []string{"anchored the centreline to the sketch origin"}
	}
	return nil
}

// pointAtY returns a sketch point whose Y coordinate is within coincideEps of y, or nil.
func pointAtY(sk *sketch.Sketch, y float64) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.Y)-y) < coincideEps {
			return q
		}
	}
	return nil
}

// pointAtX returns a sketch point whose X coordinate is within coincideEps of x, or nil.
func pointAtX(sk *sketch.Sketch, x float64) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.X)-x) < coincideEps {
			return q
		}
	}
	return nil
}

// pointAtCoord returns the sketch point at coordinate p (within coincideEps), or nil.
func pointAtCoord(sk *sketch.Sketch, p ipt.Point2D) *sketch.Point {
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		if q := pts.Item(i); math.Abs(float64(q.X)-p.X) < coincideEps && math.Abs(float64(q.Y)-p.Y) < coincideEps {
			return q
		}
	}
	return nil
}

// lineAtCoords returns the sketch line whose endpoints match e (in either order), or nil.
func lineAtCoords(sk *sketch.Sketch, e [2]ipt.Point2D) *sketch.Line {
	lines := sk.Lines()
	for i := 0; i < lines.Count(); i++ {
		l := lines.Item(i)
		if (samePt(l.A, e[0]) && samePt(l.B, e[1])) || (samePt(l.A, e[1]) && samePt(l.B, e[0])) {
			return l
		}
	}
	return nil
}

// samePt reports whether a sketch point sits at coordinate p (within coincideEps).
func samePt(q *sketch.Point, p ipt.Point2D) bool {
	return math.Abs(float64(q.X)-p.X) < coincideEps && math.Abs(float64(q.Y)-p.Y) < coincideEps
}

// sharedPoints returns a coordinate→Point allocator over one sketch: the first time a coordinate
// is seen it mints a sketch Point; a later corner within coincideEps of it reuses the same Point.
// This makes touching profile corners structurally coincident (the original's endpoint coincidence
// constraints), so the rebuilt sketch has the same DOF instead of independent duplicated endpoints.
func sharedPoints(sk *sketch.Sketch) func(ipt.Point2D) *sketch.Point {
	type cached struct {
		p  ipt.Point2D
		pt *sketch.Point
	}
	var cache []cached
	return func(p ipt.Point2D) *sketch.Point {
		for _, e := range cache {
			if math.Abs(e.p.X-p.X) < coincideEps && math.Abs(e.p.Y-p.Y) < coincideEps {
				return e.pt
			}
		}
		pt := sk.Points().Add(m.P2(p.X, p.Y))
		cache = append(cache, cached{p, pt})
		return pt
	}
}

// coincideEps is the coordinate tolerance (cm) below which two profile corners are treated as one
// coincident sketch point.
const coincideEps = 1e-6

// operationOf maps a decoded Inventor boolean operation to the kernel operation.
func operationOf(op int) ops.PartFeatureOperation {
	switch op {
	case ipt.OpCut:
		return ops.Cut
	case ipt.OpJoin:
		return ops.Join
	case ipt.OpIntersect:
		return ops.Intersect
	default:
		return ops.NewBody
	}
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

// profileIndex returns the sketch index extrude i consumes. It falls back to i only when the
// profile list is absent entirely (an older decode path), never when the graph resolved a profile
// and simply could not bind one — that case must skip rather than build on a guess.
func profileIndex(profiles []int, i int) int {
	if len(profiles) == 0 {
		return i
	}
	if i >= len(profiles) {
		return -1
	}
	return profiles[i]
}

// sketchEntityCount totals the curves and points across decoded sketches — the signal that a decode
// actually read geometry rather than producing empty shells.
func sketchEntityCount(sketches []ipt.Sketch) int {
	n := 0
	for _, s := range sketches {
		n += len(s.Points) + len(s.Lines) + len(s.Circles) + len(s.Arcs) + len(s.Ellipses) + len(s.Splines)
	}
	return n
}

// extrudeRegionAt returns the decoded region for extrude i, or nil when the decode produced none.
func extrudeRegionAt(regions [][]ipt.RegionLoop, i int) []ipt.RegionLoop {
	if i >= len(regions) {
		return nil
	}
	return regions[i]
}

// extentOf turns a decoded extrude's termination into the feature engine's extent. A through-all
// extrude carries no distance — it runs until it leaves the material — so it must NOT be built as a
// length: its depth parameter decodes as 0, and a 0-length extrude is a degenerate zero-thickness
// body (that is what made BigChunkyPlate a surface rather than a solid).
func extentOf(ex ipt.Extrude) feature.Extent {
	if ex.ThroughAll {
		return feature.Extent{Type: feature.ThroughAllExtent, Direction: directionOf(ex)}
	}
	// A "To <face>" extrude terminates AT a plane and its Distance is a stale leftover, so it must be
	// built from the target, never as a length. Direction is deliberately left at its zero value:
	// toPlaneSpan derives the span from the SIGNED distance to the target, so which way it runs is
	// already decided by where that target is (see ipt.toTargetPlane).
	if ex.ToPlaneOK {
		if pl, ok := planeOf(ex.ToPlane); ok {
			return feature.Extent{Type: feature.ToFaceExtent, ToPlane: feature.NewFixedWorkPlane(pl)}
		}
	}
	dist := ex.Distance
	e := feature.Extent{
		Type:      feature.DistanceExtent,
		Direction: directionOf(ex),
		Distance:  func() float64 { return dist },
	}
	// A two-sided extrude grows dimLength2 the other way. Its own direction stays PositiveDir:
	// Distance2 IS the negative side, so pairing it with NegativeDir would grow both spans the
	// same way.
	if ex.Distance2 != 0 && !ex.Midplane {
		d2 := ex.Distance2
		e.Direction = feature.PositiveDir
		e.Distance2 = func() float64 { return d2 }
	}
	return e
}

// directionOf maps the extrude's own direction operands onto the extent direction. Midplane wins
// over reversed: straddling the sketch plane is symmetric, so which way it "grows" is moot.
//
// The extent direction is expressed in the SKETCH's frame, not the world's: buildExtrusionShell
// grows the prism along plane.Normal() scaled by the span, so PositiveDir means "along this
// sketch's own normal" wherever that points. Measured on all 517 corpus sketch placements, the
// DirectionAxis is the sketch's own normal in world coordinates (dot = +1.000 on every extrude of
// every part probed), so `dir` adds nothing to the comparison — only `reversed` decides, by
// flipping that vector to run against the normal.
//
// This deliberately replaces `sign(dir.z)` vs world +Z, which was only ever a shortcut valid while
// every sketch was forced onto XY (before ee6ac047 decoded the real planes). That shortcut is
// silently BLIND on any sketch whose normal is perpendicular to Z: dir.z is then 0, so it returns
// PositiveDir and ignores `reversed` entirely. CompressionRollerArmActuatorScrew is built from
// ±X-facing sketches, and its screwdriver-slot cut was placed OUTSIDE the head, removing nothing —
// scoring 1.033x on a no-op (proven: the volume is bit-identical across that feature).
func directionOf(ex ipt.Extrude) feature.ExtentDirection {
	if ex.Midplane {
		return feature.SymmetricDir
	}
	if ex.Reversed {
		return feature.NegativeDir
	}
	return feature.PositiveDir
}
