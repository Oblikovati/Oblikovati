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

// CSG-demotion exactness guard (M2 Phase 3, Oblikovati/Oblikovati#1336 — "retire BSP-CSG as the primary
// curved path"). The triangle-soup CSG is now only a last-resort fallback: booleanGeneral tries every
// exact analytic path first (curvedExactPaths) and reaches booleanCSG only when none applies. This is the
// regression net that keeps it that way — it runs a representative boolean from EVERY exact-path family
// through the public ops.Boolean and asserts the result is the EXACT analytic B-rep, not faceted CSG.
//
// The witness is the result faces: the CSG fallback (trianglesToBody) emits ONLY planar triangle facets —
// hundreds of them, zero analytic curved faces. So a curved-boolean result that carries an analytic
// cylinder/cone/sphere face, no non-analytic face, and a small face count provably took the exact path.
// If a refactor makes any exact path silently return ok=false, its case here flips from a handful of
// analytic faces to triangle soup and the test fails — catching the regression the umbrella worries about.

// exactCurvedFaceCeiling bounds an exact curved-boolean result's face count. Every exact case here is a
// few analytic faces (≤ ~10); the CSG fallback for these inputs is hundreds of triangles, so any result
// above this ceiling is faceted soup, not an exact B-rep.
const exactCurvedFaceCeiling = 40

// assertExactCurved fails unless res is a watertight solid that took the exact analytic curved path: at
// least one analytic curved face, no non-analytic face, and a small face count (not CSG triangle soup).
func assertExactCurved(t *testing.T, res *topo.Body, name string) {
	t.Helper()
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("%s: result is not a watertight solid: %+v", name, v)
	}
	curved, nonAnalytic := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Cone, geom.Sphere, geom.Torus:
			curved++
		case geom.Plane:
		default:
			nonAnalytic++
		}
	}
	if curved == 0 {
		t.Errorf("%s: result has no analytic curved face across %d faces — it fell back to faceted CSG, not the exact path", name, len(res.Faces()))
	}
	if nonAnalytic > 0 {
		t.Errorf("%s: result has %d non-analytic faces (want only analytic surfaces on the exact path)", name, nonAnalytic)
	}
	if n := len(res.Faces()); n > exactCurvedFaceCeiling {
		t.Errorf("%s: result has %d faces (> %d) — that is faceted CSG soup, not an exact B-rep", name, n, exactCurvedFaceCeiling)
	}
}

// curvedExactCase is one (operation, target, tool) whose result must take an exact analytic path.
type curvedExactCase struct {
	name  string
	op    ops.PartFeatureOperation
	build func(t *testing.T) (target, tool *topo.Body)
}

// TestCurvedBooleansStayExact runs a representative boolean from every exact-path family and asserts none
// silently degrades to CSG — the standing guard that BSP-CSG is the last-resort fallback, not the primary
// curved path.
func TestCurvedBooleansStayExact(t *testing.T) {
	for _, c := range curvedExactCases() {
		t.Run(c.name, func(t *testing.T) {
			target, tool := c.build(t)
			res, err := ops.Boolean(c.op, target, tool)
			if err != nil {
				t.Fatalf("%s: Boolean(%s): %v", c.name, c.op, err)
			}
			assertExactCurved(t, res, c.name)
		})
	}
}

