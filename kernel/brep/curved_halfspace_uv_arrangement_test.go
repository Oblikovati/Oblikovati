// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cylinderRuledUV builds a minimal radius-R cylinder ruledUV about the z-axis (base at the origin) for the
// (u,v)-arrangement unit tests: point3(u,v) = (R·cos u, R·sin u, v), so paramOf must invert it exactly.
func cylinderRuledUV(r, vMin, vMax float64) ruledUV {
	axis, ref := math.V3(0, 0, 1), math.V3(1, 0, 0)
	bottom, top := math.P3(0, 0, vMin), math.P3(0, 0, vMax)
	botCirc, _ := geom.NewCircle(bottom, axis, r)
	topCirc, _ := geom.NewCircle(top, axis, r)
	return ruledUV{
		base: math.P3(0, 0, 0), axis: axis, ref: ref, binor: axis.Cross(ref),
		radSlope: 0, radConst: r,
		band: coneSideBand_{
			bottom: bottom, top: top, bottomCirc: botCirc, topCirc: topCirc,
			vMin: vMin, vMax: vMax, rBot: r, rTop: r,
		},
	}
}

// TestRuledParamOfInvertsPoint3: paramOf is the exact inverse of point3 over the band — a sampled (u,v)
// round-trips through the surface point and back to itself (Oblikovati#1405).
func TestRuledParamOfInvertsPoint3(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	for i := range 16 {
		u := 2 * stdmath.Pi * float64(i) / 16
		for _, v := range []float64{-4, -1, 0, 2.5, 4} {
			uv := c.paramOf(c.point3(u, v))
			du := stdmath.Abs(uv.X - u)
			if du > stdmath.Pi { // both near the seam (0 vs 2π) — compare wrapped
				du = 2*stdmath.Pi - du
			}
			if du > 1e-9 || stdmath.Abs(uv.Y-v) > 1e-9 {
				t.Errorf("paramOf(point3(%.4f,%.4f)) = (%.6f,%.6f), want (%.4f,%.4f)", u, v, uv.X, uv.Y, u, v)
			}
		}
	}
}

// TestSampleImprintUVOnRimCircle: sampling the bottom-rim circle (v=vMin) as an imprint yields tagged
// (u,v) segments that all sit at v=vMin, carry the circle as their source curve, and whose endpoint
// parameters map back (via the circle) to the segment's (u,v) — the round-trip the boundary re-emission
// relies on (Oblikovati#1405).
func TestSampleImprintUVOnRimCircle(t *testing.T) {
	t.Parallel()
	const r, vMin = 3.0, -5.0
	c := cylinderRuledUV(r, vMin, 5)
	rim, _ := geom.NewCircle(math.P3(0, 0, vMin), math.V3(0, 0, 1), r) // the bottom rim, z=vMin
	segs := c.sampleImprintUV(rim)
	if len(segs) != imprintSampleCount {
		t.Fatalf("sampled %d segments, want %d", len(segs), imprintSampleCount)
	}
	for i, s := range segs {
		if stdmath.Abs(s.a.Y-vMin) > 1e-9 || stdmath.Abs(s.b.Y-vMin) > 1e-9 {
			t.Errorf("segment %d not on the rim v=%.1f: a.v=%.6f b.v=%.6f", i, vMin, s.a.Y, s.b.Y)
		}
		if s.curve != rim {
			t.Errorf("segment %d lost its source curve tag", i)
		}
		// The tagged parameter at b must reproduce the segment's (u,v) endpoint through the surface.
		if got := c.paramOf(s.curve.PointAt(s.tB)); stdmath.Abs(got.Y-s.b.Y) > 1e-9 {
			t.Errorf("segment %d param tB=%.4f maps to v=%.6f, want %.6f", i, s.tB, got.Y, s.b.Y)
		}
	}
}

