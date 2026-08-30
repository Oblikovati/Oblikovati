// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft feature — SECTION RESOLUTION and SKIN helpers (M48 #2240 split of sweep_loft.go). Resolves each
// loft section to its model loops and normal (a sketch loop, a point, or a model face's boundary), and
// skins/tubes the resolved loops into the loft tool body (with the hollow-by-cut inner-loop handling).
// The loft feature and collection live in loft.go.

// hollowByCut skins the outer body and cuts each bore out of it — the fallback for lofts with
// more than one hole, where a multiply-connected end cap can't be a simple annular strip. Each
// bore is extended past the body's end caps (extendEnds) so the through-cut is not coplanar. Guides
// shape the outer skin only (the bores skin unguided). rec collects the bore cuts' boolean-fallback
// diagnostics (#1601; nil discards).
func hollowByCut(outers [][]math.Point3, inners [][][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides, rec *diag.Recorder) (*topo.Body, error) {
	tool, err := skinLoops(outers, closed, feat, ends, guides)
	if err != nil {
		return nil, err
	}
	eps := 0.0
	if !closed {
		eps = loftOvershoot(outers)
	}
	for h := 0; h < numHoles(inners); h++ {
		ring := extendEnds(holeRing(inners, h), eps)
		hole, herr := skinLoops(ring, closed, feat+"-hole", ends, loftGuides{})
		if herr != nil {
			return nil, herr
		}
		if tool, err = ops.BooleanWithDiagnostics(ops.Cut, tool, hole, rec); err != nil {
			return nil, err
		}
	}
	return tool, nil
}

// holeRing collects hole h's loop from every section into one section sequence.
func holeRing(inners [][][]math.Point3, h int) [][]math.Point3 {
	ring := make([][]math.Point3, len(inners))
	for i := range inners {
		ring[i] = inners[i][h]
	}
	return ring
}

// resolveSections resolves each section into its outer loop, inner (hole) loops, and plane/face
// normal in model space. A section is a sketch profile, a point (apex; single-point loop, no
// holes, valid only at an end), or an existing body face (resolved against bodies — its boundary
// is the loop, its surface gives the normal for Tangent/Smooth). At least one section must be a
// real profile/face, and all sections must share their inner-loop count.
func (l *LoftFeature) resolveSections(bodies []*topo.Body) (outers [][]math.Point3, inners [][][]math.Point3, normals []math.UnitVector3, surfs []geom.Surface, err error) {
	if len(l.def.Sections) < 2 {
		return nil, nil, nil, nil, fmt.Errorf("loft: %d sections, need at least 2", len(l.def.Sections))
	}
	if err := l.validatePointSections(); err != nil {
		return nil, nil, nil, nil, err
	}
	for i, s := range l.def.Sections {
		outer, holes, n, surf, e := resolveSection(s, bodies)
		if e != nil {
			return nil, nil, nil, nil, fmt.Errorf("loft section %d: %w", i, e)
		}
		outers, inners, normals, surfs = append(outers, outer), append(inners, holes), append(normals, n), append(surfs, surf)
	}
	for i, h := range inners {
		if len(h) != len(inners[0]) {
			return nil, nil, nil, nil, fmt.Errorf("loft: section %d has %d holes, want %d (hole counts must match across sections; a point section cannot pair with a hollow one)", i, len(h), len(inners[0]))
		}
	}
	return outers, inners, normals, surfs, nil
}

// resolveSection resolves one section's outer loop, inner (hole) loops, normal, and — for a face
// section — the face's surface (nil otherwise; the skinner reads it for real face continuity).
func resolveSection(s LoftSection, bodies []*topo.Body) ([]math.Point3, [][]math.Point3, math.UnitVector3, geom.Surface, error) {
	switch {
	case s.IsPoint():
		return []math.Point3{*s.Point}, nil, sectionNormal(s), nil, nil
	case s.IsFace():
		f, ok := findFace(bodies, s.FaceKey)
		if !ok {
			return nil, nil, math.UnitVector3{}, nil, fmt.Errorf("face reference is lost (no running body has it)")
		}
		outer, holes := faceLoopsModel(f)
		if len(outer) < 3 {
			return nil, nil, math.UnitVector3{}, nil, fmt.Errorf("face has a degenerate boundary (%d points)", len(outer))
		}
		return outer, holes, faceNormal(f, outer), f.Geometry(), nil
	default:
		prof, e := resolveSingleProfile(s.Sketch, s.ProfileIndex, "loft")
		if e != nil {
			return nil, nil, math.UnitVector3{}, nil, e
		}
		outer := loopToModel(prof.OuterLoop(), s.Sketch.Plane())
		var holes [][]math.Point3
		for _, il := range prof.InnerLoops() {
			holes = append(holes, loopToModel(il, s.Sketch.Plane()))
		}
		return outer, holes, s.Sketch.Plane().Normal(), nil, nil
	}
}

// sectionNormal is a sketch/point section's plane normal — for a point (apex) section the tangent
// plane a TangentToPlane condition domes against. Falls back to +Z for a bare 3D point.
func sectionNormal(s LoftSection) math.UnitVector3 {
	if s.Sketch != nil {
		return s.Sketch.Plane().Normal()
	}
	return math.V3(0, 0, 1).AsUnit()
}

// findFace resolves a face reference key against the running bodies (persistent naming).
func findFace(bodies []*topo.Body, key []byte) (*topo.Face, bool) {
	for _, b := range bodies {
		if f, ok := FindOrRecoverFace(b, key); ok {
			return f, true
		}
	}
	return nil, false
}

// faceLoopsModel returns a face's outer boundary loop and its inner (hole) loops as ordered
// model-space polygons (the "from" vertex of each oriented edge use).
func faceLoopsModel(f *topo.Face) (outer []math.Point3, inners [][]math.Point3) {
	for _, l := range f.Loops() {
		poly := loopUseStarts(l)
		if l.IsOuter() {
			outer = poly
		} else if len(poly) >= 3 {
			inners = append(inners, poly)
		}
	}
	return outer, inners
}

// loopUseStarts is the loop's boundary as an ordered point ring, each oriented edge sampled in
// traversal order. A straight edge contributes just its start point (the polygon corner); a curved
// edge (a circle/arc — e.g. the rim of an analytic cylinder cap) is sampled into many points so the
// ring is a real polygon, not a single vertex. The closing point is dropped (it equals the next
// edge's start).
func loopUseStarts(l *topo.Loop) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		c := u.Edge().Geometry()
		lo, hi := c.Domain()
		n := edgeRingSamples(c)
		for i := range n { // [0,n): exclude the endpoint shared with the next edge's start
			f := float64(i) / float64(n)
			t := lo + (hi-lo)*f
			if u.Reversed() {
				t = hi - (hi-lo)*f
			}
			pts = append(pts, c.PointAt(t))
		}
	}
	return pts
}

