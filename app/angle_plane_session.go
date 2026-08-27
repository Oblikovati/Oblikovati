// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the Angle Plane tool's property window.

// ActiveAnglePlane returns the running Angle to Plane tool, or nil when the active tool is not
// an angle-plane (or there is none).
func (s *Session) ActiveAnglePlane() *AngleWorkPlaneTool {
	return s.activeTool[*AngleWorkPlaneTool]()
}
