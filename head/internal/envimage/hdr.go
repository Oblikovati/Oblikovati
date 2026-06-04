// SPDX-License-Identifier: GPL-2.0-only

package envimage

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
)

// DecodeHDR reads a Radiance RGBE (.hdr) equirectangular image into a linear-RGB [Equirect].
// It supports the common new-style run-length encoding and a flat-scanline fallback; the RGBE
// shared exponent is expanded to float per the standard ldexp rule. This is a small in-repo
// decoder (no third-party dependency) so the loadable-environment path stays self-contained.
func DecodeHDR(path string) (Equirect, error) {
	f, err := os.Open(path)
	if err != nil {
		return Equirect{}, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	w, h, err := readHDRHeader(r)
	if err != nil {
		return Equirect{}, err
	}
	out := newEquirect(w, h)
	row := make([]byte, w*4) // one scanline of RGBE
	for y := 0; y < h; y++ {
		if err := readHDRScanline(r, row, w); err != nil {
			return Equirect{}, fmt.Errorf("scanline %d: %w", y, err)
		}
		for x := 0; x < w; x++ {
			rr, gg, bb := rgbeToFloat(row[x*4], row[x*4+1], row[x*4+2], row[x*4+3])
			out.set(x, y, rr, gg, bb)
		}
	}
	return out, nil
}

// readHDRHeader consumes the text header and the resolution line, returning width and height.
// Only the standard "-Y H +X W" orientation is accepted (top-to-bottom rows).
func readHDRHeader(r *bufio.Reader) (int, int, error) {
	first, err := r.ReadString('\n')
	if err != nil {
		return 0, 0, err
	}
	if !strings.HasPrefix(first, "#?") {
		return 0, 0, fmt.Errorf("envimage: not a Radiance HDR file (magic %q)", strings.TrimSpace(first))
	}
	for { // skip header variables until the blank separator line
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, 0, err
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}
	res, err := r.ReadString('\n')
	if err != nil {
		return 0, 0, err
	}
	var h, w int
	if _, err := fmt.Sscanf(strings.TrimSpace(res), "-Y %d +X %d", &h, &w); err != nil {
		return 0, 0, fmt.Errorf("envimage: unsupported resolution line %q", strings.TrimSpace(res))
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("envimage: bad HDR size %dx%d", w, h)
	}
	return w, h, nil
}

// readHDRScanline fills row (w×4 RGBE bytes) with one scanline, decoding new-style RLE when the
// 4-byte header marks it, else reading flat RGBE pixels.
func readHDRScanline(r *bufio.Reader, row []byte, w int) error {
	var hdr [4]byte
	if _, err := readFull(r, hdr[:]); err != nil {
		return err
	}
	rle := hdr[0] == 2 && hdr[1] == 2 && int(hdr[2])<<8|int(hdr[3]) == w && w >= 8 && w < 0x8000
	if !rle { // flat: hdr is the first pixel, read the rest
		copy(row[:4], hdr[:])
		_, err := readFull(r, row[4:])
		return err
	}
	for ch := 0; ch < 4; ch++ { // four RLE-encoded channel planes
		if err := decodeRLEChannel(r, row, ch, w); err != nil {
			return err
		}
	}
	return nil
}

// decodeRLEChannel decodes one run-length-encoded channel plane into row at stride 4.
func decodeRLEChannel(r *bufio.Reader, row []byte, ch, w int) error {
	for x := 0; x < w; {
		count, err := r.ReadByte()
		if err != nil {
			return err
		}
		if count > 128 { // a run of (count-128) identical bytes
			val, err := r.ReadByte()
			if err != nil {
				return err
			}
			for n := int(count) - 128; n > 0 && x < w; n-- {
				row[x*4+ch] = val
				x++
			}
			continue
		}
		for n := int(count); n > 0 && x < w; n-- { // count literal bytes
			val, err := r.ReadByte()
			if err != nil {
				return err
			}
			row[x*4+ch] = val
			x++
		}
	}
	return nil
}

// rgbeToFloat expands a shared-exponent RGBE pixel to linear float RGB.
func rgbeToFloat(r, g, b, e byte) (float32, float32, float32) {
	if e == 0 {
		return 0, 0, 0
	}
	f := math.Ldexp(1.0, int(e)-(128+8))
	return float32(float64(r) * f), float32(float64(g) * f), float32(float64(b) * f)
}

// readFull reads len(buf) bytes or returns an error (bufio.Reader has no io.ReadFull shortcut
// that surfaces short reads as cleanly here).
func readFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
