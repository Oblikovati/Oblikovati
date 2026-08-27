// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Measure tool's readout panel.

// ActiveMeasure returns the running Measure tool, or nil when the active tool is not a measure (or
// there is none).
func (s *Session) ActiveMeasure() *MeasureTool {
	return s.activeTool[*MeasureTool]()
}
