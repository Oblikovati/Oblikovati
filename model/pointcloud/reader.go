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
// true physical size in a cm or inch document (#1636). Malformed records are skipped with a
// per-record warning like the DWG decoder (#1646); a structural failure (or zero decodable
// points) is an error naming the offending input. An extension no registered reader handles
// errors, naming it.
//
// Example:
//
//	pts, warns, err := pointcloud.ReadScan("survey.las", data, exchange.TranslationOptions{TargetUnitMM: 10})
func ReadScan(filename string, data []byte, opts exchange.TranslationOptions) ([]math.Point3, []string, error) {
	samples, warns, err := ReadScanSamples(filename, data, opts)
	if err != nil {
		return nil, nil, err
	}
	points := make([]math.Point3, len(samples))
	for i, s := range samples {
		points[i] = s.Point
	}
	return points, warns, nil
}

// ReadScanSamples decodes a scan file into cloud-local samples, preserving any RGB or intensity
// channels the reader exposes.
func ReadScanSamples(filename string, data []byte, opts exchange.TranslationOptions) ([]PointSample, []string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, r := range registeredReaders {
		for _, e := range r.Extensions() {
			if e == ext {
				return readScaled(r, data, opts)
			}
		}
	}
	return nil, nil, fmt.Errorf("pointcloud: no reader for scan extension %q (file %q)", ext, filename)
}

// readScaled decodes with r and applies the single file→database unit factor to every point —
// the one shared scaling seam for all registered readers (#1636), mirroring meshio's scaleRaw.
func readScaled(r PointReader, data []byte, opts exchange.TranslationOptions) ([]PointSample, []string, error) {
	if err := opts.Report("points", 0, 0); err != nil { // #1647: honour a first-call cancel before decode
		return nil, nil, err
	}
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	if err := opts.Report("points", len(samples), len(samples)); err != nil { // #1647: report the record count
		return nil, nil, err
	}
	f := math.Scalar(opts.ImportScale(fileUnitMM(r, data)))
	if f == 1 {
		return samples, warns, nil
	}
	for i, s := range samples {
		s.Point = math.P3(s.Point.X*f, s.Point.Y*f, s.Point.Z*f)
		samples[i] = s
	}
	return samples, warns, nil
}

// samplesFromChannels zips a format reader's decoded columns — positions plus optional RGB and
// intensity aligned 1:1 with the points — into cloud-local samples. It is the single seam where the
// E57/PLY/LAS readers turn their kernel format package's ScanData into PointSamples, so the
// column→sample mapping lives in one place instead of each reader re-parsing bytes (#1788). A nil rgb
// or intensity column marks that channel absent for every point; a present column is 1:1 with points.
func samplesFromChannels(points []math.Point3, rgb [][3]float32, intensity []float64) []PointSample {
	out := make([]PointSample, len(points))
	for i, p := range points {
		s := PointSample{Point: p}
		if i < len(rgb) {
			s.HasRGB = true
			s.RGB = rgb[i]
		}
		if i < len(intensity) {
			s.HasIntensity = true
			s.Intensity = intensity[i]
		}
		out[i] = s
	}
	return out
}

// pointsOf projects samples onto their positions for the point-only Read path shared by every reader.
func pointsOf(samples []PointSample) []math.Point3 {
	out := make([]math.Point3, len(samples))
	for i, s := range samples {
		out[i] = s.Point
	}
	return out
}

// perFileUnitReader is an optional PointReader that derives the file's length unit from the decoded
// content, overriding the static FileUnitMM. It exists for formats whose spec unit is unreliable in
// the wild — an E57 that stores millimetre coordinates in the metre-typed cartesian field
// (#1789). A reader that does not implement it keeps its static FileUnitMM.
type perFileUnitReader interface {
	fileUnitMM(data []byte) (mm float64, ok bool)
}

// fileUnitMM resolves the file→millimetre unit for r's data: a reader that inspects the content
// (perFileUnitReader) wins when it can decide, else the reader's static FileUnitMM stands.
func fileUnitMM(r PointReader, data []byte) float64 {
	if pf, ok := r.(perFileUnitReader); ok {
		if mm, ok := pf.fileUnitMM(data); ok {
			return mm
		}
	}
	return r.FileUnitMM()
}

