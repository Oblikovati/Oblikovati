// SPDX-License-Identifier: GPL-2.0-only

package app

// sketchPlacement is defined here with the drag-to-create state machine that owns it.
type sketchPlacement struct {
	active  bool
	pressX  float64
	pressY  float64
	dragged bool
}
