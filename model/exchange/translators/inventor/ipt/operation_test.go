// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"testing"
)

// TestEnumNodeRealMarker locks the operation-encoding generalization for real parts: their
// enum nodes carry a schema marker (+12) that is NOT the 0x0C20 of corpus parts (0x0A96,
// 0x0B5C, …). Keying on 0x0C20 found zero operation nodes on real parts, so no extrude/revolve
// could bind its boolean op. This builds two enum nodes (a kind-5 operation and a kind-3 hole
// type) with the real 0x0A96 marker and asserts they decode by structure, not marker.
func TestEnumNodeRealMarker(t *testing.T) {
	const realMarker = 0x00000A96
	enumNode := func(id uint32, kind, value uint16) []byte {
		b := make([]byte, 24)
		binary.LittleEndian.PutUint32(b[0:], dcNodeTag)
		binary.LittleEndian.PutUint32(b[4:], id)
		binary.LittleEndian.PutUint32(b[8:], nullRef)
		binary.LittleEndian.PutUint32(b[12:], realMarker)
		binary.LittleEndian.PutUint16(b[16:], kind)
		binary.LittleEndian.PutUint16(b[18:], value)
		binary.LittleEndian.PutUint32(b[20:], enumTrailer)
		return b
	}
	var seg []byte
	seg = append(seg, enumNode(1, 5, uint16(OpCut))...)     // operation = Cut
	seg = append(seg, enumNode(2, 3, 2)...)                 // hole type = 2
	seg = append(seg, enumNode(3, 5, uint16(OpNewBody))...) // operation = NewBody

	if ops := DecodeOperations(seg); len(ops) != 2 || ops[0] != OpCut || ops[1] != OpNewBody {
		t.Errorf("DecodeOperations = %v, want [%d %d]", ops, OpCut, OpNewBody)
	}
	if hk := enumNodeValues(seg, 3); len(hk) != 1 || hk[0] != 2 {
		t.Errorf("enumNodeValues(kind=3) = %v, want [2]", hk)
	}
}

func TestDecodeOperations(t *testing.T) {
	cases := []struct {
		file string
		want []int
	}{
		{"10_box.ipt", []int{OpNewBody}},
		{"17_box_cut.ipt", []int{OpNewBody, OpCut}},
		{"14_box_two.ipt", []int{OpNewBody, OpJoin}},
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, _ := d.Segment("PmDCSegment")
		got := DecodeOperations(seg)
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.file, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: op[%d]=%d, want %d", tc.file, i, got[i], tc.want[i])
			}
		}
	}
}
