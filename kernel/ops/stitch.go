// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	"sort"

	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// defaultStitchTolerance is the coincidence grid used when a feature passes 0.
const defaultStitchTolerance = 1e-7

// Stitch welds independently-built surface bodies into one quilt by exact-coincidence
// matching: vertices that quantize to the same tolerance grid cell become one, and
// the boundary edges they share are merged so each is used by both faces. When every
// edge ends up used by exactly two faces the quilt is closed, and (unless
// maintainSurface) the result is a solid — this is the "stitch closed surfaces → a
// solid" path. Tolerant matching of near-but-not-coincident edges (a real gap within
// a tolerance band) is the phase-D tolerant-topology case, still deferred (see Sew).
func Stitch(bodies []*topo.Body, tolerance float64, maintainSurface bool, feat string) (*topo.Body, error) {
	if len(bodies) == 0 {
		return nil, errors.New("stitch: no surface bodies to stitch")
	}
	tol := tolerance
	if tol <= 0 {
		tol = defaultStitchTolerance
	}
	w := newWeld(tol)
	for _, b := range bodies {
		for _, f := range b.Faces() {
			w.addFace(f)
		}
	}
	return w.build(maintainSurface, feat), nil
}

// vKey is a vertex position quantized to the coincidence grid; eKey is an undirected
// edge keyed by its (canonically ordered) endpoint cells.
type (
	vKey [3]int64
	eKey [2]vKey
)

// weld accumulates the coincidence-welded vertices, edges and faces to rebuild.
type weld struct {
	tol    float64
	points map[vKey]math.Point3
	curves map[eKey]geom.Curve3
	uses   map[eKey]int
	faces  []weldedFace
}

type weldedFace struct {
	surface geom.Surface
	lineage topo.Lineage
	loops   []weldedLoop
}
type weldedLoop struct {
	outer bool
	uses  []weldedUse
}
type weldedUse struct {
	key      eKey
	reversed bool // traversal runs against the canonical (a→b) edge direction
}

func newWeld(tol float64) *weld {
	return &weld{tol: tol, points: map[vKey]math.Point3{}, curves: map[eKey]geom.Curve3{}, uses: map[eKey]int{}}
}

// addFace captures a face (surface + lineage + welded loops) for rebuild.
func (w *weld) addFace(f *topo.Face) {
	wf := weldedFace{surface: f.Geometry(), lineage: f.Lineage()}
	for _, l := range f.Loops() {
		wf.loops = append(wf.loops, w.addLoop(l))
	}
	w.faces = append(w.faces, wf)
}

func (w *weld) addLoop(l *topo.Loop) weldedLoop {
	wl := weldedLoop{outer: l.IsOuter()}
	for _, u := range l.EdgeUses() {
		wl.uses = append(wl.uses, w.addUse(u))
	}
	return wl
}

// addUse registers an edge use, welding its endpoints/curve and counting the use so
// closedness (every edge used twice) can be decided.
func (w *weld) addUse(u *topo.EdgeUse) weldedUse {
	e := u.Edge()
	sk, ek := w.record(e)
	from, to := sk, ek
	if u.Reversed() {
		from, to = ek, sk
	}
	key, reversed := canonical(from, to)
	w.uses[key]++
	return weldedUse{key: key, reversed: reversed}
}

// record welds an edge's endpoint positions and stores a representative curve.
func (w *weld) record(e *topo.Edge) (vKey, vKey) {
	sk := w.vkey(e.StartVertex().Point())
	ek := w.vkey(e.EndVertex().Point())
	w.points[sk] = e.StartVertex().Point()
	w.points[ek] = e.EndVertex().Point()
	key, _ := canonical(sk, ek)
	if _, ok := w.curves[key]; !ok {
		w.curves[key] = e.Geometry()
	}
	return sk, ek
}

func (w *weld) vkey(p math.Point3) vKey { return vKey{w.q(p.X), w.q(p.Y), w.q(p.Z)} }
func (w *weld) q(v float64) int64       { return int64(stdmath.Round(v / w.tol)) }

// isClosed reports whether every welded edge is used by exactly two faces.
func (w *weld) isClosed() bool {
	if len(w.uses) == 0 {
		return false
	}
	for _, c := range w.uses {
		if c != 2 {
			return false
		}
	}
	return true
}

// build reconstructs the single welded body (solid iff closed and not held as a
// surface), preserving each source face's lineage so reference keys still resolve.
func (w *weld) build(maintainSurface bool, feat string) *topo.Body {
	solid := w.isClosed() && !maintainSurface
	bld := topo.NewBuilder(solid, topo.NewLineage(topo.Tok(feat, "body", 0)))
	verts := w.buildVertices(bld, feat)
	edges := w.buildEdges(bld, verts, feat)
	for _, wf := range w.faces {
		bld.AddFace(wf.surface, wf.lineage, w.faceLoops(wf, edges)...)
	}
	return bld.Build()
}

// buildVertices creates one shared vertex per welded cell, in sorted-key order so
// the synthesized lineage (and thus reference keys) is stable across recompute.
func (w *weld) buildVertices(bld *topo.Builder, feat string) map[vKey]*topo.Vertex {
	keys := make([]vKey, 0, len(w.points))
	for k := range w.points {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return lessV(keys[i], keys[j]) })
	out := make(map[vKey]*topo.Vertex, len(keys))
	for i, k := range keys {
		out[k] = bld.AddVertex(w.points[k], topo.NewLineage(topo.Tok(feat, "weld-vertex", i)))
	}
	return out
}

// buildEdges creates one shared edge per welded cell pair, in sorted-key order.
func (w *weld) buildEdges(bld *topo.Builder, verts map[vKey]*topo.Vertex, feat string) map[eKey]*topo.Edge {
	keys := make([]eKey, 0, len(w.curves))
	for k := range w.curves {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return lessE(keys[i], keys[j]) })
	out := make(map[eKey]*topo.Edge, len(keys))
	for i, k := range keys {
		out[k] = bld.AddEdge(w.curves[k], verts[k[0]], verts[k[1]], topo.NewLineage(topo.Tok(feat, "weld-edge", i)))
	}
	return out
}

// faceLoops turns a welded face's loops back into builder loop specs that reference
// the shared edges with the original traversal orientation.
func (w *weld) faceLoops(wf weldedFace, edges map[eKey]*topo.Edge) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, len(wf.loops))
	for i, wl := range wf.loops {
		uses := make([]topo.Use, len(wl.uses))
		for j, wu := range wl.uses {
			uses[j] = topo.Use{Edge: edges[wu.key], Reversed: wu.reversed}
		}
		if wl.outer {
			specs[i] = topo.OuterLoop(uses...)
		} else {
			specs[i] = topo.InnerLoop(uses...)
		}
	}
	return specs
}

// canonical orders an endpoint pair (a≤b) and reports whether the original from→to
// direction runs against that canonical order.
func canonical(a, b vKey) (eKey, bool) {
	if lessV(a, b) {
		return eKey{a, b}, false
	}
	return eKey{b, a}, true
}

func lessV(a, b vKey) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func lessE(a, b eKey) bool {
	if a[0] != b[0] {
		return lessV(a[0], b[0])
	}
	return lessV(a[1], b[1])
}
