// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"hash/crc32"
	"testing"
)

// naiveCRC16ARC is an independent bit-by-bit CRC-16/ARC reference. It exists only
// to cross-check the table-driven crc16 so a table-generation bug cannot pass
// unnoticed (the table-driven and bitwise paths must agree on arbitrary data).
func naiveCRC16ARC(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func TestCRC16CheckValue(t *testing.T) {
	// CRC-16/ARC standard check value for "123456789" is 0xBB3D.
	if got := crc16(0, []byte("123456789")); got != 0xBB3D {
		t.Fatalf("crc16 check = %#04x, want 0xBB3D", got)
	}
}

func TestCRC16MatchesNaive(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0x00},
		{0xFF, 0x00, 0xAA, 0x55},
		[]byte("the quick brown fox"),
	}
	for _, in := range inputs {
		if got, want := crc16(0, in), naiveCRC16ARC(in); got != want {
			t.Errorf("crc16(%q) = %#04x, naive = %#04x", in, got, want)
		}
	}
}

func TestCRC16Seeded(t *testing.T) {
	// Seeding must equal restarting the running checksum, so a section's CRC can be
	// continued across a prefix.
	data := []byte("abcdefgh")
	full := crc16(0, data)
	part := crc16(0, data[:3])
	if got := crc16(part, data[3:]); got != full {
		t.Fatalf("seeded continuation = %#04x, want %#04x", got, full)
	}
}

func TestCRC32MatchesStdlib(t *testing.T) {
	data := []byte("the quick brown fox jumps over the lazy dog")
	if got, want := crc32sum(0, data), crc32.ChecksumIEEE(data); got != want {
		t.Fatalf("crc32sum = %#08x, stdlib = %#08x", got, want)
	}
	// Standard CRC-32 check value for "123456789".
	if got := crc32sum(0, []byte("123456789")); got != 0xCBF43926 {
		t.Fatalf("crc32 check = %#08x, want 0xCBF43926", got)
	}
}

func TestReedSolomonDeinterleave(t *testing.T) {
	const blocks, dataSize = 2, 3
	src := make([]byte, blocks*rsBlockSize)
	copy(src[0*rsBlockSize:], []byte{10, 11, 12}) // block 0 data
	copy(src[1*rsBlockSize:], []byte{20, 21, 22}) // block 1 data
	got, err := reedSolomonDeinterleave(src, blocks, dataSize)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []byte{10, 20, 11, 21, 12, 22}
	if !bytesEqual(got, want) {
		t.Fatalf("deinterleave = %v, want %v", got, want)
	}
}

func TestReedSolomonRejectsBadShape(t *testing.T) {
	if _, err := reedSolomonDeinterleave(make([]byte, 255), 0, 10); err == nil {
		t.Error("expected error for zero blockCount")
	}
	if _, err := reedSolomonDeinterleave(make([]byte, 10), 1, 239); err == nil {
		t.Error("expected error for short input")
	}
}
