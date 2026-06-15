// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/model/assembly"
)

// The Drive command (M12-F03, Oblikovati/Oblikovati#366) animates a selected joint through its
// range of motion: it precomputes the motion frames and plays them back (drive_anim.go). It
// lives on the Joints panel and enables only when a drivable joint (rotational/slider/
// cylindrical) is selected in the browser's Joints folder.

// driveFrameCount is how many steps a default sweep is divided into.
const driveFrameCount = 36

// defaultSlideCm is the default linear sweep length (cm) for a slider with no limits.
const defaultSlideCm = 5.0

// driveCommand builds the Joints-panel Drive command.
func driveCommand() *CommandDefinition {
	return NewCommand("Assembly.Drive", "Drive", "Joints", func(s *Session) error {
		joint, ok := selectedDrivableJoint(s)
		if !ok {
			return fmt.Errorf("select a drivable joint (rotational, slider, or cylindrical) in the browser to drive")
		}
		return s.StartDriveAnimation(joint.ID(), defaultDriveSettings(joint))
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasSelectedDrivableJoint).
		WithIcon("joint-drive").WithButtonStyle(CompactIconButton).
		WithTooltip("Drive — animate a joint through its range of motion.")
}

// hasSelectedDrivableJoint enables Drive only when a drivable joint is selected.
func hasSelectedDrivableJoint(s *Session) bool {
	_, ok := selectedDrivableJoint(s)
	return ok
}

// selectedDrivableJoint returns the selected joint when it is one whose motion can be driven.
func selectedDrivableJoint(s *Session) (assembly.Joint, bool) {
	h, ok := s.Selection().First().(AssemblyJointHandle)
	if !ok || !isDrivableKind(h.Joint.Type()) {
		return nil, false
	}
	return h.Joint, true
}

// isDrivableKind reports whether a joint kind has a single driven variable to sweep.
func isDrivableKind(k types.AssemblyJointType) bool {
	return k == types.JointRotational || k == types.JointSlider || k == types.JointCylindrical
}

// defaultDriveSettings builds a sensible sweep for a joint: a slider sweeps its linear range,
// every other drivable kind its angular range. The sweep oscillates (there-and-back) so the
// looped playback reads as smooth motion.
func defaultDriveSettings(j assembly.Joint) assembly.DriveSettings {
	if j.Type() == types.JointSlider {
		start, end := linearRange(j)
		return assembly.NewDriveSettings(types.DriveLinear, start, end, stepFor(start, end), 2, true, false)
	}
	start, end := angularRange(j)
	return assembly.NewDriveSettings(types.DriveAngular, start, end, stepFor(start, end), 2, true, false)
}

// angularRange is the joint's angular limits, or a full turn when unbounded.
func angularRange(j assembly.Joint) (start, end float64) {
	if lim := j.Limits(); lim != nil {
		if lo, hasLo := lim.AngularMinimum(); hasLo {
			if hi, hasHi := lim.AngularMaximum(); hasHi {
				return lo, hi
			}
		}
		if hi, hasHi := lim.AngularMaximum(); hasHi {
			return 0, hi
		}
	}
	return 0, 2 * math.Pi
}

// linearRange is the joint's linear limits, or a default slide when unbounded.
func linearRange(j assembly.Joint) (start, end float64) {
	if lim := j.Limits(); lim != nil {
		if lo, hasLo := lim.LinearMinimum(); hasLo {
			if hi, hasHi := lim.LinearMaximum(); hasHi {
				return lo, hi
			}
		}
		if hi, hasHi := lim.LinearMaximum(); hasHi {
			return 0, hi
		}
	}
	return 0, defaultSlideCm
}

// stepFor divides a span into driveFrameCount steps (always positive; a zero span degenerates
// to a unit step so the sweep still has its endpoints).
func stepFor(start, end float64) float64 {
	span := math.Abs(end - start)
	if span == 0 {
		return 1
	}
	return span / driveFrameCount
}
