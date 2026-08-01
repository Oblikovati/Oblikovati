// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSeamWindingSpherePatchMeshesToClosedForm is the AREA + MANIFOLD gate on the exact population the
// periodic-seam leap touches: the corner-blend spheres whose boundary loop STARTS on the (u,v) chart's
// seam and winds a whole period about the chart's axis.
//
// WHAT THE SWEEP FOUND, AND HOW IT WAS MEASURED. Measuring function: develop each loop of every face
// whose surface has a periodic parameter direction, by inverting the SHIPPED body's own boundary
// discretization (faceOuterBoundary + faceHoleBoundaries) through Surface.ParamAt and summing the
// wrapped steps around the FULL cycle, closing step included; the residual is the loop's total winding
// about the periodic axis, which is 0 for every loop that has a development and ±2πk for every loop
// that has none. Run at ops.PropertyQuality() and again at ops.DefaultQuality() over the 124 healthy
// shipped bodies (1155 faces) of the scored corpus, it returns EXACTLY 5 loops on 5 cases: simple/A6,
// C8, E4, K6 and W1, each the case's single great-circle-bounded geom.Sphere corner blend, each on the
// u axis, each leaping exactly ±2π on the CLOSING step while its OPEN chain reads 6.067–6.263 rad —
// inside the old open-chain-only guard's 2π−1e-6 threshold by as little as 0.024 rad.
//
// ★ POPULATION RE-MEASURED (patchgridcap slice, §region): K6 LEFT the population when the void-corner
// region fix (assembleFilletFaces → assembleCornerBlendBody) uniform-flipped its shell — the corner
// loop's traversal reversed and its closing step no longer leaps a period (measured: 0 seam-winding
// sphere faces on the fixed K6 body at both gate qualities). Its corner ball — which this test's own
// doc records as "the 7/8 complement", i.e. the WRONG meshed region the flip corrected — is now
// pinned by the stricter octant gate (k6_l4_trihedral_setback_test.go: area 25π/2 + solid angle π/2,
// both against closed forms). The population here is the remaining FOUR.
//
// WHY THE MESH IS NEVERTHELESS RIGHT, AND WHY THIS TEST EXISTS ANYWAY. All five are intercepted by
// spherePatchMesh, which charts the patch on its OWN axis (gnomonic/stereographic) and never asks for
// the seam development — so the leap costs no triangle today; instrumenting both mesher call sites
// (tessellateCurvedFace, conformingCylConeMesh) across the whole kernel and model suites found ZERO
// loops rejected only by the new closing-step guard. That is routing luck, not a guarantee: the
// interception happens in specialCurvedMesh, one edit away from changing, while unwrap is shared
// production mesher code. So the guard itself is fixed in kernel/ops/tessellate_trim.go and pinned by
// TestUnwrapRejectsSeamStartFullWrap (plus its three false-positive siblings), and THIS test pins the
// property that guard protects: the five faces mesh to their own exact area, fold-free.
//
// ★ THE TARGET IS A CLOSED FORM, NOT A CAPTURED NUMBER. Each patch is a spherical polygon bounded by
// GREAT-CIRCLE arcs — asserted first, exactly: every boundary edge is a circle/arc concentric with the
// sphere and of its radius — so its area is r²·Ω with Ω the spherical excess, evaluated from the loop's
// own vertices by the Van Oosterom–Strackee formula, which is exact for geodesic edges. Independent
// corroboration: C8's cap comes out 448.3874 against c8_apex_test.go's separately-derived c8Girard
// 448.387; simple/W1 and A6 land on a spherical OCTANT (π/2·r²) and K6 on its 7/8 complement, both to
// 10 significant figures.
func TestSeamWindingSpherePatchMeshesToClosedForm(t *testing.T) {
	t.Parallel()
	dir := CorpusFixtureDir()
	for _, name := range []string{"A6", "C8", "E4", "W1"} {
		t.Run(name, func(t *testing.T) {
			body, ok := shippedCaseBody(caseRecord(t, "simple", name), dir)
			if !ok {
				t.Fatalf("simple/%s has no healthy shipped body to measure", name)
			}
			assertSeamWindingPatchMeshes(t, body)
		})
	}
}

