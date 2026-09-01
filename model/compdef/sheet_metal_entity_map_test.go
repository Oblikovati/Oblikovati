// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"bytes"
	"testing"

	gmath "oblikovati.org/math"
)

// TestEntityMapRoundTrip a folded top face maps to a real flat face, and mapping that flat
// face back returns the same folded top face — the correspondence a flat drawing dimension
// relies on across recompute.
func TestEntityMapRoundTrip(t *testing.T) {
	t.Parallel()
	d, _ := sheetWithFlange(t)
	folded := d.Features().Result()[0]
	top := dominantFace(folded, gmath.V3(0, 0, 1)) // the base top face (+Z)
	if top == nil {
		t.Fatal("no top face on the folded body")
	}
	foldedKey := top.ReferenceKey()

	flatKey, found, err := d.MapFoldedToFlat(foldedKey)
	if err != nil || !found {
		t.Fatalf("MapFoldedToFlat: found=%v err=%v", found, err)
	}
	// The mapped key resolves to a real flat face.
	fp, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	if _, ok := fp.Body.FindFaceByKey(flatKey); !ok {
		t.Error("mapped flat key does not resolve to a flat face")
	}

	backKey, found, err := d.MapFlatToFolded(flatKey)
	if err != nil || !found {
		t.Fatalf("MapFlatToFolded: found=%v err=%v", found, err)
	}
	if !bytes.Equal(backKey, foldedKey) {
		t.Errorf("round-trip key = %x, want the original folded top face %x", backKey, foldedKey)
	}
}

// TestEntityMapFrontBackDistinct the top and bottom folded faces map to distinct flat faces.
func TestEntityMapFrontBackDistinct(t *testing.T) {
	t.Parallel()
	d, _ := sheetWithFlange(t)
	folded := d.Features().Result()[0]
	topKey := dominantFace(folded, gmath.V3(0, 0, 1)).ReferenceKey()
	bottomKey := dominantFace(folded, gmath.V3(0, 0, -1)).ReferenceKey()

	front, _, _ := d.MapFoldedToFlat(topKey)
	back, _, _ := d.MapFoldedToFlat(bottomKey)
	if bytes.Equal(front, back) {
		t.Error("top and bottom folded faces mapped to the same flat face")
	}
}

// TestEntityMapUnknownKey an unresolvable key reports not-found rather than erroring.
func TestEntityMapUnknownKey(t *testing.T) {
	t.Parallel()
	d, _ := sheetWithFlange(t)
	if _, found, err := d.MapFoldedToFlat([]byte("not-a-real-key")); err != nil || found {
		t.Errorf("unknown key: found=%v err=%v, want found=false err=nil", found, err)
	}
}

// TestEntityMapNoFlat mapping on a sheet-metal part with no developable flat errors (there is
// no base Face to develop).
func TestEntityMapNoFlat(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if _, _, err := d.MapFoldedToFlat([]byte("k")); err == nil {
		t.Error("MapFoldedToFlat with no flat must error")
	}
	if _, _, err := d.MapFlatToFolded([]byte("k")); err == nil {
		t.Error("MapFlatToFolded with no flat must error")
	}
}
