// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// NurbsPcurveMesh meshes a B-spline face from its reconstructed pcurves (M25 F01) with a METRIC-AWARE
// triangulation. The over-enclosure of every prior attempt was folded triangles, not bad geometry
// (every sampled point lies exactly on OCC's surface): a plain (u,v) Delaunay twists in 3D because u
// and v have very different 3D scales (anisotropy). So the boundary pcurve + deflection-adaptive
// interior nodes are triangulated in METRIC-SCALED (u,v) — each axis scaled by its mean 3D length
// (√E, √G), making the parameter space ≈ isometric to 3D, so the Delaunay yields well-shaped 3D
// triangles. Points lift via PointAt (exact forward eval, no projection); the boundary keeps the exact
// 3D edge points (watertight). Returns nil to fall back.
func NurbsPcurveMesh(f *topo.Face, q Quality) *Mesh {
	s, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return nil
	}
	outerUV, outer3D, holesUV, holes3D := facePcurveLoops(f, q)
	if len(outer3D) < 3 {
		return nil
	}
	su, sv := faceMetricScale(f) // #2010: memoized off the per-pick hot path (metricScale is pure in the surface)
	return FoldDrivenPatch(s, su, sv, q, outer3D, outerUV, holes3D, holesUV)
}

// faceMetricScaleMemo is the ops-owned payload of topo.Face.metricScaleMemo: the face surface's cached
// per-axis (u,v) metric (√E,√G). Distinct name from the topo field it backs, mirroring the
// facePickTess/pickTess precedent. See faceMetricScale.
type faceMetricScaleMemo struct{ su, sv float64 }

// faceMetricScale returns metricScale of the face's surface, memoized on the face for its lifetime.
// metricScale is a pure function of the immutable surface derivatives (~25 DerivativesAt evals per
// call), so caching it is do-no-harm by construction (byte-identical su,sv) while removing the
// #2010 per-frame cost the interactive edge pick paid via starvedEdgeTarget → metricScale (edge
// picking is not covered by the pickTess whole-face memo). Mirrors pickFaceMesh (pick.go).
func faceMetricScale(f *topo.Face) (su, sv float64) {
	if c, ok := f.MetricScaleMemo().(faceMetricScaleMemo); ok {
		return c.su, c.sv
	}
	su, sv = MetricScale(f.Geometry())
	f.SetMetricScaleMemo(faceMetricScaleMemo{su: su, sv: sv})
	return su, sv
}

// AspectDensifyThreshold gates the #2009 starved-rail densification (densifyStarvedRail,
// edge_discretize.go) to strongly anisotropic panels only: the model-relative face aspect
// A = max(len_u,len_v)/min(len_u,len_v), where len_u/len_v are metric-scaled (metricScale × domain
// span) — the SAME per-axis 3D scale the CDT already triangulates in. Below this, starvedEdgeTarget
// returns h=0 and discretizeEdge is byte-identical to the pre-#2009 path. Recon's do-no-harm
// discriminant (aspect-mesh-recon-report.md §4): T6's obstacle patch measures A=2.13 (stays off);
// the U4 sliver that motivated this fix measures A=15.19 (fires). 4 sits with comfortable margin on
// both sides.
const AspectDensifyThreshold = 4.0

// kBoundaryCells ties a densified rail's target segment length h to the interior grid's own cell
// budget (maxInteriorCells, nurbs_interior.go): h = min(len_u,len_v)/kBoundaryCells makes a
// densified rail's segments comparable in size to an interior grid cell, so the metric CDT sees a
// well-graded point set instead of a 2-vertex rail forced against a saturated interior column
// (root cause, recon §2: a giant off-chord "fan" triangle spanning the whole starved rail).
const kBoundaryCells = 20.0

// starvedRailTarget returns the target segment length h for densifying a high-aspect panel's
// starved straight rails (#2009), or 0 when the face's aspect is at/below aspectDensifyThreshold —
// the do-no-harm gate.
func starvedRailTarget(s geom.Surface, su, sv float64) float64 {
	if FaceAspect(s, su, sv) <= AspectDensifyThreshold {
		return 0
	}
	lenU, lenV := axisExtents(s, su, sv)
	return stdmath.Min(lenU, lenV) / kBoundaryCells
}

