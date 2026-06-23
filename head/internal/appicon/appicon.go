// SPDX-License-Identifier: GPL-2.0-only

// Package appicon is the application's brand icon: one source-of-truth SVG (the
// Oblikovati mark) plus encoders that turn it into the raster forms each platform
// wants — an RGBA bitmap for the live window/taskbar icon (GLFW), PNGs for the Linux
// AppImage and the macOS .icns iconset, and a multi-resolution .ico for the Windows
// executable. All rasterization goes through head/icon, the one vetted SVG seam, so the
// glyph is identical everywhere and there are no committed per-size bitmaps to drift.
package appicon

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"

	"oblikovati.org/head/icon"
)

//go:embed oblikovati.svg
var svg []byte

// maxICOSize is the largest dimension a Windows .ico entry can encode (the width/height
// byte is 0 for 256 and cannot exceed it).
const maxICOSize = 256

// Image rasterizes the icon to a px×px RGBA bitmap on a transparent background.
//
//	img, err := appicon.Image(64) // a 64px window icon
func Image(px int) (*image.RGBA, error) {
	return icon.RenderFull(svg, px)
}

// WritePNG renders the icon at px×px and writes it as a PNG (used for the Linux
// AppImage icon and the macOS .icns iconset members).
func WritePNG(w io.Writer, px int) error {
	img, err := Image(px)
	if err != nil {
		return err
	}
	return png.Encode(w, img)
}

// WriteICO writes a Windows .ico bundling the icon at each requested size as a
// PNG-compressed entry (the Vista+ ICO form, which every supported Windows reads).
//
//	err := appicon.WriteICO(f, 16, 32, 48, 64, 128, 256)
func WriteICO(w io.Writer, sizes ...int) error {
	if len(sizes) == 0 {
		return fmt.Errorf("appicon: WriteICO needs at least one size")
	}
	pngs := make([][]byte, len(sizes))
	for i, s := range sizes {
		if s <= 0 || s > maxICOSize {
			return fmt.Errorf("appicon: ICO size must be 1..%d, got %d", maxICOSize, s)
		}
		var buf bytes.Buffer
		if err := WritePNG(&buf, s); err != nil {
			return err
		}
		pngs[i] = buf.Bytes()
	}
	return writeICOContainer(w, sizes, pngs)
}

// writeICOContainer emits the ICONDIR header, one ICONDIRENTRY per image, then the PNG
// payloads — offsets computed from the fixed 6-byte header + 16 bytes per entry.
func writeICOContainer(w io.Writer, sizes []int, pngs [][]byte) error {
	hdr := make([]byte, 6)
	binary.LittleEndian.PutUint16(hdr[2:], 1) // image type = icon
	binary.LittleEndian.PutUint16(hdr[4:], uint16(len(sizes)))
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := byte(s % maxICOSize) // 256 encodes as 0
		entry := make([]byte, 16)
		entry[0], entry[1] = dim, dim                // width, height
		binary.LittleEndian.PutUint16(entry[4:], 1)  // color planes
		binary.LittleEndian.PutUint16(entry[6:], 32) // bits per pixel
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(pngs[i])))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		if _, err := w.Write(entry); err != nil {
			return err
		}
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		if _, err := w.Write(p); err != nil {
			return err
		}
	}
	return nil
}
