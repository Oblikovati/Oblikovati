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

// TestDecodeEntityCountMatchesOracle pins the public Decode count against the sum
// of the oracle's per-type tallies for testfile-1.
func TestDecodeEntityCountMatchesOracle(t *testing.T) {
	data := loadTestFile(t, "testfile-1.dwg")
	dr, _, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	// LINE+ARC+CIRCLE+POINT+ELLIPSE+LWPOLYLINE+SPLINE = the curve entities Decode keeps.
	const want = 58062 + 1670 + 959 + 739 + 1271 + 15525 + 2898
	if len(dr.Entities) != want {
		t.Errorf("Decode kept %d entities, want %d (oracle curve total)", len(dr.Entities), want)
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
