// SPDX-License-Identifier: GPL-2.0-only

package appicon

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestImageRendersNonBlank(t *testing.T) {
	img, err := Image(64)
	if err != nil {
		t.Fatalf("Image(64): %v", err)
	}
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 64 {
		t.Fatalf("bounds = %v, want 64x64", b)
	}
	var opaque int
	for _, a := range alphas(img.Pix) {
		if a > 0 {
			opaque++
		}
	}
	if opaque == 0 {
		t.Fatal("rendered icon is fully transparent — the mark did not draw")
	}
}

func TestImageRejectsNonPositiveSize(t *testing.T) {
	if _, err := Image(0); err == nil {
		t.Fatal("Image(0) should error")
	}
}

func TestWritePNGDecodes(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePNG(&buf, 128); err != nil {
		t.Fatalf("WritePNG: %v", err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Width != 128 || cfg.Height != 128 {
		t.Fatalf("png = %dx%d, want 128x128", cfg.Width, cfg.Height)
	}
}

func TestWriteICOHasEntriesAndPNGPayloads(t *testing.T) {
	sizes := []int{16, 32, 256}
	var buf bytes.Buffer
	if err := WriteICO(&buf, sizes...); err != nil {
		t.Fatalf("WriteICO: %v", err)
	}
	b := buf.Bytes()
	if got := binary.LittleEndian.Uint16(b[2:]); got != 1 {
		t.Fatalf("ICO type = %d, want 1", got)
	}
	if got := int(binary.LittleEndian.Uint16(b[4:])); got != len(sizes) {
		t.Fatalf("ICO count = %d, want %d", got, len(sizes))
	}
	for i, s := range sizes {
		entry := b[6+16*i:]
		wantDim := byte(s % 256)
		if entry[0] != wantDim || entry[1] != wantDim {
			t.Fatalf("entry %d dim = %d, want %d (256 encodes as 0)", i, entry[0], wantDim)
		}
		off := binary.LittleEndian.Uint32(entry[12:])
		length := binary.LittleEndian.Uint32(entry[8:])
		payload := b[off : off+length]
		if _, err := png.DecodeConfig(bytes.NewReader(payload)); err != nil {
			t.Fatalf("entry %d payload is not a PNG: %v", i, err)
		}
	}
}

func TestWriteICORejectsBadSizes(t *testing.T) {
	if err := WriteICO(&bytes.Buffer{}); err == nil {
		t.Fatal("WriteICO with no sizes should error")
	}
	if err := WriteICO(&bytes.Buffer{}, 512); err == nil {
		t.Fatal("WriteICO with size > 256 should error")
	}
}

// alphas returns the alpha byte of every RGBA pixel in a row-major pix slice.
func alphas(pix []uint8) []uint8 {
	out := make([]uint8, 0, len(pix)/4)
	for i := 3; i < len(pix); i += 4 {
		out = append(out, pix[i])
	}
	return out
}
