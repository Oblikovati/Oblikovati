// SPDX-License-Identifier: GPL-2.0-only

// Package pointcloud models attached laser-scan / photogrammetry point clouds (M17-F06, #645):
// a linked, transformable reference object a design can be modeled against, not B-rep geometry.
// A cloud carries its scan points in cloud-local coordinates plus a placement transform and
// scale into model space, a display budget, and visibility. Scan decoding is pluggable behind
// PointReader so new formats (LAS/E57) drop in without touching the cloud model.
package pointcloud

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/math"
)

// registeredReaders is the decoder set keyed by lowercase extension: ASCII, PLY, E57, and LAS.
// A new format's reader registers here without touching call sites.
var registeredReaders = []PointReader{NewASCIIReader(), NewPLYReader(), NewE57Reader(), NewLASReader()}

// ReadScan decodes a scan file's bytes into cloud-local points, choosing the reader by the
// filename's extension and scaling the file's unit into the document's working (database) unit —
// the same TranslationOptions seam the mesh importers use, so a metric LAS/E57 survey lands at
// true physical size in a cm or inch document (#1636). It errors when no registered reader
// handles the extension, naming it.
//
// Example:
//
//	pts, err := pointcloud.ReadScan("survey.las", data, exchange.TranslationOptions{TargetUnitMM: 10})
func ReadScan(filename string, data []byte, opts exchange.TranslationOptions) ([]math.Point3, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, r := range registeredReaders {
		for _, e := range r.Extensions() {
			if e == ext {
				return readScaled(r, data, opts)
			}
		}
	}
	return nil, fmt.Errorf("pointcloud: no reader for scan extension %q (file %q)", ext, filename)
}

// readScaled decodes with r and applies the single file→database unit factor to every point —
// the one shared scaling seam for all registered readers (#1636), mirroring meshio's scaleRaw.
func readScaled(r PointReader, data []byte, opts exchange.TranslationOptions) ([]math.Point3, error) {
	points, err := r.Read(data)
	if err != nil {
		return nil, err
	}
	f := math.Scalar(opts.ImportScale(r.FileUnitMM()))
	if f == 1 {
		return points, nil
	}
	for i, p := range points {
		points[i] = math.P3(p.X*f, p.Y*f, p.Z*f)
	}
	return points, nil
}

// IsScanFile reports whether a path's extension is a 3D-scan point-cloud format handled by a
// registered reader (.xyz/.pts/.asc/.txt/.ply/.e57/.las). The import flow routes such files to the
// point-cloud attach path — the appropriate home for scan data — rather than the body/sketch
// importers (#645).
func IsScanFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, r := range registeredReaders {
		for _, e := range r.Extensions() {
			if e == ext {
				return true
			}
		}
	}
	return false
}

// ScanExtensions returns every file extension a registered scan reader handles (for the import
// file dialog's filter).
func ScanExtensions() []string {
	var out []string
	for _, r := range registeredReaders {
		out = append(out, r.Extensions()...)
	}
	return out
}

// PointReader decodes one scan-file format's bytes into cloud-local points. One implementation
// per format keeps the cloud model format-agnostic; the host wraps any third-party decoder
// behind this project-owned seam (CLAUDE.md "Dependencies"). The first reader is clean-room
// ASCII XYZ/PTS; LAS/E57 are future readers registered the same way.
type PointReader interface {
	// Extensions returns the lowercase file extensions (with leading dot) this reader handles.
	Extensions() []string
	// Read parses scan bytes into cloud-local points, erroring with the offending input.
	// Coordinates are returned in the FILE's unit; ReadScan applies the unit scale.
	Read(data []byte) ([]math.Point3, error)
	// FileUnitMM is the millimetre size of the format's length unit: E57 is metres by the ASTM
	// E2807 spec, LAS metres by ASPRS convention, and the unitless formats (ASCII XYZ/PTS, PLY)
	// follow the same declared millimetre convention as unitless meshes (STL/OBJ) (#1636).
	FileUnitMM() float64
}

// asciiReader parses the open ASCII point formats — XYZ and PTS — where each non-empty,
// non-comment line is "x y z" optionally followed by intensity / r g b columns (ignored for
// now). A lone leading integer (a PTS point-count header) is skipped. The formats carry no unit,
// so they follow the millimetre convention (FileUnitMM = 1), like unitless meshes (#1636).
type asciiReader struct{}

// NewASCIIReader returns the clean-room reader for .xyz/.pts/.asc/.txt scan files.
func NewASCIIReader() PointReader { return asciiReader{} }

func (asciiReader) Extensions() []string { return []string{".xyz", ".pts", ".asc", ".txt"} }

// FileUnitMM: ASCII scan formats are unitless — the declared millimetre convention applies.
func (asciiReader) FileUnitMM() float64 { return 1 }

// Read parses each coordinate line into a point. A line with fewer than three numeric fields
// that is the FIRST data line is treated as a PTS count header and skipped; any other malformed
// line is an error citing the line number and content.
func (asciiReader) Read(data []byte) ([]math.Point3, error) {
	var points []math.Point3
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // tolerate long lines
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if skippableLine(text) {
			continue
		}
		p, ok := parseXYZ(text)
		if ok {
			points = append(points, p)
			continue
		}
		if len(points) == 0 && isCountHeader(text) {
			continue // a PTS file's leading point-count line
		}
		return nil, fmt.Errorf("pointcloud: line %d is not \"x y z\": %q", line, text)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("pointcloud: reading scan: %w", err)
	}
	return points, nil
}

// skippableLine reports whether a line carries no coordinate (blank or a comment).
func skippableLine(text string) bool {
	return text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "//")
}

// parseXYZ reads the first three whitespace-separated fields as x, y, z, ignoring any trailing
// columns (intensity/colour). ok is false when the line lacks three parseable floats.
func parseXYZ(text string) (math.Point3, bool) {
	f := strings.Fields(text)
	if len(f) < 3 {
		return math.Point3{}, false
	}
	x, ex := strconv.ParseFloat(f[0], 64)
	y, ey := strconv.ParseFloat(f[1], 64)
	z, ez := strconv.ParseFloat(f[2], 64)
	if ex != nil || ey != nil || ez != nil {
		return math.Point3{}, false
	}
	return math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)), true
}

// isCountHeader reports whether a line is a single non-negative integer (a PTS count header).
func isCountHeader(text string) bool {
	f := strings.Fields(text)
	if len(f) != 1 {
		return false
	}
	n, err := strconv.Atoi(f[0])
	return err == nil && n >= 0
}
