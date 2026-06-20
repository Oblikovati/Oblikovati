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

	"oblikovati.org/math"
)

// registeredReaders is the decoder set keyed by lowercase extension. ASCII formats ship now;
// LAS/E57 readers register here later without touching call sites.
var registeredReaders = []PointReader{NewASCIIReader()}

// ReadScan decodes a scan file's bytes into cloud-local points, choosing the reader by the
// filename's extension. It errors when no registered reader handles the extension, naming it.
func ReadScan(filename string, data []byte) ([]math.Point3, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, r := range registeredReaders {
		for _, e := range r.Extensions() {
			if e == ext {
				return r.Read(data)
			}
		}
	}
	return nil, fmt.Errorf("pointcloud: no reader for scan extension %q (file %q)", ext, filename)
}

// PointReader decodes one scan-file format's bytes into cloud-local points. One implementation
// per format keeps the cloud model format-agnostic; the host wraps any third-party decoder
// behind this project-owned seam (CLAUDE.md "Dependencies"). The first reader is clean-room
// ASCII XYZ/PTS; LAS/E57 are future readers registered the same way.
type PointReader interface {
	// Extensions returns the lowercase file extensions (with leading dot) this reader handles.
	Extensions() []string
	// Read parses scan bytes into cloud-local points, erroring with the offending input.
	Read(data []byte) ([]math.Point3, error)
}

// asciiReader parses the open ASCII point formats — XYZ and PTS — where each non-empty,
// non-comment line is "x y z" optionally followed by intensity / r g b columns (ignored for
// now). A lone leading integer (a PTS point-count header) is skipped. Coordinates are read in
// the file's own units; the owning PointCloud's UnitsFactor/scale maps them to model space.
type asciiReader struct{}

// NewASCIIReader returns the clean-room reader for .xyz/.pts/.asc/.txt scan files.
func NewASCIIReader() PointReader { return asciiReader{} }

func (asciiReader) Extensions() []string { return []string{".xyz", ".pts", ".asc", ".txt"} }

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
