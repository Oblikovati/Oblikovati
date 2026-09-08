// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The evidence behind the feature engine's choice of level (ADR-0061 stage 6). The engine's
// post-condition runs on every body every feature builds, so the difference between the two levels
// is paid thousands of times over a rebuild — and unlike a suite wall time it is not affected by
// whatever else the machine is doing.
//
//	go test ./kernel/ops/validate/ -run XXX -bench Level -benchtime 300x

func BenchmarkLevelTopologyOnly(b *testing.B) {
	body := drilledPlate(b)
	b.ResetTimer()
	for b.Loop() {
		_ = ValidateTopology(body)
	}
}

func BenchmarkLevelFullValidate(b *testing.B) {
	body := drilledPlate(b)
	b.ResetTimer()
	for b.Loop() {
		_ = Validate(body)
	}
}

// drilledPlate is a block with two cylindrical bores: its top and bottom faces each carry an outer
// loop and two hole loops, which is exactly the shape checkHoleContainment does work on.
func drilledPlate(b *testing.B) *topo.Body {
	b.Helper()
	plate, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 6, 2), "plate")
	if err != nil {
		b.Fatalf("plate: %v", err)
	}
	for i, at := range []math.Point3{math.P3(3, 3, -1), math.P3(7, 3, -1)} {
		drill, err := brep.SolidCylinder(at, math.V3(0, 0, 1), 1, 4)
		if err != nil {
			b.Fatalf("drill %d: %v", i, err)
		}
		bored, err := brep.Boolean(brep.Difference, plate, drill)
		if err != nil {
			b.Fatalf("bore %d: %v", i, err)
		}
		plate = bored
	}
	return plate
}