// FaceAspect returns the model-relative aspect ratio of a surface's (clamped) domain, judged in
// METRIC-SCALED 3D extents (su,sv from metricScale × domain span) rather than raw parameter units —
// so a cone's angular u (which spans a small raw range but a large 3D circumference) is compared
// fairly against its axial v.
func FaceAspect(s geom.Surface, su, sv float64) float64 {
	lenU, lenV := axisExtents(s, su, sv)
	if lenU <= 0 || lenV <= 0 {
		return 1
	}
	if lenU > lenV {
		return lenU / lenV
	}
	return lenV / lenU
}

// axisExtents returns the metric-scaled 3D length of the surface's (clamped) u and v domain spans:
// su/sv (mean |∂P/∂u|, |∂P/∂v|, from metricScale) times the domain's parameter extent.
func axisExtents(s geom.Surface, su, sv float64) (lenU, lenV float64) {
	ulo, uhi := clampSpan(s.UDomain())
	vlo, vhi := clampSpan(s.VDomain())
	return su * (uhi - ulo), sv * (vhi - vlo)
}

// densifyStarvedRail extends discretizeEdge's chord-sagitta output for an edge that is (a) reduced
// to just its two endpoints — the documented contract for a straight/near-straight curve
// (edge_discretize.go: "A straight edge yields just its two endpoints") — and (b) shared by a
// high-aspect B-spline face (#2009): such a face's metric CDT would otherwise be forced to fan ONE
// giant off-chord triangle across the starved rail against its saturated interior grid (root cause,
// aspect-mesh-recon-report.md §2: +216% area on the U4 sliver, never converging as tolerance
// tightens because the missing vertices are on the CONSTRAINED BOUNDARY). An edge discretizeEdge
// already sampled to >2 points (curved, already sagitta-adaptive) is returned UNCHANGED.
//
// CALLER-INDEPENDENCE (the watertightness proof). This is a pure function of the edge's own
// topology (starvedEdgeTarget scans e.Faces(), never a caller-supplied face), so EVERY caller of
// discretizeEdge — the high-aspect B-spline face's own concatLoopPcurve AND any lower-aspect or
// even non-B-spline (cone/cylinder/plane) NEIGHBOUR face's loopBoundary — computes the IDENTICAL
// denser polyline for this edge. Both sides of a shared rail therefore stay in TRUE
// per-triangle-edge conformance (meshOpenEdges/freeEdgeCount, the codebase's existing strict
// watertightness bar — watertight_test.go, fillet_test.go's assertWatertight — stay at 0 free
// edges), not merely a geometric "no gap" argument.
//
// An earlier draft densified only inside the B-spline face's OWN loop assembly (a face-local
// post-step, gated on that face's own aspect) and argued "watertight by collinearity" — the
// inserted points lie exactly on the straight line a coarser neighbour already chords. That
// argument is geometrically true but insufficient: TestFilletRunOutToZero's meshOpenEdges gate
// (a plain box+run-out-fillet solid whose taper closes via a high-aspect B-spline cap, A=8.76)
// caught it directly — 177 open (non-2-incident) mesh edges, because the cap's cone/plane
// neighbours never saw the extra vertices. Routing the decision through discretizeEdge itself (so
// it is truly caller-independent) closes that gap; see aspect-mesh-fix-report.md for the full
// investigation.
func densifyStarvedRail(e *topo.Edge, pts []math.Point3) []math.Point3 {
	if len(pts) != 2 {
		return pts // curved: discretizeEdge already sampled it finer, never a candidate
	}
	h := starvedEdgeTarget(e)
	if h <= 0 {
		return pts
	}
	dense := DensifyStraightEdgeCurve(e, h)
	if len(dense) <= len(pts) {
		return pts // too short to need subdivision at this h
	}
	return dense
}

// starvedEdgeTarget returns edge e's #2009 densification target h, or 0 if none of e's faces is a
// high-aspect B-spline panel (aspectDensifyThreshold). When more than one qualifying face shares
// the edge, the SMALLEST (densest) target wins, so every qualifying face's CDT gets at least the
// resolution it needs. Scanning e.Faces() — not a caller-supplied face — is what makes
// densifyStarvedRail's decision independent of which face happens to be calling discretizeEdge.
func starvedEdgeTarget(e *topo.Edge) float64 {
	var h float64
	for _, f := range e.Faces() {
		s, ok := f.Geometry().(geom.BSplineSurface)
		if !ok {
			continue
		}
		su, sv := faceMetricScale(f) // #2010: THE per-pick hot site — memoize per face, don't re-evaluate every frame
		if fh := starvedRailTarget(s, su, sv); fh > 0 && (h == 0 || fh < h) {
			h = fh
		}
	}
	return h
}

