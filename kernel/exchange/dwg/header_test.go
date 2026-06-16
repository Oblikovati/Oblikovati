// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"encoding/binary"
	"testing"
)

// buildR2000Header synthesises a minimal but valid R2000 file header with the
// given locator records, so the parser is unit-tested without the multi-MB corpus.
func buildR2000Header(t *testing.T, recs []SectionLocator) []byte {
	t.Helper()
	const recStart = 0x19
	buf := make([]byte, recStart)
	copy(buf, "AC1015")
	buf[0x13] = 30 // codepage ANSI_1252, low byte
	binary.LittleEndian.PutUint32(buf[0x15:], uint32(len(recs)))
	for _, r := range recs {
		rec := make([]byte, 9)
		rec[0] = r.ID
		binary.LittleEndian.PutUint32(rec[1:], uint32(r.Address))
		binary.LittleEndian.PutUint32(rec[5:], uint32(r.Size))
		buf = append(buf, rec...)
	}
	crc := crc16(0xC0C1, buf) // CRC runs from byte 0 over the whole header
	buf = binary.LittleEndian.AppendUint16(buf, crc)
	buf = append(buf, sentinelHeaderEndR2000[:]...)
	return buf
}

func TestParseR2000HeaderSynthetic(t *testing.T) {
	recs := []SectionLocator{
		{ID: secHeaderVars, Address: 0x5397, Size: 585},
		{ID: secClasses, Address: 0x55e0, Size: 44375},
		{ID: secObjectMap, Address: 0x174b0d6, Size: 0xCC876},
	}
	h, err := ParseFileHeader(buildR2000Header(t, recs))
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	if h.Version != R2000 {
		t.Errorf("version = %v, want R2000", h.Version)
	}
	if h.Codepage != 30 {
		t.Errorf("codepage = %d, want 30", h.Codepage)
	}
	if len(h.Sections) != 3 {
		t.Fatalf("sections = %d, want 3", len(h.Sections))
	}
	obj, ok := h.Locator(secObjectMap)
	if !ok || obj.Address != 0x174b0d6 || obj.Size != 0xCC876 {
		t.Errorf("object-map locator = %+v (ok=%v), want addr 0x174b0d6 size 0xCC876", obj, ok)
	}
}

func TestParseR2000HeaderRejectsBadCRC(t *testing.T) {
	buf := buildR2000Header(t, []SectionLocator{{ID: secHeaderVars, Address: 1, Size: 2}})
	buf[0x10] ^= 0xFF // corrupt a header byte the CRC covers
	if _, err := ParseFileHeader(buf); err == nil {
		t.Fatal("expected CRC mismatch error")
	}
}

// TestParseR2000HeaderRealFile pins the parser against the known layout of the
// AC1015 corpus file, hand-decoded from its header bytes.
func TestParseR2000HeaderRealFile(t *testing.T) {
	data := loadTestFile(t, "testfile-2.dwg")
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	if h.Version != R2000 {
		t.Fatalf("version = %v, want R2000", h.Version)
	}
	if len(h.Sections) != 6 {
		t.Fatalf("sections = %d, want 6", len(h.Sections))
	}
	want := map[byte]SectionLocator{
		secHeaderVars: {ID: secHeaderVars, Address: 0x5397, Size: 585},
		secClasses:    {ID: secClasses, Address: 0x55e0, Size: 44375},
		secObjectMap:  {ID: secObjectMap, Address: 0x174b0d6, Size: 0xCC876},
	}
	for id, exp := range want {
		got, ok := h.Locator(id)
		if !ok || got != exp {
			t.Errorf("locator %d = %+v (ok=%v), want %+v", id, got, ok, exp)
		}
	}
	// The header-vars section must be addressable and CRC-bounded within the file.
	if _, err := h.SectionBytes(data, secObjectMap); err != nil {
		t.Errorf("object-map section bytes: %v", err)
	}
}

func TestParseR2004MalformedFailsLoudly(t *testing.T) {
	// A truncated/garbage AC1032 file must fail at header decryption (no marker),
	// not parse into nonsense.
	hdr := append([]byte("AC1032"), make([]byte, 200)...)
	if _, err := ParseFileHeader(hdr); err == nil {
		t.Fatal("expected decryption-marker error for malformed paged container")
	}
}