// scanUnitMM maps a "coordinates are really millimetres" verdict to the file's length unit in
// millimetres: a scan whose quantisation is too coarse to be metres is millimetres (#1789),
// otherwise the format's metre convention (E57 ASTM E2807, LAS ASPRS) stands. Shared by the E57 and
// LAS per-file overrides.
func scanUnitMM(millimetres bool) float64 {
	if millimetres {
		return 1 // millimetres
	}
	return 1000 // metres (the format's convention)
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
	// Read parses scan bytes into point-only coordinates for legacy callers.
	Read(data []byte) ([]math.Point3, []string, error)
	// ReadSamples parses scan bytes into cloud-local samples, erroring with the offending input on a
	// structural failure and warning per skipped record (#1646). Coordinates are returned in
	// the FILE's unit; ReadScan applies the unit scale.
	ReadSamples(data []byte) ([]PointSample, []string, error)
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

// Read parses each coordinate line into a sample. A line with fewer than three numeric fields
// that is the FIRST data line is treated as a PTS count header and skipped. Any other malformed
// line is SKIPPED with a warning citing the line number and content — the DWG decoder's
// warn-and-continue policy (#1646), so one bad record never sinks a million good points; a scan
// where nothing decodes errors instead.
func (asciiReader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	var samples []PointSample
	var warns []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // tolerate long lines
	line := 0
	for sc.Scan() {
		line++
		if w := scanLine(sc.Text(), line, &samples, len(warns)); w != "" {
			warns = append(warns, w)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("pointcloud: reading scan: %w", err)
	}
	if len(samples) == 0 && len(warns) > 0 {
		return nil, nil, fmt.Errorf("pointcloud: no decodable points; %s", warns[0])
	}
	return samples, warns, nil
}

// Read returns point-only coordinates for callers that do not need channels.
func (r asciiReader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	return pointsOf(samples), warns, nil
}

// scanLine parses one line into samples, returning a warning for a skipped malformed record (""
// when the line is fine). warnCount distinguishes a leading PTS count header (only valid before
// any point or warning) from a malformed record.
func scanLine(raw string, line int, samples *[]PointSample, warnCount int) string {
	text := strings.TrimSpace(raw)
	if skippableLine(text) {
		return ""
	}
	if s, ok := parseSample(text); ok {
		*samples = append(*samples, s)
		return ""
	}
	if len(*samples) == 0 && warnCount == 0 && isCountHeader(text) {
		return "" // a PTS file's leading point-count line
	}
	return fmt.Sprintf("pointcloud: skipped line %d, not \"x y z\": %q", line, text)
}

// skippableLine reports whether a line carries no coordinate (blank or a comment).
func skippableLine(text string) bool {
	return text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "//")
}

// parseSample reads a line into a sample, preserving any intensity or RGB columns.
func parseSample(text string) (PointSample, bool) {
	f := strings.Fields(text)
	if len(f) < 3 {
		return PointSample{}, false
	}
	pt, ok := parseXYZ(f)
	if !ok {
		return PointSample{}, false
	}
	s := PointSample{Point: pt}
	applySampleChannels(&s, f)
	return s, true
}

// parseXYZ parses the first three whitespace fields as the point's position.
func parseXYZ(f []string) (math.Point3, bool) {
	x, ex := strconv.ParseFloat(f[0], 64)
	y, ey := strconv.ParseFloat(f[1], 64)
	z, ez := strconv.ParseFloat(f[2], 64)
	if ex != nil || ey != nil || ez != nil {
		return math.Point3{}, false
	}
	return math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)), true
}

// applySampleChannels reads the optional intensity / RGB columns from the recognised ASCII layouts:
// 3 = xyz only, 4 = +intensity, 6 = +rgb, 7 = +intensity+rgb. Any other column count keeps just the
// xyz already parsed and ignores the extra columns — a 5-column (e.g. classification/return) or a
// 9-column (xyz+rgb+normals) scan imported before #645 and must keep importing (regression guard).
func applySampleChannels(s *PointSample, f []string) {
	switch len(f) {
	case 4:
		setSampleIntensity(s, f[3])
	case 6:
		setSampleRGB(s, f[3:6])
	case 7:
		setSampleIntensity(s, f[3])
		setSampleRGB(s, f[4:7])
	}
}

// setSampleIntensity sets the intensity channel from an ASCII field, leaving it unset if unparsable.
func setSampleIntensity(s *PointSample, field string) {
	if v, err := strconv.ParseFloat(field, 64); err == nil {
		s.HasIntensity = true
		s.Intensity = v
	}
}

// asciiColorMax is the ASCII scan colour convention's channel maximum; RGB is normalised to 0..1 by
// it at decode so the renderer never guesses the bit depth per point (#1787).
const asciiColorMax = 255

// setSampleRGB sets the RGB channel from three ASCII fields, leaving it unset if any is unparsable.
func setSampleRGB(s *PointSample, fields []string) {
	if rgb, ok := parseRGBFields(fields); ok {
		s.HasRGB = true
		s.RGB = rgb
	}
}

// parseRGBFields parses three ASCII colour columns, normalising each to 0..1 from the 0..255 scan
// convention (#1787). Returns false if any field is not numeric.
func parseRGBFields(fields []string) ([3]float32, bool) {
	var rgb [3]float32
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return [3]float32{}, false
		}
		rgb[i] = float32(v) / asciiColorMax
	}
	return rgb, true
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
