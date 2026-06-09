// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// boxNamed builds a box whose faces carry the given feature name in their lineage, so two
// operands can have distinct reference keys.
func boxNamed(name string, px, py, pz, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(px, py, pz))
	}
	return subd.ToBody(m, name)
}

// faceWithNormal returns the first face whose outward normal points along dir.
func faceWithNormal(b *topo.Body, dir math.Vector3) *topo.Face {
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Dot(dir) > 0.9 {
			return f
		}
	}
	return nil
}

// K1a: a face that survives a boolean unchanged keeps its reference key, so a pick made
// before the boolean rebinds to the result face afterwards.
func TestBooleanPreservesSurvivingFaceKey(t *testing.T) {
	a := boxNamed("partA", 0, 0, 0, 2, 2, 2)
	b := boxNamed("partB", 5, 0, 0, 2, 2, 2) // disjoint: every A face survives whole

	key := a.Faces()[0].ReferenceKey()
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.FindFaceByKey(key); !ok {
		t.Fatal("a surviving face's reference key did not carry through the union (K1a)")
	}
}

// A face untouched by a Cut keeps its key (the part-side pick survives a hole/cut edit).
func TestBooleanCutPreservesUntouchedFaceKey(t *testing.T) {
	// A=[0,2]³; B overlaps the top, leaving A's bottom face (z=0, normal −Z) untouched.
	a := boxNamed("part", 0, 0, 0, 2, 2, 2)
	b := boxNamed("tool", 1, 0.5, 0.5, 2, 2, 2)
	bottom := faceWithNormal(a, math.V3(0, 0, -1))
	if bottom == nil {
		t.Fatal("no bottom face on A")
	}
	key := bottom.ReferenceKey()

	res, err := brep.Boolean(brep.Difference, a, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.FindFaceByKey(key); !ok {
		t.Fatal("the untouched bottom face's key did not survive the cut (K1a)")
	}
}

// Sanity: the disjoint union is still a valid two-shell solid (volume 16).
func TestBooleanDisjointUnionValid(t *testing.T) {
	a := boxNamed("a", 0, 0, 0, 2, 2, 2)
	b := boxNamed("b", 5, 0, 0, 2, 2, 2)
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil || res == nil {
		t.Fatalf("disjoint union: %v", err)
	}
	if v := vol(res); stdmath.Abs(v-16) > 1e-6 {
		t.Errorf("disjoint union volume = %g, want 16", v)
	}
}