// caseRecord finds one corpus record by grid and case name.
func caseRecord(t *testing.T, grid, name string) Record {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == name {
			return r
		}
	}
	t.Fatalf("no corpus record %s/%s", grid, name)
	return Record{}
}

// spherePatchAreaTol is how far the chordal mesh may fall short of the exact spherical area. A mesh
// INSCRIBED in a sphere always under-reports; the measured worst of the population was 0.196 % (K6's
// then-7/8 patch, since removed — see the ★ population note; the density fix shrank the remaining
// four's deficits further), so 0.5 % separates chordal deficit from a development defect. The
// upper bound is not a tolerance at all: an inscribed triangulation cannot exceed the true area, so
// anything above it is a fold or a self-overlap.
const spherePatchAreaTol = 0.005

// assertSeamWindingPatchMeshes finds the body's seam-winding sphere patch — there must be exactly one,
// which is the population claim itself — and measures its mesh against its own exact spherical area.
func assertSeamWindingPatchMeshes(t *testing.T, body *topo.Body) {
	t.Helper()
	var hits []*topo.Face
	for _, f := range body.Faces() {
		sph, isSphere := f.Geometry().(geom.Sphere)
		if isSphere && outerLoopSeamWinding(f, sph) != 0 {
			hits = append(hits, f)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("%d sphere faces develop with a nonzero seam winding, want exactly 1 — the population moved", len(hits))
	}
	f := hits[0]
	sph := f.Geometry().(geom.Sphere)
	want := exactSphericalPolygonArea(t, f, sph)
	m := ops.TessellateFace(f, ops.PropertyQuality())
	got := ops.MeshArea(m)
	if got > want*(1+1e-9) {
		t.Errorf("face %d meshes %.10g, ABOVE its exact spherical area %.10g — an inscribed mesh cannot, so the patch overlaps itself",
			f.ID(), got, want)
	}
	if rel := (want - got) / want; rel > spherePatchAreaTol {
		t.Errorf("face %d meshes %.10g against the exact spherical area %.10g (short by %.4g %%, budget %.4g %%)",
			f.ID(), got, want, rel*100, spherePatchAreaTol*100)
	}
	assertFaceFoldFreeAtEveryQuality(t, fmt.Sprintf("face %d", f.ID()), f, m)
}

// outerLoopSeamWinding is how many whole periods the face's outer loop winds about the sphere's polar
// (u) axis: the sum of the wrapped u steps around the CLOSED ring, in periods. It reproduces the
// kernel's own criterion from the same discretization the mesher uses, so the test names its population
// by provenance rather than by a hard-coded face id.
func outerLoopSeamWinding(f *topo.Face, sph geom.Sphere) int {
	loop := denseOuterLoop(f)
	if len(loop) < 3 {
		return 0
	}
	total := 0.0
	for i := range loop {
		u0, _ := sph.ParamAt(loop[i])
		u1, _ := sph.ParamAt(loop[(i+1)%len(loop)])
		total += wrapToPi(u1 - u0)
	}
	return int(stdmath.Round(total / (2 * stdmath.Pi)))
}

// wrapToPi folds an angle into (−π, π] — the signed shortest step between two periodic samples.
func wrapToPi(a float64) float64 {
	for a > stdmath.Pi {
		a -= 2 * stdmath.Pi
	}
	for a <= -stdmath.Pi {
		a += 2 * stdmath.Pi
	}
	return a
}

// denseOuterLoop is the face's outer loop as a point ring, each edge sampled the way the mesher samples
// it, oriented to the edge use, with the shared vertices not repeated.
func denseOuterLoop(f *topo.Face) []math.Point3 {
	var out []math.Point3
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		for _, u := range l.EdgeUses() {
			pts := ops.TessellateEdge(u.Edge(), ops.PropertyQuality())
			if u.Reversed() {
				pts = reversedPoints(pts)
			}
			if len(out) > 0 {
				pts = pts[1:]
			}
			out = append(out, pts...)
		}
	}
	if n := len(out); n > 1 && out[0].DistanceTo(out[n-1]) < 1e-9 {
		out = out[:n-1]
	}
	return out
}

