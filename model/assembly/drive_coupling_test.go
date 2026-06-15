// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestDriveRotateRotatePropagatesGearRatio is the #883 regression: driving one gear of a
// rotate-rotate–coupled pair must turn the other gear THROUGH the ratio. A driver and a driven
// gear each spin on a grounded frame (their own rotational joints); a rotate-rotate constraint
// gears them 2:1. Driving the driver π/4 must turn the driven gear 2×π/4. Before the
// drive-coupling fix the driven gear did not move at all (rotate-rotate contributes no static
// residual, so the static re-solve left it behind).
func TestDriveRotateRotatePropagatesGearRatio(t *testing.T) {
	occs := occurrence.NewOccurrences()
	ground := place(occs, "ground:1", math.Identity4())
	ground.SetGrounded(true)
	driver := place(occs, "driver:1", math.Identity4())
	driven := place(occs, "driven:1", math.Translation4(math.V3(10, 0, 0)))

	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	zAxis := func(o *occurrence.Occurrence, x float64) Ref {
		return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(x, 0, 0), unit(t, 0, 0, 1))}
	}
	driverJoint := js.AddRotational(zAxis(ground, 0), zAxis(driver, 0))
	js.AddRotational(zAxis(ground, 10), zAxis(driven, 10))
	cs.AddRotateRotate(zAxis(driver, 0), zAxis(driven, 10), 2.0)

	span := stdmath.Pi / 4
	res, err := DriveJoint(occs, cs, js, driverJoint.ID(), NewDriveSettings(types.DriveAngular, 0, span, span/4, 1, false, false))
	if err != nil {
		t.Fatalf("DriveJoint: %v", err)
	}

	roll := func(occID uint64, frame int) float64 {
		for _, p := range res.Frames[frame].Placements {
			if p.Occurrence == occID {
				return rollAboutZ(p.Transform)
			}
		}
		t.Fatalf("occurrence %d absent from frame %d", occID, frame)
		return 0
	}
	last := len(res.Frames) - 1
	driverSwept := stdmath.Abs(roll(driver.ID(), last) - roll(driver.ID(), 0))
	drivenSwept := stdmath.Abs(roll(driven.ID(), last) - roll(driven.ID(), 0))
	if stdmath.Abs(driverSwept-span) > 1e-3 {
		t.Errorf("driver swept %.4f rad, want %.4f", driverSwept, span)
	}
	if stdmath.Abs(drivenSwept-2*span) > 1e-3 {
		t.Errorf("driven gear swept %.4f rad, want %.4f (2:1 gear ratio propagated by the drive)", drivenSwept, 2*span)
	}
}
