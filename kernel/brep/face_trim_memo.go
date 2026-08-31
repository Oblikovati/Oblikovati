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
// (ops.ReplaceFaceSurface, despite its name, rebuilds the whole body through topo.NewBuilder). So the
// development is a pure function of the face and cannot go stale under it. That is the same property
// topo.Face's pickTess and metricScaleMemo rest on, and this follows their contract: the payload is
// opaque to topo, it lives exactly as long as the face, and it is written on the same single-threaded
// classification path per body.
//
// The memo changes no verdict — it caches an input, not a decision — which
// TestTrimUVMemoChangesNoVerdict asserts directly by classifying with the memo cold and warm.

// faceTrimUV is one face flattened for repeated containment queries: the curvedFace itself, its loops
// already developed into (u,v), and the cast-axis choice those rings were developed for.
type faceTrimUV struct {
	face     curvedFace
	rings    [][]math.Point2 // one per loop, in f.loops order; nil when the face defers to the winding
	uPer     bool
	vPer     bool
	alongV   bool
	castable bool // false ⇒ no exterior axis to cast toward; the geodesic winding decides instead
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
// repeat per query.
func developFaceTrim(cf curvedFace) *faceTrimUV {
	m := &faceTrimUV{face: cf}
	if len(cf.loops) == 0 {
		return m
	}
	m.uPer, m.vPer = surfacePeriodic(cf.surface)
	m.alongV, m.castable = castAxis(cf.surface, m.uPer, m.vPer)
	if !m.castable {
		return m
	}
	for _, loop := range cf.loops {
		m.rings = append(m.rings, loopToUV(cf.surface, loop, m.uPer, m.vPer))
	}
	return m
}

// contains reports whether p (on the face's surface) lies within the trimmed region, from the
// developed rings. It is pointInTrimUV's decision with the projection already made.
func (m *faceTrimUV) contains(p math.Point3) bool {
	if len(m.face.loops) == 0 {
		return true // a boundary-less closed face (a whole sphere/torus) contains every surface point
	}
	if !m.castable {
		return pointInCurvedFace(m.face, p) // sphere / torus: no exterior axis; pole-free geodesic winding
	}
	up, vp := m.face.surface.ParamAt(p)
	return trimRingParity(m.rings, math.P2(up, vp), m.uPer, m.vPer, m.alongV)
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
