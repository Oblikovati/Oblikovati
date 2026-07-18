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
	for _, o := range sphereCornerOracles {
		t.Run(o.name, func(t *testing.T) {
			body := corpusFixture(t, o.step)
			v := vertexNearest(t, body, o.vertex)
			faces := facesAtVertex(v)
			assertSphereHostSet(t, faces)
			cb, err := solveBlend(v, faces, 10)
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
	body := corpusFixture(t, sphereCornerOracles[0].step) // D5, host sphere R = 150
	v := vertexNearest(t, body, sphereCornerOracles[0].vertex)
	faces := facesAtVertex(v)
	if _, err := solveBlend(v, faces, 200); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("spindle r=200 on a sphere-host corner: got err %v, want %q", err, "fillet: corner face must be planar")
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
