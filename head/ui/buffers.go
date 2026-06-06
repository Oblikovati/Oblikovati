// SPDX-License-Identifier: GPL-2.0-only

package ui

import "bytes"

// bufString reads the NUL-terminated text ImGui wrote into an InputText buffer.
func bufString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// clearBuf zeroes an InputText buffer so the field empties after a successful action.
func clearBuf(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
