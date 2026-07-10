// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Two more associative sketch-curve sources (#1873), alongside the edge/vertex/work-datum sources
// (reference_source.go / work_reference_source.go): the section curves where the sketch plane cuts
// a solid ("project cut edges") and a face's silhouette on the sketch plane ("project silhouette").
// Both re-run their kernel op against the part's current bodies on every read, so they recompute
// with the model like every other projected reference, and both bind to one sketch plane fixed at
// construction — matching model/sketch projection.go and WorkPlaneRefSource.

// CutEdgeRefSource adapts one section loop (the sketch plane × a part body) to a sketch curve
// source, keyed by body index + loop index so it re-associates to the same loop after recompute.
type CutEdgeRefSource struct {
	bodies    func() []*topo.Body
	plane     sketch.Plane
	bodyIndex int
	wireIndex int
}

// NewCutEdgeRefSource binds an associative curve source to the wireIndex-th section loop of the
// bodyIndex-th body cut by the sketch plane.
func NewCutEdgeRefSource(part *PartComponentDefinition, plane sketch.Plane, bodyIndex, wireIndex int) CutEdgeRefSource {
	return CutEdgeRefSource{
		bodies: func() []*topo.Body { return part.SurfaceBodies().All() },
		plane:  plane, bodyIndex: bodyIndex, wireIndex: wireIndex,
	}
}

// SourceID encodes the (body, loop) indices this source re-sections to.
func (s CutEdgeRefSource) SourceID() string { return fmt.Sprintf("%d:%d", s.bodyIndex, s.wireIndex) }

// SourceKind tags this as a cut-edge reference for projection persistence/rebind (#1873).
func (s CutEdgeRefSource) SourceKind() string { return "cutEdge" }

// SamplePoints re-sections the body by the sketch plane and samples the loop; ok=false when the
// body or loop is gone (the plane no longer cuts it there), which freezes the projection.
func (s CutEdgeRefSource) SamplePoints() ([]math.Point3, bool) {
	bodies := s.bodies()
	if s.bodyIndex < 0 || s.bodyIndex >= len(bodies) {
		return nil, false
	}
	wires, ok := sectionWires(bodies[s.bodyIndex], s.plane)
	if !ok || s.wireIndex < 0 || s.wireIndex >= len(wires) {
		return nil, false
	}
	return sampleWireCurve(wires[s.wireIndex]), true
}

// CutEdgeSources sections every part body by the sketch plane and returns one associative curve
// source per section loop, in body-then-loop order — the supply for [sketch.Sketch.ProjectCutEdges]
// (#1873). Empty when the plane misses every body.
func (d *PartComponentDefinition) CutEdgeSources(plane sketch.Plane) []sketch.CurveSource {
	var out []sketch.CurveSource
	for bi, b := range d.SurfaceBodies().All() {
		wires, ok := sectionWires(b, plane)
		if !ok {
			continue
		}
		for wi := range wires {
			out = append(out, NewCutEdgeRefSource(d, plane, bi, wi))
		}
	}
	return out
}

// sectionWires sections one body by the sketch plane and returns its section wires; ok=false when
// the plane misses the body or the section is degenerate.
func sectionWires(b *topo.Body, plane sketch.Plane) ([]*topo.Wire, bool) {
	sec, err := ops.SectionWithPlane(b, plane.Origin(), plane.Normal().AsVector(), ops.DefaultQuality())
	if err != nil {
		return nil, false
	}
	return sec.Wires(), true
}

// SilhouetteRefSource adapts a face's silhouette on the sketch plane to a curve source: it
// re-resolves the face by reference key, re-traces its silhouette along the plane normal, and
// returns the loop nearest the proximity point.
type SilhouetteRefSource struct {
	ref             string
	bodies          func() []*topo.Body
	plane           sketch.Plane
	proximity       math.Point3
	includeBoundary bool
}

// NewSilhouetteRefSource binds an associative curve source to the silhouette of the face with
// reference key ref, viewed along plane's normal; proximity selects the loop when several exist.
func NewSilhouetteRefSource(part *PartComponentDefinition, ref string, plane sketch.Plane, proximity math.Point3, includeBoundary bool) SilhouetteRefSource {
	return SilhouetteRefSource{
		ref:    ref,
		bodies: func() []*topo.Body { return part.SurfaceBodies().All() },
		plane:  plane, proximity: proximity, includeBoundary: includeBoundary,
	}
}

// SourceKind tags this as a silhouette reference for projection persistence/rebind (#1873).
func (s SilhouetteRefSource) SourceKind() string { return "silhouette" }

