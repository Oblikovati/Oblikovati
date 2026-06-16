// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// TestDecodeCorpus exercises the public entry point on the whole corpus: every
// file decodes without a fatal error, yields a populated entity set, and produces
// no per-entity warnings for the curve types that are implemented.
func TestDecodeCorpus(t *testing.T) {
	files := append([]string{"testfile-2.dwg"}, r2018Corpus...)
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			data := loadTestFile(t, name)
			dr, warns, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if len(dr.Entities) < 100 {
				t.Fatalf("decoded only %d entities", len(dr.Entities))
			}
			if len(warns) != 0 {
				t.Errorf("unexpected %d decode warnings (first: %s)", len(warns), warns[0])
			}
		})
	}
}

// TestDecodeClassifiesEverySpace checks that space classification accounts for every
// curve entity: the model-space, block-definition, and paper-space buckets sum to the
// oracle's total curve count for testfile-1. This is the invariant behind the
// model-space import — no curve is lost or double-counted when it is sorted by entmode.
func TestDecodeClassifiesEverySpace(t *testing.T) {
	data := loadTestFile(t, "testfile-1.dwg")
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatal(err)
	}
	od, _ := h.ObjectData(data)
	omb, _ := h.ObjectMapBytes(data)
	refs, _ := parseObjectMap(omb)
	c := &collector{data: od, version: h.Version, blockEntities: map[uint64][]Entity{}, blockInserts: map[uint64][]*Insert{}}
	c.collect(refs)

	blockCurves := 0
	for _, es := range c.blockEntities {
		blockCurves += len(es)
	}
	got := len(c.modelEntities) + blockCurves + c.paperCurves
	// LINE+ARC+CIRCLE+POINT+ELLIPSE+LWPOLYLINE+SPLINE oracle tallies for testfile-1.
	const wantTotal = 58062 + 1670 + 959 + 739 + 1271 + 15525 + 2898
	if got != wantTotal {
		t.Errorf("classified %d curves (model %d + block %d + paper %d), want oracle total %d",
			got, len(c.modelEntities), blockCurves, c.paperCurves, wantTotal)
	}
}

// TestExpandInsertsNested checks block-insert expansion: a model-space insert places its
// block's geometry transformed, and a nested insert (a block that inserts another block)
// composes transforms — the property that reconstructs a real drawing from its blocks.
func TestExpandInsertsNested(t *testing.T) {
	const inner, outer = uint64(1), uint64(2)
	c := &collector{
		blockEntities: map[uint64][]Entity{
			inner: {&Point{Position: [3]float64{1, 0, 0}}}, // a point at (1,0) in the inner block
		},
		blockInserts: map[uint64][]*Insert{
			// the outer block places the inner block, offset by (10,0)
			outer: {{BlockHeader: inner, Insertion: [3]float64{10, 0, 0}, Scale: [3]float64{1, 1, 1}}},
		},
		// model space places the outer block, offset by (0, 100)
		modelInserts: []*Insert{{BlockHeader: outer, Insertion: [3]float64{0, 100, 0}, Scale: [3]float64{1, 1, 1}}},
	}
	out := c.resolve()
	if len(out) != 1 {
		t.Fatalf("expanded to %d entities, want 1", len(out))
	}
	// (1,0) + inner-offset (10,0) + outer-offset (0,100) = (11,100).
	if p := out[0].(*Point); p.Position != [3]float64{11, 100, 0} {
		t.Errorf("nested insert placed point at %v, want (11,100,0)", p.Position)
	}
}

// TestExpandInsertsCycleSafe checks a block that (transitively) inserts itself does not
// loop forever — the cycle guard stops the recursion.
func TestExpandInsertsCycleSafe(t *testing.T) {
	const a = uint64(1)
	c := &collector{
		blockEntities: map[uint64][]Entity{a: {&Point{}}},
		blockInserts:  map[uint64][]*Insert{a: {{BlockHeader: a, Scale: [3]float64{1, 1, 1}}}}, // self-reference
		modelInserts:  []*Insert{{BlockHeader: a, Scale: [3]float64{1, 1, 1}}},
	}
	out := c.resolve() // must terminate
	if len(out) == 0 {
		t.Error("self-referential block produced no geometry")
	}
}

// TestDrawingPlanar checks the 2D/3D routing helper on synthetic drawings.
func TestDrawingPlanar(t *testing.T) {
	flat := &Drawing{Entities: []Entity{
		&Line{Start: [3]float64{0, 0, 5}, End: [3]float64{1, 1, 5}},
		&Circle{Center: [3]float64{2, 2, 5}},
	}}
	if z, ok := flat.Planar(1e-9); !ok || z != 5 {
		t.Errorf("flat drawing Planar = (%v,%v), want (5,true)", z, ok)
	}
	bumpy := &Drawing{Entities: []Entity{
		&Line{Start: [3]float64{0, 0, 0}, End: [3]float64{1, 1, 9}},
	}}
	if _, ok := bumpy.Planar(1e-9); ok {
		t.Error("3D drawing wrongly reported planar")
	}
}
