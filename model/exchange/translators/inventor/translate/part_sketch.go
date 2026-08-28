// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"

	m "oblikovati.org/math"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// Inventor .ipt part translator — SKETCH TRANSLATION (M48 #2231 split of part.go). Extracting the
// decoded 2D sketches, resolving their placement plane, and emitting them (with their line entities) onto
// the document — deliberately independent of the feature that will consume them.

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
				if graph := ipt.GraphSketches(d); sketchEntityCount(graph) > 0 && (revolveBinds(graph) || graphRevolveCandidate(graph)) {
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

// planeIsXY reports whether a sketch plane is the world XY plane (its normal is ±Z), the case where
// addHole's top-face fallback places a hole correctly.
func planeIsXY(p sketch.Plane) bool {
	return math.Abs(float64(p.Normal().AsVector().Z)) > 0.999
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

// sketchEntityCount totals the curves and points across decoded sketches — the signal that a decode
// actually read geometry rather than producing empty shells.
func sketchEntityCount(sketches []ipt.Sketch) int {
	n := 0
	for _, s := range sketches {
		n += len(s.Points) + len(s.Lines) + len(s.Circles) + len(s.Arcs) + len(s.Ellipses) + len(s.Splines)
	}
	return n
}
