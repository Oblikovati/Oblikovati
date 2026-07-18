// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func openDoc(t *testing.T, name string) *Document {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	d, err := Open(data)
	if err != nil {
		t.Fatalf("Open %s: %v", name, err)
	}
	return d
}

func TestSegmentsDecodeWithExpectedSizes(t *testing.T) {
	cases := []struct {
		file, segment string
		wantLen       int
	}{
		{"blank_a.ipt", "PmDCSegment", 3073},
		{"10_box.ipt", "PmBRepSegment", 12065},
		{"15_cylinder.ipt", "PmBRepSegment", 4261},
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, ok := d.Segment(tc.segment)
		if !ok {
			t.Errorf("%s: missing segment %s (have %v)", tc.file, tc.segment, d.SegmentNames())
			continue
		}
		if len(seg) != tc.wantLen {
			t.Errorf("%s %s: decompressed len = %d, want %d", tc.file, tc.segment, len(seg), tc.wantLen)
		}
	}
}

// TestDecompressHandlesZlibAndZstd guards the real-world case: Inventor releases before 2027
// zlib-compress their segment payloads (0x78 header), not Zstd. Every .ipt in a typical
// design (e.g. the ReelToReel deck) is zlib — before this path they all failed with
// "no readable segments". decompress must inflate both, and reject anything else.
func TestDecompressHandlesZlibAndZstd(t *testing.T) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()
	want := []byte("PmDCSegment payload \x00\x01\x02 abc")
	hdr := make([]byte, bStreamStart) // 18-byte B-stream header, contents irrelevant to decompress

	var zb bytes.Buffer
	zw := zlib.NewWriter(&zb)
	zw.Write(want)
	zw.Close()
	if got, err := decompress(dec, append(hdr, zb.Bytes()...)); err != nil || !bytes.Equal(got, want) {
		t.Errorf("zlib: got %q, err %v; want %q", got, err, want)
	}

	zst, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	packed := zst.EncodeAll(want, nil)
	if got, err := decompress(dec, append(hdr, packed...)); err != nil || !bytes.Equal(got, want) {
		t.Errorf("zstd: got %q, err %v; want %q", got, err, want)
	}

	garbage := append(hdr, []byte{0x00, 0x11, 0x22, 0x33, 0x44}...)
	if _, err := decompress(dec, garbage); err == nil {
		t.Error("unknown compression magic should error, got nil")
	}
}

func TestExpectedSegmentSetPresent(t *testing.T) {
	d := openDoc(t, "10_box.ipt")
	for _, name := range []string{"PmDCSegment", "PmBRepSegment", "PmGraphicsSegment", "PmBrowserSegment"} {
		if _, ok := d.Segment(name); !ok {
			t.Errorf("missing expected segment %s", name)
		}
	}
}
