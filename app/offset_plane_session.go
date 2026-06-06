// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati/model/param"

// Session bridge for the Offset Plane tool's UI: the head reads/sets the offset in the
// document's display unit (e.g. mm) without touching database units or the tool's
// internals — the same shape as the Extrude bridge.

// ActiveOffsetPlane returns the running Offset Plane tool, or nil when the active tool is
// not an offset-plane (or there is none).
func (s *Session) ActiveOffsetPlane() *OffsetWorkPlaneTool {
	if s.tool == nil {
		return nil
	}
	t, _ := s.tool.tool.(*OffsetWorkPlaneTool)
	return t
}

// OffsetDistanceDisplay returns the active offset tool's distance in the document's length
// unit (the value the input field shows), or 0 when no offset tool is active.
func (s *Session) OffsetDistanceDisplay() float64 {
	t := s.ActiveOffsetPlane()
	if t == nil {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(t.Distance(), param.Length))
}

// SetOffsetDistanceDisplay sets the active offset tool's distance from a value given in the
// document's length unit (e.g. 25 mm), converting to database units. A no-op when no offset
// tool is active.
func (s *Session) SetOffsetDistanceDisplay(value float64) {
	if t := s.ActiveOffsetPlane(); t != nil {
		t.SetDistance(s.DocumentUnits().FromPreferred(value, param.Length).Value)
	}
}
