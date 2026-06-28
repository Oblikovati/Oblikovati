// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func approx(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// boxFaces returns the six outward-oriented quad surface bodies of a box of the
// given dimensions, anchored at the origin (reusing quadBody from stitch_test.go).
func boxFaces(sx, sy, sz float64) []*topo.Body {
	p := math.P3
	return []*topo.Body{
		quadBody("bottom", p(0, 0, 0), p(0, sy, 0), p(sx, sy, 0), p(sx, 0, 0)),
		quadBody("top", p(0, 0, sz), p(sx, 0, sz), p(sx, sy, sz), p(0, sy, sz)),
		quadBody("front", p(0, 0, 0), p(sx, 0, 0), p(sx, 0, sz), p(0, 0, sz)),
		quadBody("back", p(0, sy, 0), p(0, sy, sz), p(sx, sy, sz), p(sx, sy, 0)),
		quadBody("left", p(0, 0, 0), p(0, 0, sz), p(0, sy, sz), p(0, sy, 0)),
		quadBody("right", p(sx, 0, 0), p(sx, sy, 0), p(sx, sy, sz), p(sx, 0, sz)),
	}
}

func TestTrimByPlaneKeepsHalf(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	trimmed, err := TrimByPlane(patch, math.P3(2, 0, 0), math.V3(1, 0, 0), true, "trim")
	if err != nil {
		t.Fatalf("TrimByPlane: %v", err)
	}
	if trimmed.IsSolid() || len(trimmed.Faces()) != 1 {
		t.Errorf("trim result solid=%v faces=%d, want surface/1", trimmed.IsSolid(), len(trimmed.Faces()))
	}
	box := trimmed.RangeBox()
	if !approx(box.Min.X, 2) || !approx(box.Max.X, 4) {
		t.Errorf("trimmed x-span = [%v,%v], want [2,4]", box.Min.X, box.Max.X)
	}
}

// twoQuadSheet is a 2-face coplanar quilt (two unit-deep quads sharing the x=2 edge on z=0).
func twoQuadSheet(t *testing.T) *topo.Body {
	t.Helper()
	q1 := quadBody("q1", math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0))
	q2 := quadBody("q2", math.P3(2, 0, 0), math.P3(4, 0, 0), math.P3(4, 2, 0), math.P3(2, 2, 0))
	sheet, err := Stitch([]*topo.Body{q1, q2}, 0, true, "sheet")
	if err != nil {
		t.Fatalf("Stitch setup: %v", err)
	}
	if len(sheet.Faces()) != 2 {
		t.Fatalf("setup sheet has %d faces, want 2", len(sheet.Faces()))
	}
	return sheet
}

// K5: trimming a MULTI-FACE planar sheet clips each face and welds the kept ones back.
func TestTrimMultiFaceSheet(t *testing.T) {
	trimmed, err := TrimByPlane(twoQuadSheet(t), math.P3(1, 0, 0), math.V3(1, 0, 0), true, "trim")
	if err != nil {
		t.Fatalf("TrimByPlane multi-face: %v", err)
	}
	if len(trimmed.Faces()) != 2 {
		t.Errorf("trimmed sheet has %d faces, want 2 (clipped quad + whole quad)", len(trimmed.Faces()))
	}
	box := trimmed.RangeBox()
	if !approx(box.Min.X, 1) || !approx(box.Max.X, 4) {
		t.Errorf("trimmed x-span = [%v,%v], want [1,4]", box.Min.X, box.Max.X)
	}
}

// TestSheetEdgesAreProvenanceNamed is ADR-0043: a sheet built by buildSheet (here via OffsetSurface)
// used to mint its welded edges with a build-order weld counter (feat:edge#N) that renumbers under an
// upstream edit. They must now be named by their patch provenance — a shared seam by its two patches,
// a boundary edge by its one patch — so a reference to a sheet edge survives a re-weld.
func TestSheetEdgesAreProvenanceNamed(t *testing.T) {
	off, err := OffsetSurface(twoQuadSheet(t), 0.5, "off")
	if err != nil {
		t.Fatalf("OffsetSurface: %v", err)
	}
	seams := 0
	for _, e := range off.Edges() {
		k := string(e.ReferenceKey())
		if strings.Contains(k, "off:edge#") {
			t.Errorf("sheet edge kept a weld-counter ordinal: %q (provenance naming missing)", k)
		}
		// The one shared seam borders both patches, so it carries the separator between them.
		if strings.Contains(k, "off:patch#0/off:x#0/off:patch#1") {
			seams++
		}
	}
	if seams != 1 {
		t.Errorf("shared seam named by its two patches: got %d such edges, want 1", seams)
	}
}

