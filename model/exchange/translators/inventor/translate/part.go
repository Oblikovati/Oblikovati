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
// user parameters become Oblikovati parameters, and (interim) the solid body is
// reconstructed for parts we can rebuild. Sketches and the feature tree are the next
// decode stages. Returns any non-fatal translation warnings.
func FromInventor(iptBytes []byte, outPath string) ([]string, error) {
	d, err := ipt.Open(iptBytes)
	if err != nil {
		return nil, err
	}
	if d.IsAssembly() {
		return nil, assemblyError(d)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, warns, err := buildPart(ws, outPath, d)
	if err != nil {
		return warns, err
	}
	if err := ws.Save(document); err != nil {
		return warns, err
	}
	return warns, nil
}

// buildPart adds a part document to ws at outPath and populates it from the decoded .ipt —
// parameters, then the parametric feature tree (falling back to the ACIS body). It stops
// short of saving so callers (a standalone part, or an assembly component) control that.
func buildPart(ws *doc.Workspace, outPath string, d *ipt.Document) (*doc.Document, []string, error) {
	document, err := compdef.AddPart(ws, outPath, true)
	if err != nil {
		return nil, nil, err
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	warns := addParameters(def, d)
	built := addFeatures(def, d)
	def.Recompute()
	// Fall back to Inventor's stored display tessellation (curved faces + holes already meshed)
	// when the parametric path built nothing OR produced no solid — real interactively-modelled
	// parts routinely exceed the feature decode, and a static body beats an empty one.
	if !built || !hasSolidBody(def) {
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

// addFeatures decodes the part's sketches and features and builds them. Each extrude
// consumes its own sketch (matched by order); a revolve consumes the first sketch.
// Returns true if at least one feature was built. v1: extrude(distance,operation) and
// full-revolve; feature order is assumed to match sketch/parameter order.
func addFeatures(def *compdef.PartComponentDefinition, d *ipt.Document) bool {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return false
	}
	sketches := ipt.DecodeSketches(seg)
	// A loft blends 2+ profile sketches on parallel planes stacked along +Z; its sections need
	// custom plane placement, so build it before the on-XY emit path.
	if heights, ok := ipt.LoftSectionHeights(seg, len(sketches)); ok {
		if addLoft(def, sketches, heights) {
			return true
		}
	}
	// A sweep pushes a profile along a 3D path — the path is a curve, not an on-XY profile.
	if sw, ok := ipt.DecodeSweep(seg); ok {
		if addSweep(def, sw) {
			return true
		}
	}
	// A revolve is handled before the general emit path: it emits only the chosen profile + axis.
	// Build it only when RevolveProfile validates a CLOSED, one-sided profile about an unambiguous
	// axis; otherwise fall back to the mesh body (a merely-"Resolved" cluster can be an incomplete
	// open chain that would revolve to a wrong solid).
	if ipt.HasRevolve(seg) {
		// Reconstruct the profile from point incidence — exact connectivity that reunites a profile
		// split across the 800-byte cluster gap (the clustered DecodeSketches leaves those as open
		// chains). Fall back to the clustered sketches when incidence yields nothing (e.g. an
		// arc-bearing profile it declines).
		profiles := ipt.LineProfiles(seg)
		if len(profiles) == 0 {
			profiles = sketches
		}
		b, ok := ipt.RevolveProfile(profiles)
		if !ok {
			return false
		}
		var angle func() float64 // nil ⇒ full 360°
		if a, ok := ipt.RevolveAngle(seg); ok {
			a := a
			angle = func() float64 { return a }
		}
		return addRevolve(def, profiles, b, angle)
	}
	// Emit every decoded sketch first (a sketch-only part keeps its sketch even with no feature),
	// then build features that consume them by order.
	emitted := make([]*sketch.Sketch, len(sketches))
	for i := range sketches {
		emitted[i], _ = emitSketchOn(def, sketches[i], sketch.XYPlane())
	}
	built := false
	extrudes := ipt.DecodeExtrudes(seg)
	var lastExtrude *feature.PartFeature
	for i, ex := range extrudes {
		if i >= len(emitted) || emitted[i] == nil {
			continue
		}
		dist := ex.Distance
		lastExtrude = feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(emitted[i], 0, operationOf(ex.Operation), func() float64 { return dist })
		built = true
	}
	// A drilled hole cuts the base solid: place it on the extrude's top face (analytic),
	// drilling at the profile centroid. Needs the base extrude to have built the body first.
	if h, ok := ipt.DecodeHole(seg); ok && len(extrudes) > 0 && len(sketches) > 0 {
		cx, cy := profileCentroid(sketches[0])
		addHole(def, h, cx, cy, extrudes[0].Distance)
		built = true
	}
	// A pattern or mirror replicates the last feature. It must run after the source feature
	// has been added so its occurrences re-cut the running body. Rectangular (grid), circular
	// (about Z), and mirror are mutually exclusive per their node name.
	if rp, ok := ipt.DecodeRectPattern(seg); ok && lastExtrude != nil {
		addRectPattern(def, lastExtrude, rp)
		built = true
	} else if cp, ok := ipt.DecodeCircPattern(seg); ok && lastExtrude != nil {
		addCircPattern(def, lastExtrude, cp)
		built = true
	} else if mir, ok := ipt.DecodeMirror(d); ok && lastExtrude != nil {
		addMirror(def, lastExtrude, mir)
		built = true
	}
	return built
}

// addRevolve emits the chosen profile (and, for a separate-sketch axis, the axis sketch) and builds
// a revolve about the decoded centreline — the axisLine ipt.RevolveProfile validated as the axis of
// a closed one-sided profile, which may live in a different sketch than the profile. Revolving about
// that line (rather than the fixed X origin axis, which spun a mis-picked profile into a blob)
// reproduces the real solid of revolution. Returns false when the profile emits no geometry. angle
// is nil ⇒ full turn.
func addRevolve(def *compdef.PartComponentDefinition, sketches []ipt.Sketch, b ipt.RevolveBinding, angle func() float64) bool {
	profile, profLines := emitSketchOn(def, sketches[b.ProfileSketch], sketch.XYPlane())
	if profile == nil {
		return false
	}
	axisSk, axisLines := profile, profLines
	if b.AxisSketch != b.ProfileSketch {
		axisSk, axisLines = emitSketchOn(def, sketches[b.AxisSketch], sketch.XYPlane())
	}
	if b.AxisLine >= len(axisLines) || axisLines[b.AxisLine] == nil {
		return false
	}
	feature.NewRevolveFeatures(def.Features()).AddAboutCenterlineLine(profile, 0, axisSk, axisLines[b.AxisLine], angle, ops.NewBody)
	return true
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
func emitSketchOn(def *compdef.PartComponentDefinition, s ipt.Sketch, plane sketch.Plane) (*sketch.Sketch, []*sketch.Line) {
	if len(s.Points) == 0 && len(s.Lines) == 0 && len(s.Circles) == 0 {
		return nil, nil
	}
	sk := def.Sketches().Add(plane)
	for _, p := range s.Points {
		sk.Points().Add(m.P2(p.X, p.Y))
	}
	lines := make([]*sketch.Line, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = sk.Lines().AddByTwoPoints(m.P2(l.A.X, l.A.Y), m.P2(l.B.X, l.B.Y))
	}
	for _, c := range s.Circles {
		sk.Circles().AddByCenterRadius(m.P2(c.Center.X, c.Center.Y), m.Scalar(c.Radius))
	}
	return sk, lines
}

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
