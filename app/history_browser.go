// SPDX-License-Identifier: GPL-2.0-only

package app

// History Browser window state (Edit ▸ History Browser): a panel that complements the
// undo/redo buttons for long histories and multi-document assemblies — it shows each open
// document's whole timeline since it opened (with save checkpoints) and jumps the cursor to any
// point. The window reads timelines through DocumentHistoryView and navigates through
// JumpDocumentTo; this file only owns the open/closed state.

// OpenHistoryBrowser opens the History Browser window.
func (s *Session) OpenHistoryBrowser() { s.historyBrowserOpen = true }

// CloseHistoryBrowser closes the History Browser window.
func (s *Session) CloseHistoryBrowser() { s.historyBrowserOpen = false }

// ToggleHistoryBrowser flips the History Browser window open/closed — the Edit menu item.
func (s *Session) ToggleHistoryBrowser() { s.historyBrowserOpen = !s.historyBrowserOpen }

// HistoryBrowserOpen reports whether the History Browser window is open.
func (s *Session) HistoryBrowserOpen() bool { return s.historyBrowserOpen }
