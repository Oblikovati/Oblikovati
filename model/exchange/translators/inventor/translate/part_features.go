// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"

	m "oblikovati.org/math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Inventor .ipt part translator — the PER-FEATURE-KIND builders (M48 #2231 split of part.go): extrude
// (the general extract-then-build path) plus mirror, rectangular/circular pattern, hole, loft and sweep.

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
	if h, ok := ipt.DecodeHole(d); ok {
		if len(extrudes) > 0 && len(placed) > 0 && len(emitted) > 0 && emitted[0].sk != nil {
			cx, cy := profileCentroid(placed[0].geom)
			addHole(def, h, placed[0].plane, cx, cy, extrudes[0].Distance)
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
func addHole(def *compdef.PartComponentDefinition, h ipt.Hole, plane sketch.Plane, cx, cy, thickness float64) {
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
	// A PLAIN drilled hole uses its OWN placement (its transform, decoded into h.Center/Axis): GeomFace
	// finds the bored face and Center pins the exact drill point on it, so the bore lands where the
	// file put it even when it is not centred on the base profile (CapstainNut's Ø1.7 bore sits at the
	// origin with a +X axis, off the hex centroid — the profile-centroid guess missed it entirely). A
	// plain bore is symmetric about its axis, so which face the placement resolves to is immaterial.
	//
	// The placement transform's translation is the hole's own coordinate-system ORIGIN, which equals
	// the drill point only when the hole is centred there (as CapstainNut's is); on a hole offset from
	// that datum (the generated box fixtures place a centred bore whose datum sits at a corner) it is
	// NOT the bore centre. So the placement is used only where the top-face fallback is itself
	// unreliable — a base sketch that is NOT the XY plane, the one case (CapstainNut, a hex extruded
	// along +X) the fallback cannot place. XY-base holes (every fixture, MainFrame, WheelSlider) keep
	// the fallback, which matches Inventor there. A plain bore is symmetric about its axis, so which
	// face the placement resolves to is immaterial; a directional counterbore/countersink is excluded.
	if h.Placed && h.Type == ipt.DrilledHole && !planeIsXY(plane) {
		center := m.P3(m.Scalar(h.Center[0]), m.Scalar(h.Center[1]), m.Scalar(h.Center[2]))
		hd.GeomFace = &topo.GeometricFaceRef{Centroid: center, Normal: m.Vector3{X: m.Scalar(h.Axis[0]), Y: m.Scalar(h.Axis[1]), Z: m.Scalar(h.Axis[2])}}
		hd.Center = &center
		return
	}
	// Fallback: drill on the base extrude's top face — the sketch-plane point (cx,cy) lifted into 3D
	// and advanced along the plane normal by the extrude thickness. Hardcoding z=thickness / +Z drilled
	// in empty space on any base sketch that was not the XY plane; this reduces to those old values on
	// an XY sketch.
	normal := plane.Normal()
	top := plane.ToModel(m.P2(m.Scalar(cx), m.Scalar(cy)))
	center := top.TranslateBy(normal.AsVector().Scale(m.Scalar(thickness)))
	hd.GeomFace = &topo.GeometricFaceRef{Centroid: center, Normal: normal.AsVector()}
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

// extrudeRegionAt returns the decoded region for extrude i, or nil when the decode produced none.
func extrudeRegionAt(regions [][]ipt.RegionLoop, i int) []ipt.RegionLoop {
	if i >= len(regions) {
		return nil
	}
	return regions[i]
}