// TestSplitSeamCrossingSplitsAtSeam: an imprint segment whose endpoints straddle the seam (its short arc
// crosses u=0≡2π) is split into two segments meeting exactly at the seam, with v and the curve parameter
// interpolated; a non-straddling segment passes through unchanged (Oblikovati#1405).
func TestSplitSeamCrossingSplitsAtSeam(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	// climbs past 2π: a=(6.0, 1) -> b=(0.2, 3); the short arc crosses the seam at u=2π.
	up := uvSeg{a: math.P2(6.0, 1), b: math.P2(0.2, 3), curve: nil, tA: 0, tB: 1, kind: segImprint}
	got := splitSeamCrossing(up)
	if len(got) != 2 {
		t.Fatalf("seam-straddling segment split into %d, want 2", len(got))
	}
	if stdmath.Abs(got[0].b.X-twoPi) > 1e-12 || got[1].a.X != 0 {
		t.Errorf("split not anchored at the seam: end of first=%.4f, start of second=%.4f (want 2π, 0)", got[0].b.X, got[1].a.X)
	}
	if stdmath.Abs(got[0].b.Y-got[1].a.Y) > 1e-12 {
		t.Errorf("v discontinuous across the seam: %.6f vs %.6f", got[0].b.Y, got[1].a.Y)
	}
	// continuity of the curve parameter: first runs tA->tSeam, second tSeam->tB
	if got[0].tB != got[1].tA {
		t.Errorf("curve parameter discontinuous across the seam: %.6f vs %.6f", got[0].tB, got[1].tA)
	}
	// a non-straddling segment is unchanged
	inside := uvSeg{a: math.P2(1.0, 0), b: math.P2(2.0, 1), kind: segImprint}
	if s := splitSeamCrossing(inside); len(s) != 1 || s[0] != inside {
		t.Errorf("a non-straddling segment was altered: %+v", s)
	}
}

// TestSplitVSeamCrossingSplitsAtTubeSeam: a segment whose endpoints straddle the TUBE seam (v=0≡2π) is split
// into two meeting AT v=0/2π, with u and the curve parameter interpolated — the v-analogue of the azimuth
// split, needed for the two-oval band whose ovals wrap the tube (Oblikovati#1406).
func TestSplitVSeamCrossingSplitsAtTubeSeam(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	// climbs past 2π in v: a=(1, 6.0) -> b=(3, 0.2); the short arc crosses the tube seam at v=2π.
	up := uvSeg{a: math.P2(1, 6.0), b: math.P2(3, 0.2), tA: 0, tB: 1, kind: segImprint}
	got := splitVSeamCrossing(up)
	if len(got) != 2 {
		t.Fatalf("v-seam-straddling segment split into %d, want 2", len(got))
	}
	if stdmath.Abs(float64(got[0].b.Y)-twoPi) > 1e-12 || got[1].a.Y != 0 {
		t.Errorf("split not anchored at the tube seam: end-v=%.4f start-v=%.4f (want 2π, 0)", got[0].b.Y, got[1].a.Y)
	}
	if stdmath.Abs(float64(got[0].b.X)-float64(got[1].a.X)) > 1e-12 {
		t.Errorf("u discontinuous across the tube seam: %.6f vs %.6f", got[0].b.X, got[1].a.X)
	}
	if got[0].tB != got[1].tA {
		t.Errorf("curve parameter discontinuous across the tube seam: %.6f vs %.6f", got[0].tB, got[1].tA)
	}
	// a segment not straddling the tube seam passes through unchanged.
	inside := uvSeg{a: math.P2(1, 1.0), b: math.P2(2, 2.0), kind: segImprint}
	if s := splitVSeamCrossing(inside); len(s) != 1 || s[0] != inside {
		t.Errorf("a non-straddling segment was altered: %+v", s)
	}
}

