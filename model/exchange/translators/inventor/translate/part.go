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
	warns = append(warns, applyRevolveRadii(def, d)...)
	warns = append(warns, applyAxialLengths(def, d)...)
	warns = append(warns, applyCentrelineAnchor(def, d)...)
	def.Recompute()
	// Inventor's display mesh hides the sketch/feature history behind a single imported body,
	// so it is imported ONLY on explicit opt-in — otherwise the partial parametric tree stands.
	if meshFallback && (!built || !hasSolidBody(def)) {
		warns = append(warns, addBodyIfPresent(def, d)...)
		def.Recompute()
	}
	return document, warns, nil
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
			return true, nil
		}
	}
	if sw, ok := ipt.DecodeSweep(seg); ok {
		if addSweep(def, sw) {
			return true, nil
		}
	}
	// Decoupled path: extract + emit all sketches unconditionally, then build features over them.
	placed := extractSketches(seg)
	emitted := emitSketches(def, placed)
	if ipt.HasRevolve(seg) {
		return buildRevolve(def, seg, placed, emitted)
	}
	return buildExtrudeFeatures(def, d, seg, placed, emitted)
}

// extractSketches decodes the part's 2D sketch geometry into placed sketches, WITHOUT regard to
// the feature that will consume them. A revolve profile is reconstructed from point incidence
// (exact connectivity that reunites a profile split across the 800-byte cluster gap); everything
// else uses the clustered decode. All land on the XY origin plane. Keeping extraction separate
// from the feature build is what lets a part's sketches always reach the document.
func extractSketches(seg []byte) []placedSketch {
	decoded := ipt.DecodeSketches(seg)
	if ipt.HasRevolve(seg) {
		// Incidence connectivity beats the clustered decode for a revolve profile; fall back to
		// the clustered sketches when it declines (e.g. an arc-bearing profile).
		if profiles := ipt.LineProfiles(seg); len(profiles) > 0 {
			// Reunite a separate vertical centreline into the profile sketch (as Inventor authored
			// it) so the revolve's radius dimensions can bind to the profile in one sketch.
			decoded = ipt.ReuniteRevolveAxis(profiles)
		}
	}
	placed := make([]placedSketch, len(decoded))
	for i := range decoded {
		placed[i] = placedSketch{geom: decoded[i], plane: sketch.XYPlane()}
	}
	return placed
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
func buildExtrudeFeatures(def *compdef.PartComponentDefinition, d *ipt.Document, seg []byte, placed []placedSketch, emitted []emittedSketch) (bool, []string) {
	built := false
	var notes []string
	extrudes := ipt.DecodeExtrudes(seg)
	var lastExtrude *feature.PartFeature
	for i, ex := range extrudes {
		if i >= len(emitted) || emitted[i].sk == nil {
			notes = append(notes, fmt.Sprintf("extrude %d has no sketch to consume — skipped", i))
			continue
		}
		dist := ex.Distance
		lastExtrude = feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(emitted[i].sk, 0, operationOf(ex.Operation), func() float64 { return dist })
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
	// A pattern or mirror replicates the last feature; it must run after the source feature so its
	// occurrences re-cut the running body. Rectangular / circular / mirror are mutually exclusive.
	if rp, ok := ipt.DecodeRectPattern(seg); ok {
		if lastExtrude != nil {
			addRectPattern(def, lastExtrude, rp)
			built = true
		} else {
			notes = append(notes, "rectangular pattern decoded but no source feature — skipped")
		}
	} else if cp, ok := ipt.DecodeCircPattern(seg); ok {
		if lastExtrude != nil {
			addCircPattern(def, lastExtrude, cp)
			built = true
		} else {
			notes = append(notes, "circular pattern decoded but no source feature — skipped")
		}
	} else if mir, ok := ipt.DecodeMirror(d); ok {
		if lastExtrude != nil {
			addMirror(def, lastExtrude, mir)
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
	if len(s.Points) == 0 && len(s.Lines) == 0 && len(s.Circles) == 0 && len(s.Arcs) == 0 && len(s.Ellipses) == 0 {
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
	}
	for _, a := range s.Arcs {
		// Share the arc's endpoints with the adjacent lines (pointAt) so a filleted profile stays a
		// closed loop, exactly as a shared line corner does. Consumers decode the MINOR arc, so its
		// sweep direction (CCW when the CCW span start→end is the shorter one) is derived here.
		sk.Arcs().Add(pointAt(a.Center), pointAt(a.Start), pointAt(a.End), minorArcCCW(a))
	}
	for _, c := range s.Circles {
		sk.Circles().AddByCenterRadius(m.P2(c.Center.X, c.Center.Y), m.Scalar(c.Radius))
	}
	for _, e := range s.Ellipses {
		// Share the centre with any coincident corner (pointAt), matching how Inventor stores an
		// ellipse's centre by reference. The major-axis direction and both semi-axes are verbatim.
		sk.Ellipses().AddWithCenter(pointAt(e.Center), m.V2(e.MajorAxis.X, e.MajorAxis.Y), m.Scalar(e.MajorR), m.Scalar(e.MinorR))
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

// applyDistanceDim adds one distance dimension to the first sketch containing both endpoints.
func applyDistanceDim(def *compdef.PartComponentDefinition, dm ipt.DistanceDim) bool {
	for k := 0; k < def.Sketches().Count(); k++ {
		sk := def.Sketches().Item(k)
		pa, pb := pointAtCoord(sk, dm.A), pointAtCoord(sk, dm.B)
		if pa == nil || pb == nil || sk.DegreesOfFreedom() <= 0 {
			continue
		}
		if _, err := sk.DimensionConstraints().AddDistance(pa, pb, fmt.Sprintf("%g cm", dm.Value)); err != nil {
			return false
		}
		return true
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

// applyRadiusDim adds one radius dimension to the first sketch that holds its circle or arc.
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
		if _, err := sk.DimensionConstraints().AddRadius(c, fmt.Sprintf("%g cm", rd.Radius)); err != nil {
			return false
		}
		return true
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
	if raw := SoupFromMesh(ipt.GraphicsMesh(d)); raw.TriangleCount() >= 4 {
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
