// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"testing"
)

// TestPadContentHeaderShiftsFields checks that inserting the missing pre-2023 gap moves a node's
// entity fields to the 2027 offsets: a coordinate a pre-2023 SketchPoint holds at 38 must read back
// at 42 after normalisation, and the bytes before the gap must be preserved.
func TestPadContentHeaderShiftsFields(t *testing.T) {
	gap := contentHeaderGapFor(point2DNodeType) // a sketch entity: gap at 22
	pay := make([]byte, 60)
	for i := 0; i < gap; i++ {
		pay[i] = byte(i + 1) // distinctive header bytes to prove they survive
	}
	binary.LittleEndian.PutUint64(pay[38:], math.Float64bits(-13.75)) // x at the pre-2023 offset
	binary.LittleEndian.PutUint64(pay[46:], math.Float64bits(29.75))  // y

	out := padContentHeader(pay, point2DNodeType)
	if len(out) != len(pay)+4 {
		t.Fatalf("padded length = %d, want %d", len(out), len(pay)+4)
	}
	for i := 0; i < gap; i++ {
		if out[i] != byte(i+1) {
			t.Fatalf("header byte %d = %d, want %d (pre-gap bytes must survive)", i, out[i], i+1)
		}
	}
	if x := math.Float64frombits(binary.LittleEndian.Uint64(out[42:])); x != -13.75 {
		t.Errorf("x reads %.4f at offset 42 after padding, want -13.75", x)
	}
	if y := math.Float64frombits(binary.LittleEndian.Uint64(out[50:])); y != 29.75 {
		t.Errorf("y reads %.4f at offset 50 after padding, want 29.75", y)
	}
}

// TestOlderContentHeaderDetectsLayout checks the pre-2023 detector: a feature node whose property
// List2 marker sits at offset 30 is older; at 34 is 2027; an unclassifiable set defaults to 2027.
func TestOlderContentHeaderDetectsLayout(t *testing.T) {
	featureWithMarkerAt := func(off int) dcNode {
		pay := make([]byte, 64)
		binary.LittleEndian.PutUint16(pay[off:], list2Marker)
		binary.LittleEndian.PutUint16(pay[off+2:], list2Owner)
		return dcNode{typ: featureNodeType, payload: pay}
	}
	older := []dcNode{featureWithMarkerAt(30), featureWithMarkerAt(30), featureWithMarkerAt(30)}
	if !olderContentHeader(older) {
		t.Error("markers at offset 30 not detected as pre-2023")
	}
	newer := []dcNode{featureWithMarkerAt(34), featureWithMarkerAt(34)}
	if olderContentHeader(newer) {
		t.Error("markers at offset 34 wrongly detected as pre-2023")
	}
	if olderContentHeader(nil) {
		t.Error("empty segment must default to 2027 (no normalisation)")
	}
}

// TestContentHeaderGapPerType pins the node-type-dependent gap positions: a Loop's shorter header
// carries the gap at 14 (its operation@10 stays put, its edge list realigns 14→18), a
// SketchEntityRef at 18 (associative id 18→22), and every other type at 22. A single uniform offset
// leaves the low-offset region nodes misaligned, so no pre-2023 part's extrudes resolve a region.
func TestContentHeaderGapPerType(t *testing.T) {
	cases := []struct {
		typ  uint32
		want int
	}{
		{loopNodeType, 14}, {loop3DNodeType, 14}, {entityRefNodeType, 18},
		{point2DNodeType, 22}, {featureNodeType, 22}, {faceBoundNodeType, 22},
	}
	for _, c := range cases {
		if got := contentHeaderGapFor(c.typ); got != c.want {
			t.Errorf("contentHeaderGapFor(0x%08X) = %d, want %d", c.typ, got, c.want)
		}
	}
	// A Loop's edge-list marker at the pre-2023 offset 14 must land at 18 after padding.
	loop := make([]byte, 40)
	binary.LittleEndian.PutUint16(loop[14:], list2Marker)
	binary.LittleEndian.PutUint16(loop[16:], list2Owner)
	out := padContentHeader(loop, loopNodeType)
	if !hasList2Marker(out, loopEdgesOffset) {
		t.Errorf("Loop edge marker not at %d after padding", loopEdgesOffset)
	}
}
