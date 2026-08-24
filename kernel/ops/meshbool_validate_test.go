// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// TestValidateCrossingCylinders drives the mesh-arrangement engine on real curved
// geometry: two crossing cylinders, each tessellated to ~124 triangles. It guards
// two things. First, the adapter's vertex weld — TessellateBody meshes each face
// independently, so the cap/wall rim vertices differ by ~1 ulp and the raw soup is
// NOT watertight (16 cracks); without the weld the boolean inherits an open input.
// Second, that CO-REFINEMENT is watertight: the robust Delaunay CDT (layer 2) must
// split both operands so each stays a closed mesh (aOut/bOut have no open edge) —
// the property whose absence tears #2084, now guaranteed here as on the coil.
//
// Third, that the FULL result is a valid closed solid. This near-tangent case once
// left a couple of open edges from generalized-winding-number classification of a
// face whose centroid sits almost on the other cylinder's surface; the exact
// ray-cast point-in-solid classifier removed that residual, so the union now
// validates end to end.
func TestValidateCrossingCylinders(t *testing.T) {
	a, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("cylinder a: %v", err)
	}
	b, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatalf("cylinder b: %v", err)
	}
	q := DefaultQuality()
	sa, sb := bodyToSoup(a, q), bodyToSoup(b, q)

	// The weld must make the tessellated operand soups watertight.
	if soupOpenEdges(sa) != 0 || soupOpenEdges(sb) != 0 {
		t.Fatalf("adapter soup not watertight after weld: a=%d b=%d open edges", soupOpenEdges(sa), soupOpenEdges(sb))
	}
	// The CDT co-refinement must keep both operands watertight (the #2084 fix).
	aOut, bOut := meshbool.CoRefine(sa, sb)
	if soupOpenEdges(aOut) != 0 || soupOpenEdges(bOut) != 0 {
		t.Fatalf("co-refinement is not watertight: aOut=%d bOut=%d open edges", soupOpenEdges(aOut), soupOpenEdges(bOut))
	}
	// With exact ray-cast classification the full union is a valid closed solid.
	res := booleanViaMeshbool(a, b, meshbool.Union, q, "op")
	if !res.IsSolid() {
		t.Fatal("union result is not a solid")
	}
	if r := Validate(res); !r.Valid {
		t.Fatalf("union result invalid: %+v", r)
	}
	volA := BodyGeometryProperties(a, q).Volume
	volB := BodyGeometryProperties(b, q).Volume
	if volU := BodyGeometryProperties(res, q).Volume; volU < volA || volU > volA+volB {
		t.Fatalf("union volume %.3f out of range [%.3f, %.3f]", volU, volA, volA+volB)
	}
}

// soupOpenEdges counts directed edges of a soup whose reverse is absent (a
// watertight mesh has zero).
func soupOpenEdges(soup [][3]meshbool.Point) int {
	edge := map[string]int{}
	key := func(p, q meshbool.Point) string { return pointKey(p) + "->" + pointKey(q) }
	for _, tri := range soup {
		for e := 0; e < 3; e++ {
			edge[key(tri[e], tri[(e+1)%3])]++
		}
	}
	open := 0
	for _, tri := range soup {
		for e := 0; e < 3; e++ {
			if edge[pointKey(tri[(e+1)%3])+"->"+pointKey(tri[e])] == 0 {
				open++
			}
		}
	}
	return open
}