// TestAssembleBandSegmentsClosesRectangle: assembling an imprint with the rim+seam frame yields a closed
// parameter rectangle — the four frame edges span [0,2π]×[vMin,vMax] and share the rectangle corners — and
// no assembled segment spans the seam discontinuity (Oblikovati#1405).
func TestAssembleBandSegmentsClosesRectangle(t *testing.T) {
	t.Parallel()
	const vMin, vMax = -5.0, 5.0
	c := cylinderRuledUV(3, vMin, vMax)
	rim, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3) // a v=0 horizontal cut imprint
	segs := c.assembleBandSegments(c.sampleImprintUV(rim))

	rims, seams := 0, 0
	for _, s := range segs {
		if du := stdmath.Abs(s.b.X - s.a.X); du > stdmath.Pi && s.kind == segImprint {
			t.Errorf("an imprint segment spans the seam: a=%v b=%v", s.a, s.b)
		}
		switch s.kind {
		case segRim:
			rims++
			if s.a.X != 0 || stdmath.Abs(s.b.X-2*stdmath.Pi) > 1e-12 {
				t.Errorf("rim segment does not span [0,2π]: a=%v b=%v", s.a, s.b)
			}
		case segSeam:
			seams++
			if stdmath.Abs(s.a.Y-vMin) > 1e-12 || stdmath.Abs(s.b.Y-vMax) > 1e-12 {
				t.Errorf("seam segment does not span [vMin,vMax]: a=%v b=%v", s.a, s.b)
			}
		}
	}
	if rims != 2 || seams != 2 {
		t.Errorf("frame has %d rims and %d seams, want 2 and 2", rims, seams)
	}
}

// horizontalCutImprint samples the circle at axial height v as a (u,v) imprint (a horizontal line v=const).
func (c ruledUV) horizontalCutImprint(t *testing.T, v float64) []uvSeg {
	t.Helper()
	circ, err := geom.NewCircle(math.P3(0, 0, v), math.V3(0, 0, 1), c.radConst)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	return c.sampleImprintUV(circ)
}

// TestKeptCellsClassifiesByMaterial: two horizontal imprint cuts split the band into three v-bands; the
// material predicate selects which are kept — the middle band (one cell) or the two outer bands (#1405).
func TestKeptCellsClassifiesByMaterial(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	imprint := append(c.horizontalCutImprint(t, -2), c.horizontalCutImprint(t, 2)...)
	cells := arrangeBand(c.assembleBandSegments(imprint))
	if len(cells) != 3 {
		t.Fatalf("got %d cells, want 3 (v<-2, -2<v<2, v>2)", len(cells))
	}
	mid := keptCells(cells, func(uv math.Point2) bool { return uv.Y > -2 && uv.Y < 2 })
	if len(mid) != 1 {
		t.Errorf("middle-band material kept %d cells, want 1", len(mid))
	}
	outer := keptCells(cells, func(uv math.Point2) bool { return uv.Y < -2 || uv.Y > 2 })
	if len(outer) != 2 {
		t.Errorf("outer material kept %d cells, want 2", len(outer))
	}
}

// TestHalfSpaceMaterialKeepsNegativeSide: with g(u,v)=v the half-space predicate keeps exactly the v<0
// cell of a v=0 cut — the plane-cut classification the analytic walk produced, now via the arrangement.
func TestHalfSpaceMaterialKeepsNegativeSide(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	c.s = 1 // g(u,v) = p + v·s = v  -> kept where v < 0
	cells := arrangeBand(c.assembleBandSegments(c.horizontalCutImprint(t, 0)))
	kept := keptCells(cells, c.halfSpaceMaterial())
	if len(kept) != 1 {
		t.Fatalf("kept %d cells, want 1 (the v<0 half)", len(kept))
	}
	if p, _ := interiorPointOf(kept[0].Outer); p.Y >= 0 {
		t.Errorf("kept cell interior v=%.3f, want <0", p.Y)
	}
}

