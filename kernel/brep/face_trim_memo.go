// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The per-face memo of a trimmed face's boundary DEVELOPED into its surface's (u,v) domain
// (M48/C3, Oblikovati/Oblikovati#3477).
//
// WHY IT EXISTS. pointInTrimUV classifies a point by ray-casting against the face's loops projected
// into (u,v), and building that projection walks every loop edge at trimUVSamples stations, inverting
// each station through geom.Surface.ParamAt. For a B-spline surface ParamAt is a multistart
// nearest-seed search, so one containment query on a four-edge patch costs ~128 NURBS inversions — and
// the query is asked once per point. Nothing about the projection depends on the point, so every call
// after the first was repeating work that was never needed twice.
//
// It is not a micro-optimization: brep.EntityDistance minimises a point-to-face distance OVER a curve,
// so it pays that cost per evaluation of the objective, inside a minimiser, inside the face-face
// recursion. Measured on the OCCT blend-parity corpus, a self-intersection scan that called
// EntityDistance on B-spline face pairs did not finish in ten minutes with the whole scan doing no
// surface-surface marching at all; the goroutine dump sat in unwrapLoopRing → ParamAt → nearestSeed.
//
// WHY IT IS SAFE. A topo.Face is immutable once its body is built: the loops are written only by
// topo's builder, and every operation that changes geometry BUILDS NEW FACES rather than mutating them
// (transform.ReplaceFaceSurface, despite its name, rebuilds the whole body through topo.NewBuilder). So the
// development is a pure function of the face and cannot go stale under it. That is the same property
// topo.Face's pickTess and metricScaleMemo rest on, and this follows their contract: the payload is
// opaque to topo, it lives exactly as long as the face, and it is written on the same single-threaded
// classification path per body.
//
// The memo changes no verdict — it caches an input, not a decision — which
// TestTrimUVMemoChangesNoVerdict asserts directly by classifying with the memo cold and warm.

// faceTrimUV is one face flattened for repeated containment queries: the curvedFace itself, its loops
// already developed into (u,v), the cast-axis choice those rings were developed for, and which side of
// the rings the face is.
type faceTrimUV struct {
	face       curvedFace
	rings      [][]math.Point2 // one per loop, in f.loops order; nil when the face defers to the foot test
	uPer       bool
	vPer       bool
	alongV     bool
	castable   bool // false ⇒ no exterior axis to cast toward: a sphere, a torus
	ringsBound bool // the rings close in the covering plane AND nest, so they bound one region there
	complement bool // the face is the region OUTSIDE its rings (an outerless closed-surface face)
}

// faceTrimUVOf returns f's development, building it on first use and reusing it thereafter.
//
// Example:
//
//	m := faceTrimUVOf(f)
//	inside := m.contains(p)
func faceTrimUVOf(f *topo.Face) *faceTrimUV {
	if m, ok := f.TrimUVMemo().(*faceTrimUV); ok && m != nil {
		return m
	}
	m := developFaceTrim(curvedFaceOf(f))
	f.SetTrimUVMemo(m)
	return m
}

// developFaceTrim projects a curved face's loops into (u,v) once — the work pointInTrimUV used to
// repeat per query — and settles, also once, which side of those rings the face is.
func developFaceTrim(cf curvedFace) *faceTrimUV {
	m := &faceTrimUV{face: cf}
	if len(cf.loops) == 0 {
		return m
	}
	m.uPer, m.vPer = surfacePeriodic(cf.surface)
	m.alongV, m.castable = castAxis(cf.surface, m.uPer, m.vPer)
	for _, loop := range cf.loops {
		m.rings = append(m.rings, loopToUV(cf.surface, loop, m.uPer, m.vPer))
	}
	if !m.castable {
		m.alongV = true // no axis reaches an exterior, but a ring that CLOSES is left by either ray
		m.ringsBound = ringsBoundOneRegion(m.rings, m.uPer, m.vPer)
		m.complement = cf.outerless
	}
	return m
}

