// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/exchange/step/geommap"
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// addFace maps one ADVANCED_FACE onto the builder. An unsupported surface is a
// warned skip (the face is omitted) rather than a fatal error, per the plan's
// fallback policy — the resulting body's open edges are then reported by Validate.
func (a *assembler) addFace(faceID int) error {
	ent, err := a.g.Lookup(faceID)
	if err != nil {
		return err
	}
	if ent.Keyword != "ADVANCED_FACE" {
		return fmt.Errorf("topomap: #%d is %s, want ADVANCED_FACE", faceID, ent.Keyword)
	}
	extruded, err := a.addExtrudedFace(ent)
	if err != nil || extruded {
		return err
	}
	surface, skipped, err := a.faceSurface(ent)
	if err != nil || skipped {
		return err
	}
	sameSense, err := faceSameSense(ent)
	if err != nil {
		return err
	}
	return a.buildFaceLoops(ent, surface, sameSense)
}

// addExtrudedFace maps an ADVANCED_FACE whose surface is a SURFACE_OF_LINEAR_EXTRUSION of a B-spline
// profile, returning handled=true when it did. The STEP extrusion surface is infinite, so the sweep
// range is derived from the face's own extent along the sweep direction (extrusionSweepRange) to size
// the bounded kernel B-spline patch. This recovers the swept side face the importer used to skip,
// closing the open-shell import of the OCCT blend-parity base bodies G3–H1 (corpus resurvey §4). Any
// other surface returns handled=false so the common path (faceSurface) is unchanged (do-no-harm).
func (a *assembler) addExtrudedFace(ent *part21.RawEntity) (handled bool, err error) {
	surfID, ok, err := faceSurfaceRef(ent) // Finding 4: shared guard+ref with faceSurface (no dup)
	if err != nil || !ok {
		return false, err // malformed: let the common path (faceSurface) report the shape error
	}
	profile, dir, ok, err := geommap.LinearExtrusionBSpline(a.g, surfID, a.scale)
	if err != nil || !ok {
		return false, err
	}
	sameSense, err := faceSameSense(ent)
	if err != nil {
		return false, err
	}
	bounds, err := a.faceBounds(ent)
	if err != nil {
		return false, err
	}
	a.emitExtrudedFace(ent, profile, dir, sameSense, bounds)
	return true, nil
}

// emitExtrudedFace sizes the finite B-spline patch to the face's swept extent along dir
// (extrusionSweepRange) and emits it. A build failure is a warned skip — the face is dropped and
// the import continues, so a downstream Validate reports the open edge rather than aborting.
func (a *assembler) emitExtrudedFace(ent *part21.RawEntity, profile geom.BSplineCurve, dir math.Vector3, sameSense bool, bounds []boundLoop) {
	lo, hi := extrusionSweepRange(bounds, profile.Ctrl, dir)
	surface, err := geommap.NewExtrudedBSplineSurface(profile, dir, lo, hi)
	if err != nil {
		a.warn("skipped extruded face #%d: %v", ent.ID, err)
		return
	}
	a.emitFace(surface, sameSense, bounds)
}

// faceSurfaceRef resolves an ADVANCED_FACE's surface reference (parameter 2). ok=false flags a
// malformed entity (fewer than the 4 required params) so each caller decides how to react — the
// extrusion path defers to the common path, which reports the descriptive shape error (Finding 4).
func faceSurfaceRef(ent *part21.RawEntity) (surfID int, ok bool, err error) {
	if len(ent.Params) < 4 {
		return 0, false, nil
	}
	surfID, err = ent.Params[2].AsRef()
	return surfID, err == nil, err
}

// extrusionSweepRange bounds the interval [lo,hi] the face occupies along the unit sweep direction d,
// so the finite B-spline patch covers the (3D-loop-trimmed) face. It over-covers deliberately — the
// face is trimmed by its loops — using the profile's convex-hull bound: a boundary point is p=C(u)+v·d,
// and C(u)·d lies within the profile control points' d-projections, so lo=min(p·d)−max(ctrl·d) and
// hi=max(p·d)−min(ctrl·d) is a guaranteed superset of the true v-range (both rails and the sides).
func extrusionSweepRange(bounds []boundLoop, ctrl []math.Point3, d math.Vector3) (lo, hi float64) {
	pMin, pMax := projRange(loopPoints(bounds), d)
	cMin, cMax := projRange(ctrl, d)
	return pMin - cMax, pMax - cMin
}

