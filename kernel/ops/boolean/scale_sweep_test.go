// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestScaleSweepInvariance is the acceptance gate for ADR-0042 Phase 1
// (Oblikovati/Oblikovati#1244): the SAME part modelled at a range of unit scales must yield
// the same topology and the same size-normalised volume. Phase 1's model-relative tolerances
// (#1243) make this hold across the kernel's full single-model working range — here 1 µm to
// 1 km, 9 orders of magnitude — where the old cm-anchored absolute tolerances failed (sub-µm
// parts were welded out of existence).
//
// The nm/pm extreme is intentionally NOT covered: below ~1 µm the floor is no longer a
// kernel weld tolerance but the fundamental vector-normalisation epsilon in primitive
// construction (a cross-product of pm-scale edges underflows), which only Phase 2
// (working-scale storage, coordinates kept O(1)) can resolve. See ADR-0042 §Phase 2.
//
// Scales are the cm value of one unit: 1 µm = 1e-4 cm … 1 km = 1e5 cm.
func TestScaleSweepInvariance(t *testing.T) {
	t.Parallel()
	scales := []struct {
		name string
		s    float64
	}{
		{"1µm", 1e-4}, {"1mm", 0.1}, {"1cm", 1}, {"1m", 100}, {"1km", 1e5},
	}

	type sig struct {
		faces, edges, verts int
		volNorm             float64 // volume / s³ — scale-independent
	}
	got := make([]sig, len(scales))
	for i, sc := range scales {
		body := scaledFilletedPart(t, sc.s)
		if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
			t.Fatalf("%s: not a valid solid: %+v", sc.name, r)
		}
		if open := ops.BoundaryEdges(body); len(open) != 0 {
			t.Fatalf("%s: %d boundary edges, want 0 (watertight)", sc.name, len(open))
		}
		vol := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
		got[i] = sig{len(body.Faces()), len(body.Edges()), len(body.Vertices()), vol / (sc.s * sc.s * sc.s)}
	}

	ref := got[1] // the mm reference
	for i, sc := range scales {
		g := got[i]
		if g.faces != ref.faces || g.edges != ref.edges || g.verts != ref.verts {
			t.Errorf("%s topology = %d/%d/%d faces/edges/verts, want %d/%d/%d (scale-variant!)",
				sc.name, g.faces, g.edges, g.verts, ref.faces, ref.edges, ref.verts)
		}
		// 0.5% band: absorbs the absolute tessellation chord tolerance's tiny facet-count
		// drift across scales (a measurement artefact, not a modelling one).
		if rel := stdmath.Abs(g.volNorm-ref.volNorm) / ref.volNorm; rel > 5e-3 {
			t.Errorf("%s normalised volume = %.9g, want %.9g (rel %.2e, scale-variant!)",
				sc.name, g.volNorm, ref.volNorm, rel)
		}
	}
}

// scaledFilletedPart builds one representative part — a box with a corner notched out by a
// boolean cut, then a vertical edge rounded by a fillet — at characteristic size s. It
// exercises the boolean (CSG weld), the fillet (curved-surface build + weld) and the
// tessellation that measures the result, all of which #1243 made model-relative.
func scaledFilletedPart(t *testing.T, s float64) *topo.Body {
	t.Helper()
	base, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(4*s, 4*s, 2*s), "base")
	if err != nil {
		t.Fatalf("SolidBlock base (s=%g): %v", s, err)
	}
	tool, err := brep.SolidBlock(math.P3(2.5*s, 2.5*s, -s), math.P3(5*s, 5*s, 3*s), "tool")
	if err != nil {
		t.Fatalf("SolidBlock tool (s=%g): %v", s, err)
	}
	cut, err := ops.Boolean(ops.Cut, base, tool)
	if err != nil {
		t.Fatalf("Boolean Cut (s=%g): %v", s, err)
	}
	filleted, err := blend.FilletEdges(cut, [][]byte{longestVerticalEdge(t, cut, s)}, 0.5*s)
	if err != nil {
		t.Fatalf("blend.FilletEdges (s=%g): %v", s, err)
	}
	return filleted
}