// reversedPoints returns the points back to front (new slice).
func reversedPoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// exactSphericalPolygonArea is the CLOSED FORM: r² times the signed solid angle the face's outer loop
// subtends at the sphere's centre, taking the complement when the loop runs the other way round. It
// asserts every boundary edge is a GREAT circle first, which is what makes the formula exact.
func exactSphericalPolygonArea(t *testing.T, f *topo.Face, sph geom.Sphere) float64 {
	t.Helper()
	dirs := greatCircleLoopDirections(t, f, sph)
	if len(dirs) < 3 {
		t.Fatalf("face %d outer loop has %d vertices, too few for a spherical polygon", f.ID(), len(dirs))
	}
	omega := solidAngleFan(dirs)
	if omega < 0 {
		omega += 4 * stdmath.Pi // the loop bounds the COMPLEMENT of the polygon it traverses
	}
	return sph.Radius * sph.Radius * omega
}

// greatCircleLoopDirections returns the outer loop's vertices as unit directions from the sphere's
// centre, in traversal order, failing the test if any boundary edge is not a great-circle arc (a small
// circle would make the spherical-excess formula inexact, so the target would not be a closed form).
func greatCircleLoopDirections(t *testing.T, f *topo.Face, sph geom.Sphere) []math.Vector3 {
	t.Helper()
	var dirs []math.Vector3
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		for _, u := range l.EdgeUses() {
			e := u.Edge()
			assertGreatCircleEdge(t, f, e, sph)
			p := e.StartVertex().Point()
			if u.Reversed() {
				p = e.EndVertex().Point()
			}
			dirs = append(dirs, sph.Center.VectorTo(p).AsUnit().AsVector())
		}
	}
	return dirs
}

// assertGreatCircleEdge fails unless the edge's curve is a circle or arc concentric with the sphere and
// of the sphere's own radius — i.e. a geodesic on it.
func assertGreatCircleEdge(t *testing.T, f *topo.Face, e *topo.Edge, sph geom.Sphere) {
	t.Helper()
	tol := 1e-9 * stdmath.Max(1, sph.Radius)
	var center math.Point3
	var radius float64
	switch c := e.Geometry().(type) {
	case geom.Circle:
		center, radius = c.Center, c.Radius
	case geom.Arc3d:
		center, radius = c.Center, c.Radius
	default:
		t.Fatalf("face %d edge %d carries %T, not a circular arc — no exact spherical excess", f.ID(), e.ID(), c)
	}
	if d := float64(center.DistanceTo(sph.Center)); d > tol {
		t.Fatalf("face %d edge %d is a SMALL circle: its centre is %.6g off the sphere's (tol %.3g)", f.ID(), e.ID(), d, tol)
	}
	if d := stdmath.Abs(radius - sph.Radius); d > tol {
		t.Fatalf("face %d edge %d radius %.10g against the sphere's %.10g (tol %.3g)", f.ID(), e.ID(), radius, sph.Radius, tol)
	}
}

// solidAngleFan is the SIGNED solid angle of a spherical polygon, fanned from its first vertex with the
// Van Oosterom–Strackee formula. It is exact for geodesic edges, so extra vertices lying on an arc
// change nothing — which is what lets the loop's own vertex list stand in for the polygon's corners.
func solidAngleFan(dirs []math.Vector3) float64 {
	sum := 0.0
	for i := 1; i+1 < len(dirs); i++ {
		a, b, c := dirs[0], dirs[i], dirs[i+1]
		num := float64(a.Dot(b.Cross(c)))
		den := 1 + float64(a.Dot(b)) + float64(b.Dot(c)) + float64(c.Dot(a))
		sum += 2 * stdmath.Atan2(num, den)
	}
	return sum
}