// TestInteriorPointOfConcave: interiorPointOf returns a point genuinely inside even for a concave (L-shaped)
// polygon whose centroid lies outside it — the robustness keptCells relies on for non-convex cells.
func TestInteriorPointOfConcave(t *testing.T) {
	t.Parallel()
	lShape := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 1), math.P2(1, 1), math.P2(1, 4), math.P2(0, 4),
	}
	p, ok := interiorPointOf(lShape)
	if !ok || !pointInPolygon2D(p, lShape) {
		t.Errorf("interiorPointOf returned %v (ok=%v), not inside the L-shape", p, ok)
	}
}

// TestKeptBoundaryWrappingBandTwoLoops: a v=0 cut keeping the v<0 half is a band that WRAPS the seam — its
// boundary must be two closed loops (the bottom rim and the section), with the artificial seam edges
// dissolved by the cross-seam cancellation (Oblikovati#1405).
func TestKeptBoundaryWrappingBandTwoLoops(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	c.s = 1 // g(u,v)=v -> keep v<0
	cells := arrangeBand(c.assembleBandSegments(c.horizontalCutImprint(t, 0)))
	loops := chainLoops(keptBoundaryEdges(keptCells(cells, c.halfSpaceMaterial()), true, false))
	if len(loops) != 2 {
		t.Fatalf("wrapping band: %d boundary loops, want 2 (rim + section)", len(loops))
	}
	atRim, atSection := false, false
	for _, lp := range loops {
		allRim, allSec := true, true
		for _, e := range lp {
			if stdmath.Abs(float64(e.a.Y)+5) > 1e-6 {
				allRim = false
			}
			if stdmath.Abs(float64(e.a.Y)) > 1e-6 {
				allSec = false
			}
		}
		atRim, atSection = atRim || allRim, atSection || allSec
	}
	if !atRim || !atSection {
		t.Errorf("loops are not {bottom rim, section}: atRim=%v atSection=%v", atRim, atSection)
	}
}

// TestKeptBoundaryTongueSingleLoop: two vertical-ruling imprints carve a u-span that does NOT touch the
// seam; keeping the middle span is a non-wrapping tongue whose boundary is a single loop (two rulings + two
// rim arcs), with no seam edge involved (Oblikovati#1405).
func TestKeptBoundaryTongueSingleLoop(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	left := c.sampleImprintUV(geom.NewLineSegment(c.point3(stdmath.Pi/2, -5), c.point3(stdmath.Pi/2, 5)))
	right := c.sampleImprintUV(geom.NewLineSegment(c.point3(3*stdmath.Pi/2, -5), c.point3(3*stdmath.Pi/2, 5)))
	cells := arrangeBand(c.assembleBandSegments(append(left, right...)))
	kept := keptCells(cells, func(uv math.Point2) bool { return uv.X > stdmath.Pi/2 && uv.X < 3*stdmath.Pi/2 })
	if len(kept) != 1 {
		t.Fatalf("tongue: %d kept cells, want 1", len(kept))
	}
	if loops := chainLoops(keptBoundaryEdges(kept, true, false)); len(loops) != 1 {
		t.Fatalf("tongue: %d boundary loops, want 1", len(loops))
	}
}

// TestTrimByImprintProducesValidFace: the end-to-end arrangement trim of a wrapping cut yields one kept
// curvedFace with two boundary loops whose edges lie on the cylinder, plus a non-empty lid section — the
// whole project→assemble→subdivide→classify→boundary→re-emit pipeline (Oblikovati#1405).
func TestTrimByImprintProducesValidFace(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	c.s = 1 // g(u,v)=v -> keep v<0
	surf, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	cf := curvedFace{surface: surf}
	circ, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	faces, lid, err := trimByImprint(&c, cf, surf, []geom.Curve3{circ}, ruledMaterial(&c))
	if err != nil || len(faces) != 1 {
		t.Fatalf("trimByImprint: err=%v faces=%d, want 1 face", err, len(faces))
	}
	if len(faces[0].loops) != 2 {
		t.Fatalf("kept face has %d loops, want 2 (the wrapping band)", len(faces[0].loops))
	}
	for li, lp := range faces[0].loops {
		for ei, e := range lp.edges {
			if r := distFromAxis(e.start(), c); stdmath.Abs(r-3) > 1e-6 {
				t.Errorf("loop %d edge %d start off the cylinder: radius %.6f", li, ei, r)
			}
		}
	}
	if len(lid) == 0 {
		t.Error("trimByImprint returned no lid section arcs")
	}
}