// DensifyStraightEdgeCurve samples edge e's own curve into ⌈L3d/h⌉+1 evenly-parameterized points,
// start vertex → end vertex (discretizeEdge's natural order) — called only when e is already known
// to be (near-)straight (densifyStarvedRail's len(pts)==2 gate: discretizeEdge's own chord-sagitta
// test already proved it flat to within q.Tol()). Endpoints are the edge's own vertices, exactly
// matching discretizeEdge, so every inserted point sits on the SAME edge curve every caller — on
// either side of the shared boundary — already agrees on.
func DensifyStraightEdgeCurve(e *topo.Edge, h float64) []math.Point3 {
	p0, p1 := e.StartVertex().Point(), e.EndVertex().Point()
	l3d := float64(p0.DistanceTo(p1))
	pieces := int(stdmath.Ceil(l3d / h))
	if pieces < 2 {
		return []math.Point3{p0, p1}
	}
	c := e.Geometry()
	lo, hi := c.Domain()
	pts := make([]math.Point3, 0, pieces+1)
	pts = append(pts, p0)
	for k := 1; k < pieces; k++ {
		t := lo + (hi-lo)*float64(k)/float64(pieces)
		pts = append(pts, c.PointAt(t))
	}
	return append(pts, p1)
}

// nurbsRefineFactors are the interior-grid density multipliers the fold-driven loop tries in turn
// (1 = the curvature-adaptive grid, then 2×, 4× denser). A B-spline lip whose curvature the chord
// estimate under-resolves leaves large triangles that fold across it; a denser interior grid shrinks
// them below the fold threshold. validate.RepairFolds handles the rest; refinement stops as soon as a density
// is fold-free (#585).
var nurbsRefineFactors = []float64{1, 0.5, 0.25}

// FoldDrivenPatch triangulates a B-spline trim at increasing interior density, keeping the first
// fold-free result (or the least-folded one if none reaches zero). Each attempt is an independent
// metric-scaled CDT of the SAME exact boundary loops (so neighbouring faces still stitch watertight,
// whatever density wins) plus a denser interior node set — built by the shared metricCDTPatch.
func FoldDrivenPatch(s geom.BSplineSurface, su, sv float64, q Quality, outer3D []math.Point3, outerUV []math.Point2, holes3D [][]math.Point3, holesUV [][]math.Point2) *Mesh {
	want := boundaryEdgeCount(outer3D, holes3D)
	var best, bestAny *Mesh
	bestFolds, bestAnyFolds := 1<<30, 1<<30
	for _, refine := range nurbsRefineFactors {
		m, _ := MetricCDTPatch(s, su, sv, q, outer3D, outerUV, holes3D, holesUV, refine)
		if m == nil {
			continue
		}
		folds := validate.FoldEdgeCount(m)
		if folds < bestAnyFolds {
			bestAny, bestAnyFolds = m, folds
		}
		if FreeEdgeCount(m) != want {
			continue // does not cover its own boundary — a partial triangulation, not a candidate
		}
		if folds < bestFolds {
			best, bestFolds = m, folds
		}
		if bestFolds == 0 {
			break
		}
	}
	if best != nil {
		return best
	}
	return bestAny
}

// boundaryEdgeCount is how many free edges a patch covering exactly this trim must have: every loop is
// closed, so it contributes one edge per point. A conformant triangulation's free edges ARE its
// boundary — more means an interior hole, fewer means it never reached the trim's rim.
func boundaryEdgeCount(outer3D []math.Point3, holes3D [][]math.Point3) int {
	n := len(outer3D)
	for _, h := range holes3D {
		n += len(h)
	}
	return n
}

// MetricScale returns the mean 3D length of a unit step in u and in v (√E, √G of the first
// fundamental form), sampled over the domain — the per-axis scale that makes (u,v) ≈ isometric to 3D.
// Generalised from B-splines to any surface (a cylinder/cone has a strongly anisotropic (u,v): u is an
// angle, v a length). Infinite analytic domains (a cylinder's axial v) are clamped to a finite window.
func MetricScale(s geom.Surface) (su, sv float64) {
	ulo, uhi := clampSpan(s.UDomain())
	vlo, vhi := clampSpan(s.VDomain())
	var sumU, sumV float64
	const n = 4
	for i := 0; i <= n; i++ {
		for j := 0; j <= n; j++ {
			du, dv := s.DerivativesAt(ulo+(uhi-ulo)*float64(i)/n, vlo+(vhi-vlo)*float64(j)/n)
			sumU += du.Length()
			sumV += dv.Length()
		}
	}
	su, sv = sumU/float64((n+1)*(n+1)), sumV/float64((n+1)*(n+1))
	if su <= 0 {
		su = 1
	}
	if sv <= 0 {
		sv = 1
	}
	return su, sv
}