// contains reports whether p (on the face's surface) lies within the trimmed region, from the
// developed rings. It is pointInTrimUV's decision with the projection already made.
//
// ★ ON A CLOSED SURFACE IT DOES NOT READ THE LOOPS' HANDEDNESS WHERE THE RINGS ALREADY BOUND A REGION
// (Oblikovati/Oblikovati#3477). A sphere or a torus has no parameter axis reaching an exterior, so the
// rings' interior and its complement are both admissible faces, and the classifier used to pick between
// them from the loop's traversal direction ([pointInCurvedFace]). That direction is NOT a carrier this
// kernel maintains — kernel/ops/analytic_face_region.go says so of the same corpus ("a producer may wind
// a closed-surface face's loops either way … every torus band in the corpus comes out clockwise whichever
// side it covers") — and an OCCT-parity fillet body (corpus simple/B3) and a blend body (bfuseblend/A6)
// both carry torus and sphere PATCHES wound clockwise while the face is the rings' INTERIOR. Every such
// face claimed the far side of its own surface, so ops.SelfIntersections found "crossings" at points
// outside both faces' range boxes and refused a genuine watertight solid.
//
// Rings that CLOSE in the covering plane and NEST bound exactly one region there, and that region is the
// face — the same orientation-free even-odd reading every open domain already takes here, with
// [curvedFace.outerless] (a topological datum topo carries on the loop, recorded by the builder that
// wound it) naming the one case where the face is their complement. A ring system that bounds no such
// region — a ring wrapping a whole period, or two DISJOINT rings, where the face is genuinely either of
// two two-sided regions (a sphere zone: the belt, or the two caps its rims equally bound) — has nothing
// but the winding to go on, and keeps reading it.
func (m *faceTrimUV) contains(p math.Point3) bool {
	if len(m.face.loops) == 0 {
		return true // a boundary-less closed face (a whole sphere/torus) contains every surface point
	}
	if !m.castable && !m.ringsBound {
		return pointInCurvedFace(m.face, p)
	}
	up, vp := m.face.surface.ParamAt(p)
	return trimRingParity(m.rings, math.P2(up, vp), m.uPer, m.vPer, m.alongV) != m.complement
}

// ringsBoundOneRegion reports whether the developed rings enclose exactly one region of the covering
// plane: every ring returns to its own start (rather than circling a periodic axis a full turn, which
// leaves an open polyline enclosing nothing), and ONE of them contains all the rest (an outer ring with
// its holes). Two disjoint rings fail it — they bound two regions and their complement equally, and only
// the winding says which the face is.
func ringsBoundOneRegion(rings [][]math.Point2, uPer, vPer bool) bool {
	for _, ring := range rings {
		if len(ring) < 3 {
			return false
		}
		if uPer && ringMissingTurns(ring, false) != 0 {
			return false
		}
		if vPer && ringMissingTurns(ring, true) != 0 {
			return false
		}
	}
	return len(rings) > 0 && ringsHaveOneContainer(rings)
}

// ringsHaveOneContainer reports whether some ring holds every other ring inside it.
func ringsHaveOneContainer(rings [][]math.Point2) bool {
	for i, outer := range rings {
		if ringHoldsTheRest(rings, i, outer) {
			return true
		}
	}
	return false
}

// ringHoldsTheRest reports whether ring outer contains one vertex of every other ring — enough, since
// trim rings of one face do not cross.
func ringHoldsTheRest(rings [][]math.Point2, skip int, outer []math.Point2) bool {
	for j, ring := range rings {
		if j != skip && !pointInPolygon2D(ring[0], outer) {
			return false
		}
	}
	return true
}

// trimRingParity is the even-odd verdict over already-developed loop rings — the one place the
// crossing count is turned into a containment answer, shared by the memoized and un-memoized paths so
// they cannot drift apart.
func trimRingParity(rings []([]math.Point2), q math.Point2, uPer, vPer, alongV bool) bool {
	total := 0
	for _, poly := range rings {
		if len(poly) < 2 {
			continue
		}
		total += loopRayCrossings(q, poly, uPer, vPer, alongV)
	}
	return total%2 == 1
}