// TestTrimByImprintReversesTopRim: when the source side traverses its top rim reversed (topRimReversed) and
// the kept wrapping band's hi boundary is the PURE top rim, orientLoops reverses that loop so the rebuilt
// rim stays opposite its cap — the analytic splitSide convention reproduced through the arrangement (#1405).
func TestTrimByImprintReversesTopRim(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	c.s = -1                     // g(u,v) = -v -> keep v>0, so the hi boundary is the top rim
	c.band.topRimReversed = true // the source side traverses the top rim reversed
	surf, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	circ, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	faces, _, err := trimByImprint(&c, curvedFace{surface: surf}, surf, []geom.Curve3{circ}, ruledMaterial(&c))
	if err != nil || len(faces) != 1 || len(faces[0].loops) != 2 {
		t.Fatalf("trimByImprint: err=%v faces=%d", err, len(faces))
	}
	// loops[0] is the hi boundary = the top rim (a full circle); it must be present and on the cylinder.
	hi := faces[0].loops[0].edges
	if !allRimEdges(hi) {
		t.Errorf("hi loop is not the pure top rim (edges: %d)", len(hi))
	}
}

// TestClipParamsMultiArmHyperbola: a plane parallel to a cone's axis cuts it in a hyperbola whose two arms
// both lie in the band (the joining vertex falls below it), so clipParams must return TWO parameter ranges,
// each spanning the band height — the multi-arm windowing a cone cut needs (Oblikovati#1405).
func TestClipParamsMultiArmHyperbola(t *testing.T) {
	t.Parallel()
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6, "c")
	plane, _ := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0)) // axis-parallel → hyperbola, two arms
	var sf *topo.Face
	for _, f := range cone.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			sf = f
		}
	}
	cf := curvedFace{surface: sf.Geometry(), reversed: sf.Reversed(), loops: loopsOf(sf), lineage: sf.Lineage()}
	curves, _ := curvedImprint(cf, curvedFace{surface: plane}, geom.ResolutionForBox(cone.RangeBox()))
	if len(curves) != 1 {
		t.Fatalf("want one conic section, got %d", len(curves))
	}
	_, band, _ := fullConeSideBand(cf)
	c := newConeUV(sf.Geometry().(geom.Cone), band, plane, math.V3(1, 0, 0))
	ranges := c.clipParams(curves[0])
	if len(ranges) != 2 {
		t.Fatalf("hyperbola has two arms in the band, clipParams returned %d ranges", len(ranges))
	}
	for i, r := range ranges {
		v0, v1 := c.curveV(curves[0], r[0]), c.curveV(curves[0], r[1])
		lo, hi := stdmath.Min(v0, v1), stdmath.Max(v0, v1)
		if lo > band.vMin+0.5 || hi < band.vMax-0.5 {
			t.Errorf("arm %d spans v[%.2f,%.2f], should cover the band [%.2f,%.2f]", i, lo, hi, band.vMin, band.vMax)
		}
	}
}

