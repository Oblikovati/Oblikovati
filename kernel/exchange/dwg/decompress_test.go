// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"encoding/binary"
	"testing"
)

// TestDecompressR2004SynthRunLength exercises the length>offset run-length case
// (a back-reference that reads bytes it is producing) with a hand-built stream.
//
// Leading literal: 0x00 0x01 -> 0x0F+1+3 = 0x13 = 19? Too long; instead use a
// short low-nibble literal length. 0x01 -> 1+3 = 4 literals "ABCD". Then opcode
// 0x21 (compressedBytes = 0x21-0x1E = 3) with a two-byte offset of (0,litCount0):
// firstByte 0x00 -> offset 0, litCount 0 -> read next literal length; make it the
// terminator path by ending with opcode 0x11.
func TestDecompressR2004SynthRunLength(t *testing.T) {
	// literals "ABCD"; copy 3 bytes from offset 0 (== last byte 'D') -> "DDD";
	// then a 0-litCount that reads a literal length whose high nibble is set, which
	// doubles as the terminator opcode 0x11.
	src := []byte{
		0x01,               // leading literal length -> 4
		'A', 'B', 'C', 'D', // 4 literals
		0x21,       // opcode: compressedBytes = 3
		0x00, 0x00, // two-byte offset: offset 0, litCount 0
		0x11, // literal length byte: high nibble set -> 0 lits, next opcode
		// 0x11 is now the next opcode -> terminator
	}
	out, err := decompressR2004(src, 7)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if string(out) != "ABCDDDD" {
		t.Fatalf("decompressed = %q, want %q", out, "ABCDDDD")
	}
}

// TestDecompressR2004PageMap validates the decompressor against the real AC1032
// corpus: the section-page-map page decompresses to its declared size and begins
// with the known ascending (pageNumber, size) entries.
func TestDecompressR2004PageMap(t *testing.T) {
	data := loadTestFile(t, "testfile-1.dwg")
	const pageHdr = 0x18c8320 // page-map address (0x18c8220) + 0x100
	if binary.LittleEndian.Uint32(data[pageHdr:]) != 0x41630e3b {
		t.Fatalf("page-map magic = %#x, want 0x41630e3b", binary.LittleEndian.Uint32(data[pageHdr:]))
	}
	decompSize := int(binary.LittleEndian.Uint32(data[pageHdr+4:]))
	compSize := int(binary.LittleEndian.Uint32(data[pageHdr+8:]))
	body := data[pageHdr+20 : pageHdr+20+compSize]

	out, err := decompressR2004(body, decompSize)
	if err != nil {
		t.Fatalf("decompress page map: %v", err)
	}
	if len(out) != decompSize {
		t.Fatalf("decompressed %d bytes, want %d", len(out), decompSize)
	}
	// First three page entries are number/size pairs 1/0xA0, 2/0x820, 3/0x1A0.
	want := []struct{ num, size uint32 }{{1, 0xA0}, {2, 0x820}, {3, 0x1A0}}
	for i, w := range want {
		num := binary.LittleEndian.Uint32(out[i*8:])
		size := binary.LittleEndian.Uint32(out[i*8+4:])
		if num != w.num || size != w.size {
			t.Errorf("entry %d = (num %d, size %#x), want (%d, %#x)", i, num, size, w.num, w.size)
		}
	}
}
