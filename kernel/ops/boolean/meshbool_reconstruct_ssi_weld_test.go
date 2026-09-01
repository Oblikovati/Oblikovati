// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// ADR-0056 Layer 4 — SSI-edge weld. An oblique plane∩cylinder cut is an ELLIPSE. The
// endpoints+midpoint weld key (curved_stitch.edgeKey) fuses the two incident faces'
// copies of the ellipse into ONE topo.Edge (IntersectSurfacesAnalytic canonicalises the
// curve, so both copies are bit-identical), and orientRunEdge picks the arc HALF through
// the run's interior vertex so the stored elliptical arc is the boundary the run actually
// traces. A CLEAN oblique cut (one ellipse per pierced face) now rebuilds analytically;
// only a cut that crosses an operand CORNER (its ellipse split across two faces, an
// Euler-inadmissible assembly) still declines to the exact faceted fallback.

// hasEllipseEdge reports whether the body carries an analytic elliptical boundary edge.
func hasEllipseEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		switch e.Geometry().(type) {
		case geom.EllipticalArc, geom.EllipseFull:
			return true
		}
	}
	return false
}

// facetedVolume is the exact mesh-arrangement volume of `target op tool`, built straight from
// the soup (soupToBody) — independent of the reconstruction path, so it is a valid oracle for
// the reconstructed body's volume.
func facetedVolume(t *testing.T, op PartFeatureOperation, target, tool *topo.Body) float64 {
	t.Helper()
	faceted := meshArrangementFallback(op, target, tool, &diag.Recorder{})
	if faceted == nil {
		t.Fatal("faceted oracle produced no body")
	}
	return query.BodyGeometryProperties(faceted, PropertyQuality()).Volume
}

// TestReconstructObliqueBoreRebuilds: a tilted cylinder bored cleanly through a slab's top and
// bottom faces (an ellipse at each) reconstructs to a valid analytic genus-1 solid — both
// elliptical rims analytic, the bore wall a single cylinder face, the exact volume.
func TestReconstructObliqueBoreRebuilds(t *testing.T) {
	t.Parallel()
	slab, err := brep.SolidBlock(m.P3(-1, -1, 0), m.P3(1, 1, 0.5), "slab")
	if err != nil {
		t.Fatalf("slab: %v", err)
	}
	bore, err := brep.SolidCylinder(m.P3(0, 0, -0.2), m.V3(0.3, 0, 1), 0.2, 1.0)
	if err != nil {
		t.Fatalf("bore: %v", err)
	}
	recon, ok := reconstructBoolean(slab, bore, meshbool.Difference, DefaultQuality())
	if !ok {
		t.Fatal("clean oblique bore declined; the ellipse SSI must reconstruct (ADR-0056 Layer 4)")
	}
	if r := Validate(recon); !r.Valid || !r.Closed || !r.Manifold || !recon.IsSolid() {
		t.Fatalf("oblique bore is not a valid closed manifold solid: %+v", r)
	}
	if n := cylinderFaceCount(recon); n != 1 {
		t.Fatalf("oblique bore kept %d cylinder walls, want 1", n)
	}
	if !hasEllipseEdge(recon) {
		t.Fatal("oblique bore has no analytic elliptical rim edge — it faceted the ellipse")
	}
	want := facetedVolume(t, Cut, slab, bore)
	if v := query.BodyGeometryProperties(recon, PropertyQuality()).Volume; stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("oblique bore volume = %.5f, want ~%.5f (the exact mesh union)", v, want)
	}
}

// TestReconstructObliqueCylinderBoxUnion: a tilted cylinder UNIONed with a box, poking out
// through the box top at an angle, reconstructs analytically — the ellipse where the cylinder
// wall meets the box top is a shared analytic edge, and the exposed stub keeps its cylinder wall
// and circular cap. This is the #2167-sibling "cyl∪box" the Layer-4 weld unlocks.
func TestReconstructObliqueCylinderBoxUnion(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(m.P3(-1, -1, 0), m.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	stub, err := brep.SolidCylinder(m.P3(0, 0, 0.5), m.V3(0.4, 0, 1), 0.3, 1.2)
	if err != nil {
		t.Fatalf("stub: %v", err)
	}
	recon, ok := reconstructBoolean(box, stub, meshbool.Union, DefaultQuality())
	if !ok {
		t.Fatal("oblique cyl∪box declined; the ellipse seam must reconstruct (ADR-0056 Layer 4)")
	}
	if r := Validate(recon); !r.Valid || !r.Closed || !r.Manifold || !recon.IsSolid() {
		t.Fatalf("cyl∪box union is not a valid closed manifold solid: %+v", r)
	}
	if n := cylinderFaceCount(recon); n < 1 {
		t.Fatalf("cyl∪box union kept %d cylinder walls, want >=1", n)
	}
	if !hasEllipseEdge(recon) {
		t.Fatal("cyl∪box union has no analytic elliptical seam edge — it faceted the ellipse")
	}
	want := facetedVolume(t, Join, box, stub)
	if v := query.BodyGeometryProperties(recon, PropertyQuality()).Volume; stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("cyl∪box union volume = %.5f, want ~%.5f (the exact mesh union)", v, want)
	}
}