// edgeRingSamples is how many points to sample an edge into for a loop ring: 1 for a straight
// segment (the start corner), more for a curved edge so it reads as a polygon.
func edgeRingSamples(c geom.Curve3) int {
	if _, ok := c.(geom.LineSegment); ok {
		return 1
	}
	return 48
}

// faceNormal is the source face's surface normal used to aim a Tangent/Smooth takeoff: exact for
// a planar face (its plane normal), otherwise the boundary's best-fit (Newell) normal — so face
// continuity is exact for planar source faces and a sensible approximation for curved ones. Sign
// is irrelevant (the takeoff is re-oriented outward by the skinner).
func faceNormal(f *topo.Face, outer []math.Point3) math.UnitVector3 {
	if pl, ok := f.Geometry().(geom.Plane); ok {
		return pl.Normal().AsUnit()
	}
	return boundaryNormal(outer)
}

// boundaryNormal is the Newell normal of a (near-)planar polygon (+Z when degenerate).
func boundaryNormal(poly []math.Point3) math.UnitVector3 {
	var n math.Vector3
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		n = n.Add(math.V3((a.Y-b.Y)*(a.Z+b.Z), (a.Z-b.Z)*(a.X+b.X), (a.X-b.X)*(a.Y+b.Y)))
	}
	if n.Length() < 1e-12 {
		return math.V3(0, 0, 1).AsUnit()
	}
	return n.AsUnit()
}

