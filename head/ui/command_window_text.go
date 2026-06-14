// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app/cmdline"
)

// selectionText renders the selection count as a short status (e.g. "1 selected") — shown in
// the command window's control row (it moved here when the status bar was removed, M26).
func selectionText(n int) string {
	return strconv.Itoa(n) + " selected"
}

// Pure presentation helpers for the Command Window (M26 F04), split out with no cgo build
// tag so they are unit-tested without the native layer (mirrors update_text.go).

// severityColor maps a scrollback line's severity to its RGBA text colour: echoed input
// reads blue (the user's own typing), prompts amber, warnings orange, errors red, and
// ordinary output the default light grey.
func severityColor(sev cmdline.Severity) [4]float32 {
	switch sev {
	case cmdline.Echo:
		return [4]float32{0.55, 0.78, 1.0, 1}
	case cmdline.Prompt:
		return [4]float32{0.92, 0.85, 0.42, 1}
	case cmdline.Warning:
		return [4]float32{1.0, 0.72, 0.2, 1}
	case cmdline.Error:
		return [4]float32{1.0, 0.42, 0.42, 1}
	default:
		return [4]float32{0.82, 0.82, 0.82, 1}
	}
}

// completionColor is the text colour for an autocomplete suggestion: the highlighted one reads
// in the accent blue, the rest are dimmed.
func completionColor(selected bool) [4]float32 {
	if selected {
		return [4]float32{0.55, 0.78, 1.0, 1}
	}
	return [4]float32{0.6, 0.6, 0.6, 1}
}

// completionLabel prefixes an autocomplete suggestion with a marker so the highlighted one
// (Up/Down, Tab-completes) stands out even without colour.
func completionLabel(word string, selected bool) string {
	if selected {
		return "▸ " + word
	}
	return "  " + word
}
