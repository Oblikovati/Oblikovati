// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/param"

// Session bridge for the Extrude tool's UI: the head reads/sets the distance in the
// document's display unit (e.g. mm) without touching database units or the tool's
// internals, and asks whether a profile has been picked so it can prompt vs. highlight.

// ActiveExtrude returns the running Extrude tool, or nil when the active tool is not an
// extrude (or there is none).
func (s *Session) ActiveExtrude() *ExtrudeTool {
	if s.tool == nil {
		return nil
	}
	ext, _ := s.tool.tool.(*ExtrudeTool)
	return ext
}

// ExtrudeDistanceDisplay returns the active extrude tool's distance in the document's
// length unit (the value an input field shows), or 0 when no extrude is active.
func (s *Session) ExtrudeDistanceDisplay() float64 {
	ext := s.ActiveExtrude()
	if ext == nil {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(ext.Distance(), param.Length))
}

// SetExtrudeDistanceDisplay sets the active extrude tool's distance from a value given in
// the document's length unit (e.g. 25 mm), converting to database units. A no-op when no
// extrude is active.
func (s *Session) SetExtrudeDistanceDisplay(value float64) {
	if ext := s.ActiveExtrude(); ext != nil {
		ext.SetDistance(s.DocumentUnits().FromPreferred(value, param.Length).Value)
	}
}

// LengthUnitName returns the document's preferred length-unit name (e.g. "mm"), to label
// the distance field.
func (s *Session) LengthUnitName() string {
	return s.DocumentUnits().PreferredName(param.Length)
}

// AngleUnitName returns the document's preferred angle-unit name (e.g. "deg"), to label
// the taper field.
func (s *Session) AngleUnitName() string {
	return s.DocumentUnits().PreferredName(param.Angle)
}

// ExtrudeSecondDistanceDisplay / SetExtrudeSecondDistanceDisplay read/write the active
// extrude's asymmetric second-direction depth in the document's length unit.
func (s *Session) ExtrudeSecondDistanceDisplay() float64 {
	ext := s.ActiveExtrude()
	if ext == nil {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(ext.SecondDistance(), param.Length))
}

func (s *Session) SetExtrudeSecondDistanceDisplay(value float64) {
	if ext := s.ActiveExtrude(); ext != nil {
		ext.SetSecondDistance(s.DocumentUnits().FromPreferred(value, param.Length).Value)
	}
}

// ExtrudeTaperDisplay / SetExtrudeTaperDisplay read/write the active extrude's draft
// taper in the document's angle unit (e.g. degrees).
func (s *Session) ExtrudeTaperDisplay() float64 {
	ext := s.ActiveExtrude()
	if ext == nil {
		return 0
	}
	return s.DocumentUnits().ToPreferred(param.Q(ext.Taper(), param.Angle))
}

func (s *Session) SetExtrudeTaperDisplay(value float64) {
	if ext := s.ActiveExtrude(); ext != nil {
		ext.SetTaper(s.DocumentUnits().FromPreferred(value, param.Angle).Value)
	}
}