// numHoles returns the per-section inner-loop count (all sections share it).
func numHoles(inners [][][]math.Point3) int {
	if len(inners) == 0 {
		return 0
	}
	return len(inners[0])
}

// loopToModel maps a sketch loop's polygon into model space on the sketch plane.
func loopToModel(loop sketch.Loop, plane sketch.Plane) []math.Point3 {
	poly := loop.Polygon()
	out := make([]math.Point3, len(poly))
	for i, p := range poly {
		out[i] = plane.ToModel(p)
	}
	return out
}

// skinLoops resamples a set of section loops to a common point count, corresponds them
// (minimize twist), blends them with a Catmull-Rom spline (a loft is not a straight blend),
// and meshes the swept solid. See loft_skin.go.
func skinLoops(loops [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	return sweptSolid(skinnedSections(loops, maxLoopCount(loops), closed, ends, guides), closed, feat)
}

// tubeLoops skins corresponding outer and inner section loops to a common point count, then
// meshes them directly into a hollow tube. Outer and inner share that point count so their
// rings pair up across the annular end caps; both run through the same correspondence + spline
// blend, so the pipe wall reads as smooth as a solid loft. Guides shape the outer wall only.
func tubeLoops(outers, inners [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeSolid(skinnedSections(outers, n, closed, ends, guides), skinnedSections(inners, n, closed, ends, loftGuides{}), closed, feat)
}

// skinShell is skinLoops' open counterpart for the surface operation: the skinned sections meshed
// as an OPEN sheet (no end caps) via sweptShell (#1858).
func skinShell(loops [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	return sweptShell(skinnedSections(loops, maxLoopCount(loops), closed, ends, guides), closed, feat)
}

// tubeShellLoops is tubeLoops' open counterpart: the nested outer/inner skinned sections meshed as
// an open pipe surface (no annular end caps) via tubeShell (#1858).
func tubeShellLoops(outers, inners [][]math.Point3, closed bool, feat string, ends loftEnds, guides loftGuides) (*topo.Body, error) {
	n := maxLoopCount(outers, inners)
	return tubeShell(skinnedSections(outers, n, closed, ends, guides), skinnedSections(inners, n, closed, ends, loftGuides{}), closed, feat)
}

// skinnedSections resamples loops to n points, corresponds them (minimize twist), blends them with
// a tangent-driven Hermite spline (Catmull-Rom interior, end conditions at the ends), bends the
// spine to any centerline, then pulls the result toward any guide rails — the densified section
// sequence ready to mesh. See loft_skin.go.
func skinnedSections(loops [][]math.Point3, n int, closed bool, ends loftEnds, guides loftGuides) [][]math.Point3 {
	resampled := make([][]math.Point3, len(loops))
	for i, lp := range loops {
		resampled[i] = resampleLoop(lp, n)
	}
	aligned := mapAlign(resampled, guides.mapCurves)
	// A twisting loft's skin quads are a full section-edge wide, so they warp steeply and facet
	// even when finely sampled along the length. Subdivide each section edge (corner-preserving)
	// proportional to the twist, so the skin is narrow enough across its width to read smooth.
	// The wrap's correspondence is offset by the monodromy (a 180° half-twist returns shifted by
	// half the points); the around-subdivision and the spline blend both measure the wrap twist
	// against the start REINDEXED by that shift, else a Möbius/twisted closure reads its seam as a
	// ~180° twist and over-subdivides every section (the seam notch's cousin — a 12× mesh blow-up).
	shift := closureShift(aligned, closed)
	aligned = densifyAround(aligned, aroundSubdivisions(aligned, closed, shift))
	secs := splineSections(aligned, closed, ends, closureShift(aligned, closed))
	secs = areaGraphScale(secs, guides.areaGraph)
	secs = centerlineGuide(secs, guides.centerline)
	return railGuide(secs, guides.rails)
}

// maxLoopCount returns the largest point count across every loop in the given sets, the common
// resample resolution so the densest section is not coarsened.
func maxLoopCount(loopSets ...[][]math.Point3) int {
	n := 0
	for _, set := range loopSets {
		for _, lp := range set {
			if len(lp) > n {
				n = len(lp)
			}
		}
	}
	return n
}
