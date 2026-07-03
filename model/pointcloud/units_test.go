// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"encoding/binary"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/math"
)

// Point-cloud unit scaling tests (M40 audit S1, #1636): scan files scale from their file unit
// into the document's working (database) unit exactly like the mesh importers do, so a metric
// LAS survey attached to a cm or inch document lands at true physical size.

// cmDoc is the standard centimetre document translation (1 working unit = 10 mm, ADR-0042).
var cmDoc = exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM}

// inchDoc is an inch document translation (1 working unit = 25.4 mm).
var inchDoc = exchange.TranslationOptions{TargetUnitMM: 25.4}

func approxEq(a, b float64) bool { return stdmath.Abs(a-b) < 1e-9*(1+stdmath.Abs(b)) }

// TestScanReaderFileUnits pins each registered reader's declared file unit: E57 is metres by the
// ASTM E2807 spec, LAS is metres by ASPRS convention, and the unitless formats (ASCII XYZ/PTS,
// PLY) follow the same millimetre convention as unitless meshes (STL/OBJ) so it cannot drift.
func TestScanReaderFileUnits(t *testing.T) {
	want := map[string]float64{".xyz": 1, ".pts": 1, ".asc": 1, ".txt": 1, ".ply": 1, ".e57": 1000, ".las": 1000}
	for _, r := range registeredReaders {
		for _, ext := range r.Extensions() {
			if got := r.FileUnitMM(); got != want[ext] {
				t.Errorf("reader for %s: FileUnitMM = %v, want %v", ext, got, want[ext])
			}
		}
	}
}

// minimalLAS assembles a tiny valid uncompressed LAS 1.2 file (a named fake for the on-disk
// format): the 227-byte public header plus format-0 point records with the given real-coordinate
// points, stored as scaled integers (scale 0.001 m, zero offset).
func minimalLAS(points [][3]float64) []byte {
	const headerSize, recLen, scale = 227, 20, 0.001
	hdr := make([]byte, headerSize)
	copy(hdr, "LASF")
	hdr[24], hdr[25] = 1, 2
	binary.LittleEndian.PutUint16(hdr[94:], uint16(headerSize))
	binary.LittleEndian.PutUint32(hdr[96:], uint32(headerSize))
	binary.LittleEndian.PutUint16(hdr[105:], recLen)
	binary.LittleEndian.PutUint32(hdr[107:], uint32(len(points)))
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint64(hdr[131+i*8:], stdmath.Float64bits(scale))
	}
	out := hdr
	for _, p := range points {
		rec := make([]byte, recLen)
		for c := 0; c < 3; c++ {
			binary.LittleEndian.PutUint32(rec[c*4:], uint32(int32(stdmath.Round(p[c]/scale))))
		}
		out = append(out, rec...)
	}
	return out
}

// TestReadScanScalesLASIntoWorkingUnits: a metric LAS (metres) lands at true physical scale in
// both a centimetre and an inch document — a 2 m inter-point distance is 200 cm and 78.740… in.
func TestReadScanScalesLASIntoWorkingUnits(t *testing.T) {
	data := minimalLAS([][3]float64{{0, 0, 0}, {2, 0, 0}})
	for _, tc := range []struct {
		name string
		opts exchange.TranslationOptions
		want float64 // distance in working units
	}{
		{"cm document", cmDoc, 200},
		{"inch document", inchDoc, 2000 / 25.4},
	} {
		pts, _, err := ReadScan("survey.las", data, tc.opts)
		if err != nil || len(pts) != 2 {
			t.Fatalf("%s: ReadScan = %d points, err %v", tc.name, len(pts), err)
		}
		if got := float64(pts[0].DistanceTo(pts[1])); !approxEq(got, tc.want) {
			t.Errorf("%s: 2 m LAS distance = %v working units, want %v", tc.name, got, tc.want)
		}
	}
}

// TestReadScanUnitlessMillimetreConvention pins the declared fallback for unitless scan formats:
// like STL/OBJ meshes, an ASCII XYZ is read as millimetres, so 10 units become 1 cm.
func TestReadScanUnitlessMillimetreConvention(t *testing.T) {
	pts, _, err := ReadScan("scan.xyz", []byte("10 20 30\n"), cmDoc)
	if err != nil || len(pts) != 1 {
		t.Fatalf("ReadScan = %d points, err %v", len(pts), err)
	}
	want := math.P3(1, 2, 3)
	if !approxEq(float64(pts[0].X), float64(want.X)) || !approxEq(float64(pts[0].Y), float64(want.Y)) || !approxEq(float64(pts[0].Z), float64(want.Z)) {
		t.Errorf("10 20 30 mm in a cm document = %v, want %v", pts[0], want)
	}
}

// TestPLYCloudMatchesMeshScale: the .ply symmetry guard (#1636) — a PLY point imported as a cloud
// gets exactly the coordinate scale a mesh import applies to the same unitless millimetre data,
// verified end to end against a one-triangle STL welded through meshio.
func TestPLYCloudMatchesMeshScale(t *testing.T) {
	// One triangle with vertices (0,0,0) (40,0,0) (0,40,0), in file millimetres.
	tris := [][3][3]float32{{{0, 0, 0}, {40, 0, 0}, {0, 40, 0}}}
	body, _, err := meshio.ImportBody("stl", binarySTL(tris), "import:stl#0", 0, cmDoc)
	if err != nil {
		t.Fatalf("meshio.ImportBody: %v", err)
	}
	ply := "ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nend_header\n0 0 0\n40 0 0\n0 40 0\n"
	pts, _, err := ReadScan("scan.ply", []byte(ply), cmDoc)
	if err != nil || len(pts) != 3 {
		t.Fatalf("ReadScan(.ply) = %d points, err %v", len(pts), err)
	}
	cloudBox := math.EmptyBox()
	for _, p := range pts {
		cloudBox = cloudBox.ExtendPoint(p)
	}
	meshBox := body.RangeBox()
	if !approxEq(float64(cloudBox.Max.X), float64(meshBox.Max.X)) || !approxEq(float64(cloudBox.Max.Y), float64(meshBox.Max.Y)) {
		t.Errorf("PLY cloud box %v != mesh box %v for identical source coordinates", cloudBox, meshBox)
	}
}

// binarySTL encodes triangles as a minimal binary STL (80-byte header, count, 50-byte records).
func binarySTL(tris [][3][3]float32) []byte {
	out := make([]byte, 80, 84+50*len(tris))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(tris)))
	for _, tri := range tris {
		out = append(out, make([]byte, 12)...) // zero normal; readers recompute
		for _, v := range tri {
			for _, c := range v {
				out = binary.LittleEndian.AppendUint32(out, stdmath.Float32bits(c))
			}
		}
		out = append(out, 0, 0) // attribute byte count
	}
	return out
}
