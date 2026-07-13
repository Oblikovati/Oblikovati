// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestIsAssembly distinguishes an assembly (.iam) from a part (.ipt) by its content segment.
func TestIsAssembly(t *testing.T) {
	if !openDoc(t, "asm_two_boxes.iam").IsAssembly() {
		t.Error("asm_two_boxes.iam not detected as an assembly")
	}
	if openDoc(t, "10_box.ipt").IsAssembly() {
		t.Error("10_box.ipt wrongly detected as an assembly")
	}
}

// TestDecodeOccurrences recovers the two placed instances of the box component.
func TestDecodeOccurrences(t *testing.T) {
	d := openDoc(t, "asm_two_boxes.iam")
	seg, ok := d.Segment("AmDcSegment")
	if !ok {
		t.Fatal("no AmDcSegment")
	}
	occ := DecodeOccurrences(seg)
	if len(occ) != 2 {
		t.Fatalf("got %d occurrences, want 2: %+v", len(occ), occ)
	}
	for i, want := range []Occurrence{{"asm_box", 1}, {"asm_box", 2}} {
		if occ[i] != want {
			t.Errorf("occurrence[%d] = %+v, want %+v", i, occ[i], want)
		}
	}
	if refs := ComponentRefs(occ); len(refs) != 1 || refs[0] != "asm_box" {
		t.Errorf("component refs = %v, want [asm_box]", refs)
	}
}

// TestDecodeAssemblyTransforms decodes the placed occurrences of the distinctive-translation
// assembly: two box instances, one at the origin (identity) and one translated (13,17,19) cm.
// This exercises the full node-graph path (M-stream metadata → block walk → 232792BC →
// Transformation3D).
func TestDecodeAssemblyTransforms(t *testing.T) {
	d := openDoc(t, "asm_distinct.iam")
	placed := DecodeAssembly(d)
	if len(placed) != 2 {
		t.Fatalf("got %d placed occurrences, want 2: %+v", len(placed), placed)
	}
	tx := func(p PlacedOccurrence) (float64, float64, float64) {
		return p.Transform[3], p.Transform[7], p.Transform[11]
	}
	x0, y0, z0 := tx(placed[0])
	if x0 != 0 || y0 != 0 || z0 != 0 {
		t.Errorf("occurrence[0] translation = (%.3f,%.3f,%.3f), want origin", x0, y0, z0)
	}
	x1, y1, z1 := tx(placed[1])
	if x1 != 13 || y1 != 17 || z1 != 19 {
		t.Errorf("occurrence[1] translation = (%.3f,%.3f,%.3f), want (13,17,19)", x1, y1, z1)
	}
	// rotation is identity for both
	for i, p := range placed {
		if p.Transform[0] != 1 || p.Transform[5] != 1 || p.Transform[10] != 1 || p.Transform[15] != 1 {
			t.Errorf("occurrence[%d] rotation not identity: %v", i, p.Transform)
		}
	}
}

// TestDecodeSubAssembly checks both levels of a nested assembly decode: the top places two
// "sub" occurrences at (0,0,0)/(0,6,0), and the sub-assembly places two "leaf" occurrences
// at (0,0,0)/(3,0,0). The component of a sub-assembly occurrence is another .iam.
func TestDecodeSubAssembly(t *testing.T) {
	check := func(file, component string, wantYs, wantXs [2]float64) {
		placed := DecodeAssembly(openDoc(t, file))
		if len(placed) != 2 {
			t.Fatalf("%s: %d occurrences, want 2", file, len(placed))
		}
		for _, p := range placed {
			if p.Component != component {
				t.Errorf("%s: component %q, want %q", file, p.Component, component)
			}
		}
	}
	check("top.iam", "sub", [2]float64{0, 6}, [2]float64{0, 0})
	check("sub.iam", "leaf", [2]float64{0, 0}, [2]float64{0, 3})
	// spot-check the top's Y placements and the sub's X placements
	top := DecodeAssembly(openDoc(t, "top.iam"))
	if !(top[0].Transform[7] == 0 && top[1].Transform[7] == 6) {
		t.Errorf("top Y placements = %g,%g, want 0,6", top[0].Transform[7], top[1].Transform[7])
	}
	sub := DecodeAssembly(openDoc(t, "sub.iam"))
	if !(sub[0].Transform[3] == 0 && sub[1].Transform[3] == 3) {
		t.Errorf("sub X placements = %g,%g, want 0,3", sub[0].Transform[3], sub[1].Transform[3])
	}
}