// curvedExactCases lists one boolean per exact-path family (#1334/#1335/#1336), each built from a
// known-good geometry borrowed from that family's own test.
func curvedExactCases() []curvedExactCase {
	return []curvedExactCase{
		{"cylinder ∩ box", ops.Intersect, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10), demoBlock(t, math.P3(-2.5, -2.5, -5), math.P3(2.5, 2.5, 15))
		}},
		{"cylinder − box tunnel", ops.Cut, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 3.5, 4), csgBox(math.P3(-1, -1, -1), 2, 2, 6)
		}},
		{"box − cylinder (drilled plate)", ops.Cut, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoBlock(t, math.P3(-10, -10, 0), math.P3(10, 10, 6)), demoCyl(t, math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5, 8)
		}},
		{"crossing cylinders ∩", ops.Intersect, crossingCylinders},
		{"crossing cylinders − (drill)", ops.Cut, crossingCylinders},
		{"crossing cylinders ∪", ops.Join, crossingCylinders},
		{"equal-radius Steinmetz ∩", ops.Intersect, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12), demoCyl(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
		}},
		{"cone ∩ cylinder", ops.Intersect, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12), demoCone(t, math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5)
		}},
		{"cone ∩ cone", ops.Intersect, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCone(t, math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5), demoCone(t, math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4)
		}},
		{"partial penetration ∩", ops.Intersect, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12), demoCyl(t, math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)
		}},
		{"coaxial cylinders ∪", ops.Join, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoCyl(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4), demoCyl(t, math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
		}},
		{"cylinder boss ∪", ops.Join, func(t *testing.T) (*topo.Body, *topo.Body) {
			return demoBlock(t, math.P3(-5, -5, 0), math.P3(5, 5, 2)), demoCyl(t, math.P3(0, 0, 2), math.V3(0, 0, 1), 1.5, 3)
		}},
		{"cone ∩ box (axis-∥ flat)", ops.Intersect, coneFlatBox},
		{"cone − box (axis-∥ flat)", ops.Cut, coneFlatBox},
		{"cone ∩ box (vertex-inside flat)", ops.Intersect, coneVertexInsideFlatBox},
		{"cone − box (vertex-inside flat)", ops.Cut, coneVertexInsideFlatBox},
		{"cone ∩ box (oblique ellipse)", ops.Intersect, coneObliqueEllipseBox},
		{"cone − box (oblique ellipse)", ops.Cut, coneObliqueEllipseBox},
		{"cone ∩ box (oblique hyperbola)", ops.Intersect, coneObliqueHyperbolaBox},
		{"cone − box (oblique hyperbola)", ops.Cut, coneObliqueHyperbolaBox},
		{"cone ∩ box (parabola)", ops.Intersect, coneParabolaBox},
		{"cone − box (parabola)", ops.Cut, coneParabolaBox},
		{"cone ∩ box (oblique vertex-inside)", ops.Intersect, coneObliqueVertexInsideBox},
		{"cone − box (oblique vertex-inside)", ops.Cut, coneObliqueVertexInsideBox},
		{"cone − box (clips rim annulus)", ops.Cut, coneClipsRimBox},
		{"cone ∩ box (clips rim tongue)", ops.Intersect, coneClipsRimBox},
		{"cylinder ∩ box (oblique ellipse)", ops.Intersect, cylinderObliqueEllipseBox},
		{"cylinder − box (oblique ellipse)", ops.Cut, cylinderObliqueEllipseBox},
		{"cylinder − box (clips rim annulus)", ops.Cut, cylinderClipsRimBox},
		{"cylinder ∩ box (clips rim tongue)", ops.Intersect, cylinderClipsRimBox},
		{"cone apex − box (oblique, apex dropped)", ops.Cut, coneApexObliqueBox},
		{"cone apex ∩ box (oblique, apex kept)", ops.Intersect, coneApexObliqueBox},
		{"torus − box (perp half)", ops.Cut, torusPerpHalfBox},
		{"torus ∩ box (perp half)", ops.Intersect, torusPerpHalfBox},
		{"torus − box (perp off-centre)", ops.Cut, torusPerpOffCentreBox},
		{"torus ∩ box (axis-∥ oval cap)", ops.Intersect, torusAxisParallelOvalBox},
	}
}

// torusPerpHalfBox is a torus (R=5, r=2, axis +z) cut by the box face z=0 — PERPENDICULAR to the axis at
// the mid-plane: the section is two concentric circles, and the kept half is a trimmed torus band capped
// by a planar annulus. The only torus cut with an analytic section (an oblique plane cuts a quartic spiric
// curve, still CSG); the box's far parallel faces clear the torus and compose (Oblikovati#1375).
func torusPerpHalfBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2), demoBlock(t, math.P3(-20, -20, 0), math.P3(20, 20, 20))
}

// torusPerpOffCentreBox cuts the same torus by z=1 (off the mid-plane): the kept band's two section
// circles have unequal tube parameters, exercising the seam-direction loft over the wrapping tube arc.
func torusPerpOffCentreBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2), demoBlock(t, math.P3(-20, -20, 1), math.P3(20, 20, 20))
}

// torusAxisParallelOvalBox cuts the same torus by the box face y=6 — PARALLEL to the axis, offset between
// the inner (R−r=3) and outer (R+r=7) tube radii: the section is a single quartic SPIRIC oval, and the
// ∩ keeps the small outer-tube CAP poking past y=6 (one analytic torus face inside the oval + a planar
// oval lid). The first exact OBLIQUE torus cut (Oblikovati#1375); the box's other faces clear and compose.
func torusAxisParallelOvalBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2), demoBlock(t, math.P3(-20, 6, -20), math.P3(20, 20, 20))
}

func demoTorus(t *testing.T, center math.Point3, axis math.Vector3, major, minor float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidTorus(center, axis, major, minor, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	return b
}

// coneApexObliqueBox is a FULL cone (apex at the top, base radius 3) tilted (axis (0,0.6,0.8)) cut by the
// box face z=4. The section is an oblique ELLIPSE wholly between the base rim and the apex: the Cut keeps
// the base side (apex DROPPED → a frustum-like band) and the ∩ keeps the tip (apex KEPT → a one-loop cone
// closing to its apex pole, capped by the elliptical cut). The full-cone-to-apex case (Oblikovati#1375).
func coneApexObliqueBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 0), demoBlock(t, math.P3(-20, -20, 4), math.P3(20, 20, 30))
}

