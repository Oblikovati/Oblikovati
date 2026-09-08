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
// bored by a coaxial shaft. Their B-reps are exact and certified against independent oracles; it is the
// MESH that is not there yet, and this row is what keeps that visible until a chart-driven mesher lands.

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

// TestADiscardedTrimIsReported: the ring∩ball lens is a torus band wrapping the tube between two
// section curves. No mesher charts it, so the router meshes the whole torus — 227 mm³ where the face
// carries 24 — and the body must report the degradation.
func TestADiscardedTrimIsReported(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	ball, err := brep.SolidSphere(math.P3(5, 0, 0), 2, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	lens, err := ops.Boolean(ops.Intersect, ring, ball)
	if err != nil {
		t.Fatalf("ring ∩ ball: %v", err)
	}
	// The B-rep is right — that is the point of reporting the MESH rather than refusing the boolean.
	if v := ops.Validate(lens); !v.Valid || !v.Closed || !v.Manifold {
		t.Fatalf("the lens B-rep is not a valid closed manifold solid: %+v", v)
	}
	if !hasIgnoredTrim(t, lens) {
		t.Error("the lens meshed over the torus's whole domain without recording the discarded trim")
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
