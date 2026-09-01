// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// sphereCornerOracle is one sphere-host trihedral corner from OCCT tests/blend/simple, with the
// picked corner VERTEX (nearest-body-vertex locator) and the DRAWEXE-measured corner-ball CENTRE
// (sphere-host-corner-derivation.md §"DRAWEXE-verified numbers"). r = 10 in every case.
type sphereCornerOracle struct {
	name   string
	step   string
	vertex math.Point3
	center math.Point3 // OCCT analytic corner-face centre, radius 10
}

// sphereCornerOracles are the three sphere-host corners this slice (SP2) solves analytically. Before
// SP2 each falls to solvePlanarBlend → "corner face must be planar"; after SP2 solveBlend returns the
// analytic geom.Sphere corner whose centre matches DRAWEXE to ~1e-9 (residuals reported by the test).
var sphereCornerOracles = []sphereCornerOracle{
	{"D5", "simple/D5.step", math.P3(-75, 0, 129.9038105676658), math.P3(-71.5756677325, 10, 119.9038105676658)},
	{"D9", "simple/D9.step", math.P3(0, -75, 129.9038105676658), math.P3(-10, -71.5756677325, 119.9038105676658)},
	{"E4", "simple/E4.step", math.P3(0, 0, -150), math.P3(-10, 10, -139.2838827718)},
}

// corpusFixture reads a STEP fixture from the occtparity corpus (the sphere-host inputs live there,
// not under kernel/ops/testdata) as our body — the SAME import path the corpus harness uses.
func corpusFixture(t *testing.T, rel string) *topo.Body {
	t.Helper()
	path := filepath.Join("..", "..", "model", "feature", "occtparity", "fixtures", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		t.Fatalf("import %s: %v", path, err)
	}
	if len(bodies) != 1 {
		t.Fatalf("%s produced %d bodies, want 1", path, len(bodies))
	}
	return bodies[0]
}

// vertexNearest returns the body vertex closest to p — the corner-vertex locator for the direct
// solveBlend drive (the corpus harness locates by picked edges; here we pin the shared vertex).
func vertexNearest(t *testing.T, b *topo.Body, p math.Point3) *topo.Vertex {
	t.Helper()
	var best *topo.Vertex
	bestD := stdmath.Inf(1)
	for _, v := range b.Vertices() {
		if d := float64(v.Point().DistanceTo(p)); d < bestD {
			bestD, best = d, v
		}
	}
	if best == nil {
		t.Fatalf("body has no vertices")
	}
	return best
}

// TestSphereHostCornerCentre drives solveBlend on the sphere-host trihedral corner of each real
// D5/D9/E4 body and asserts the analytic corner-ball centre matches DRAWEXE (SP2). RED before SP2
// (sphere host → solvePlanarBlend → "corner face must be planar"); GREEN after the sphere corner
// solve lands. It also asserts the host set at the vertex is exactly {1 sphere, 2 planes}.
func TestSphereHostCornerCentre(t *testing.T) {
	t.Parallel()
	for _, o := range sphereCornerOracles {
		t.Run(o.name, func(t *testing.T) {
			body := corpusFixture(t, o.step)
			v := vertexNearest(t, body, o.vertex)
			faces := facesAtVertex(v)
			assertSphereHostSet(t, faces)
			cb, err := solveBlend(nil, v, faces, 10)
			if err != nil {
				t.Fatalf("%s: solveBlend declined the sphere-host corner: %v", o.name, err)
			}
			res := float64(cb.center.DistanceTo(o.center))
			t.Logf("%s corner centre %v, oracle %v, residual %.3e", o.name, cb.center, o.center, res)
			if res > 1e-6 {
				t.Fatalf("%s: corner centre %v != oracle %v (residual %.3e > 1e-6)", o.name, cb.center, o.center, res)
			}
			if cb.sphere.Radius != 10 {
				t.Fatalf("%s: corner sphere radius %g, want 10", o.name, cb.sphere.Radius)
			}
		})
	}
}

// TestSphereHostCornerSpindleRejects asserts do-no-harm: a spindle radius r > R on a real sphere-host
// corner (ρ = R−r ≤ 0, the ball engulfs the host) honest-rejects with the EXACT historical string, so an
// unsolvable sphere corner still errors exactly as an all-planar/cylinder decline would.
func TestSphereHostCornerSpindleRejects(t *testing.T) {
	t.Parallel()
	body := corpusFixture(t, sphereCornerOracles[0].step) // D5, host sphere R = 150
	v := vertexNearest(t, body, sphereCornerOracles[0].vertex)
	faces := facesAtVertex(v)
	if _, err := solveBlend(nil, v, faces, 200); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("spindle r=200 on a sphere-host corner: got err %v, want %q", err, "fillet: corner face must be planar")
	}
}

