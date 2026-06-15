// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/api/types"

// DriveSettings describes a joint drive (M12-F03, Oblikovati/Oblikovati#366): which driven
// variable to sweep, the value range and (positive) step, how many times to repeat, whether
// to ping-pong (end→start on alternate passes), and whether a collision halts the sweep.
// Values are in the variable's units (radians for angular, centimetres for linear). It is a
// value object built by the router from the wire DTO and implements contract.DriveSettings.
type DriveSettings struct {
	variable   types.DriveVariable
	start, end float64
	step       float64
	reps       int
	pingPong   bool
	collision  bool
}

// NewDriveSettings builds drive settings from plain values (the router's bridge from the wire
// DTO), e.g. NewDriveSettings(types.DriveAngular, 0, math.Pi, math.Pi/18, 1, false, true).
func NewDriveSettings(variable types.DriveVariable, start, end, step float64, reps int, pingPong, collision bool) DriveSettings {
	return DriveSettings{variable: variable, start: start, end: end, step: step, reps: reps, pingPong: pingPong, collision: collision}
}

// Variable selects which driven variable the drive sweeps.
func (s DriveSettings) Variable() types.DriveVariable { return s.variable }

// Range returns the start and end values of the sweep.
func (s DriveSettings) Range() (start, end float64) { return s.start, s.end }

// Step returns the increment between consecutive frames (always reported positive).
func (s DriveSettings) Step() float64 { return s.step }

// RepetitionCount returns how many times the sweep repeats (1 = a single pass).
func (s DriveSettings) RepetitionCount() int { return s.reps }

// RepetitionStartEndStart reports whether each repetition also plays back end→start.
func (s DriveSettings) RepetitionStartEndStart() bool { return s.pingPong }

// CollisionDetection reports whether the drive halts on interference.
func (s DriveSettings) CollisionDetection() bool { return s.collision }