// TestTrimByImprintIslandHole: a CLOSED imprint loop inside a ruled face (not reaching a rim) with the
// material kept OUTSIDE it produces a trimmed face carrying that loop as an inner HOLE — the island case
// the general arrangement handles that the single-valued analytic walk never could (Oblikovati#1405, the
// curved∩curved generality). The material is given as a seam-aware 3D test (point3 handles the shifted
// frame), the form a curved∩curved membership predicate takes.
func TestTrimByImprintIslandHole(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	corners := []math.Point2{math.P2(2, -1), math.P2(3, -1), math.P2(3, 1), math.P2(2, 1)}
	var curves []geom.Curve3
	for i := range 4 {
		a := c.point3(float64(corners[i].X), float64(corners[i].Y))
		b := c.point3(float64(corners[(i+1)%4].X), float64(corners[(i+1)%4].Y))
		curves = append(curves, geom.NewLineSegment(a, b))
	}
	center := c.point3(2.5, 0) // 3D centre of the island (absolute frame)
	// keepOutside reads c.point3 LATE (after trimByImprint shifts c's seam): point3 adds seamU back, so the
	// 3D test stays in the absolute frame the fixed centre was computed in — the curved∩curved predicate form.
	keepOutside := func() materialPredicate {
		return func(uv math.Point2) bool {
			// near the island centre on the cylinder ≈ inside the island; keep everything else.
			return float64(c.point3(uv.X, uv.Y).DistanceTo(center)) > 1.6
		}
	}
	surf, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	faces, _, err := trimByImprint(&c, curvedFace{surface: surf}, surf, curves, keepOutside)
	if err != nil || len(faces) != 1 {
		t.Fatalf("trimByImprint: err=%v faces=%d, want 1", err, len(faces))
	}
	// the kept band has its two rim loops PLUS the island as an inner loop.
	if got := len(faces[0].loops); got != 3 {
		t.Fatalf("kept face has %d loops, want 3 (two rims + the island hole)", got)
	}
	// exactly one loop is the island (none of its edges touch a rim level vMin/vMax).
	islands := 0
	for _, lp := range faces[0].loops {
		interior := true
		for _, e := range lp.edges {
			if v := float64(c.paramOf(e.start()).Y); stdmath.Abs(v-5) < 0.1 || stdmath.Abs(v+5) < 0.1 {
				interior = false
			}
		}
		if interior {
			islands++
		}
	}
	if islands != 1 {
		t.Errorf("found %d interior (island) loops, want 1", islands)
	}
}

// distFromAxis returns p's perpendicular distance from the side's axis (its cylinder radius).
func distFromAxis(p math.Point3, c ruledUV) float64 {
	d := c.base.VectorTo(p)
	radial := d.Sub(c.axis.Scale(d.Dot(c.axis)))
	return float64(radial.Length())
}

// TestEmitLoopEdgesStructurallyValid: re-emitting a wrapping-band boundary yields analytic loopEdges whose
// 3D endpoints lie exactly on the cylinder, whose consecutive edges connect, and that close — the bridge
// from the (u,v) arrangement back to a valid B-rep boundary (Oblikovati#1405).
func TestEmitLoopEdgesStructurallyValid(t *testing.T) {
	t.Parallel()
	c := cylinderRuledUV(3, -5, 5)
	c.s = 1 // g(u,v)=v -> keep v<0
	segs := c.assembleBandSegments(c.horizontalCutImprint(t, 0))
	loops := chainLoops(keptBoundaryEdges(keptCells(arrangeBand(segs), c.halfSpaceMaterial()), true, false))
	if len(loops) != 2 {
		t.Fatalf("want 2 boundary loops, got %d", len(loops))
	}
	for li, lp := range loops {
		edges, _, ok := emitLoopEdges(&c, lp, newUVSegIndex(segs))
		if !ok || len(edges) == 0 {
			t.Fatalf("loop %d: emit failed (ok=%v, %d edges)", li, ok, len(edges))
		}
		for ei, e := range edges {
			if r := distFromAxis(e.start(), c); stdmath.Abs(r-3) > 1e-6 {
				t.Errorf("loop %d edge %d start off the cylinder: radius %.6f, want 3", li, ei, r)
			}
			next := edges[(ei+1)%len(edges)]
			if gap := float64(e.end().DistanceTo(next.start())); gap > 1e-6 {
				t.Errorf("loop %d edge %d end does not meet the next edge's start: gap %.2e", li, ei, gap)
			}
		}
	}
}
