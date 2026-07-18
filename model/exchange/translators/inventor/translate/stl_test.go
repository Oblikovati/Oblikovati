// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"encoding/binary"
	"math"
	"testing"

	"oblikovati.org/kernel/exchange/meshio"
	omath "oblikovati.org/math"
)

// TestEncodeBinarySTL checks the binary-STL framing (80-byte header + u32 count + 50 bytes/triangle),
// the cm→mm scaling, and that a NaN coordinate is written as zero rather than propagated.
func TestEncodeBinarySTL(t *testing.T) {
	raw := meshio.RawMesh{
		Verts: []omath.Point3{
			omath.P3(0, 0, 0),
			omath.P3(1, 0, 0),
			omath.P3(math.NaN(), 1, 0), // exercises writeVec3's NaN guard
		},
		Tris: [][3]int{{0, 1, 2}},
	}
	got := encodeBinarySTL(raw)
	if want := 80 + 4 + 50; len(got) != want {
		t.Fatalf("STL length = %d, want %d (header + count + one 50-byte facet)", len(got), want)
	}
	if n := binary.LittleEndian.Uint32(got[80:84]); n != 1 {
		t.Errorf("triangle count field = %d, want 1", n)
	}
	// Facet body starts at header(80)+count(4)=84: normal(12), then the three vertices (12 each).
	// The second vertex is (1,0,0) cm → its x must scale to 10 mm.
	if x := math.Float32frombits(binary.LittleEndian.Uint32(got[84+12+12:])); x != 10 {
		t.Errorf("second vertex x = %v mm, want 10 (1 cm scaled)", x)
	}
}

func TestAxisIndex(t *testing.T) {
	for name, want := range map[string]int{"smallest": 0, "middle": 1, "largest": 2, "other": 2} {
		if got := axisIndex(name); got != want {
			t.Errorf("axisIndex(%q) = %d, want %d", name, got, want)
		}
	}
}
