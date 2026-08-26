// SPDX-License-Identifier: GPL-2.0-only

package ui

import "bytes"

// bufString reads the NUL-terminated text ImGui wrote into an InputText buffer.
func bufString(b []byte) string {
	if before, _, ok := bytes.Cut(b, []byte{0}); ok {
		return string(before)
	}
	return string(b)
}

// clearBuf zeroes an InputText buffer so the field empties after a successful action.
func clearBuf(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// setBuf replaces an InputText buffer's contents with s (NUL-terminated, truncated to fit) —
// used to drop a clicked autocomplete suggestion into the command input.
func setBuf(b []byte, s string) {
	clearBuf(b)
	n := copy(b, s)
	if n < len(b) {
		b[n] = 0
	}
}