// loopPoints collects the RANGE-BOX corners of every bound loop. Review Finding 1 (2026-07-24):
// bounding the sweep from edge StartVertex/EndVertex alone under-covers a curved trim edge that
// bulges in the sweep direction — its d-extremum sits at a curve INTERIOR, not a vertex, so a
// vertex-only bound can under-size the patch and SILENTLY re-open the shell. The edge RangeBox
// samples each curve's interior (kernel/topo curveSamplesPerEdge), so its corners bound any
// interior excursion; projecting them onto d yields a guaranteed superset of the face's v-range.
func loopPoints(bounds []boundLoop) []math.Point3 {
	var pts []math.Point3
	for _, b := range bounds {
		corners := loopBox(b.uses).Corners()
		pts = append(pts, corners[:]...)
	}
	return pts
}

// projRange returns the min (lo) and max (hi) scalar projection of the points onto direction d.
// Named lo/hi (not min/max) to avoid shadowing the builtins and to seed the extrema once (Finding 3).
func projRange(pts []math.Point3, d math.Vector3) (lo, hi float64) {
	if len(pts) == 0 {
		return 0, 0
	}
	lo = float64(pts[0].AsVector().Dot(d))
	hi = lo
	for _, p := range pts[1:] {
		s := float64(p.AsVector().Dot(d))
		if s < lo {
			lo = s
		}
		if s > hi {
			hi = s
		}
	}
	return lo, hi
}

// faceSurface resolves an ADVANCED_FACE's surface (parameter 2). An unsupported
// surface type is warned and reported as skipped (no error), so the import
// continues and the missing face shows up as boundary edges in Validate.
func (a *assembler) faceSurface(ent *part21.RawEntity) (geom.Surface, bool, error) {
	surfID, ok, err := faceSurfaceRef(ent) // Finding 4: shared guard+ref with addExtrudedFace
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, fmt.Errorf("topomap: ADVANCED_FACE #%d wants 4 params, got %d", ent.ID, len(ent.Params))
	}
	surf, err := geommap.Surface(a.g, surfID, a.scale)
	if err != nil {
		if _, ok := errors.AsType[geommap.ErrUnsupportedSurface](err); ok {
			a.warn("skipped face #%d: %v", ent.ID, err)
			return nil, true, nil
		}
		return nil, false, err
	}
	return surf, false, nil
}

// faceSameSense reads ADVANCED_FACE.same_sense (the last boolean parameter): true
// means the face's outward sense agrees with its surface normal.
func faceSameSense(ent *part21.RawEntity) (bool, error) {
	return ent.Params[3].AsBool()
}

// buildFaceLoops builds every bound loop and adds the face. The face is added
// reversed when same_sense is false (its material side opposes the surface normal),
// matching topo.AddReversedFace's contract.
func (a *assembler) buildFaceLoops(ent *part21.RawEntity, surface geom.Surface, sameSense bool) error {
	bounds, err := a.faceBounds(ent)
	if err != nil {
		return err
	}
	a.emitFace(surface, sameSense, bounds)
	return nil
}

// faceBounds builds every FACE_(OUTER_)BOUND of an ADVANCED_FACE into boundLoops, sharing the
// per-import edge/vertex maps (so the addExtrudedFace path can measure the loops before choosing the
// surface without rebuilding topology).
func (a *assembler) faceBounds(ent *part21.RawEntity) ([]boundLoop, error) {
	boundRefs, err := refListValues(ent.Params[1])
	if err != nil {
		return nil, err
	}
	bounds := make([]boundLoop, 0, len(boundRefs))
	for _, boundID := range boundRefs {
		b, err := a.buildBound(boundID)
		if err != nil {
			return nil, err
		}
		bounds = append(bounds, b)
	}
	return bounds, nil
}

// emitFace finalizes the outer-loop choice (ensureOuterLoop) and adds the face, dropping degenerate
// VERTEX_LOOP bounds (a pole/apex imposes no boundary).
func (a *assembler) emitFace(surface geom.Surface, sameSense bool, bounds []boundLoop) {
	ensureOuterLoop(bounds)
	loops := make([]topo.LoopSpec, 0, len(bounds))
	for _, b := range bounds {
		if b.degenerate {
			continue
		}
		loops = append(loops, loopSpec(b.outer, b.uses))
	}
	a.addBuiltFace(surface, sameSense, loops)
}

// ensureOuterLoop guarantees the face has an outer loop flagged. STEP makes
// FACE_OUTER_BOUND optional and some exporters (OpenCASCADE) write FACE_BOUND for EVERY
// bound, which would leave a face with no outer loop — its planar triangulation then yields
// nothing (a missing cap/plate). When no bound was declared outer, infer it: the loop whose
// edges span the largest bounding box encloses the rest, so it is the outer one.
func ensureOuterLoop(bounds []boundLoop) {
	for _, b := range bounds {
		if b.outer {
			return // STEP already declared the outer loop
		}
	}
	best, bestSize := -1, -1.0
	for i := range bounds {
		if bounds[i].degenerate {
			continue // a vertex loop never encloses anything
		}
		if s := loopBoxDiagonal(bounds[i].uses); s > bestSize {
			best, bestSize = i, s
		}
	}
	if best >= 0 {
		bounds[best].outer = true
	}
}

