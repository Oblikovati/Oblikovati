// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridge for the free-form cage editor's property window.

// ActiveCageEdit returns the running Edit Freeform Cage tool, or nil when the active tool is not
// the cage editor (or there is none).
func (s *Session) ActiveCageEdit() *FreeformCageEditTool {
	return s.activeTool[*FreeformCageEditTool]()
}

// CanEditFreeformCage reports whether the active part holds a free-form body to edit — the
// enable predicate for the ribbon command, so the button greys out on a part with no cage
// rather than starting a tool that can draw nothing.
func (s *Session) CanEditFreeformCage() bool {
	_, ok := activeFreeformBody(s)
	return ok
}

// ApplyActiveCageLevel pushes the cage tool's level onto the body — what the panel's level field
// calls when the user changes it. A no-op when the tool is not running.
func (s *Session) ApplyActiveCageLevel() bool {
	t := s.ActiveCageEdit()
	if t == nil {
		return false
	}
	return s.ApplyCageLevel(t.Level())
}

// CreaseActiveCageHandle creases the edges around the cage tool's last dragged handle at its
// current sharpness — the panel's Crease action. It reports false when no handle has been
// dragged yet, which is what keeps the button disabled.
func (s *Session) CreaseActiveCageHandle() bool {
	t := s.ActiveCageEdit()
	if t == nil || t.LastVertex() < 0 {
		return false
	}
	return s.CreaseCageEdgesAround(t.LastVertex(), t.Sharpness())
}