// TestDecodeConstraintAndFilteredOccurrences covers the constrained assembly: its mate
// constraint decodes as a mate, and filtering occurrences to the real component (asm_cube)
// drops the spurious "hash:N" name that the constraint's geometry selection emits — leaving
// two occurrences correctly matched to their solved transforms (the mate pulls occ2 to z=4).
func TestDecodeConstraintAndFilteredOccurrences(t *testing.T) {
	d := openDoc(t, "asm_mate.iam")

	kinds := DecodeConstraintKinds(d)
	if len(kinds) != 1 || kinds[0] != ConstraintMate {
		t.Fatalf("constraint kinds = %v, want [mate]", kinds)
	}

	// Unfiltered decode over-counts (the spurious constraint-selection name).
	if all := DecodeAssembly(d); len(all) < 3 {
		t.Errorf("unfiltered decode = %d occurrences, expected the spurious one included", len(all))
	}
	placed := d.PlacedOccurrences(func(c string) bool { return c == "asm_cube" })
	if len(placed) != 2 {
		t.Fatalf("filtered occurrences = %d, want 2: %+v", len(placed), placed)
	}
	var haveOrigin, haveZ4 bool
	for _, p := range placed {
		if p.Component != "asm_cube" {
			t.Errorf("unexpected component %q", p.Component)
		}
		switch {
		case p.Transform[3] == 0 && p.Transform[7] == 0 && p.Transform[11] == 0:
			haveOrigin = true
		case p.Transform[3] == 0 && p.Transform[7] == 0 && absf(p.Transform[11]-4) < 1e-9:
			haveZ4 = true
		}
	}
	if !haveOrigin || !haveZ4 {
		t.Errorf("placements origin=%v z=4=%v (want both) — mate-solved positions", haveOrigin, haveZ4)
	}
}

// TestDecodeRotatedPlacement locks down non-identity rotation decode: a bar placed rotated
// 90° CCW about +Z then translated (7,0,0) must decode to the exact row-major matrix (a
// transpose or wrong axis convention would flip the off-diagonal signs).
func TestDecodeRotatedPlacement(t *testing.T) {
	placed := DecodeAssembly(openDoc(t, "asm_rotated.iam"))
	if len(placed) != 2 {
		t.Fatalf("got %d placed occurrences, want 2", len(placed))
	}
	want := Matrix4{
		0, -1, 0, 7,
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	for j, v := range placed[1].Transform {
		if absf(v-want[j]) > 1e-9 {
			t.Errorf("occ2 transform cell %d = %.3f, want %.3f\n  got  %v\n  want %v", j, v, want[j], placed[1].Transform, want)
			break
		}
	}
}

// TestOccurrenceScanFields covers the backward identifier + forward integer scan on a
// synthesized AmDc-style buffer: a name embedded in a longer run still resolves, and a
// bare ":N" with no identifier is rejected.
func TestOccurrenceScanFields(t *testing.T) {
	mk := func(s string) []byte {
		b := make([]byte, 2*len(s))
		for i, c := range s {
			b[2*i] = byte(c)
		}
		return b
	}
	// "Bracket:1" then a break, "Bracket:2" glued to trailing "X", then a bare ":9".
	seg := append(mk("Bracket:1"), 0, 0)
	seg = append(seg, mk("Bracket:2X")...)
	seg = append(seg, 0, 0)
	seg = append(seg, mk(":9")...)
	occ := DecodeOccurrences(seg)
	if len(occ) != 2 {
		t.Fatalf("got %d occurrences, want 2: %+v", len(occ), occ)
	}
	for i, want := range []Occurrence{{"Bracket", 1}, {"Bracket", 2}} {
		if occ[i] != want {
			t.Errorf("occurrence[%d] = %+v, want %+v", i, occ[i], want)
		}
	}
}