// SourceID packs everything needed to rebuild the source after a reload: the include-boundary
// flag, the proximity point, and the face key last — so a key containing the '|' delimiter
// survives a SplitN.
func (s SilhouetteRefSource) SourceID() string {
	return fmt.Sprintf("%t|%g|%g|%g|%s", s.includeBoundary,
		float64(s.proximity.X), float64(s.proximity.Y), float64(s.proximity.Z), s.ref)
}

// SamplePoints re-resolves the face, re-traces its silhouette along the sketch-plane normal and
// samples the loop nearest the proximity point; ok=false when the face or its silhouette is gone.
func (s SilhouetteRefSource) SamplePoints() ([]math.Point3, bool) {
	face, ok := s.face()
	if !ok {
		return nil, false
	}
	sil, err := ops.FaceSilhouetteWires(face, s.plane.Normal().AsVector(), s.includeBoundary, ops.DefaultQuality())
	if err != nil {
		return nil, false
	}
	w := nearestWire(sil.Wires(), s.proximity)
	if w == nil {
		return nil, false
	}
	return sampleWireCurve(w), true
}

// face re-resolves the silhouette's face by reference key among the part's current bodies.
func (s SilhouetteRefSource) face() (*topo.Face, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if f, ok := feature.FindOrRecoverFace(b, key); ok {
			return f, true
		}
	}
	return nil, false
}

// SilhouetteSource builds the associative silhouette curve source for the face with reference key
// faceRef and reports whether it currently yields geometry (the face resolves and has a silhouette
// on this plane), so the router can reject a dead pick cleanly (#1873).
func (d *PartComponentDefinition) SilhouetteSource(faceRef string, plane sketch.Plane, proximity math.Point3, includeBoundary bool) (sketch.CurveSource, bool) {
	src := NewSilhouetteRefSource(d, faceRef, plane, proximity, includeBoundary)
	_, ok := src.SamplePoints()
	return src, ok
}

// sampleWireCurve samples every edge of a section/silhouette wire into a single model-space
// polyline (mirrors the router's wire-payload sampling, but yields points for projection).
func sampleWireCurve(w *topo.Wire) []math.Point3 {
	var pts []math.Point3
	for _, u := range w.Uses() {
		pts = append(pts, sampleUseCurve(u)...)
	}
	return pts
}

// sampleUseCurve samples one edge use over its domain into referenceSampleSteps+1 points,
// honouring the use's direction so consecutive edges chain head-to-tail.
func sampleUseCurve(u topo.Use) []math.Point3 {
	c := u.Edge.Geometry()
	lo, hi := c.Domain()
	out := make([]math.Point3, referenceSampleSteps+1)
	for i := range out {
		t := float64(i) / float64(referenceSampleSteps)
		if u.Reversed {
			t = 1 - t
		}
		out[i] = c.PointAt(lo + (hi-lo)*t)
	}
	return out
}

// nearestWire returns the wire whose sampled polyline passes closest to p — how a silhouette pick
// selects among several loops (Inventor's ProximityPoint). nil for an empty set.
func nearestWire(wires []*topo.Wire, p math.Point3) *topo.Wire {
	var best *topo.Wire
	bestD := stdmath.Inf(1)
	for _, w := range wires {
		if d := wireDistanceTo(w, p); d < bestD {
			best, bestD = w, d
		}
	}
	return best
}

// wireDistanceTo returns the minimum distance from p to the wire's sampled polyline.
func wireDistanceTo(w *topo.Wire, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, q := range sampleWireCurve(w) {
		if d := float64(p.DistanceTo(q)); d < best {
			best = d
		}
	}
	return best
}

// parseCutEdgeSourceID splits a cut-edge SourceID ("<body>:<loop>") back into its indices.
func parseCutEdgeSourceID(id string) (bodyIndex, wireIndex int, ok bool) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	bi, err1 := strconv.Atoi(parts[0])
	wi, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return bi, wi, true
}

// parseSilhouetteSourceID unpacks a silhouette SourceID ("<incl>|<px>|<py>|<pz>|<faceKey>").
func parseSilhouetteSourceID(id string) (faceRef string, proximity math.Point3, includeBoundary, ok bool) {
	parts := strings.SplitN(id, "|", 5)
	if len(parts) != 5 {
		return "", math.Point3{}, false, false
	}
	incl, err0 := strconv.ParseBool(parts[0])
	x, err1 := strconv.ParseFloat(parts[1], 64)
	y, err2 := strconv.ParseFloat(parts[2], 64)
	z, err3 := strconv.ParseFloat(parts[3], 64)
	if err0 != nil || err1 != nil || err2 != nil || err3 != nil {
		return "", math.Point3{}, false, false
	}
	return parts[4], math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)), incl, true
}