// loopBox is the 3D bounding box of a loop's edges. Each edge's RangeBox samples its curve
// interior (not just endpoints), so the box encloses any bulge of a curved edge. Shared by
// loopBoxDiagonal (outer-loop pick) and loopPoints (sweep-range bound) — one corner-enumeration path.
func loopBox(uses []topo.Use) math.Box {
	box := math.EmptyBox()
	for _, u := range uses {
		box = box.Union(u.Edge.RangeBox())
	}
	return box
}

// loopBoxDiagonal is the diagonal length of the bounding box of a loop's edges — a
// rotation-tolerant size used to pick the enclosing (outer) loop.
func loopBoxDiagonal(uses []topo.Use) float64 {
	return float64(loopBox(uses).Diagonal().Length())
}

// addBuiltFace adds the face forward or reversed, with a stable imported lineage, and records the
// face's parametric trim when its surface is periodic (ADR-0063, #3550).
//
// An imported face used to arrive with chart = nil, always. On a PERIODIC surface the 3-D loops alone
// do not say which of two complementary regions the face is — a torus band's two rims bound the strip
// between them and the strip the other way round the seam — so the general chart-driven mesher
// declines such a face outright and it falls to whatever bespoke arm recognises its shape. Measured on
// occtparity simple/J3 and bfuseblend/A4: every face of both bodies recorded chart = 0, not only the
// tori. brep.ChartOfFace derives the contours from the loops this assembler has just wound, which is
// the producer ADR-0063 asks for, and DECLINES rather than guessing where they do not determine one.
func (a *assembler) addBuiltFace(surface geom.Surface, sameSense bool, loops []topo.LoopSpec) {
	lineage := topo.NewLineage(topo.Tok(a.feat, "face", a.nextF))
	a.nextF++
	recordImportedChart(a.addOrientedFace(surface, sameSense, lineage, loops))
}

// addOrientedFace adds the face in the sense STEP declared and returns it.
func (a *assembler) addOrientedFace(surface geom.Surface, sameSense bool, lineage topo.Lineage,
	loops []topo.LoopSpec) *topo.Face {
	if sameSense {
		return a.builder.AddFace(surface, lineage, loops...)
	}
	return a.builder.AddReversedFace(surface, lineage, loops...)
}

// recordImportedChart derives and stores a periodic face's chart, or leaves it nil.
//
// IT IS OFF, and the reason is a measured regression rather than a doubt about the chart (#3549).
// brep.ChartOfFace is right — kernel/brep's own rows hold it, and with it switched on bfuseblend/A4's
// host torus meshes through the GENERAL chart-driven mesher at 292 849.88 mm² against DRAWEXE's
// 292 920 (rel −2.4e-4, zero diagnostics) where the bespoke loft reads 291 968.09 (rel −3.3e-3). But a
// chart does not only describe a face, it ROUTES it: chartFaceMesh declines an uncharted face and
// accepts a charted one, so recording charts here sends every seam-crossing imported face to the
// covering mesher, which triangulates the face's whole grid replicated a period either way.
//
// Measured on the OCC fixtures by TestTessellationBudget, which is the gate that has to go green
// before this switch flips: **0.08 s off, 6.37 s on, against a 2.15 s budget**. The faces that move
// were previously DEGRADED — they fell to the flat patch CDT or the surface's whole domain and said so
// — so the covering is doing the right thing and the price is the covering's own cost. That cost is
// #3549, and the anomaly under it is measured: on the J3 host chart the insertion allocates THIRTY
// triangles per point (23 606 516 for 788 250 points, 1 574 591 live) where a Bowyer-Watson insert
// should allocate about three, and on a jittered grid of 319 229 points the same code allocates two.
func recordImportedChart(f *topo.Face) {
	if !chartImportedFaces {
		return
	}
	if chart, ok := brep.ChartOfFace(f); ok {
		f.SetChart(chart)
	}
}

// chartImportedFaces switches the producer on. It is one bool because the producer is BUILT and
// tested (kernel/brep's ChartOfFace rows, and TestAnImportedPeriodicFaceCarriesItsChart here drives
// this path directly); what is not ready is the mesher the chart routes to. See recordImportedChart.
const chartImportedFaces = false
