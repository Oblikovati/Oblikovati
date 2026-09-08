// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A trim the mesher discards must SAY so (ADR-0061 stage 5).
//
// The curved-face router ends at the surface's full parametric domain for a face whose boundary wraps a
// seam no wrapping mesher recognised. That grid is the whole surface, so a TRIMMED face meshed with it
// carries material the face does not have and omits its own boundary — the neighbours meet nothing
// along it. The mesh still ships, because a wrong covering beats a missing face in a viewport, but the
// ground rules do not let it ship silently.
//
// The bodies below are the ones ADR-0061 stage 5 made buildable: a ring meeting a ball, and a ring
// bored by a coaxial shaft. Both are meshed from their own region now — the tube-wrapping band by the
// band loft, the azimuth-wrapping band by the chart-driven mesher — so these rows assert the mesh, and
// the reporter's own row moved to chart_face_cover_test.go, where a face that carries NO chart is what
// still reaches it.

// hasIgnoredTrim reports whether a body's mesh recorded the discarded-trim defect.
func hasIgnoredTrim(t *testing.T, b *topo.Body) bool {
	t.Helper()
	mesh, _ := tessellate.TessellateBody(b, ops.DefaultQuality())
	for _, d := range mesh.Diagnostics {
		if d.Code == tessellate.CodeTrimIgnoredFullDomain {
			return true
		}
	}
	return false
}

// TestATubeWrappingBandMeshesItsOwnRegion: a ball swallowing a stretch of a ring's tube leaves a band
// that WRAPS the tube between two torus∩quadric sections. Nothing charted it before — the router meshed
// the whole torus, 227 mm³ where the intersection carries 24 — and the band loft now does, because what
// it accepts is the SHAPE (a torus face with two edges that each go the whole way round the tube)
// rather than the curve kind its boundaries happen to be.
//
// Both sides of the same section are asserted, because the loft has to choose WHICH of the two bands
// the boundaries bound is the face's, and choosing wrong is invisible in one of them alone: the cut and
// the intersect are complementary, so a mesher that always takes the same side meshes them identically.
func TestATubeWrappingBandMeshesItsOwnRegion(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	ball, err := brep.SolidSphere(math.P3(5, 0, 0), 2, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	for _, c := range []struct {
		name string
		op   ops.PartFeatureOperation
		want float64 // the analytic volume, from the boolean's own certified corpus
	}{
		{"ring ∩ ball", ops.Intersect, 23.86935},
		{"ring − ball", ops.Cut, 198.19675},
	} {
		body, err := ops.Boolean(c.op, ring, ball)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		mesh, _ := tessellate.TessellateBody(body, ops.DefaultQuality())
		if free := tessellate.FreeEdgeCount(mesh); free != 0 {
			t.Errorf("%s meshed with %d free edges, want a watertight mesh", c.name, free)
		}
		if hasIgnoredTrim(t, body) {
			t.Errorf("%s reported a discarded trim; the band loft charts it now", c.name)
		}
		// A chord deficit at this quality is a couple of percent — the bare torus's own is 1.3%. What
		// this separates it from is the WRONG band, which misses by a factor.
		got := tessellate.MeshGeometryProperties(mesh).Volume
		if rel := stdmath.Abs(got-c.want) / c.want; rel > 0.05 {
			t.Errorf("%s meshes to %.5f against an analytic %.5f (rel %.4f); that is not a chord deficit",
				c.name, got, c.want, rel)
		}
	}
}

// TestABoredRingCarriesItsOwnTrim: a ring bored by a COAXIAL shaft leaves two torus bands that wrap
// the ring's AZIMUTH rather than its tube, which no wrapping mesher charted — the router meshed the
// whole torus (144.78 mm3 against 203.59) and the body reported the discarded trim.
//
// The chart-driven mesher takes it now: region from the face's carried chart, points from the shared
// edges (ADR-0061). So this row asserts the opposite of what it used to — a watertight body, a volume
// that is a chord deficit, and NO defect. What keeps the reporter honest is
// TestAFaceWithoutAChartStillReportsItsDiscardedTrim, beside recordIgnoredTrim itself.
func TestABoredRingCarriesItsOwnTrim(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	shaft, err := brep.SolidCylinder(math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8)
	if err != nil {
		t.Fatalf("shaft: %v", err)
	}
	bored, err := ops.Boolean(ops.Cut, ring, shaft)
	if err != nil {
		t.Fatalf("ring - coaxial shaft: %v", err)
	}
	if v := ops.Validate(bored); !v.Valid || !v.Closed || !v.Manifold {
		t.Fatalf("the bored ring's B-rep is not a valid closed manifold solid: %+v", v)
	}
	mesh, _ := tessellate.TessellateBody(bored, ops.DefaultQuality())
	if free := tessellate.FreeEdgeCount(mesh); free != 0 {
		t.Errorf("the bored ring meshed with %d free edges, want a watertight mesh", free)
	}
	if hasIgnoredTrim(t, bored) {
		t.Error("the bored ring still reports a discarded trim; the chart mesher charts it now")
	}
	const want = 203.59 // the boolean's own certified corpus (TestBoredRingIsExact)
	if got := tessellate.MeshGeometryProperties(mesh).Volume; stdmath.Abs(got-want) > 0.05*want {
		t.Errorf("the bored ring meshes to %.5f against an analytic %.5f; that is not a chord deficit", got, want)
	}
}

// TestAnUntrimmedFaceReportsNothing is the control that keeps the signal meaningful: a bare torus, a
// bare sphere and a drilled plate all mesh correctly, and the same full-domain grid is exactly right
// for an untrimmed closed surface — a face with no boundary loops IS its whole domain.
func TestAnUntrimmedFaceReportsNothing(t *testing.T) {
	t.Parallel()
	ring, _ := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	ball, _ := brep.SolidSphere(math.P3(0, 0, 0), 2, "ball")
	plate, _ := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	bit, _ := brep.SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 2, 4)
	drilled, err := ops.Boolean(ops.Cut, plate, bit)
	if err != nil {
		t.Fatalf("drilled plate: %v", err)
	}
	for _, c := range []struct {
		name string
		body *topo.Body
		want float64
	}{
		{"bare torus", ring, 2 * stdmath.Pi * stdmath.Pi * 5 * 1.5 * 1.5},
		{"bare sphere", ball, 4.0 / 3.0 * stdmath.Pi * 8},
		{"drilled plate", drilled, 200 - stdmath.Pi*4*2},
	} {
		if hasIgnoredTrim(t, c.body) {
			t.Errorf("%s reported a discarded trim; its mesh is correct and the signal must stay meaningful", c.name)
		}
		mesh, _ := tessellate.TessellateBody(c.body, ops.DefaultQuality())
		if free := tessellate.FreeEdgeCount(mesh); free != 0 {
			t.Errorf("%s meshed with %d free edges, want a watertight mesh", c.name, free)
		}
		// A correct mesh is within a chord deficit of the analytic volume — a few percent at this quality,
		// either way (a chorded bore removes slightly LESS than the true cylinder, so a drilled body reads
		// slightly high). What that distinguishes it from is a mesh of the WRONG region, which the lens
		// above misses by a factor of nine.
		if got := tessellate.MeshGeometryProperties(mesh).Volume; stdmath.Abs(got-c.want) > 0.05*c.want {
			t.Errorf("%s meshes to %.5f against an analytic %.5f; a chord deficit is a few percent, not this", c.name, got, c.want)
		}
	}
}