// patchBuilder accumulates the mesh vertices: exact/ on-surface 3D positions + normals, plus the
// metric-scaled (u,v) coordinates the CDT triangulates in.
type patchBuilder struct {
	s      geom.Surface
	su, sv float64
	pos    []math.Point3
	nrm    []math.Vector3
	scaled [][2]float64
}

func newPatchBuilder(s geom.Surface, su, sv float64) *patchBuilder {
	return &patchBuilder{s: s, su: su, sv: sv}
}

// addLoop appends a boundary loop: exact 3D edge points (watertight) with normal + scaled (u,v) from
// the pcurve. Returns the loop's vertex indices.
func (b *patchBuilder) addLoop(loop3D []math.Point3, loop2D []math.Point2) []int {
	idx := make([]int, len(loop3D))
	for i, p := range loop3D {
		idx[i] = b.add(p, float64(loop2D[i].X), float64(loop2D[i].Y))
	}
	return idx
}

// addInterior appends an interior node at parameters g, lifted on-surface via PointAt.
func (b *patchBuilder) addInterior(g [2]float64) {
	b.add(b.s.PointAt(g[0], g[1]), g[0], g[1])
}

func (b *patchBuilder) add(p math.Point3, u, v float64) int {
	idx := len(b.pos)
	b.pos = append(b.pos, p)
	b.nrm = append(b.nrm, b.s.NormalAt(u, v))
	b.scaled = append(b.scaled, [2]float64{u * b.su, v * b.sv})
	return idx
}

// facePcurveLoops returns each loop's (u,v) pcurve and matching exact 3D polyline, concatenating the
// loop's edge-uses (pcurve from healing, 3D from the same edge discretization), dropping the point
// shared with the previous edge and the closing duplicate — like loopBoundary.
func facePcurveLoops(f *topo.Face, q Quality) (outerUV []math.Point2, outer3D []math.Point3, holesUV [][]math.Point2, holes3D [][]math.Point3) {
	for _, l := range f.Loops() {
		uv, p3 := concatLoopPcurve(f.Geometry(), l, q)
		if l.IsOuter() {
			outerUV, outer3D = uv, p3
		} else {
			holesUV = append(holesUV, uv)
			holes3D = append(holes3D, p3)
		}
	}
	return outerUV, outer3D, holesUV, holes3D
}

func concatLoopPcurve(s geom.Surface, l *topo.Loop, q Quality) (uv []math.Point2, p3 []math.Point3) {
	needProject := false
	for _, u := range l.EdgeUses() {
		pts := DiscretizeEdge(u.Edge(), q) // #2009: may already carry starved-rail densification (edge-level, both faces see it)
		if u.Reversed() {
			pts = probe.ReversedPoints(pts)
		}
		pc := u.Pcurve()
		if len(pc) != len(pts) {
			needProject = true
			pc = geom.ProjectCurveToSurface(s, pts) // no/stale pcurve, OR a densified rail: reconstruct on the fly
		}
		if len(p3) > 0 {
			pc, pts = pc[1:], pts[1:] // drop the point shared with the previous edge
		}
		uv = append(uv, pc...)
		p3 = append(p3, pts...)
	}
	if n := len(p3); n > 1 && p3[0].DistanceTo(p3[n-1]) < geom.ResolutionForPoints(p3).Weld() {
		uv, p3 = uv[:n-1], p3[:n-1] // model-relative loop closure (ADR-0042, #1399)
	}
	// Per-edge projection re-seeds a fresh GLOBAL closest point at each edge's start; on a self-proximal
	// NURBS (the EDF bell-mouth lip) that start can snap to the wrong sheet, so the concatenated (u,v)
	// loop self-intersects — and constrainedDelaunay then silently drops those crossing boundary
	// constraints (insertConstraint gives up), cracking the face against its neighbours. When any edge
	// had to be projected, re-derive the WHOLE loop's (u,v) in ONE continuous march (each point seeded
	// from the previous, so it stays on one sheet across edge joins). Stored pcurves (a healed body,
	// every edge with a matching pcurve) are kept verbatim.
	if needProject {
		uv = geom.ProjectCurveToSurface(s, p3)
	}
	return uv, p3
}
