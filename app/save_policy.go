// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/app/options"
)

// The save policy half of M03-F09 (#610): the "save" option group bound to its
// live readers — the head's thumbnail capture, the session's dependent-save
// flow, and the package store's old-version retention. Setters reject what
// this host cannot perform (no dead settings, the M05-F11 rule).

// supportedThumbnailModes are the capture modes this host implements: none,
// and capturing the active viewport when a save completes (the head's
// framebuffer readback). The iso-view and import-from-file modes await a
// headless render path.
var supportedThumbnailModes = map[types.ThumbnailSaveOption]bool{
	types.ThumbnailNone:               true,
	types.ThumbnailActiveWindowOnSave: true,
}

// SetSaveOptions validates, applies and stores the save group.
func (s *Session) SetSaveOptions(o options.Save) error {
	if !supportedThumbnailModes[o.Thumbnail] {
		return fmt.Errorf("app: thumbnail mode %q is not implemented by this host (none|activeWindowOnSave)", o.Thumbnail)
	}
	if o.OldVersionsToKeep < 0 {
		return fmt.Errorf("app: oldVersionsToKeep %d is negative (0 disables retention)", o.OldVersionsToKeep)
	}
	s.appOptions.Save = o
	s.applyOldVersionRetention(o.OldVersionsToKeep)
	return s.saveOptions()
}

// oldVersionRetainer is what the package store implements; the session only
// knows the doc.Store seam, so retention forwards through this assertion.
type oldVersionRetainer interface {
	SetOldVersionsToKeep(count int)
}

// applyOldVersionRetention forwards the retention count to a store that
// supports it (a memory-only session has nothing to retain).
func (s *Session) applyOldVersionRetention(count int) {
	if r, ok := s.store.(oldVersionRetainer); ok {
		r.SetOldVersionsToKeep(count)
	}
}

// SavePolicyControl is the [contract.SaveOptions] view over the session's
// save group.
type SavePolicyControl struct{ s *Session }

var _ contract.SaveOptions = SavePolicyControl{}

// SavePolicy returns the in-process save-options control.
func (s *Session) SavePolicy() SavePolicyControl { return SavePolicyControl{s: s} }

// Thumbnail returns how a preview thumbnail is captured on save.
func (c SavePolicyControl) Thumbnail() types.ThumbnailSaveOption {
	return c.s.appOptions.Save.Thumbnail
}

// SetThumbnail selects the capture mode, erroring on unimplemented modes.
func (c SavePolicyControl) SetThumbnail(option types.ThumbnailSaveOption) error {
	o := c.s.appOptions.Save
	o.Thumbnail = option
	return c.s.SetSaveOptions(o)
}

// SaveDependents reports whether saving also saves dirty referenced documents.
func (c SavePolicyControl) SaveDependents() bool { return c.s.appOptions.Save.SaveDependents }

// SetSaveDependents toggles dependent saving. The toggle always applies to the
// running session; persisting it is best-effort (a read-only config dir must
// not break an in-memory policy change — the contract setter carries no error).
func (c SavePolicyControl) SetSaveDependents(save bool) {
	c.s.appOptions.Save.SaveDependents = save
	_ = c.s.saveOptions()
}

// OldVersionsToKeep returns the save-time retention count, 0 when off.
func (c SavePolicyControl) OldVersionsToKeep() int { return c.s.appOptions.Save.OldVersionsToKeep }

// SetOldVersionsToKeep sets the retention count, erroring on negatives.
func (c SavePolicyControl) SetOldVersionsToKeep(count int) error {
	o := c.s.appOptions.Save
	o.OldVersionsToKeep = count
	return c.s.SetSaveOptions(o)
}