// longestVerticalEdge returns the reference key of the longest vertical edge (endpoints share
// X and Y to a scale-relative tolerance) — a deterministic, convex edge to round, identical
// across scales.
func longestVerticalEdge(t *testing.T, b *topo.Body, s float64) []byte {
	t.Helper()
	var best []byte
	bestDZ := 0.0
	tol := 1e-6 * s
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		dz := stdmath.Abs(a.Z - c.Z)
		if stdmath.Abs(a.X-c.X) < tol && stdmath.Abs(a.Y-c.Y) < tol && dz > bestDZ {
			best, bestDZ = e.ReferenceKey(), dz
		}
	}
	if best == nil {
		t.Fatalf("no vertical edge to fillet (s=%g)", s)
	}
	return best
}

// TestFarFromOriginDrilledPlateWatertight is the far-from-origin arm of the #1399 acceptance
// gate: a curved boolean run on a part whose coordinates sit far from the origin must stay
// watertight AND keep the exact analytic path (the hole wall a geom.Cylinder, not a faceted CSG
// prism). The model-relative weld (ADR-0042) scales with the part's extent, so coincident section
// points still merge once their separation grows with the coordinate magnitude; a cm-anchored weld
// would fail to chain the section arcs far out and demote the cut to CSG (no cylinder face).
//
// The offsets stay within float64's single-model range (≤ ~1e6 cm, the issue's spec). Beyond ~1e8
// the coordinate ULP itself exceeds a usable feature tolerance and corrupts the geometry — a
// working-scale-storage concern (ADR-0042 §Phase 2), orthogonal to #1399's relative tolerances.
func TestFarFromOriginDrilledPlateWatertight(t *testing.T) {
	t.Parallel()
	const r = 1.5
	const wantVol = 20*20*6 - stdmath.Pi*r*r*6 // analytic slab − πr²·thickness
	for _, off := range []float64{0, 1e4, 1e6} {
		slab, err := brep.SolidBlock(math.P3(off-10, off-10, 0), math.P3(off+10, off+10, 6), "slab")
		if err != nil {
			t.Fatalf("off %g: SolidBlock: %v", off, err)
		}
		rod, err := brep.SolidCylinder(math.P3(off, off, -1), math.V3(0, 0, 1), r, 8)
		if err != nil {
			t.Fatalf("off %g: SolidCylinder: %v", off, err)
		}
		drilled, err := ops.Boolean(ops.Cut, slab, rod)
		if err != nil {
			t.Fatalf("off %g: Boolean(Cut): %v", off, err)
		}
		if rr := ops.Validate(drilled); !rr.Valid || !rr.Closed || !rr.Manifold || !drilled.IsSolid() {
			t.Errorf("off %g: drilled plate not a watertight solid: %+v", off, rr)
		}
		if len(ops.BoundaryEdges(drilled)) != 0 {
			t.Errorf("off %g: %d boundary edges, want 0 (watertight)", off, len(ops.BoundaryEdges(drilled)))
		}
		if !hasCylinderFace(drilled) {
			t.Errorf("off %g: hole wall is not an analytic cylinder — the cut demoted to CSG far from origin", off)
		}
		if v := query.BodyGeometryProperties(drilled, ops.DefaultQuality()).Volume; stdmath.Abs(v-wantVol)/wantVol > 0.01 {
			t.Errorf("off %g: volume %.4f, want %.4f within 1%%", off, v, wantVol)
		}
	}
}

// TestScaleSweepSewDefaultGap checks the sew arm of the gate: Sew's default (tolerance 0) gap
// is now model-relative (ADR-0042 res.Sew()), so a quilt sews shut at any scale. A box built
// at each scale is already a solid; re-sewing it must keep it a watertight solid rather than
// rejecting or over-merging it because an absolute 1e-4 cm gap no longer suits the size.
func TestScaleSweepSewDefaultGap(t *testing.T) {
	t.Parallel()
	for _, sc := range []struct {
		name string
		s    float64
	}{{"1µm", 1e-4}, {"1mm", 0.1}, {"1m", 100}, {"1km", 1e5}} {
		box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(sc.s, sc.s, sc.s), "box")
		if err != nil {
			t.Fatalf("%s: SolidBlock: %v", sc.name, err)
		}
		sewn, err := ops.Sew(box, 0) // 0 ⇒ model-relative default gap
		if err != nil {
			t.Errorf("%s: Sew: %v", sc.name, err)
			continue
		}
		if !sewn.IsSolid() || len(ops.BoundaryEdges(sewn)) != 0 {
			t.Errorf("%s: sewn body not a watertight solid (open=%d)", sc.name, len(ops.BoundaryEdges(sewn)))
		}
	}
}
