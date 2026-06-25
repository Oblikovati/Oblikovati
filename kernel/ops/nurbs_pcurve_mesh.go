// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nurbsPcurveMesh meshes a B-spline face from its reconstructed pcurves (M25 F01) with a METRIC-AWARE
// triangulation. The over-enclosure of every prior attempt was folded triangles, not bad geometry
// (every sampled point lies exactly on OCC's surface): a plain (u,v) Delaunay twists in 3D because u
// and v have very different 3D scales (anisotropy). So the boundary pcurve + deflection-adaptive
// interior nodes are triangulated in METRIC-SCALED (u,v) — each axis scaled by its mean 3D length
// (√E, √G), making the parameter space ≈ isometric to 3D, so the Delaunay yields well-shaped 3D
// triangles. Points lift via PointAt (exact forward eval, no projection); the boundary keeps the exact
// 3D edge points (watertight). Returns nil to fall back.
func nurbsPcurveMesh(f *topo.Face, q Quality) *Mesh {
	s, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return nil
	}
	outerUV, outer3D, holesUV, holes3D := facePcurveLoops(f, q)
	if len(outer3D) < 3 {
		return nil
	}
	su, sv := metricScale(s)
	return foldDrivenPatch(s, su, sv, q, outer3D, outerUV, holes3D, holesUV)
}

// nurbsRefineFactors are the interior-grid density multipliers the fold-driven loop tries in turn
// (1 = the curvature-adaptive grid, then 2×, 4× denser). A B-spline lip whose curvature the chord
// estimate under-resolves leaves large triangles that fold across it; a denser interior grid shrinks
// them below the fold threshold. repairFolds handles the rest; refinement stops as soon as a density
// is fold-free (#585).
var nurbsRefineFactors = []float64{1, 0.5, 0.25}

// foldDrivenPatch triangulates a B-spline trim at increasing interior density, keeping the first
// fold-free result (or the least-folded one if none reaches zero). Each attempt is an independent
// metric-scaled CDT of the SAME exact boundary loops (so neighbouring faces still stitch watertight,
// whatever density wins) plus a denser interior node set — built by the shared metricCDTPatch.
func foldDrivenPatch(s geom.BSplineSurface, su, sv float64, q Quality, outer3D []math.Point3, outerUV []math.Point2, holes3D [][]math.Point3, holesUV [][]math.Point2) *Mesh {
	var best *Mesh
	bestFolds := 1 << 30
	for _, refine := range nurbsRefineFactors {
		m, _ := metricCDTPatch(s, su, sv, q, outer3D, outerUV, holes3D, holesUV, refine)
		if m == nil {
			continue
		}
		if folds := FoldEdgeCount(m); folds < bestFolds {
			best, bestFolds = m, folds
		}
		if bestFolds == 0 {
			break
		}
	}
	return best
}

// metricScale returns the mean 3D length of a unit step in u and in v (√E, √G of the first
// fundamental form), sampled over the domain — the per-axis scale that makes (u,v) ≈ isometric to 3D.
// Generalised from B-splines to any surface (a cylinder/cone has a strongly anisotropic (u,v): u is an
// angle, v a length). Infinite analytic domains (a cylinder's axial v) are clamped to a finite window.
func metricScale(s geom.Surface) (su, sv float64) {
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
		pts := discretizeEdge(u.Edge(), q)
		if u.Reversed() {
			pts = reverse3(pts)
		}
		pc := u.Pcurve()
		if len(pc) != len(pts) {
			needProject = true
			pc = geom.ProjectCurveToSurface(s, pts) // no/stale pcurve: reconstruct on the fly
		}
		if len(p3) > 0 {
			pc, pts = pc[1:], pts[1:] // drop the point shared with the previous edge
		}
		uv = append(uv, pc...)
		p3 = append(p3, pts...)
	}
	if n := len(p3); n > 1 && p3[0].DistanceTo(p3[n-1]) < ResolutionForPoints(p3).Weld() {
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
