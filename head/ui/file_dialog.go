// SPDX-License-Identifier: GPL-2.0-only

package ui

import "bytes"

// fileDialogMode is which file operation the path modal will perform on confirm. The
// zero value (dialogClosed) means the modal is not showing.
type fileDialogMode int

const (
	dialogClosed  fileDialogMode = iota
	dialogOpen                   // File ▸ Open
	dialogSaveAs                 // File ▸ Save As
	dialogLoadHDR                // View ▸ Load HDR (environment image)
)

// pathBufferLen bounds the path the user can type. ImGui edits the buffer in place,
// so it is a fixed array, not a growable slice; 512 covers any realistic file path.
const pathBufferLen = 512

// fileDialog is the head's path-entry modal state: which operation is pending and the
// path being typed. It is deliberately free of any native/cgo dependency so the
// open→type→confirm/cancel transitions are unit-testable; the draw method that renders
// it lives in the cgo file file_dialog_draw.go (the navigate.go split, ADR-0014).
type fileDialog struct {
	mode fileDialogMode
	path [pathBufferLen]byte
}

// fileAction is what a confirmed dialog asks the chrome to perform. Kind ==
// dialogClosed means "nothing this frame" (a cancel, or OK on an empty path).
type fileAction struct {
	Kind fileDialogMode
	Path string
}

// openFor arms the dialog for mode with an empty path. Calling it again re-arms the
// dialog (e.g. switching from Open to Save As) and clears any prior text.
func (d *fileDialog) openFor(mode fileDialogMode) {
	d.mode = mode
	d.path = [pathBufferLen]byte{}
}

// isOpen reports whether the modal should render this frame.
func (d *fileDialog) isOpen() bool { return d.mode != dialogClosed }

// title is the window heading for the current mode.
func (d *fileDialog) title() string {
	switch d.mode {
	case dialogSaveAs:
		return "Save As"
	case dialogLoadHDR:
		return "Load HDR"
	default:
		return "Open"
	}
}

// text returns the NUL-trimmed path the user has typed into the buffer.
func (d *fileDialog) text() string {
	if i := bytes.IndexByte(d.path[:], 0); i >= 0 {
		return string(d.path[:i])
	}
	return string(d.path[:])
}

// confirm dismisses the dialog and returns the action for its mode and typed path. A
// blank path yields a dialogClosed action so the caller never acts on "".
func (d *fileDialog) confirm() fileAction {
	path := d.text()
	mode := d.mode
	d.cancel()
	if path == "" {
		return fileAction{Kind: dialogClosed}
	}
	return fileAction{Kind: mode, Path: path}
}

// cancel dismisses the dialog without an action and clears the typed path.
func (d *fileDialog) cancel() {
	d.mode = dialogClosed
	d.path = [pathBufferLen]byte{}
}