// K5: offsetting a MULTI-FACE coplanar quilt translates every face along the shared normal.
func TestOffsetMultiFaceCoplanar(t *testing.T) {
	off, err := OffsetSurface(twoQuadSheet(t), 0.5, "off")
	if err != nil {
		t.Fatalf("OffsetSurface multi-face: %v", err)
	}
	if len(off.Faces()) != 2 {
		t.Errorf("offset quilt has %d faces, want 2", len(off.Faces()))
	}
	box := off.RangeBox()
	if !approx(box.Min.Z, 0.5) || !approx(box.Max.Z, 0.5) {
		t.Errorf("offset z = [%v,%v], want flat at 0.5", box.Min.Z, box.Max.Z)
	}
}

// K5: extending a planar surface's boundary edge grows the face outward by the distance.
func TestExtendByEdgeGrowsFace(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	var key []byte // the bottom edge (both endpoints at y=0)
	for _, e := range patch.Edges() {
		if approx(float64(e.StartVertex().Point().Y), 0) && approx(float64(e.EndVertex().Point().Y), 0) {
			key = e.ReferenceKey()
		}
	}
	if key == nil {
		t.Fatal("no bottom edge found")
	}
	ext, err := ExtendByEdge(patch, key, 2, "ext")
	if err != nil {
		t.Fatalf("ExtendByEdge: %v", err)
	}
	box := ext.RangeBox()
	if !approx(box.Min.Y, -2) || !approx(box.Max.Y, 4) {
		t.Errorf("extended y-span = [%v,%v], want [-2,4]", box.Min.Y, box.Max.Y)
	}
	if !approx(box.Min.X, 0) || !approx(box.Max.X, 4) {
		t.Errorf("extended x-span = [%v,%v], want [0,4] (unchanged)", box.Min.X, box.Max.X)
	}
}

func TestExtendByEdgeLostEdgeErrors(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	if _, err := ExtendByEdge(patch, []byte("ghost"), 2, "ext"); err == nil {
		t.Error("extending a lost edge should error")
	}
}

func TestTrimByPlaneEmptyErrors(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	// Keep the x ≥ 10 side — nothing remains.
	if _, err := TrimByPlane(patch, math.P3(10, 0, 0), math.V3(1, 0, 0), true, "trim"); err == nil {
		t.Error("trimming away the whole patch should error")
	}
}

func TestOffsetSurfaceMovesAlongNormal(t *testing.T) {
	patch := quadBody("p", math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0))
	offset, err := OffsetSurface(patch, 3, "offset")
	if err != nil {
		t.Fatalf("OffsetSurface: %v", err)
	}
	box := offset.RangeBox()
	if !approx(box.Min.Z, 3) || !approx(box.Max.Z, 3) {
		t.Errorf("offset z = [%v,%v], want flat at 3", box.Min.Z, box.Max.Z)
	}
}

func TestMidSurfacesThinBox(t *testing.T) {
	// A 4×4×1 thin plate: only the top/bottom faces (separation 1) are a thin pair.
	solid, _ := Stitch(boxFaces(4, 4, 1), 0, false, "box")
	patches, err := MidSurfaces(solid, 2, "mid")
	if err != nil {
		t.Fatalf("MidSurfaces: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("got %d mid-surfaces, want 1 (the thin pair)", len(patches))
	}
	if !approx(patches[0].Thickness, 1) {
		t.Errorf("recorded thickness = %v, want 1", patches[0].Thickness)
	}
	// The mid patch sits on z = 0.5, midway between the caps.
	box := patches[0].Body.RangeBox()
	if !approx(box.Min.Z, 0.5) || !approx(box.Max.Z, 0.5) {
		t.Errorf("mid patch z = [%v,%v], want flat at 0.5", box.Min.Z, box.Max.Z)
	}
}

func TestMidSurfacesNoThinPairErrors(t *testing.T) {
	solid, _ := Stitch(boxFaces(1, 1, 1), 0, false, "box")
	if _, err := MidSurfaces(solid, 0.5, "mid"); err == nil {
		t.Error("a cube with all separations 1 should have no pair within 0.5")
	}
}
