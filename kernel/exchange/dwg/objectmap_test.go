// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// TestParseObjectMapSynthetic builds one delta-encoded section (plus the size-2
// terminator) and checks handle accumulation and signed offset deltas.
func TestParseObjectMapSynthetic(t *testing.T) {
	// pairs: (h+1, loc+10), (h+1, loc-4) -> (1,10) then (2,6)
	pairs := []byte{0x01, 0x0A, 0x01, 0x44} // 0x44 = MC -4 (0x40 sign | 4)
	size := 2 + len(pairs)                  // size field counts itself
	data := []byte{byte(size >> 8), byte(size)}
	data = append(data, pairs...)
	data = append(data, 0x00, 0x00) // CRC (not verified by the parser)
	data = append(data, 0x00, 0x02) // terminator section (size == 2)

	refs, err := parseObjectMap(data)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	want := []ObjectRef{{Handle: 1, Offset: 10}, {Handle: 2, Offset: 6}}
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d", len(refs), len(want))
	}
	for i, w := range want {
		if refs[i] != w {
			t.Errorf("ref %d = %+v, want %+v", i, refs[i], w)
		}
	}
}

// TestParseObjectMapResetsPerSection guards the per-section accumulator reset: two
// sections each starting their own delta chain, so the second section's first
// offset is absolute (not continued from the first section's running total).
func TestParseObjectMapResetsPerSection(t *testing.T) {
	sec := func(pairs []byte) []byte {
		size := 2 + len(pairs)
		out := []byte{byte(size >> 8), byte(size)}
		out = append(out, pairs...)
		return append(out, 0x00, 0x00) // CRC placeholder
	}
	data := sec([]byte{0x01, 0x05})                 // section 0: (1, 5)
	data = append(data, sec([]byte{0x02, 0x09})...) // section 1: (2, 9) — reset, not (3,14)
	data = append(data, 0x00, 0x02)                 // terminator

	refs, err := parseObjectMap(data)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	want := []ObjectRef{{Handle: 1, Offset: 5}, {Handle: 2, Offset: 9}}
	if len(refs) != 2 || refs[0] != want[0] || refs[1] != want[1] {
		t.Fatalf("refs = %+v, want %+v (accumulators must reset per section)", refs, want)
	}
}

// TestObjectMapCorpusInRange decodes the real object maps and requires every
// recorded offset to land inside the object-data stream — the whole-map integrity
// check. R2004 offsets index AcDb:AcDbObjects; R2000 offsets index the file.
func TestObjectMapCorpusInRange(t *testing.T) {
	t.Run("R2004", func(t *testing.T) {
		data := loadTestFile(t, "testfile-1.dwg")
		h, err := ParseFileHeader(data)
		if err != nil {
			t.Fatal(err)
		}
		handles, err := h.LogicalSection(data, "AcDb:Handles")
		if err != nil {
			t.Fatal(err)
		}
		objs, err := h.LogicalSection(data, "AcDb:AcDbObjects")
		if err != nil {
			t.Fatal(err)
		}
		assertRefsInRange(t, mustParseObjectMap(t, handles), int64(len(objs)))
	})
	t.Run("R2000", func(t *testing.T) {
		data := loadTestFile(t, "testfile-2.dwg")
		h, err := ParseFileHeader(data)
		if err != nil {
			t.Fatal(err)
		}
		sec, err := h.SectionBytes(data, secObjectMap)
		if err != nil {
			t.Fatal(err)
		}
		assertRefsInRange(t, mustParseObjectMap(t, sec), int64(len(data)))
	})
}

func mustParseObjectMap(t *testing.T, data []byte) []ObjectRef {
	t.Helper()
	refs, err := parseObjectMap(data)
	if err != nil {
		t.Fatalf("parseObjectMap: %v", err)
	}
	if len(refs) < 100 {
		t.Fatalf("only %d objects decoded; expected a populated drawing", len(refs))
	}
	return refs
}

func assertRefsInRange(t *testing.T, refs []ObjectRef, limit int64) {
	t.Helper()
	for _, r := range refs {
		if r.Offset < 0 || r.Offset >= limit {
			t.Fatalf("object handle %d offset %d out of [0,%d)", r.Handle, r.Offset, limit)
		}
	}
}