// concaveSphereCornerFixture builds a synthetic sphere-host corner with the SAME geometry as the D5
// oracle (sphere-host-corner-derivation.md) — sphere O=(0,0,0) R=150, planes y=0 (outward −ŷ) and
// z=129.9038105676658 (outward +ẑ), vertex (−75,0,129.9038105676658) — but the sphere face REVERSED:
// material OUTSIDE the sphere, a concave-bore corner (SP1's dimpleRimFixtureEdge pattern generalised
// from a two-face edge to the three-face trihedral corner). solveBlend only reads v.Point() and the
// three faces' Geometry()/Reversed() (no edge/loop wiring), so this minimal fixture drives the real
// solveBlend→sphereHostCorner→solveSphereBlend path without a full topological body.
func concaveSphereCornerFixture(t *testing.T) (*topo.Vertex, []*topo.Face) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "sphere-corner-concave", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(sphereCornerOracles[0].vertex, lin)
	sphereFace := bld.AddReversedFace(d5r150(t), lin) // material OUTSIDE the sphere: concave bore
	plane0 := bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, -1, 0)), lin)
	plane1 := bld.AddFace(planeOn(t, math.P3(0, 0, 129.9038105676658), math.V3(0, 0, 1)), lin)
	return v, []*topo.Face{sphereFace, plane0, plane1}
}

// TestSphereHostCornerConcaveRejects is the SP2-review regression: a concave-bore sphere host (material
// OUTSIDE the sphere, Reversed face) must honest-reject with the exact historical string via the s-gate
// in sphereCornerRho (s = (v−O)·n̂_host,out ≤ 0), rather than solve a ρ = R+r corner ball — that concave
// case is explicitly out of SP2's scope (sphereCornerRho's doc comment: "a follow-on slice"). Mutation
// witness: flipping the gate's sign (s<=0 → s>=0) would let this fixture's s=−150 through and go on to
// build a wrong-side (ρ=R−r instead of R+r) ball instead of rejecting; this test pins the reject.
func TestSphereHostCornerConcaveRejects(t *testing.T) {
	t.Parallel()
	v, faces := concaveSphereCornerFixture(t)
	if _, err := solveBlend(nil, v, faces, 10); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("concave sphere-host corner: got err %v, want %q", err, "fillet: corner face must be planar")
	}
}

// TestSphereRootsSeparated_CollapsedRootsReject is the SP2-review regression for the grazing/no-real-root
// branch. A cleanly-negative discriminant (the plane-pair line clearing the offset sphere outright) is
// NOT a discriminating fixture: sphereLineParam's √(negative) already propagates NaN through nearerRoot,
// and sphereCornerConsistent's final `< res.Weld()` comparison is false for any NaN operand — so a
// disabled discriminant guard is silently backstopped by that unrelated consistency check and the test
// would still pass even with sphereRootsSeparated's own logic gutted (verified live: stubbing it to
// `return true` and also removing nearerRoot's `disc<0` guard still left the corner rejected via NaN
// propagation). The ONE case that is genuinely unique to sphereRootsSeparated is a near-tangent line
// whose discriminant is small and POSITIVE — a real (non-NaN), consistency-passing pair of roots that
// simply sit closer together than curvedCornerBandK·res.Weld() (sphereLineParam's doc comment: "the
// near/far pick is noise"). Constructing that exact regime from real plane/sphere geometry is itself
// ill-conditioned (it sits at the float64 precision floor by definition), so — per this task's documented
// fallback — this drives sphereRootsSeparated directly with hand-picked qa/qb/qc reproducing a small,
// clean, reproducible positive discriminant, rather than reaching it through the full solveBlend pipeline.
//
// rho=1, u=(x0,0,0) with x0=1−1e−12, d=(0,0,1): qc = x0²−rho² ≈ −2e−12, disc = qb²−4·qa·qc ≈ 8e−12,
// sep = √disc/|d| ≈ 2.83e−6 — a genuine disc>0 pair, but far inside the band=4e−4 a 1e5-unit-model
// Resolution gives (curvedCornerBandK·1e−9·1e5), so it must still reject as collapsed.
func TestSphereRootsSeparated_CollapsedRootsReject(t *testing.T) {
	t.Parallel()
	x0 := 1 - 1e-12
	qa, qb, qc := 1.0, 0.0, x0*x0-1
	d := math.V3(0, 0, 1)
	res := ResolutionForSize(1e5)
	if disc := qb*qb - 4*qa*qc; disc <= 0 {
		t.Fatalf("fixture invalid: discriminant %.3e is not positive (want a real near-tangent pair)", disc)
	}
	if sphereRootsSeparated(qa, qb, qc, d, res) {
		t.Fatalf("collapsed near-tangent roots accepted as separated (qa=%v qb=%v qc=%v), want reject", qa, qb, qc)
	}
}

// assertSphereHostSet checks the corner vertex bounds exactly one sphere host and two planes — the
// recognizer's precondition (sphereHostCorner), so a locator drift is caught here, not downstream.
func assertSphereHostSet(t *testing.T, faces []*topo.Face) {
	t.Helper()
	nSph, nPl := 0, 0
	for _, f := range faces {
		switch f.Geometry().(type) {
		case geom.Sphere:
			nSph++
		case geom.Plane:
			nPl++
		}
	}
	if nSph != 1 || nPl != 2 || len(faces) != 3 {
		t.Fatalf("corner host set = %d faces (%d sphere, %d plane), want 1 sphere + 2 planes", len(faces), nSph, nPl)
	}
}
