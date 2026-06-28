// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Exact drilled-plate boolean (M2 Phase 3, Oblikovati/Oblikovati#1336). A straight cylinder drilling a
// clean through-hole in an all-planar slab must come back as an EXACT curved B-rep — the cylinder surface
// preserved as the hole wall — not triangle-soup CSG. This is the reverse of the #1334 cylinder − box
// case and the most common curved cut in real parts (a drilled plate). The guards: the result carries a
// geom.Cylinder wall face, its volume is the analytic slab − πr²h (NOT a faceted approximation), and it is
// a watertight manifold solid.

// slabBody is a validated all-planar block, the plate the tests drill.
func slabBody(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(-10, -10, 0), math.P3(10, 10, 6), "slab")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// throughRod is a cylinder long enough to pierce the slab's two z-faces, centred at (cx, cy).
func throughRod(t *testing.T, cx, cy, radius float64) *topo.Body {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(cx, cy, -1), math.V3(0, 0, 1), radius, 8)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return rod
}

// hasCylinderFace reports whether any face of the body is an analytic cylinder — proof the hole wall is an
// exact surface, not a faceted prism.
func hasCylinderFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return true
		}
	}
	return false
}

// TestDrilledPlateIsExact drills one through-hole and checks the result is an exact curved solid: a
// cylinder wall face survives, the volume is the analytic slab minus πr²·thickness (a faceted CSG hole
// would be a few percent off), and it is watertight and manifold.
func TestDrilledPlateIsExact(t *testing.T) {
	slab := slabBody(t)
	v0 := ops.BodyGeometryProperties(slab, ops.DefaultQuality()).Volume
	const r = 1.5

	res, err := ops.Boolean(ops.Cut, slab, throughRod(t, 0, 0, r))
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if rr := ops.Validate(res); !rr.Valid || !rr.Closed || !rr.Manifold || !res.IsSolid() {
		t.Fatalf("drilled plate not a watertight solid: %+v", rr)
	}
	if !hasCylinderFace(res) {
		t.Error("drilled plate has no geom.Cylinder face — the hole wall fell back to a faceted prism (CSG)")
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := v0 - stdmath.Pi*r*r*6 // slab − πr²·thickness, exact
	// The analytic hole is exact; only the tessellation of the curved wall for the volume integral carries
	// chord error, well under 1% at DefaultQuality. A CSG faceted hole would miss by the inscribed-polygon
	// deficit (several %), so this tight bound is what distinguishes the exact path.
	if rel := stdmath.Abs(got-want) / want; rel > 0.005 {
		t.Errorf("drilled-plate volume %.5f, want %.5f (rel %.4f > 0.005); hole is not exact", got, want, rel)
	}
}

// TestCurvedBooleanKeepsTargetEdgeIdentity is ADR-0043 P4: the exact analytic curved cut keeps the
// hole's cylinder wall AND lets the target's untouched boundaries keep their identity — a slab edge
// the drill never reached stays bound to its slab:* key rather than a fresh curvedbool:e#N ordinal,
// so a selection on it survives the operation.
func TestCurvedBooleanKeepsTargetEdgeIdentity(t *testing.T) {
	slab := slabBody(t)
	in := map[string]bool{}
	for _, e := range slab.Edges() {
		in[string(e.ReferenceKey())] = true
	}
	res, err := ops.Boolean(ops.Cut, slab, throughRod(t, 0, 0, 1.5))
	if err != nil {
		t.Fatal(err)
	}
	if !hasCylinderFace(res) {
		t.Fatal("not the analytic curved path (no cylinder face) — test premise invalid")
	}
	kept := 0
	for _, e := range res.Edges() {
		if in[string(e.ReferenceKey())] {
			kept++
		}
	}
	if kept == 0 {
		t.Error("no slab edge kept its identity through the curved cut — all renamed to curvedbool:e#N")
	}
}

// TestDrilledPlateClippedDefers: a hole whose circle clips the slab edge is NOT a clean through-hole, so
// the exact path declines and the general boolean still produces a valid solid (the fallback is intact).
func TestDrilledPlateClippedDefers(t *testing.T) {
	slab := slabBody(t)
	res, err := ops.Boolean(ops.Cut, slab, throughRod(t, 9.5, 0, 1.5)) // circle spills past x=10
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if rr := ops.Validate(res); !rr.Valid || !res.IsSolid() {
		t.Fatalf("clipped-hole fallback not a valid solid: %+v", rr)
	}
}
