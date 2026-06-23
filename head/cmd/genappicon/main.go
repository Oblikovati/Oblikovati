// SPDX-License-Identifier: GPL-2.0-only

// Command genappicon renders the embedded application icon to a platform raster file.
// The release packaging scripts call it so the Linux AppImage PNG and the macOS .icns
// iconset members come straight from the source SVG, with no committed per-size
// bitmaps to drift out of sync.
//
//	go run ./cmd/genappicon -format png -size 256 -out icon.png
//	go run ./cmd/genappicon -format ico -sizes 16,32,48,64,128,256 -out oblikovati.ico
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"oblikovati.org/head/internal/appicon"
)

func main() {
	format := flag.String("format", "png", "output format: png or ico")
	size := flag.Int("size", 256, "PNG edge length in pixels")
	sizes := flag.String("sizes", "16,32,48,64,128,256", "comma-separated ICO sizes")
	out := flag.String("out", "", "output file path (required)")
	flag.Parse()
	if err := generate(*format, *size, *sizes, *out); err != nil {
		fmt.Fprintln(os.Stderr, "genappicon:", err)
		os.Exit(1)
	}
}

// generate writes the requested raster of the app icon to out, returning the file's
// close error so a truncated/short write is reported rather than swallowed.
func generate(format string, size int, sizes, out string) error {
	if out == "" {
		return fmt.Errorf("-out is required")
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := encode(f, format, size, sizes); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// encode dispatches on format, writing the icon to w.
func encode(w *os.File, format string, size int, sizes string) error {
	switch format {
	case "png":
		return appicon.WritePNG(w, size)
	case "ico":
		parsed, err := parseSizes(sizes)
		if err != nil {
			return err
		}
		return appicon.WriteICO(w, parsed...)
	default:
		return fmt.Errorf("unknown -format %q (want png or ico)", format)
	}
}

// parseSizes turns "16,32,256" into [16 32 256].
func parseSizes(csv string) ([]int, error) {
	parts := strings.Split(csv, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("bad size %q: %w", p, err)
		}
		out = append(out, n)
	}
	return out, nil
}
