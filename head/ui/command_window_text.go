// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/app/cmdline"

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