// cylinderObliqueEllipseBox is a tilted cylinder (axis (0,0.6,0.8)) cut by the axis-aligned box face z=4 —
// tilted relative to the cylinder axis, so the section is an oblique ELLIPSE wholly within the band (the
// case the old line-only cylinder split deferred to CSG, now built by the unified (u,v) split, #1375).
func cylinderObliqueEllipseBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCyl(t, math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 3, 10), demoBlock(t, math.P3(-20, -20, 4), math.P3(20, 20, 30))
}

// cylinderClipsRimBox is the same tilted cylinder cut by the box face z=7, which slices THROUGH the tilted
// top rim (its z-range is [6.2, 9.8]): the Cut keeps the clips-rim ANNULUS (ellipse arc + surviving rim
// arc), the ∩ keeps the non-wrapping TONGUE above it — both built by the unified (u,v) split (#1375).
func cylinderClipsRimBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCyl(t, math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 3, 10), demoBlock(t, math.P3(-30, -30, 7), math.P3(30, 30, 40))
}

// coneClipsRimBox is the oblique-ellipse frustum (axis (0,0.6,0.8)) cut by a box whose face z=6 slices
// THROUGH the tilted top rim (the rim z-range is [4.4, 11.6]): the section ellipse clips the top rim, so
// the kept annulus's upper boundary is the ellipse arc PLUS the surviving top-rim arc — the clips-rim
// arrangement built exactly by the (u,v) split (Oblikovati/Oblikovati#1375). The complementary ∩ keeps
// the non-wrapping TONGUE above z=6 (the surviving top-rim arc plus the section arc, pinching where the
// section meets the rim) — also built exactly by the (u,v) split's coneSideUVTongue.
func coneClipsRimBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 6), demoBlock(t, math.P3(-30, -30, 6), math.P3(30, 30, 40))
}

// coneObliqueVertexInsideBox is the tilted frustum (axis (0.2,0,0.98)) cut by the box face x=5 — far
// enough out that the oblique hyperbola's vertex falls inside the band (the oblique analogue of #1374).
func coneObliqueVertexInsideBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(2, 0, 9.797958971), 3, 6), demoBlock(t, math.P3(5, -20, -20), math.P3(40, 20, 40))
}

// coneParabolaBox is a frustum tilted by its own half-angle (one vertical generator) so the box's x=2
// face is parallel to that generator — the PARABOLIC boundary tilt (Oblikovati/Oblikovati#1375).
func coneParabolaBox(t *testing.T) (*topo.Body, *topo.Body) {
	a := stdmath.Atan(0.3)
	top := math.P3(math.Scalar(stdmath.Sin(a)*10), 0, math.Scalar(stdmath.Cos(a)*10))
	return demoCone(t, math.P3(0, 0, 0), top, 3, 6), demoBlock(t, math.P3(2, -20, -20), math.P3(40, 20, 40))
}

// coneObliqueHyperbolaBox is a frustum tilted so its axis (0.2,0,0.98) is shallower than the box's x=2
// face relative to that axis — the oblique HYPERBOLA section (vertex below the band, arms crossing both
// rims), the case that also exercises the cap-seam crossing fix (Oblikovati/Oblikovati#1375).
func coneObliqueHyperbolaBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(2, 0, 9.797958971), 3, 6), demoBlock(t, math.P3(2, -20, -20), math.P3(40, 20, 40))
}

// coneObliqueEllipseBox is a frustum tilted so its axis is (0,0.6,0.8) and an axis-aligned box whose
// only cutting face (z=4) is tilted steeper than the generators — the oblique ELLIPSE section, wholly
// within the band (Oblikovati/Oblikovati#1375).
func coneObliqueEllipseBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 6), demoBlock(t, math.P3(-20, -20, 4), math.P3(20, 20, 30))
}

// coneVertexInsideFlatBox is the same frustum with a box whose face x=4 has |D|=4 BETWEEN the bottom
// radius (3) and the top radius (6): the flat fades out before the small rim, so the imprint turns
// through the hyperbola vertex inside the side — the vertex-inside-band case (Oblikovati/Oblikovati#1374).
func coneVertexInsideFlatBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6), demoBlock(t, math.P3(4, -20, -5), math.P3(20, 20, 15))
}

// coneFlatBox is the frustum (tanα=0.3, bottom r=3, top r=6) and an oversized box whose face x=2 is
// parallel to the cone axis and cuts every cross-section — the hyperbolic-imprint case (#1372).
func coneFlatBox(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCone(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6), demoBlock(t, math.P3(2, -20, -5), math.P3(20, 20, 15))
}

// crossingCylinders is the shared thin-through-fat pair used by the three crossing-cylinder cases.
func crossingCylinders(t *testing.T) (*topo.Body, *topo.Body) {
	return demoCyl(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12), demoCyl(t, math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
}

func demoCyl(t *testing.T, base math.Point3, axis math.Vector3, r, h float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(base, axis, r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return b
}

func demoCone(t *testing.T, bottom, top math.Point3, rb, rt float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinderCone(bottom, top, rb, rt, "cone")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

func demoBlock(t *testing.T, min, max math.Point3) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(min, max, "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}
